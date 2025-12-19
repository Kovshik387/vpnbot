package admin

import (
	"VpnBot/internal/app/usecases"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"strings"
	"time"
)

func SetPaymentDateHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase, args string) {
	parts := strings.Fields(args)
	if len(parts) < 2 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"❌ Формат: /setdate username дата\n"+
				"Пример: /setdate <user> 2024-12-25")
		_, _ = bot.Send(msg)
		return
	}

	username := parts[0]
	dateStr := strings.Join(parts[1:], " ")

	var paymentDate time.Time
	var err error

	formats := []string{
		"2006-01-02",
		"02.01.2006",
		"02/01/2006",
		"02-01-2006",
		"02.01.06",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		paymentDate, err = time.Parse(format, dateStr)
		if err == nil {
			break
		}
	}

	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"❌ Не удалось распознать дату.\n"+
				"Поддерживаемые форматы:\n"+
				"• 2024-12-25\n"+
				"• 25.12.2024\n"+
				"• 25/12/2024\n"+
				"• 25.12.24")
		_, _ = bot.Send(msg)
		return
	}

	err = userUC.UpdatePaymentDate(username, paymentDate)
	if err != nil {
		log.Printf("Ошибка установки даты для %s: %v", username, err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			fmt.Sprintf("❌ Ошибка: %v", err))
		_, _ = bot.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID,
		fmt.Sprintf("✅ Дата оплаты для `%s` установлена:\n"+
			"📅 *%s* (формат: 02.01.2006)\n"+
			"🗓️ *%s* (формат: YYYY-MM-DD)",
			username,
			paymentDate.Format("02.01.2006"),
			paymentDate.Format("2006-01-02")))
	msg.ParseMode = "Markdown"
	_, _ = bot.Send(msg)
}
