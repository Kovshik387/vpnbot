package admin

import (
	"VpnBot/internal/app/usecases"
	interfaces "VpnBot/internal/interfaces/http"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

func AddUserHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase, username string) {
	user, err := userUC.AddUser(username)
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка при создании: "+err.Error())
		_, _ = bot.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Пользователь %s успешно создан ✅", username))
	_, _ = bot.Send(msg)

	mb := interfaces.NewMessageBuilder()
	response, err := mb.SendUserInfo(user)
	if err != nil {
		log.Println("Ошибка при формировании сообщения")
	}

	inf := tgbotapi.NewMessage(update.Message.Chat.ID, response)
	inf.ParseMode = "HTML"

	_, sendErr := bot.Send(inf)
	if sendErr != nil {
		log.Println("Ошибка при отправке пользователя:", sendErr)
	}
}
