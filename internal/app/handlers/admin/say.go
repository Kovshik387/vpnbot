package admin

import (
	"VpnBot/internal/app/usecases"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

func SayHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase, text string) {
	users, err := userUC.ListActive()
	if err != nil {
		log.Println(err)
		return
	}

	for _, u := range users {
		if u.Uid != 0 {
			msg := tgbotapi.NewMessage(u.Uid, text)
			msg.DisableNotification = true
			_, sendErr := bot.Send(msg)
			if sendErr != nil {
				log.Printf("Не удалось отправить сообщение пользователю %s (%d): %v",
					u.Username, u.Uid, sendErr)
			}
		}
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Сообщение отправлено всем активным пользователям ✅")
	_, _ = bot.Send(msg)
}
