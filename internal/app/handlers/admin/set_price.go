package admin

import (
	"VpnBot/internal/app/usecases"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"strconv"
	"strings"
)

func SetPriceHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase, args string) {
	parts := strings.Fields(args)
	if len(parts) < 2 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"❌ Неправильный формат команды.\n"+
				"Используйте: /setprice username цена\n"+
				"Пример: /setprice <user> 1500.00")
		_, _ = bot.Send(msg)
		return
	}

	username := parts[0]
	priceStr := parts[1]

	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil || price < 0 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"❌ Неверная цена. Используйте положительное число.\n"+
				"Пример: 500.00 или 1000")
		_, _ = bot.Send(msg)
		return
	}

	err = userUC.UpdatePrice(username, price)
	if err != nil {
		log.Printf("Ошибка установки цены для %s: %v", username, err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			fmt.Sprintf("❌ Ошибка: %v", err))
		_, _ = bot.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID,
		fmt.Sprintf("✅ Цена для пользователя `%s` установлена: *%.2f*", username, price))
	msg.ParseMode = "Markdown"
	_, _ = bot.Send(msg)
}
