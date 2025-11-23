package admin

import (
	"VpnBot/internal/app/usecases"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"strings"
)

func PollListHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase) {
	if update.Message == nil {
		return
	}

	list, err := userUC.GetAllPolls()
	if err != nil {
		_, _ = bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID,
			"Ошибка при получении списка опросов ❌"))
		return
	}

	if len(list) == 0 {
		_, _ = bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID,
			"Опросов пока нет."))
		return
	}

	var sb strings.Builder
	sb.WriteString("📋 *Список опросов:*\n\n")

	for _, p := range list {
		sb.WriteString(fmt.Sprintf(
			"• *%s*\n`\n/poll_result %s\n`\n\n",
			p.Question,
			p.PollID,
		))
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, sb.String())
	msg.ParseMode = "Markdown"
	_, _ = bot.Send(msg)
}
