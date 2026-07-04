package admin

import (
	"fmt"
	"log"

	"VpnBot/internal/app/usecases"
	interfaces "VpnBot/internal/interfaces/http"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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
	price, err := userUC.GetPriceByUsername(user.Username)
	if err != nil {
		log.Println("Ошибка при получении цены пользователя:", err)
		price = 0
	}

	response, _, err := mb.SendUserInfo(user, price)
	if err != nil {
		log.Println("Ошибка при формировании сообщения")
	}
	if paidUntil, err := userUC.GetPaymentDateByUsername(user.Username); err == nil && paidUntil != nil {
		response += fmt.Sprintf("\n<b>Оплачен до</b> %s", paidUntil.Format("02.01.2006"))
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, response)
	msg.ParseMode = "HTML"

	_, sendErr := bot.Send(msg)
	if sendErr != nil {
		log.Println("Ошибка при отправке списка пользователей:", sendErr)
	}
}
