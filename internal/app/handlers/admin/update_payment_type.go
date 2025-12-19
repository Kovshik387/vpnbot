package admin

import (
	"VpnBot/internal/app/usecases"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"strconv"
	"strings"
)

func UpdateTypePaymentHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase, args string) {
	parts := strings.Fields(args)
	if len(parts) < 2 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"❌ Неправильный формат.\n"+
				"Используйте: /setfree username true/false\n"+
				"Пример: /setfree <user> true")
		_, _ = bot.Send(msg)
		return
	}

	username := parts[0]
	isFreeStr := parts[1]

	isFree, err := strconv.ParseBool(isFreeStr)
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"❌ Неверное значение. Используйте true или false")
		_, _ = bot.Send(msg)
		return
	}

	err = userUC.UpdateTypePayment(username, isFree)
	if err != nil {
		log.Printf("Ошибка установки статуса для %s: %v", username, err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			fmt.Sprintf("❌ Ошибка: %v", err))
		_, _ = bot.Send(msg)
		return
	}

	status := "платным"
	if isFree {
		status = "бесплатным"
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID,
		fmt.Sprintf("✅ Пользователь `%s` теперь стал *%s*", username, status))
	msg.ParseMode = "Markdown"
	_, _ = bot.Send(msg)
}
