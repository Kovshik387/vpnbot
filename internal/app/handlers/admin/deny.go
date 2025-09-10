package admin

import (
	"VpnBot/internal/app/usecases"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"strconv"
	"strings"
)

func DenyHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, cdUC *usecases.CooldownUsecase, adminId int64) {
	param := strings.TrimPrefix(update.CallbackQuery.Data, "deny:")
	id := strings.Split(param, " ")

	targetID, err := strconv.ParseInt(id[0], 10, 64)
	if err != nil {
		_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Ошибка: некорректный id"))
		return
	}

	parts := strings.Split(update.CallbackQuery.Data, ":")
	if len(parts) != 2 {
		return
	}

	_ = cdUC.SetCooldown(targetID)

	_, _ = bot.Send(tgbotapi.NewMessage(targetID, "❌ Ваша заявка отклонена"))

	delMsg := tgbotapi.NewDeleteMessage(adminId, update.CallbackQuery.Message.MessageID)
	_, _ = bot.Request(delMsg)

	_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Заявка отклонена"))
	_, _ = bot.Send(tgbotapi.NewEditMessageReplyMarkup(adminId, update.CallbackQuery.Message.MessageID,
		tgbotapi.InlineKeyboardMarkup{}))
}
