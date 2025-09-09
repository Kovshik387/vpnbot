package user

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

func Start(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
	btn := tgbotapi.NewInlineKeyboardButtonData("🔑 Запросить ключ", "request_key")
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(btn),
	)

	// Создаем сообщение с кнопкой
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "KovshikVpn\nПривет! Нажмите кнопку, чтобы запросить ключ")
	msg.ReplyMarkup = keyboard

	// Отправляем сообщение
	_, err := bot.Send(msg)
	if err != nil {
		log.Fatal(err)
		return
	}
}
