package user

import (
	"VpnBot/internal/app/usecases"
	interfaces "VpnBot/internal/interfaces/http"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

func GetSubscribeHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase) {
	from := update.CallbackQuery.From
	uid := from.ID
	username := from.UserName
	if username == "" {
		username = fmt.Sprintf("id_%d", uid)
	}

	user, err := userUC.SearchUser(username)
	if err != nil {
		msg := tgbotapi.NewMessage(uid, "Ошибка при получения подписки")
		_, _ = bot.Send(msg)
		return
	}

	mb := interfaces.NewMessageBuilder()

	response, _, err := mb.SendUserInfo(user)
	if err != nil {
		log.Println("Ошибка при формировании сообщения")
	}

	msg := tgbotapi.NewMessage(uid, response)
	msg.ParseMode = "HTML"

	_, sendErr := bot.Send(msg)
	if sendErr != nil {
		log.Println("Ошибка при отправке ", sendErr)
	}
	InfoHandler(update, bot)

}
