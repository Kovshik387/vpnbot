package admin

import (
	"VpnBot/internal/app/usecases"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"strconv"
)

func UnblockUserHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase, args string) {
	id, _ := strconv.Atoi(args)

	err := userUC.Block(int64(id), false)
	if err != nil {
		log.Println(err)
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Пользователь `%d` успешно разблокирован ✅", id))
	msg.ParseMode = "Markdown2"
	_, _ = bot.Send(msg)
}
