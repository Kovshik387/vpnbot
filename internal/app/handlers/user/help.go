package user

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"VpnBot/internal/app/ui"
	"VpnBot/internal/domain/repository"
)

func HelpHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, adminId int64, pr *repository.PanelRepository) {
	var uid int64
	var chatID int64
	if update.CallbackQuery != nil {
		_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
		uid = update.CallbackQuery.From.ID
		chatID = update.CallbackQuery.Message.Chat.ID
	} else if update.Message != nil {
		uid = update.Message.From.ID
		chatID = update.Message.Chat.ID
	} else {
		return
	}

	text := "<b>Команды</b>\n\n" +
		"<b>Основное</b>\n" +
		"• <code>/start</code> — главный экран\n" +
		"• <code>/info</code> — настройка на телефоне, ПК или ТВ\n" +
		"• Кнопка <b>«Объявления»</b> — архив рассылок администратора\n"

	if adminId == chatID {
		text += "\n<b>Админ: пользователи</b>\n" +
			"• <code>/adduser</code> — добавить\n" +
			"• <code>/deleteuser</code> — удалить\n" +
			"• <code>/users [имя]</code> — список или поиск\n" +
			"• <code>/block</code> / <code>/unblock</code> — блокировка\n" +
			"• <code>/blocked</code> — заблокированные\n" +
			"• <code>/activity</code> — активные\n" +
			"• <code>/count</code> — число активных\n\n" +
			"<b>Админ: рассылки</b>\n" +
			"• <code>/say</code> — сообщение всем (попадает в «Объявления»)\n" +
			"• <code>/poll_result &lt;id&gt;</code> — итог опроса\n" +
			"• <code>/poll_list</code> — список опросов\n\n" +
			"<b>Админ: сервер</b>\n" +
			"• <code>/status</code> — нагрузка\n\n" +
			"<b>Админ: подписки</b>\n" +
			"• <code>/setprice</code> — цена\n" +
			"• <code>/setdate</code> — дата оплаты (YYYY-MM-DD)\n" +
			"• <code>/setfree</code> — бесплатный статус\n" +
			"• <code>/compensation &lt;дней&gt;</code> — компенсация\n" +
			"• <code>/override</code> — перенос дат массово\n" +
			"• <code>/normalizepay</code> — привести оплаты к текущему YYYY-MM\n" +
			"• <code>/paystats [YYYY-MM]</code> — отчёт по подтверждённым оплатам\n\n" +
			"<b>Сервис</b>\n" +
			"<code>success anal deploy</code>"
	}

	keyboard := ui.HelpActionsKeyboard()
	EditPanelFromUpdate(bot, pr, update, uid, text, &keyboard, true)
}
