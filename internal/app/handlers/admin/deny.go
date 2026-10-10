package admin

import (
	"VpnBot/internal/app/usecases"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

func DenyHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, cdUC *usecases.CooldownUsecase, adminId int64) {
	if update.CallbackQuery == nil {
		return
	}
	targetID, _, err := requestCallbackTarget(update.CallbackQuery, "deny:")
	if err != nil {
		answerRequestCallback(bot, update.CallbackQuery, "Ошибка: некорректная заявка")
		return
	}

	if err := cdUC.SetCooldown(targetID); err != nil {
		log.Println("deny set cooldown:", err)
		answerRequestCallback(bot, update.CallbackQuery, "Не удалось отклонить заявку. Попробуйте позже.")
		return
	}

	if _, err := bot.Send(tgbotapi.NewMessage(targetID, "❌ Ваша заявка отклонена")); err != nil {
		log.Println("deny notify:", err)
		answerRequestCallback(bot, update.CallbackQuery, "Заявка отклонена, но уведомление не доставлено")
		return
	}

	delMsg := tgbotapi.NewDeleteMessage(adminId, update.CallbackQuery.Message.MessageID)
	_, _ = bot.Request(delMsg)

	answerRequestCallback(bot, update.CallbackQuery, "Заявка отклонена")
}
