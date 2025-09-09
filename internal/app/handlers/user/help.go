package user

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

func Help(update tgbotapi.Update, bot *tgbotapi.BotAPI, userId int64) {
	text := "Доступные команды:\n" +
		"/start - запустить бота\n" +
		"/help - список команд\n" +
		"/ping - проверить ru сервер\n"

	if userId == update.Message.From.ID {
		text += "/adduser - добавить пользователя\n" +
			"/deleteuser - удалить пользователя\n" +
			"/users - просмотреть список пользователей, аргумент -name ищет конкретного\n"
	}

	// Кнопки для help
	btnKey := tgbotapi.NewInlineKeyboardButtonData("🔑 Запросить ключ", "request_key")
	btnPing := tgbotapi.NewInlineKeyboardButtonData("🏓 Проверить ping", "ping_server")
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(btnKey, btnPing),
	)

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
	msg.ReplyMarkup = keyboard

	_, err := bot.Send(msg)
	if err != nil {
		log.Fatal(err)
	}
}
