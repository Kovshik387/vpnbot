package user

import (
	"log"

	"VpnBot/internal/app/ui"
	"VpnBot/internal/app/usecases"
	"VpnBot/internal/domain/repository"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HasUserInput excludes Telegram service messages from input hints.
func HasUserInput(m *tgbotapi.Message) bool {
	if m == nil || m.From == nil || m.From.IsBot {
		return false
	}
	return m.Text != "" || len(m.Photo) > 0 || m.Document != nil ||
		m.Animation != nil || m.Audio != nil || m.Voice != nil ||
		m.Video != nil || m.VideoNote != nil || m.Sticker != nil ||
		m.Contact != nil || m.Location != nil || m.Venue != nil ||
		m.Poll != nil || m.Dice != nil || m.Game != nil || m.Invoice != nil
}

func UnsupportedMessageHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase,
	adminID int64, pr *repository.PanelRepository,
) {
	m := update.Message
	if !HasUserInput(m) || m.Chat == nil || !m.Chat.IsPrivate() {
		return
	}
	uid := m.From.ID
	awaiting, err := userUC.GetAwaitingPaymentScreenshot(uid)
	if err != nil {
		log.Println("unsupported input payment state:", err)
		showInputError(update, bot, pr)
		return
	}
	if awaiting {
		text := "<b>Нужен скриншот чека</b>\n\n" + paymentScreenshotInstructions +
			"\n\nЖду фото чека. Если нужна помощь, нажмите «Поддержка»."
		kb := ui.InputHelpKeyboard()
		EditPanelFromUpdate(bot, pr, update, uid, text, &kb, true)
		return
	}

	revoked, err := userUC.CheckPaymentRevoked(uid)
	if err != nil {
		log.Println("unsupported input payment access:", err)
		showInputError(update, bot, pr)
		return
	}
	text := "Бот работает через кнопки и команды. Откройте /menu и выберите действие.\n\n" +
		"Чтобы задать вопрос администратору, нажмите «Поддержка»."
	kb := ui.InputHelpKeyboard()
	switch {
	case revoked:
		text = "⏸️ <b>Доступ приостановлен</b> из-за просроченной оплаты.\n\n" +
			"Нажмите «Оплата», затем отправьте скриншот чека как фото. Если нужна помощь, нажмите «Поддержка»."
		kb = ui.PaymentRevokedKeyboard()
	case uid == adminID && (m.Text != "" || len(m.Photo) > 0 || m.Poll != nil):
		text = "Для рассылки ответьте на это сообщение командой <code>/say</code>.\n\n" +
			"Чтобы выбрать другое действие, откройте /menu."
	case len(m.Photo) > 0 || m.Document != nil:
		paid, err := userUC.IsPaidSubscription(uid)
		if err != nil {
			log.Println("unsupported input subscription:", err)
			showInputError(update, bot, pr)
			return
		}
		if paid {
			text = "Если это чек об оплате, сначала нажмите «Оплата».\n\n" +
				paymentScreenshotInstructions + "\n\nЕсли нужна помощь, нажмите «Поддержка»."
			kb = ui.PaymentRevokedKeyboard()
		}
	}
	EditPanelFromUpdate(bot, pr, update, uid, text, &kb, true)
}

func showInputError(update tgbotapi.Update, bot *tgbotapi.BotAPI, pr *repository.PanelRepository) {
	text := "Не удалось обработать сообщение. Попробуйте ещё раз или откройте /menu.\n\n" +
		"Если нужна помощь, нажмите «Поддержка»."
	kb := ui.InputHelpKeyboard()
	EditPanelFromUpdate(bot, pr, update, update.Message.From.ID, text, &kb, true)
}
