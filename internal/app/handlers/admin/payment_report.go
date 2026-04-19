package admin

import (
	"VpnBot/internal/app/usecases"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func PaymentMonthlyReportHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase, arg string) {
	text, err := userUC.PaymentMonthlyReportByArg(arg)
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"❌ Неверный формат.\nИспользование: /paystats [YYYY-MM]\nПример: /paystats 2026-04")
		_, _ = bot.Send(msg)
		return
	}
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	_, _ = bot.Send(msg)
}
