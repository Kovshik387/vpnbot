package usecases

import (
	"VpnBot/internal/domain/model"
	"log"
	"time"
)

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
	loc := now.Location()
	due := time.Date(paymentDate.Year(), paymentDate.Month(), paymentDate.Day(), 0, 0, 0, 0, loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	return int(due.Sub(today).Hours() / 24)
}

func paymentDateOneMonthFrom(now time.Time) time.Time {
	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	return today.AddDate(0, 1, 0)
}

func nextPaymentDateAfterConfirm(_ *time.Time, now time.Time) time.Time {
	return paymentDateOneMonthFrom(now)
}

func (u *UserUsecase) ListPaidBillingUsers() ([]model.TgUserModel, error) {
	return u.userRepository.ListPaidUsersForBilling()
}

func (u *UserUsecase) ProcessBillingReminders(now time.Time) (dueToday []model.TgUserModel, outs []BillingReminderOut, err error) {
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
						// Один пользователь без учётки в Marzban не должен блокировать напоминания остальным.
						log.Printf("marzban disable %q uid=%d: %v", username, usr.Uid, err)
					}
				}
				if err := u.userRepository.SetPaymentAccessRevoked(usr.Uid, true); err != nil {
					return nil, nil, err
				}
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

	return dueToday, outs, nil
}

func ReminderStageForKind(kind BillingReminderKind) (int, bool) {
	switch kind {
	case BillingRemind2d:
		return 1, true
	case BillingRemind1d:
		return 2, true
	case BillingRemindDue:
		return 3, true
	default:
		return 0, false
	}
}

func (u *UserUsecase) CommitBillingReminderStage(userID int64, kind BillingReminderKind) error {
	stage, ok := ReminderStageForKind(kind)
	if !ok {
		return nil
	}
	return u.userRepository.SetPaymentReminderStage(userID, stage)
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

func (u *UserUsecase) ConfirmExtensionAfterPayment(userID int64) error {
	old, err := u.userRepository.GetPaymentDateByUserID(userID)
	if err != nil {
		return err
	}
	next := nextPaymentDateAfterConfirm(old, time.Now())
	if err := u.userRepository.ApplyConfirmedPaymentExtension(userID, next); err != nil {
		return err
	}

	username, err := u.userRepository.GetUsernameByUserID(userID)
	if err != nil {
		return err
	}
	if username == "" {
		return nil
	}
	if err := u.marzbanClient.SetUserStatus(username, "active"); err != nil {
		return err
	}
	return u.LogConfirmedPayment(userID)
}

func (u *UserUsecase) RejectPaymentProof(userID int64) error {
	return u.userRepository.SetAwaitingPaymentScreenshot(userID, false)
}
