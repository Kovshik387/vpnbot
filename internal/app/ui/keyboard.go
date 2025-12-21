package ui

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

const BtnCheckSubscription = "✅ Проверить подписку"

func MainKeyboard() tgbotapi.ReplyKeyboardMarkup {
	kb := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(BtnCheckSubscription),
		),
	)

	kb.ResizeKeyboard = true
	kb.OneTimeKeyboard = false
	return kb
}
