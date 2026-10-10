package user

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"VpnBot/internal/app/ui"
	"VpnBot/internal/app/usecases"
	"VpnBot/internal/domain/repository"
)

func StartHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, adminId int64, pr *repository.PanelRepository, userUC *usecases.UserUsecase) {
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	messageID := update.Message.MessageID

	HomePanel(update, bot, adminId, pr, userUC)
	EnsureReplyKeyboard(bot, chatID)
	_, _ = bot.Request(tgbotapi.NewDeleteMessage(chatID, messageID))
}

func HomePanel(update tgbotapi.Update, bot *tgbotapi.BotAPI, adminId int64, pr *repository.PanelRepository, userUC *usecases.UserUsecase) {
	var uid int64
	if update.CallbackQuery != nil {
		_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
		if update.CallbackQuery.Message == nil {
			return
		}
		uid = update.CallbackQuery.From.ID
	} else if update.Message != nil {
		uid = update.Message.From.ID
	} else {
		return
	}

	if userUC != nil {
		if revoked, err := userUC.CheckPaymentRevoked(uid); err == nil && revoked {
			text := "<b>Kovshik VPN</b>\n\n" +
				"Доступ к разделам приостановлен из‑за просроченной оплаты.\n" +
				"Нажмите «Оплата» и отправьте скриншот перевода — после проверки срок продлят на месяц."
			if adminId == uid {
				text += "\n\n<i>Вы администратор — полный список команд в /help.</i>"
			}
			kb := ui.PaymentRevokedKeyboard()
			EditPanelFromUpdate(bot, pr, update, uid, text, &kb, true)
			return
		}
	}

	text := "<b>Kovshik VPN</b>\n\n" +
		"Выберите раздел ниже. Настройки, подписка и объявления открываются в этом сообщении."
	if adminId == uid {
		text += "\n\n<i>Вы администратор — полный список команд в /help.</i>"
	}
	canRequestKey := true
	if userUC != nil {
		if exists, err := userUC.UserExist(uid); err == nil && exists {
			canRequestKey = false
		}
	}
	kb := MainMenuKeyboard(userUC, uid, canRequestKey)

	EditPanelFromUpdate(bot, pr, update, uid, text, &kb, true)
}
