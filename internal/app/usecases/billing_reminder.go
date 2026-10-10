package usecases

import (
	"VpnBot/internal/domain/model"
	"VpnBot/internal/domain/ports/service"
	"errors"
	"fmt"
	"time"
)

var ErrPaymentAlreadyConfirmed = errors.New("оплата уже подтверждена")

type BillingReminderKind int

const (
	BillingRemind2d BillingReminderKind = iota
	BillingRemind1d
	BillingRemindDue
	BillingNewlyRevoked
)

type BillingReminderOut struct {
	User model.TgUserModel
	Kind BillingReminderKind
}

func calendarDaysUntilDue(paymentDate *time.Time, now time.Time) int {
	if paymentDate == nil {
		return 99999
	}
	due := time.Date(paymentDate.Year(), paymentDate.Month(), paymentDate.Day(), 0, 0, 0, 0, time.UTC)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return int(due.Sub(today).Hours() / 24)
}

func paymentDateOneMonthFrom(now time.Time) time.Time {
	loc := now.Location()
	nextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, loc)
	lastDay := time.Date(nextMonth.Year(), nextMonth.Month()+1, 0, 0, 0, 0, 0, loc).Day()
	day := now.Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(nextMonth.Year(), nextMonth.Month(), day, 0, 0, 0, 0, loc)
}

func nextPaymentDateAfterConfirm(oldDate *time.Time, now time.Time) time.Time {
	base := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if oldDate != nil {
		old := time.Date(oldDate.Year(), oldDate.Month(), oldDate.Day(), 0, 0, 0, 0, now.Location())
		if old.After(base) {
			base = old
		}
	}
	return paymentDateOneMonthFrom(base)
}

// restorePaymentAccess оставляет revoked=true при сбое, чтобы плановая проверка
// могла повторить активацию после административного изменения даты или тарифа.
func (u *UserUsecase) restorePaymentAccess(now time.Time, onlyUsername string) error {
	users, err := u.userRepository.ListUsersForBilling()
	if err != nil {
		return err
	}
	var failures []error
	for _, usr := range users {
		if !usr.PaymentAccessRevoked || (onlyUsername != "" && usr.Username != onlyUsername) {
			continue
		}
		if !usr.IsFree && (usr.PaymentDate == nil || calendarDaysUntilDue(usr.PaymentDate, now) < 0) {
			continue
		}
		if err := u.marzbanClient.SetUserStatus(usr.Username, "active"); err != nil {
			failures = append(failures, fmt.Errorf("срок/тариф сохранён, но доступ %s пока не восстановлен: %w", usr.Username, err))
			continue
		}
		if err := u.userRepository.SetPaymentAccessRevoked(usr.Uid, false); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func (u *UserUsecase) ListPaidBillingUsers() ([]model.TgUserModel, error) {
	return u.userRepository.ListPaidUsersForBilling()
}

func (u *UserUsecase) ProcessBillingReminders(now time.Time) (dueToday []model.TgUserModel, outs []BillingReminderOut, err error) {
	u.billingMu.Lock()
	defer u.billingMu.Unlock()

	restoreErr := u.restorePaymentAccess(now, "")
	failures := []error{restoreErr}
	users, err := u.userRepository.ListPaidUsersForBilling()
	if err != nil {
		return nil, nil, err
	}

	for _, usr := range users {
		d := calendarDaysUntilDue(usr.PaymentDate, now)

		if d == 0 && usr.PaymentDate != nil {
			dueToday = append(dueToday, usr)
		}

		if d < 0 {
			if !usr.PaymentAccessRevoked {
				username, err := u.userRepository.GetUsernameByUserID(usr.Uid)
				if err != nil {
					return nil, nil, err
				}
				if username != "" {
					if err := u.marzbanClient.SetUserStatus(username, "disabled"); err != nil {
						// Отсутствующая учётка уже не предоставляет VPN; временный сбой повторим при следующей проверке.
						if !errors.Is(err, service.ErrUserNotFound) {
							failures = append(failures, fmt.Errorf("отключение %s отложено: %w", username, err))
							continue
						}
					}
				}
				if err := u.userRepository.SetPaymentAccessRevoked(usr.Uid, true); err != nil {
					return nil, nil, err
				}
			}
			if usr.PaymentReminderStage < 4 {
				outs = append(outs, BillingReminderOut{User: usr, Kind: BillingNewlyRevoked})
			}
			continue
		}

		if usr.PaymentAccessRevoked {
			continue
		}

		switch {
		case d == 2 && usr.PaymentReminderStage < 1:
			outs = append(outs, BillingReminderOut{User: usr, Kind: BillingRemind2d})
		case d == 1 && usr.PaymentReminderStage < 2:
			outs = append(outs, BillingReminderOut{User: usr, Kind: BillingRemind1d})
		case d == 0 && usr.PaymentReminderStage < 3:
			outs = append(outs, BillingReminderOut{User: usr, Kind: BillingRemindDue})
		}
	}

	return dueToday, outs, errors.Join(failures...)
}

func ReminderStageForKind(kind BillingReminderKind) (int, bool) {
	switch kind {
	case BillingRemind2d:
		return 1, true
	case BillingRemind1d:
		return 2, true
	case BillingRemindDue:
		return 3, true
	case BillingNewlyRevoked:
		return 4, true
	default:
		return 0, false
	}
}

func (u *UserUsecase) CommitBillingReminderStage(userID int64, kind BillingReminderKind, expectedPaymentDates ...*time.Time) error {
	stage, ok := ReminderStageForKind(kind)
	if !ok {
		return nil
	}
	return u.userRepository.SetPaymentReminderStage(userID, stage, expectedPaymentDates...)
}

func (u *UserUsecase) CheckPaymentRevoked(userID int64) (bool, error) {
	return u.userRepository.CheckPaymentRevoked(userID)
}

func (u *UserUsecase) IsPaidSubscription(userID int64) (bool, error) {
	return u.userRepository.IsPaidSubscription(userID)
}

func (u *UserUsecase) SetAwaitingPaymentScreenshot(userID int64, v bool) error {
	return u.userRepository.SetAwaitingPaymentScreenshot(userID, v)
}

func (u *UserUsecase) GetAwaitingPaymentScreenshot(userID int64) (bool, error) {
	return u.userRepository.GetAwaitingPaymentScreenshot(userID)
}

func (u *UserUsecase) ConfirmExtensionAfterPayment(userID int64, proofMessageIDs ...int) error {
	u.billingMu.Lock()
	defer u.billingMu.Unlock()

	proofMessageID := 0
	if len(proofMessageIDs) > 0 {
		proofMessageID = proofMessageIDs[0]
	}
	if proofMessageID > 0 && u.paymentReportRepo != nil {
		confirmed, err := u.paymentReportRepo.HasConfirmedProof(userID, proofMessageID)
		if err != nil {
			return err
		}
		if confirmed {
			return ErrPaymentAlreadyConfirmed
		}
	}
	username, err := u.userRepository.GetUsernameByUserID(userID)
	if err != nil {
		return err
	}
	if username == "" {
		return fmt.Errorf("пользователь %d не найден", userID)
	}
	old, err := u.userRepository.GetPaymentDateByUserID(userID)
	if err != nil {
		return err
	}
	if err := u.marzbanClient.SetUserStatus(username, "active"); err != nil {
		return err
	}
	// Пока Marzban не подтвердил активацию, дата и флаги оплаты остаются прежними.
	next := nextPaymentDateAfterConfirm(old, time.Now())
	if u.paymentReportRepo == nil {
		return u.userRepository.ApplyConfirmedPaymentExtension(userID, next)
	}
	confirmed, err := u.paymentReportRepo.ApplyConfirmedExtension(userID, next, proofMessageID)
	if err != nil {
		return err
	}
	if !confirmed {
		return ErrPaymentAlreadyConfirmed
	}
	return nil
}

func (u *UserUsecase) RejectPaymentProof(userID int64) error {
	return u.userRepository.SetAwaitingPaymentScreenshot(userID, false)
}
