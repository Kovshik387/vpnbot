package user

import (
	"VpnBot/internal/app/usecases"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"math/rand"
)

func Skebob(update tgbotapi.Update, bot *tgbotapi.BotAPI, usUC *usecases.UserUsecase, urls []string) {
	if len(urls) == 0 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Видео пока не настроены 😢")
		_, _ = bot.Send(msg)
		return
	}

	url := urls[rand.Intn(len(urls))]

	formatUrl, err := usUC.Skebob(url)
	if err != nil {
		log.Println(err)
	}

	video := tgbotapi.NewVideo(update.Message.Chat.ID, tgbotapi.FileURL(formatUrl))
	video.Caption = "Скебоб подъехал 🚀"
	_, _ = bot.Send(video)
}
