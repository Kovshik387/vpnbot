package user

import (
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"VpnBot/internal/app/ui"
	"VpnBot/internal/domain/repository"
)

func PanelTarget(update tgbotapi.Update, pr *repository.PanelRepository, userID int64) (chatID int64, messageID int, ok bool) {
	if update.CallbackQuery != nil && update.CallbackQuery.Message != nil {
		m := update.CallbackQuery.Message
		return m.Chat.ID, m.MessageID, true
	}
	if update.Message != nil {
		ch := update.Message.Chat.ID
		storedChat, mid, has := pr.Get(userID)
		if has && storedChat == ch {
			return ch, mid, true
		}
		return ch, 0, false
	}
	return 0, 0, false
}

func needsNewPanelMessage(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "message to edit not found") ||
		strings.Contains(s, "MESSAGE_ID_INVALID") ||
		strings.Contains(s, "message can't be edited") ||
		strings.Contains(s, "there is no text in the message to edit")
}

// EditPanelHTML правит одно сообщение-панель или создаёт новое при первом обращении / потере сообщения.
func EditPanelHTML(bot *tgbotapi.BotAPI, pr *repository.PanelRepository, userID, chatID int64, messageID int,
	text string, markup *tgbotapi.InlineKeyboardMarkup, disableWebPreview bool,
) {
	if messageID == 0 {
		sendNewPanel(bot, pr, userID, chatID, text, markup)
		return
	}
	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ParseMode = tgbotapi.ModeHTML
	edit.ReplyMarkup = markup
	edit.DisableWebPagePreview = disableWebPreview
	_, err := bot.Request(edit)
	if err != nil {
		if strings.Contains(err.Error(), "message is not modified") {
			_ = pr.Set(userID, chatID, messageID)
			return
		}
		if needsNewPanelMessage(err) {
			sendNewPanel(bot, pr, userID, chatID, text, markup)
			return
		}
		log.Println("edit panel:", err)
		return
	}
	if err := pr.Set(userID, chatID, messageID); err != nil {
		log.Println("panel persist:", err)
	}
}

func sendNewPanel(bot *tgbotapi.BotAPI, pr *repository.PanelRepository, userID, chatID int64,
	text string, markup *tgbotapi.InlineKeyboardMarkup,
) {
	if oldChatID, oldMessageID, ok := pr.Get(userID); ok && oldChatID == chatID && oldMessageID != 0 {
		_, _ = bot.Request(tgbotapi.NewDeleteMessage(oldChatID, oldMessageID))
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = markup
	sent, err := bot.Send(msg)
	if err != nil {
		log.Println("send panel:", err)
		return
	}
	if err := pr.Set(userID, chatID, sent.MessageID); err != nil {
		log.Println("panel persist:", err)
	}
}

func EditPanelFromUpdate(bot *tgbotapi.BotAPI, pr *repository.PanelRepository, update tgbotapi.Update, userID int64,
	text string, markup *tgbotapi.InlineKeyboardMarkup, disableWebPreview bool,
) {
	if update.CallbackQuery != nil && update.CallbackQuery.Message != nil {
		currentChatID := update.CallbackQuery.Message.Chat.ID
		currentMessageID := update.CallbackQuery.Message.MessageID
		if oldChatID, oldMessageID, ok := pr.Get(userID); ok &&
			oldChatID == currentChatID && oldMessageID != 0 && oldMessageID != currentMessageID {
			_, _ = bot.Request(tgbotapi.NewDeleteMessage(oldChatID, oldMessageID))
		}
	}

	ch, mid, ok := PanelTarget(update, pr, userID)
	if !ok {
		sendNewPanel(bot, pr, userID, ch, text, markup)
		return
	}
	EditPanelHTML(bot, pr, userID, ch, mid, text, markup, disableWebPreview)
}

func EditPanelHTMLForUser(bot *tgbotapi.BotAPI, pr *repository.PanelRepository, userID int64,
	text string, markup *tgbotapi.InlineKeyboardMarkup, disableWebPreview bool,
) {
	chatID, messageID, ok := pr.Get(userID)
	if !ok || chatID == 0 {
		chatID = userID
		messageID = 0
	}
	EditPanelHTML(bot, pr, userID, chatID, messageID, text, markup, disableWebPreview)
}

// SendNotificationHTML шлёт отдельное сообщение с push (не трогает панель в ui_panel).
func SendNotificationHTML(bot *tgbotapi.BotAPI, chatID int64, text string, markup *tgbotapi.InlineKeyboardMarkup, disableWebPreview bool) bool {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.DisableNotification = false
	msg.ReplyMarkup = markup
	msg.DisableWebPagePreview = disableWebPreview
	_, err := bot.Send(msg)
	if err != nil {
		log.Println("send notification:", err)
		return false
	}
	return true
}

// SendNotificationHTMLForUser — уведомление в личный чат по Telegram user id.
func SendNotificationHTMLForUser(bot *tgbotapi.BotAPI, userID int64, text string, markup *tgbotapi.InlineKeyboardMarkup, disableWebPreview bool) bool {
	return SendNotificationHTML(bot, userID, text, markup, disableWebPreview)
}

// EnsureReplyKeyboard включает постоянные кнопки внизу (Telegram требует непустой text).
func EnsureReplyKeyboard(bot *tgbotapi.BotAPI, chatID int64) {
	msg := tgbotapi.NewMessage(chatID, ".")
	msg.ReplyMarkup = ui.MainKeyboard()
	msg.DisableNotification = true
	_, err := bot.Send(msg)
	if err != nil {
		log.Println("reply keyboard:", err)
	}
}
