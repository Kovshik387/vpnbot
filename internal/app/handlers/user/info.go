package user

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"VpnBot/internal/app/ui"
	"VpnBot/internal/domain/repository"
)

func InfoHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, pr *repository.PanelRepository) {
	var uid int64
	if update.CallbackQuery != nil {
		_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
		uid = update.CallbackQuery.From.ID
	} else if update.Message != nil {
		uid = update.Message.From.ID
	} else {
		return
	}

	text := "<b>Настройка VPN</b>\n\n" +
		"Выберите устройство — откроется краткая инструкция и ссылки на клиенты.\n" +
		"Токен и подписку выдаёт администратор после одобрения заявки на ключ."

	keyboard := ui.InfoRootKeyboard()
	EditPanelFromUpdate(bot, pr, update, uid, text, &keyboard, true)
}
