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
	pr.Clear(update.Message.From.ID)
	HomePanel(update, bot, adminId, pr, userUC)
}

func HomePanel(update tgbotapi.Update, bot *tgbotapi.BotAPI, adminId int64, pr *repository.PanelRepository, userUC *usecases.UserUsecase) {
	var uid int64
	var callbackChatID int64
	var callbackMessageID int
	fromCallback := false
	if update.CallbackQuery != nil {
		_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
		uid = update.CallbackQuery.From.ID
		callbackChatID = update.CallbackQuery.Message.Chat.ID
		callbackMessageID = update.CallbackQuery.Message.MessageID
		fromCallback = true
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
			if fromCallback {
				if oldChatID, oldMessageID, ok := pr.Get(uid); ok && oldChatID == callbackChatID && oldMessageID != 0 {
					EditPanelHTML(bot, pr, uid, oldChatID, oldMessageID, text, &kb, true)
					if callbackMessageID != oldMessageID {
						_, _ = bot.Request(tgbotapi.NewDeleteMessage(callbackChatID, callbackMessageID))
					}
					return
				}
				sendNewPanel(bot, pr, uid, callbackChatID, text, &kb)
				_, _ = bot.Request(tgbotapi.NewDeleteMessage(callbackChatID, callbackMessageID))
				return
			}
			EditPanelFromUpdate(bot, pr, update, uid, text, &kb, true)
			return
		}
	}

	text := "<b>Kovshik VPN</b>\n\n" +
		"Все действия — кнопками в этом сообщении (оно одно на весь чат)."
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

	// сохранённую панель и удаляем источник callback, чтобы не плодить UI.
	if fromCallback {
		if oldChatID, oldMessageID, ok := pr.Get(uid); ok && oldChatID == callbackChatID && oldMessageID != 0 {
			EditPanelHTML(bot, pr, uid, oldChatID, oldMessageID, text, &kb, true)
			if callbackMessageID != oldMessageID {
				_, _ = bot.Request(tgbotapi.NewDeleteMessage(callbackChatID, callbackMessageID))
			}
			return
		}
		sendNewPanel(bot, pr, uid, callbackChatID, text, &kb)
		_, _ = bot.Request(tgbotapi.NewDeleteMessage(callbackChatID, callbackMessageID))
		return
	}

	EditPanelFromUpdate(bot, pr, update, uid, text, &kb, true)
}
