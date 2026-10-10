package user

import (
	"fmt"
	"log"
	"math"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"VpnBot/internal/app/ui"
	"VpnBot/internal/app/usecases"
	"VpnBot/internal/domain/repository"
	"VpnBot/internal/utils"
)

func GetKeyHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, cdUC *usecases.CooldownUsecase, adminId int64, pr *repository.PanelRepository, userUC *usecases.UserUsecase) {
	if update.CallbackQuery == nil || update.CallbackQuery.Message == nil {
		return
	}
	from := update.CallbackQuery.From
	uid := from.ID
	username := from.UserName
	if username == "" {
		username = fmt.Sprintf("id_%d", uid)
	}

	exists, err := userUC.UserExist(uid)
	if err != nil {
		log.Println("request key user lookup:", err)
		_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Попробуйте позже"))
		return
	}
	if exists {
		_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Уже есть доступ"))
		t := "У вас уже есть доступ в базе. Новый ключ запрашивать не нужно."
		kb := MainMenuKeyboard(userUC, uid, false)
		EditPanelFromUpdate(bot, pr, update, uid, t, &kb, true)
		return
	}
	onCooldown, remaining, err := cdUC.IsOnCooldown(uid)
	if err != nil {
		log.Println("request key cooldown:", err)
		_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Попробуйте позже"))
		return
	}
	if onCooldown {
		_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Лимит заявок"))
		t := fmt.Sprintf("⏳ Заявку уже отправляли. Следующая попытка через <b>%d мин.</b>",
			int(math.Ceil(remaining.Minutes())))
		kb := ui.HelpActionsKeyboard()
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
	if _, err := bot.Send(adminMsg); err != nil {
		log.Println("request key send:", err)
		_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Не удалось отправить"))
		kb := MainMenuKeyboard(userUC, uid, true)
		EditPanelFromUpdate(bot, pr, update, uid,
			"Не удалось отправить заявку администратору. Попробуйте ещё раз или напишите в поддержку.", &kb, true)
		return
	}
	if err := cdUC.SetCooldown(uid); err != nil {
		log.Println("request key set cooldown:", err)
	}

	_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Отправлено"))
	userText := "✅ Заявка у администратора. Ответ придёт в этот чат."
	kb := MainMenuKeyboard(userUC, uid, false)
	EditPanelFromUpdate(bot, pr, update, uid, userText, &kb, true)
}
