package user

import (
	"fmt"
	"log"

	"VpnBot/internal/app/ui"
	"VpnBot/internal/app/usecases"
	"VpnBot/internal/domain/repository"
	"VpnBot/internal/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func PaymentSeenHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
	if update.CallbackQuery == nil || update.CallbackQuery.Message == nil {
		return
	}
	_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Ок"))
	m := update.CallbackQuery.Message
	edit := tgbotapi.NewEditMessageReplyMarkup(m.Chat.ID, m.MessageID, tgbotapi.InlineKeyboardMarkup{})
	_, err := bot.Request(edit)
	if err != nil {
		log.Println("pay_seen edit:", err)
	}
}

func PaymentFlowHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase, pr *repository.PanelRepository) {
	if update.CallbackQuery == nil {
		return
	}
	uid := update.CallbackQuery.From.ID
	paid, err := userUC.IsPaidSubscription(uid)
	if err != nil || !paid {
		_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Не требуется"))
		return
	}
	_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))

	if err := userUC.SetAwaitingPaymentScreenshot(uid, true); err != nil {
		log.Println("payment_flow awaiting:", err)
	}

	price, err := userUC.GetPriceByUserID(uid)
	if err != nil {
		log.Println("payment_flow price:", err)
		price = 0
	}
	amountLine := ""
	if price > 0 {
		amountLine = fmt.Sprintf("💰 <b>К оплате:</b> <b>%.2f</b>\n\n", price)
	}

	text := "<b>💳 Оплата</b>\n\n" +
		amountLine +
		"Сделайте перевод на реквизиты, которые вам сообщал администратор.\n" +
		"Затем <b>пришлите сюда скриншот</b> чека или платежа одним сообщением (фото).\n\n" +
		"После проверки администратор продлит срок на <b>1 месяц</b>."
	kb := ui.PaymentRevokedKeyboard()
	EditPanelFromUpdate(bot, pr, update, uid, text, &kb, true)
}

func HandlePaymentPhoto(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase, adminID int64, pr *repository.PanelRepository) bool {
	if update.Message == nil || len(update.Message.Photo) == 0 {
		return false
	}
	uid := update.Message.From.ID
	awaiting, err := userUC.GetAwaitingPaymentScreenshot(uid)
	if err != nil || !awaiting {
		return false
	}

	username := update.Message.From.UserName
	if username == "" {
		username = fmt.Sprintf("id_%d", uid)
	}

	photos := update.Message.Photo
	last := photos[len(photos)-1]
	price, err := userUC.GetPriceByUserID(uid)
	if err != nil {
		log.Println("payment photo price:", err)
		price = 0
	}

	caption := fmt.Sprintf(
		"<b>Чек об оплате</b>\nПользователь: @%s\nTelegram ID: <code>%d</code>\nСумма: <b>%.2f</b>",
		utils.HtmlEscape(username), uid, price,
	)

	okBtn := tgbotapi.NewInlineKeyboardButtonData("✅ Подтвердить", fmt.Sprintf("pc:%d", uid))
	noBtn := tgbotapi.NewInlineKeyboardButtonData("❌ Отклонить", fmt.Sprintf("pd:%d", uid))
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(okBtn, noBtn),
	)

	photo := tgbotapi.NewPhoto(adminID, tgbotapi.FileID(last.FileID))
	photo.ParseMode = tgbotapi.ModeHTML
	photo.Caption = caption
	photo.ReplyMarkup = kb
	if _, err := bot.Send(photo); err != nil {
		log.Println("forward payment photo:", err)
		text := "Не удалось отправить скриншот администратору. Попробуйте ещё раз или напишите в поддержку."
		kb := ui.PaymentRevokedKeyboard()
		EditPanelHTMLForUser(bot, pr, uid, text, &kb, true)
		return true
	}

	// Удаляем сообщение пользователя с чеком, чтобы в чате оставалась только UI-панель.
	_, _ = bot.Request(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID))

	if err := userUC.SetAwaitingPaymentScreenshot(uid, false); err != nil {
		log.Println("clear awaiting screenshot:", err)
	}

	text := "✅ <b>Чек отправлен.</b>\nОжидайте подтверждения от администратора."
	kb2 := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "panel_home"),
		),
	)
	EditPanelHTMLForUser(bot, pr, uid, text, &kb2, true)
	return true
}

func CallbackExemptWhenPaymentRevoked(data string) bool {
	return data == "payment_flow" || data == "pay_seen" || data == "panel_home"
}
