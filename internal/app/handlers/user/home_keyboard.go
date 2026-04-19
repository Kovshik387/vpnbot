package user

import (
	"VpnBot/internal/app/ui"
	"VpnBot/internal/app/usecases"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func MainMenuKeyboard(userUC *usecases.UserUsecase, uid int64, showRequestKey bool) tgbotapi.InlineKeyboardMarkup {
	if userUC == nil {
		return ui.HomeScreenKeyboard(showRequestKey, false)
	}
	revoked, err := userUC.CheckPaymentRevoked(uid)
	if err == nil && revoked {
		return ui.PaymentRevokedKeyboard()
	}
	paid, _ := userUC.IsPaidSubscription(uid)
	return ui.HomeScreenKeyboard(showRequestKey, paid)
}
