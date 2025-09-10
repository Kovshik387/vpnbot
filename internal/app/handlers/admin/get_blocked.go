package admin

import (
	"VpnBot/internal/app/usecases"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

func UserBlockedHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase) {
	us, err := userUC.ListBlocked()
	if err != nil {
		log.Println(err)
		return
	}
	separator := "----------------------------------------\n"

	for _, u := range us {
		str := fmt.Sprintf(
			"%s<b>Username:</b> %s\n"+
				"<b>Id:</b> <code>%d</code>\n%s",
			separator,
			u.Username,
			u.Uid,
			separator,
		)

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, str)
		msg.ParseMode = "HTML"

		_, sendErr := bot.Send(msg)
		if sendErr != nil {
			log.Println("Ошибка при отправке списка пользователей:", sendErr)
		}
	}
}
