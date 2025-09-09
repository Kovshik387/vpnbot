package admin

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func DenyHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
	parts := strings.Split(update.CallbackQuery.Data, ":")
	if len(parts) != 2 {
		return
	}

	username := parts[1]

	callback := tgbotapi.NewCallback(update.CallbackQuery.ID, "❌ Отклонено для @"+username)
	_, _ = bot.Request(callback)
}
