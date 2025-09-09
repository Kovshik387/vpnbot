package user

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func GetKeyHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, adminId int64) {
	user := update.Message.From
	username := user.UserName
	if username == "" {
		username = fmt.Sprintf("id_%d", user.ID) // запасной вариант
	}

	text := fmt.Sprintf("Запрос на ключ:\nПользователь: @%s\nID: %d", username, user.ID)

	approveBtn := tgbotapi.NewInlineKeyboardButtonData("✅ Одобрить", fmt.Sprintf("approve:%s", username))
	denyBtn := tgbotapi.NewInlineKeyboardButtonData("❌ Отклонить", fmt.Sprintf("deny:%s", username))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(approveBtn, denyBtn),
	)

	msg := tgbotapi.NewMessage(adminId, text)
	msg.ReplyMarkup = keyboard
	_, _ = bot.Send(msg)

	reply := tgbotapi.NewMessage(update.Message.Chat.ID, "⏳ Ваша заявка отправлена админу, ожидайте решения")
	_, _ = bot.Send(reply)
}
