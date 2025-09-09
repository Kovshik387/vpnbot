package router

import (
	"VpnBot/config"
	"VpnBot/internal/app/handlers/user"
	"VpnBot/internal/app/usecases"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CallbackHandler func(update tgbotapi.Update, bot *tgbotapi.BotAPI)

func NewCallbackRouter(userUC *usecases.UserUsecase, config *config.Config) map[string]CallbackHandler {
	return map[string]CallbackHandler{
		"approve": func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
			msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "✅ Подтверждено")
			_, _ = bot.Send(msg)
		},
		"deny": func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
			// TODO: Отклонить подписку + таймаут въебать ему
			msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "❌ Отклонено")
			_, _ = bot.Send(msg)
		},
		"request": func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
			btn := tgbotapi.NewInlineKeyboardButtonData("🔑 Запросить ключ", "request_key")
			keyboard := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(btn),
			)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Нажмите кнопку, чтобы запросить ключ")
			msg.ReplyMarkup = keyboard
			_, _ = bot.Send(msg)
		},
		// TODO нихуя не работает
		"ping": func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
			user.Ping(update, bot, config.RussianUrl)
		},
	}
}
