package user

import (
	"VpnBot/internal/app/usecases"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func GetKeyHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, cdUC *usecases.CooldownUsecase, adminId int64) {
	from := update.CallbackQuery.From
	uid := from.ID
	username := from.UserName
	if username == "" {
		username = fmt.Sprintf("id_%d", uid)
	}

	if onCooldown, remaining, _ := cdUC.IsOnCooldown(uid); onCooldown {
		msg := tgbotapi.NewMessage(uid,
			fmt.Sprintf("⏳ Вы уже отправляли заявку. Попробуйте снова через %d минут.",
				int(remaining.Minutes())))
		_, _ = bot.Send(msg)
		_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Бро ты в муте"))
		return
	}

	text := fmt.Sprintf(
		"Запрос на ключ:\n"+
			"Пользователь: @%s\n"+
			"ID: `%d`",
		username, uid,
	)
	approveBtn := tgbotapi.NewInlineKeyboardButtonData("✅ Одобрить", fmt.Sprintf("approve:%d %s",
		uid, username))
	denyBtn := tgbotapi.NewInlineKeyboardButtonData("❌ Отклонить", fmt.Sprintf("deny:%d %s",
		uid, username))
	blockBtn := tgbotapi.NewInlineKeyboardButtonData("🚫 Заблокировать", fmt.Sprintf("block:%d %s",
		uid, username))
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(approveBtn, denyBtn),
		tgbotapi.NewInlineKeyboardRow(blockBtn),
	)

	adminMsg := tgbotapi.NewMessage(adminId, text)
	adminMsg.ParseMode = "MarkdownV2"
	adminMsg.ReplyMarkup = keyboard
	_, _ = bot.Send(adminMsg)

	_, _ = bot.Request(tgbotapi.NewDeleteMessage(
		update.CallbackQuery.Message.Chat.ID,
		update.CallbackQuery.Message.MessageID,
	))

	_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Заявка отправлена админу"))
	_, _ = bot.Send(tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID,
		"⏳ Ваша заявка отправлена админу, ожидайте решения"))

	HelpHandler(update, bot, adminId)
}
