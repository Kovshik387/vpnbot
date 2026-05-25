package usecases

import (
	"VpnBot/internal/domain/model"
	"VpnBot/internal/domain/repository"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

type fakeMarzban struct {
	statusByUser map[string]string
}

func newFakeMarzban() *fakeMarzban {
	return &fakeMarzban{statusByUser: make(map[string]string)}
}

func (f *fakeMarzban) Login() error { return nil }
func (f *fakeMarzban) GetUsers() (model.UsersResponse, error) {
	return model.UsersResponse{}, nil
}
func (f *fakeMarzban) GetUser(username string) (model.User, error) {
	return model.User{}, nil
}
func (f *fakeMarzban) AddUser(username string) (model.User, error) {
	return model.User{}, nil
}
func (f *fakeMarzban) Delete(username string) error { return nil }
func (f *fakeMarzban) SetUserStatus(username string, status string) error {
	f.statusByUser[username] = status
	return nil
}

type fakeYandex struct{}

func (f *fakeYandex) GetYandexDirectLink(publicLink string) (string, error) { return publicLink, nil }

func setupBillingUsecase(t *testing.T) (*UserUsecase, *repository.UserRepository, *repository.PaymentReportRepository, *fakeMarzban, *sql.DB) {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	if err := userRepo.EnsureSchema(); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}

	pollRepo := repository.NewPollRepository(db)
	if err := pollRepo.Init(); err != nil {
		t.Fatalf("poll init: %v", err)
	}

	paymentRepo := repository.NewPaymentReportRepository(db)
	if err := paymentRepo.Init(); err != nil {
		t.Fatalf("payment report init: %v", err)
	}

	marz := newFakeMarzban()
	uc := NewUserUsecase(marz, &fakeYandex{}, userRepo, pollRepo, paymentRepo)
	return uc, userRepo, paymentRepo, marz, db
}

func TestProcessBillingReminders_Stages(t *testing.T) {
	uc, userRepo, _, _, db := setupBillingUsecase(t)
	defer func() { _ = db.Close() }()

	const uid int64 = 101
	if err := userRepo.Insert("alice", uid, time.Now()); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	base := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	if err := userRepo.UpdatePaymentDate("alice", base.AddDate(0, 0, 2)); err != nil {
		t.Fatalf("set payment date: %v", err)
	}

	dueToday, outs, err := uc.ProcessBillingReminders(base)
	if err != nil {
		t.Fatalf("process reminders d-2: %v", err)
	}
	if len(dueToday) != 0 {
		t.Fatalf("expected no dueToday on d-2, got %d", len(dueToday))
	}
	if len(outs) != 1 || outs[0].Kind != BillingRemind2d {
		t.Fatalf("expected one BillingRemind2d, got %+v", outs)
	}
	if err := uc.CommitBillingReminderStage(uid, BillingRemind2d); err != nil {
		t.Fatalf("commit stage d-2: %v", err)
	}

	_, outs, err = uc.ProcessBillingReminders(base)
	if err != nil {
		t.Fatalf("process reminders d-2 repeat: %v", err)
	}
	if len(outs) != 0 {
		t.Fatalf("expected no duplicate reminder on same day, got %+v", outs)
	}

	_, outs, err = uc.ProcessBillingReminders(base.AddDate(0, 0, 1))
	if err != nil {
		t.Fatalf("process reminders d-1: %v", err)
	}
	if len(outs) != 1 || outs[0].Kind != BillingRemind1d {
		t.Fatalf("expected one BillingRemind1d, got %+v", outs)
	}
	if err := uc.CommitBillingReminderStage(uid, BillingRemind1d); err != nil {
		t.Fatalf("commit stage d-1: %v", err)
	}

	dueToday, outs, err = uc.ProcessBillingReminders(base.AddDate(0, 0, 2))
	if err != nil {
		t.Fatalf("process reminders due day: %v", err)
	}
	if len(dueToday) != 1 || dueToday[0].Uid != uid {
		t.Fatalf("expected dueToday with uid %d, got %+v", uid, dueToday)
	}
	if len(outs) != 1 || outs[0].Kind != BillingRemindDue {
		t.Fatalf("expected one BillingRemindDue, got %+v", outs)
	}
	if err := uc.CommitBillingReminderStage(uid, BillingRemindDue); err != nil {
		t.Fatalf("commit stage due: %v", err)
	}
}

func TestProcessBillingReminders_OverdueDisablesSubscription(t *testing.T) {
	uc, userRepo, _, marz, db := setupBillingUsecase(t)
	defer func() { _ = db.Close() }()

	const uid int64 = 202
	if err := userRepo.Insert("bob", uid, time.Now()); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	now := time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC)
	if err := userRepo.UpdatePaymentDate("bob", now.AddDate(0, 0, -1)); err != nil {
		t.Fatalf("set overdue payment date: %v", err)
	}

	_, outs, err := uc.ProcessBillingReminders(now)
	if err != nil {
		t.Fatalf("process overdue reminders: %v", err)
	}
	if len(outs) != 1 || outs[0].Kind != BillingNewlyRevoked {
		t.Fatalf("expected one BillingNewlyRevoked, got %+v", outs)
	}

	revoked, err := userRepo.CheckPaymentRevoked(uid)
	if err != nil {
		t.Fatalf("check revoked: %v", err)
	}
	if !revoked {
		t.Fatalf("expected payment_access_revoked=true")
	}

	if got := marz.statusByUser["bob"]; got != "disabled" {
		t.Fatalf("expected Marzban status disabled for bob, got %q", got)
	}
}

func TestConfirmExtensionAfterPayment_EnablesSubscriptionAndLogs(t *testing.T) {
	uc, userRepo, paymentRepo, marz, db := setupBillingUsecase(t)
	defer func() { _ = db.Close() }()

	const uid int64 = 303
	if err := userRepo.Insert("carol", uid, time.Now()); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	now := time.Now()
	oldDue := now.AddDate(0, 0, -3)
	if err := userRepo.UpdatePaymentDate("carol", oldDue); err != nil {
		t.Fatalf("set old due date: %v", err)
	}
	if err := userRepo.SetPaymentAccessRevoked(uid, true); err != nil {
		t.Fatalf("set revoked: %v", err)
	}

	if err := uc.ConfirmExtensionAfterPayment(uid); err != nil {
		t.Fatalf("confirm extension: %v", err)
	}

	if got := marz.statusByUser["carol"]; got != "active" {
		t.Fatalf("expected Marzban status active for carol, got %q", got)
	}

	revoked, err := userRepo.CheckPaymentRevoked(uid)
	if err != nil {
		t.Fatalf("check revoked after confirm: %v", err)
	}
	if revoked {
		t.Fatalf("expected payment_access_revoked=false after confirm")
	}

	paidUntil, err := userRepo.GetPaymentDateByUserID(uid)
	if err != nil {
		t.Fatalf("get payment date after confirm: %v", err)
	}
	if paidUntil == nil {
		t.Fatalf("expected payment date after confirm")
	}
	expected := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 1, 0)
	got := time.Date(paidUntil.Year(), paidUntil.Month(), paidUntil.Day(), 0, 0, 0, 0, now.Location())
	if !got.Equal(expected) {
		t.Fatalf("expected next payment date %s, got %s", expected.Format("2006-01-02"), got.Format("2006-01-02"))
	}

	month := time.Now().Format("2006-01")
	entries, err := paymentRepo.Monthly(month)
	if err != nil {
		t.Fatalf("payment report monthly: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one payment report entry, got %d", len(entries))
	}
	if entries[0].UserID != uid || entries[0].Username != "carol" {
		t.Fatalf("unexpected payment entry: %+v", entries[0])
	}
}
