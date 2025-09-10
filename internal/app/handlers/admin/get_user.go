package admin

import (
	"VpnBot/internal/app/usecases"
	interfaces "VpnBot/internal/interfaces/http"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

func SearchUserHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase, name string) {
	user, err := userUC.SearchUser(name)
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка при получения пользователя: "+err.Error())
		_, _ = bot.Send(msg)
		return
	}

	if user.Username == "" {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Пользователь не найден")
		_, _ = bot.Send(msg)
		return
	}

	mb := interfaces.NewMessageBuilder()

	response, err := mb.SendUserInfo(user)
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
