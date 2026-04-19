package user

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"VpnBot/internal/app/ui"
	"VpnBot/internal/app/usecases"
	"VpnBot/internal/domain/repository"
	"VpnBot/internal/utils"
)

func GetKeyHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, cdUC *usecases.CooldownUsecase, adminId int64, pr *repository.PanelRepository, userUC *usecases.UserUsecase) {
	from := update.CallbackQuery.From
	uid := from.ID
	username := from.UserName
	if username == "" {
		username = fmt.Sprintf("id_%d", uid)
	}

	if onCooldown, remaining, _ := cdUC.IsOnCooldown(uid); onCooldown {
		_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Лимит заявок"))
		t := fmt.Sprintf("⏳ Заявку уже отправляли. Следующая попытка через <b>%d мин.</b>",
			int(remaining.Minutes()))
		kb := ui.HelpActionsKeyboard()
		EditPanelFromUpdate(bot, pr, update, uid, t, &kb, true)
		return
	}
	if exists, err := userUC.UserExist(uid); err == nil && exists {
		_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Уже есть доступ"))
		t := "У вас уже есть доступ в базе. Новый ключ запрашивать не нужно."
		kb := MainMenuKeyboard(userUC, uid, false)
		EditPanelFromUpdate(bot, pr, update, uid, t, &kb, true)
		return
	}

	text := fmt.Sprintf(
		"<b>Заявка на ключ</b>\n"+
			"Пользователь: @%s\n"+
			"Telegram ID: <code>%d</code>",
		utils.HtmlEscape(username), uid,
	)
	approveBtn := tgbotapi.NewInlineKeyboardButtonData("✅ Одобрить", fmt.Sprintf("approve:%d %s",
		uid, username))
	denyBtn := tgbotapi.NewInlineKeyboardButtonData("❌ Отклонить", fmt.Sprintf("deny:%d %s",
		uid, username))
	blockBtn := tgbotapi.NewInlineKeyboardButtonData("🚫 Заблокировать", fmt.Sprintf("block:%d %s",
		uid, username))
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(approveBtn, denyBtn),
		tgbotapi.NewInlineKeyboardRow(blockBtn),
	)

	adminMsg := tgbotapi.NewMessage(adminId, text)
	adminMsg.ParseMode = tgbotapi.ModeHTML
	adminMsg.ReplyMarkup = keyboard
	_, _ = bot.Send(adminMsg)

	_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Отправлено"))
	userText := "✅ Заявка у администратора. Ответ придёт в этот чат."
	kb := MainMenuKeyboard(userUC, uid, false)
	EditPanelFromUpdate(bot, pr, update, uid, userText, &kb, true)
}
