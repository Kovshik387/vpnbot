package user

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

func StartHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, adminId int64) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "KovshikVpn\nПривет!")
	_, err := bot.Send(msg)

	HelpHandler(update, bot, adminId)

	if err != nil {
		log.Fatal(err)
		return
	}
}
