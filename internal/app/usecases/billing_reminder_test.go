package usecases

import (
	"VpnBot/internal/domain/model"
	"VpnBot/internal/domain/ports/service"
	"VpnBot/internal/domain/repository"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

type fakeMarzban struct {
	statusByUser   map[string]string
	setStatusErr   error
	setStatusCalls int
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
	f.setStatusCalls++
	if f.setStatusErr != nil {
		return f.setStatusErr
	}
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
	db.SetMaxOpenConns(1)

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

func TestProcessBillingReminders_MarzbanMissingDoesNotBlockOthers(t *testing.T) {
	uc, userRepo, _, marz, db := setupBillingUsecase(t)
	defer func() { _ = db.Close() }()

	marz.setStatusErr = service.ErrUserNotFound

	const overdueUID int64 = 401
	const remindUID int64 = 402
	if err := userRepo.Insert("orphan", overdueUID, time.Now()); err != nil {
		t.Fatalf("insert orphan: %v", err)
	}
	if err := userRepo.Insert("payer", remindUID, time.Now()); err != nil {
		t.Fatalf("insert payer: %v", err)
	}

	now := time.Date(2026, 4, 10, 18, 0, 0, 0, time.UTC)
	if err := userRepo.UpdatePaymentDate("orphan", now.AddDate(0, 0, -2)); err != nil {
		t.Fatalf("set orphan overdue: %v", err)
	}
	if err := userRepo.UpdatePaymentDate("payer", now.AddDate(0, 0, 2)); err != nil {
		t.Fatalf("set payer due in 2 days: %v", err)
	}

	_, outs, err := uc.ProcessBillingReminders(now)
	if err != nil {
		t.Fatalf("process reminders: %v", err)
	}

	var revoked, remind2d bool
	for _, o := range outs {
		switch o.Kind {
		case BillingNewlyRevoked:
			if o.User.Uid == overdueUID {
				revoked = true
			}
		case BillingRemind2d:
			if o.User.Uid == remindUID {
				remind2d = true
			}
		}
	}
	if !revoked {
		t.Fatalf("expected revoked notice for overdue user, got %+v", outs)
	}
	if !remind2d {
		t.Fatalf("expected 2d reminder for payer, got %+v", outs)
	}

	ok, err := userRepo.CheckPaymentRevoked(overdueUID)
	if err != nil || !ok {
		t.Fatalf("expected orphan revoked in DB, revoked=%v err=%v", ok, err)
	}
}

func TestProcessBillingReminders_TransientDisableFailureRetries(t *testing.T) {
	uc, userRepo, _, marz, db := setupBillingUsecase(t)
	defer func() { _ = db.Close() }()

	const overdueUID int64 = 501
	const remindUID int64 = 502
	now := time.Date(2026, 4, 10, 18, 0, 0, 0, time.UTC)
	if err := userRepo.Insert("overdue", overdueUID, now.AddDate(0, 0, -1)); err != nil {
		t.Fatal(err)
	}
	if err := userRepo.Insert("payer", remindUID, now.AddDate(0, 0, 2)); err != nil {
		t.Fatal(err)
	}
	marz.setStatusErr = errors.New("temporary Marzban outage")
	_, outs, err := uc.ProcessBillingReminders(now)
	if err == nil {
		t.Fatal("expected a retryable disable error")
	}
	if len(outs) != 1 || outs[0].User.Uid != remindUID || outs[0].Kind != BillingRemind2d {
		t.Fatalf("other users should still receive reminders, got %+v", outs)
	}
	if revoked, err := userRepo.CheckPaymentRevoked(overdueUID); err != nil || revoked {
		t.Fatalf("failed disable must remain retryable, revoked=%v err=%v", revoked, err)
	}

	marz.setStatusErr = nil
	if _, _, err := uc.ProcessBillingReminders(now); err != nil {
		t.Fatal(err)
	}
	if marz.setStatusCalls != 2 || marz.statusByUser["overdue"] != "disabled" {
		t.Fatalf("expected successful second disable attempt, calls=%d statuses=%v", marz.setStatusCalls, marz.statusByUser)
	}
	if revoked, err := userRepo.CheckPaymentRevoked(overdueUID); err != nil || !revoked {
		t.Fatalf("successful disable must revoke access, revoked=%v err=%v", revoked, err)
	}
}

func TestProcessBillingReminders_RetriesUnsentRevokedNotice(t *testing.T) {
	uc, userRepo, _, marz, db := setupBillingUsecase(t)
	defer func() { _ = db.Close() }()
	const uid int64 = 503
	now := time.Date(2026, 4, 10, 18, 0, 0, 0, time.UTC)
	if err := userRepo.Insert("revoked-notice", uid, now.AddDate(0, 0, -1)); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		_, outs, err := uc.ProcessBillingReminders(now)
		if err != nil || len(outs) != 1 || outs[0].Kind != BillingNewlyRevoked {
			t.Fatalf("unsent revoked notice must be retried: outs=%v err=%v", outs, err)
		}
	}
	if marz.setStatusCalls != 1 {
		t.Fatalf("notice retry should not disable again, calls=%d", marz.setStatusCalls)
	}
	date, err := userRepo.GetPaymentDateByUserID(uid)
	if err != nil {
		t.Fatal(err)
	}
	if err := uc.CommitBillingReminderStage(uid, BillingNewlyRevoked, date); err != nil {
		t.Fatal(err)
	}
	if _, outs, err := uc.ProcessBillingReminders(now); err != nil || len(outs) != 0 {
		t.Fatalf("sent revoked notice should not repeat: outs=%v err=%v", outs, err)
	}
}

func TestCommitBillingReminderStage_DoesNotOverwriteNewPaymentPeriod(t *testing.T) {
	uc, userRepo, _, _, db := setupBillingUsecase(t)
	defer func() { _ = db.Close() }()
	const uid int64 = 504
	now := time.Date(2026, 4, 10, 18, 0, 0, 0, time.UTC)
	if err := userRepo.Insert("new-period", uid, now.AddDate(0, 0, 2)); err != nil {
		t.Fatal(err)
	}
	_, outs, err := uc.ProcessBillingReminders(now)
	if err != nil || len(outs) != 1 {
		t.Fatalf("expected reminder: outs=%v err=%v", outs, err)
	}
	newDate := now.AddDate(0, 1, 2)
	if err := userRepo.UpdatePaymentDate("new-period", newDate); err != nil {
		t.Fatal(err)
	}
	if err := uc.CommitBillingReminderStage(uid, outs[0].Kind, outs[0].User.PaymentDate); err != nil {
		t.Fatal(err)
	}
	var stage int
	if err := db.QueryRow(`select payment_reminder_stage from users where user_id = ?`, uid).Scan(&stage); err != nil || stage != 0 {
		t.Fatalf("stale reminder altered the new payment period: stage=%d err=%v", stage, err)
	}
	if _, outs, err := uc.ProcessBillingReminders(newDate.AddDate(0, 0, -2)); err != nil || len(outs) != 1 || outs[0].Kind != BillingRemind2d {
		t.Fatalf("new period lost its reminder: outs=%v err=%v", outs, err)
	}
}

func TestNextPaymentDatePreservesPaidDaysAndClampsMonthEnd(t *testing.T) {
	cases := []struct {
		name           string
		now, old, want time.Time
	}{
		{"early", time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC), time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC), time.Date(2026, 5, 12, 0, 0, 0, 0, time.UTC)},
		{"overdue", time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC), time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)},
		{"month end", time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC), time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)},
		{"leap year", time.Date(2028, 1, 31, 12, 0, 0, 0, time.UTC), time.Time{}, time.Date(2028, 2, 29, 0, 0, 0, 0, time.UTC)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := nextPaymentDateAfterConfirm(&tc.old, tc.now)
			if !got.Equal(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAdminBillingChangesRestoreAccessAndRetryOutages(t *testing.T) {
	cases := []struct {
		name   string
		change func(*UserUsecase, time.Time, time.Time) error
	}{
		{"setfree", func(uc *UserUsecase, now, old time.Time) error { return uc.UpdateTypePayment("restore", true) }},
		{"setdate", func(uc *UserUsecase, now, old time.Time) error {
			return uc.UpdatePaymentDate("restore", now.AddDate(0, 0, 7))
		}},
		{"compensation", func(uc *UserUsecase, now, old time.Time) error { return uc.AddCompensationDays(7) }},
		{"override", func(uc *UserUsecase, now, old time.Time) error { return uc.OverrideDate(old, now.AddDate(0, 0, 7)) }},
		{"normalizepay", func(uc *UserUsecase, now, old time.Time) error {
			_, err := uc.NormalizePaymentDatesToCurrentMonth(now)
			return err
		}},
	}
	for _, tc := range cases {
		for _, failFirst := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/outage=%t", tc.name, failFirst), func(t *testing.T) {
				uc, userRepo, _, marz, db := setupBillingUsecase(t)
				defer func() { _ = db.Close() }()
				const uid int64 = 801
				now := time.Now()
				old := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -1)
				if err := userRepo.Insert("restore", uid, old); err != nil {
					t.Fatal(err)
				}
				if err := userRepo.SetPaymentAccessRevoked(uid, true); err != nil {
					t.Fatal(err)
				}
				if failFirst {
					marz.setStatusErr = errors.New("temporary outage")
				}
				err := tc.change(uc, now, old)
				if failFirst {
					if err == nil {
						t.Fatal("administrative change must report partial failure")
					}
					if revoked, err := userRepo.CheckPaymentRevoked(uid); err != nil || !revoked {
						t.Fatalf("access must remain revoked until remote activation: revoked=%v err=%v", revoked, err)
					}
					marz.setStatusErr = nil
					if _, _, err := uc.ProcessBillingReminders(now); err != nil {
						t.Fatalf("retry restoration: %v", err)
					}
				} else if err != nil {
					t.Fatal(err)
				}
				if revoked, err := userRepo.CheckPaymentRevoked(uid); err != nil || revoked {
					t.Fatalf("access not restored after successful activation: revoked=%v err=%v", revoked, err)
				}
				if marz.statusByUser["restore"] != "active" {
					t.Fatalf("remote user not active: %v", marz.statusByUser)
				}
			})
		}
	}
}

func TestConfirmExtensionAfterPayment_ActivationFailureKeepsPaymentState(t *testing.T) {
	uc, userRepo, paymentRepo, marz, db := setupBillingUsecase(t)
	defer func() { _ = db.Close() }()

	const uid int64 = 601
	oldDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	if err := userRepo.Insert("retry", uid, oldDate); err != nil {
		t.Fatal(err)
	}
	if err := userRepo.SetPaymentAccessRevoked(uid, true); err != nil {
		t.Fatal(err)
	}
	if err := userRepo.SetAwaitingPaymentScreenshot(uid, true); err != nil {
		t.Fatal(err)
	}
	marz.setStatusErr = errors.New("activation unavailable")
	if err := uc.ConfirmExtensionAfterPayment(uid, 901); err == nil {
		t.Fatal("expected activation error")
	}
	date, err := userRepo.GetPaymentDateByUserID(uid)
	if err != nil || date == nil || !date.Equal(oldDate) {
		t.Fatalf("date changed on failed activation: date=%v err=%v", date, err)
	}
	if revoked, err := userRepo.CheckPaymentRevoked(uid); err != nil || !revoked {
		t.Fatalf("revocation cleared on failed activation: revoked=%v err=%v", revoked, err)
	}
	if awaiting, err := userRepo.GetAwaitingPaymentScreenshot(uid); err != nil || !awaiting {
		t.Fatalf("awaiting flag cleared on failed activation: awaiting=%v err=%v", awaiting, err)
	}
	if confirmed, err := paymentRepo.HasConfirmedProof(uid, 901); err != nil || confirmed {
		t.Fatalf("failed activation logged payment: confirmed=%v err=%v", confirmed, err)
	}
	marz.setStatusErr = nil
	if err := uc.ConfirmExtensionAfterPayment(uid, 901); err != nil {
		t.Fatalf("retry: %v", err)
	}
}

func TestConfirmExtensionAfterPayment_ReportFailureRollsBackAndDuplicateIsIgnored(t *testing.T) {
	uc, userRepo, paymentRepo, marz, db := setupBillingUsecase(t)
	defer func() { _ = db.Close() }()

	const uid int64 = 701
	const proofID = 902
	oldDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	if err := userRepo.Insert("atomic", uid, oldDate); err != nil {
		t.Fatal(err)
	}
	if err := userRepo.SetPaymentAccessRevoked(uid, true); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`create trigger reject_payment before insert on payment_confirms begin select raise(ABORT, 'report unavailable'); end`); err != nil {
		t.Fatal(err)
	}
	if err := uc.ConfirmExtensionAfterPayment(uid, proofID); err == nil {
		t.Fatal("expected report error")
	}
	date, err := userRepo.GetPaymentDateByUserID(uid)
	if err != nil || date == nil || !date.Equal(oldDate) {
		t.Fatalf("date update must roll back with report: date=%v err=%v", date, err)
	}
	if revoked, err := userRepo.CheckPaymentRevoked(uid); err != nil || !revoked {
		t.Fatalf("revocation must roll back with report: revoked=%v err=%v", revoked, err)
	}
	if _, err := db.Exec(`drop trigger reject_payment`); err != nil {
		t.Fatal(err)
	}
	if err := uc.ConfirmExtensionAfterPayment(uid, proofID); err != nil {
		t.Fatalf("retry after failed commit: %v", err)
	}
	if err := uc.ConfirmExtensionAfterPayment(uid, proofID); !errors.Is(err, ErrPaymentAlreadyConfirmed) {
		t.Fatalf("expected already-confirmed error, got %v", err)
	}
	if marz.setStatusCalls != 2 {
		t.Fatalf("duplicate should not activate again, got %d calls", marz.setStatusCalls)
	}
	entries, err := paymentRepo.Monthly(time.Now().UTC().Format("2006-01"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("payment report should contain exactly one event: entries=%v err=%v", entries, err)
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
