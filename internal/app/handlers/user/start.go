package user

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"VpnBot/internal/app/ui"
	"VpnBot/internal/app/usecases"
	"VpnBot/internal/domain/repository"
)

func StartHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, adminId int64, pr *repository.PanelRepository, userUC *usecases.UserUsecase) {
	if update.Message == nil || update.Message.From == nil || update.Message.Chat == nil {
		return
	}
	if _, _, hasPanel := pr.Get(update.Message.From.ID); !hasPanel {
		EnsureReplyKeyboard(bot, update.Message.Chat.ID)
	}
	MenuHandler(update, bot, adminId, pr, userUC)
}

func MenuHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, adminId int64, pr *repository.PanelRepository, userUC *usecases.UserUsecase) {
	if !showHomePanel(update, bot, adminId, pr, userUC, true) {
		return
	}
	if update.Message != nil {
		_, _ = bot.Request(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID))
	}
}

func HomePanel(update tgbotapi.Update, bot *tgbotapi.BotAPI, adminId int64, pr *repository.PanelRepository, userUC *usecases.UserUsecase) {
	showHomePanel(update, bot, adminId, pr, userUC, false)
}

func showHomePanel(update tgbotapi.Update, bot *tgbotapi.BotAPI, adminId int64, pr *repository.PanelRepository, userUC *usecases.UserUsecase, fresh bool) bool {
	var uid, chatID int64
	if update.CallbackQuery != nil && update.CallbackQuery.From != nil && update.CallbackQuery.Message != nil && update.CallbackQuery.Message.Chat != nil {
		_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
		uid = update.CallbackQuery.From.ID
		chatID = update.CallbackQuery.Message.Chat.ID
	} else if update.Message != nil && update.Message.From != nil && update.Message.Chat != nil {
		uid = update.Message.From.ID
		chatID = update.Message.Chat.ID
	} else {
		return false
	}
	render := func(text string, kb tgbotapi.InlineKeyboardMarkup) bool {
		if fresh {
			return EditPanelHTML(bot, pr, uid, chatID, 0, text, &kb, true)
		}
		return EditPanelFromUpdate(bot, pr, update, uid, text, &kb, true)
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
			return render(text, kb)
		}
	}

	text := "<b>Kovshik VPN</b>\n\n" +
		"Выберите раздел ниже. Настройки, подписка и объявления открываются в этом сообщении.\n\n" +
		"Чтобы вернуть меню в конец чата, отправьте /menu."
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
	if adminId == uid {
		kb.InlineKeyboard = append([][]tgbotapi.InlineKeyboardButton{ui.AdminUsersRow()}, kb.InlineKeyboard...)
	}

	return render(text, kb)
}
