package admin

import (
	"VpnBot/internal/app/usecases"
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func NormalizePaymentDatesHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase) {
	now := time.Now()
	affected, err := userUC.NormalizePaymentDatesToCurrentMonth(now)
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "❌ Ошибка нормализации дат: "+err.Error())
		_, _ = bot.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID,
		fmt.Sprintf("✅ Даты оплаты унифицированы.\nТекущий период: %s\nОбновлено пользователей: %d",
			now.Format("2006-01"), affected))
	_, _ = bot.Send(msg)
}
