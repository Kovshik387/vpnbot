package admin

import (
	"VpnBot/internal/app/usecases"
	interfaces "VpnBot/internal/interfaces/http"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

func UserActivityCount(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase) {
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

	count := 0
	mb := interfaces.NewMessageBuilder()
	for _, u := range users.Users {
		_, act, err := mb.SendUserInfo(u)
		if err != nil {
			log.Println("Ошибка при формировании сообщения:", err)
			continue
		}

		if act {
			count++
		}
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID,
		fmt.Sprintf("Количетсво подключённых пользователей %d\n", count))
	_, _ = bot.Send(msg)
}
