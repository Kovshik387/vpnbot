package ui

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

const BtnCheckSubscription = "✅ Проверить подписку"
const BtnShowPanel = "🏠 Показать панель"

func MainKeyboard() tgbotapi.ReplyKeyboardMarkup {
	kb := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(BtnCheckSubscription),
			tgbotapi.NewKeyboardButton(BtnShowPanel),
		),
	)

	kb.ResizeKeyboard = true
	kb.OneTimeKeyboard = false
	return kb
}
