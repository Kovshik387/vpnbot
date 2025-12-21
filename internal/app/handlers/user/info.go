package user

import (
	"VpnBot/internal/app/ui"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func InfoHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
	text := "Что делать с токеном?\n" +
		"Устройство на котором планируется использовать VPN:\n"

	btnPhone := tgbotapi.NewInlineKeyboardButtonData("📱 Телефон", "info_phone")
	btnPC := tgbotapi.NewInlineKeyboardButtonData("💻 Компьютер", "info_pc")
	btnTV := tgbotapi.NewInlineKeyboardButtonData("📺 Телевизор", "info_tv")
	btnHelp := tgbotapi.NewInlineKeyboardButtonData("📋 Список команд", "help")
	btnSupport := tgbotapi.NewInlineKeyboardButtonURL("🔧 Поддержка", "https://t.me/KovshikGo")
	btnWiki := tgbotapi.NewInlineKeyboardButtonURL("Wiki", "https://teletype.in/@yrulewet/8Qp0x8heiRS")
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(btnPhone, btnPC),
		tgbotapi.NewInlineKeyboardRow(btnTV, btnSupport),
		tgbotapi.NewInlineKeyboardRow(btnHelp, btnWiki),
	)

	var chatID int64

	if update.CallbackQuery != nil && update.CallbackQuery.Message != nil {
		chatID = update.CallbackQuery.Message.Chat.ID

		_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))

		_, _ = bot.Request(tgbotapi.NewDeleteMessage(
			update.CallbackQuery.Message.Chat.ID,
			update.CallbackQuery.Message.MessageID,
		))

	} else if update.Message != nil {
		chatID = update.Message.Chat.ID
	} else {
		return
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	_, _ = bot.Send(msg)

	msg2 := tgbotapi.NewMessage(chatID, "")
	msg2.ReplyMarkup = ui.MainKeyboard()
	_, _ = bot.Send(msg2)
}
