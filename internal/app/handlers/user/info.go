package user

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

func InfoHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
	text := "Что делать с токеном?\n" +
		"Устройство на котором планируется использовать VPN:\n"

	btnPhone := tgbotapi.NewInlineKeyboardButtonData("📱 Телефон", "info_phone")
	btnPC := tgbotapi.NewInlineKeyboardButtonData("💻 Компьютер", "info_pc")
	btnTV := tgbotapi.NewInlineKeyboardButtonData("📺 Телевизор", "info_tv")
	btnHelp := tgbotapi.NewInlineKeyboardButtonData("📋 Список команд", "help")
	btnSupport := tgbotapi.NewInlineKeyboardButtonURL("🔧 Поддержка", "https://t.me/KovshikGo")

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(btnPhone, btnPC),
		tgbotapi.NewInlineKeyboardRow(btnTV, btnSupport),
		tgbotapi.NewInlineKeyboardRow(btnHelp),
	)

	_, _ = bot.Request(tgbotapi.NewDeleteMessage(
		update.CallbackQuery.Message.Chat.ID,
		update.CallbackQuery.Message.MessageID,
	))

	msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, text)
	msg.ReplyMarkup = keyboard

	_, _ = bot.Send(msg)
}
