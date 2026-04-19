package admin

import (
	"log"

	"VpnBot/internal/app/handlers/user"
	"VpnBot/internal/app/usecases"
	"VpnBot/internal/domain/repository"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func PaymentProofConfirmHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase, adminID int64, targetUserID int64, pr *repository.PanelRepository) {
	if err := userUC.ConfirmExtensionAfterPayment(targetUserID); err != nil {
		log.Println("payment confirm:", err)
		_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Ошибка"))
		return
	}
	_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Подтверждено"))

	canRequestKey := true
	if exists, _ := userUC.UserExist(targetUserID); exists {
		canRequestKey = false
	}
	kb := user.MainMenuKeyboard(userUC, targetUserID, canRequestKey)
	text := "✅ Оплата подтверждена. Срок продлён на <b>1 месяц</b>."
	user.EditPanelHTMLForUser(bot, pr, targetUserID, text, &kb, true)

	_, _ = bot.Request(tgbotapi.NewDeleteMessage(adminID, update.CallbackQuery.Message.MessageID))
}

func PaymentProofDenyHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase, adminID int64, targetUserID int64, pr *repository.PanelRepository) {
	if err := userUC.RejectPaymentProof(targetUserID); err != nil {
		log.Println("payment deny:", err)
	}
	_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Отклонено"))

	kb := user.MainMenuKeyboard(userUC, targetUserID, false)
	text := "❌ Скриншот оплаты отклонён. При необходимости отправьте другой через «Оплата» в панели."
	user.EditPanelHTMLForUser(bot, pr, targetUserID, text, &kb, true)

	_, _ = bot.Request(tgbotapi.NewDeleteMessage(adminID, update.CallbackQuery.Message.MessageID))
}
