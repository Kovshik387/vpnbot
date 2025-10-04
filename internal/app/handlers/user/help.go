package user

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

func HelpHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, adminId int64) {
	text := "Доступные команды:\n" +
		"/start - запустить бота\n" +
		"/help - список команд\n" +
		"/ping - проверить ru сервер\n"

	var (
		chatId int64
		msgId  int
	)

	if update.Message == nil {
		chatId = update.CallbackQuery.Message.Chat.ID
		msgId = update.CallbackQuery.Message.MessageID
	} else {
		chatId = update.Message.Chat.ID
	}

	if msgId != 0 {
		_, _ = bot.Request(tgbotapi.NewDeleteMessage(chatId, msgId))
	}

	if adminId == chatId {
		text += "/adduser - добавить пользователя\n" +
			"/deleteuser - удалить пользователя\n" +
			"/users - просмотреть список пользователей, аргумент -name ищет конкретного\n" +
			"/block - заблокировать пользователя\n" +
			"/unblock - разблокировать пользователя\n" +
			"/blocked - посмотреть заблокированный пользователей\n" +
			"/activity - посмотреть активных пользователей\n" +
			"/say - оповестить пользователей\n" +
			"/count - количество активных пользователей\n" +
			"/status - нагрузка сервера\n" +
			"success anal deploy"
	}

	btnKey := tgbotapi.NewInlineKeyboardButtonData("🔑 Запросить ключ", "request_key")
	btnPing := tgbotapi.NewInlineKeyboardButtonData("🏓 Проверить ping", "ping_server")
	btnSub := tgbotapi.NewInlineKeyboardButtonData("🔓 Проверить подписку", "subscribe")
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(btnKey, btnPing),
		tgbotapi.NewInlineKeyboardRow(btnSub),
	)

	msg := tgbotapi.NewMessage(chatId, text)
	msg.ReplyMarkup = keyboard

	_, err := bot.Send(msg)
	if err != nil {
		log.Fatal(err)
	}
}
