package user

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"math/rand"
)

func Skebob(update tgbotapi.Update, bot *tgbotapi.BotAPI, urls []string) {
	if len(urls) == 0 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Видео пока не настроены 😢")
		_, _ = bot.Send(msg)
		return
	}

	url := urls[rand.Intn(len(urls))]

	video := tgbotapi.NewVideo(update.Message.Chat.ID, tgbotapi.FileURL(url))
	video.Caption = "Скебоб подъехал 🚀"
	_, _ = bot.Send(video)
}
