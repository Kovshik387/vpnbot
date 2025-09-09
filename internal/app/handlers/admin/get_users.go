package admin

import (
	"VpnBot/internal/app/usecases"
	interfaces "VpnBot/internal/interfaces/http"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

func UserList(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase) {
	users, err := userUC.ListUsers()
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка при получении пользователей: "+err.Error())
		_, _ = bot.Send(msg)
		return
	}

	if len(users.Users) == 0 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Пользователей нет")
		_, _ = bot.Send(msg)
		return
	}
	mb := interfaces.NewMessageBuilder()
	for _, u := range users.Users {

		response, err := mb.SendUserInfo(u)
		if err != nil {
			log.Println("Ошибка при формировании сообщения")
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, response)
		msg.ParseMode = "HTML"

		_, sendErr := bot.Send(msg)
		if sendErr != nil {
			log.Println("Ошибка при отправке списка пользователей:", sendErr)
		}
	}
}
