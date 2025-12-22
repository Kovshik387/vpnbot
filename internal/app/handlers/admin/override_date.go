package admin

import (
	"VpnBot/internal/app/usecases"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"time"
)

func OverrideDateHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase, oldDateStr, newDateStr string) {
	oldDate, err := time.Parse("2006-01-02", oldDateStr)
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "❌ Неверный формат старой даты")
		_, _ = bot.Send(msg)
		return
	}

	newDate, err := time.Parse("2006-01-02", newDateStr)
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "❌ Неверный формат новой даты")
		_, _ = bot.Send(msg)
		return
	}

	err = userUC.OverrideDate(oldDate, newDate)
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "❌ Ошибка: "+err.Error())
		_, _ = bot.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID,
		fmt.Sprintf("✅ Обновлено записи\nС %s на %s",
			oldDate.Format("02.01.2006"),
			newDate.Format("02.01.2006")))
	_, _ = bot.Send(msg)
}
