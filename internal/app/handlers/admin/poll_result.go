package admin

import (
	"VpnBot/internal/app/usecases"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

func PollResultHandler(
	update tgbotapi.Update,
	bot *tgbotapi.BotAPI,
	userUC *usecases.UserUsecase,
	pollID string,
) {
	if update.Message == nil {
		return
	}

	text, err := userUC.PollResult(pollID)
	if err != nil {
		log.Println("PollResult error:", err)
		_, _ = bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID,
			"Ошибка при получении результатов опроса ❌"))
		return
	}

	if text == "" {
		_, _ = bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID,
			"Опрос с таким ID не найден"))
		return
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
	msg.ParseMode = "Markdown"
	_, _ = bot.Send(msg)
}
