package admin

import (
	"strings"

	"VpnBot/internal/app/usecases"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func ApproveHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase) {
	parts := strings.Split(update.CallbackQuery.Data, ":")
	if len(parts) != 2 {
		return
	}

	username := parts[1]

	_, err := userUC.AddUser(username)
	if err != nil {
		msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Ошибка при выдаче ключа: "+err.Error())
		_, _ = bot.Send(msg)
		return
	}

	msgToUser := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "🎉 Вам выдан ключ")
	msgToUser.ParseMode = "Markdown"
	_, _ = bot.Send(msgToUser)

	callback := tgbotapi.NewCallback(update.CallbackQuery.ID, "✅ Ключ выдан")
	_, _ = bot.Request(callback)
}
