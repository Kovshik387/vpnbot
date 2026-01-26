package admin

import (
	"VpnBot/internal/app/usecases"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func CompensationHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase, daysCount int) {
	err := userUC.AddCompensationDays(daysCount)
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка при добавалении компенсационных дней: "+err.Error())
		_, _ = bot.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Платным пользователям добавлены компенсационных дней ✅"))
	_, _ = bot.Send(msg)
}
