package router

import (
	"VpnBot/config"
	"VpnBot/internal/app/handlers/admin"
	"VpnBot/internal/app/handlers/user"
	"VpnBot/internal/app/usecases"
	"errors"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CallbackHandler func(update tgbotapi.Update, bot *tgbotapi.BotAPI)

func NewCallbackRouter(userUC *usecases.UserUsecase, cdUC *usecases.CooldownUsecase, config *config.Config) map[string]CallbackHandler {
	return map[string]CallbackHandler{
		"approve:": func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
			err := checkCallbackPermission(update, bot, config.AdminId)
			if err != nil {
				return
			}
			admin.ApproveHandler(update, bot, userUC, config.AdminId)
		},
		"deny:": func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
			err := checkCallbackPermission(update, bot, config.AdminId)
			if err != nil {
				return
			}
			admin.DenyHandler(update, bot, cdUC, config.AdminId)
		},
		"request_key": func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
			user.GetKeyHandler(update, bot, cdUC, config.AdminId)
		},
		"block:": func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
			err := checkCallbackPermission(update, bot, config.AdminId)
			if err != nil {
				return
			}
			admin.BlockUserHandler(update, bot, userUC, config.AdminId)
		},
		"ping_server": func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
			user.PingHandler(update, bot, config.RussianUrl, config.AdminId)
		},
		"help": func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
			user.HelpHandler(update, bot, config.AdminId)
		},
		"subscribe": func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
			user.GetSubscribeHandler(update, bot, userUC)
		},
		"info": func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
			user.InfoHandler(update, bot)
		},
		"info_phone": func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
			deletePreview(update, bot)
			text := "📱 Инструкция для телефона:\n" +
				"1. Скачайте приложение VPN клиент\n" +
				"2. Вставьте ваш токен\n" +
				"3. Подключитесь и пользуйтесь\n" +
				"Если вы находитесь в Воронеже или другом городе, где могут блокировать интернет, можно обойти ограничения через подписку. Подписка позволяет выбрать VPN с лучшим соединением или специальную конфигурацию для обхода блокировок: скопируйте ссылку из SUB URL и вставьте её в раздел 'Добавить из буфера'.\n" +
				"Подробная инструкция по обновлению и выбору подписки доступна [здесь](https://disk.yandex.ru/i/HT13HcKOYOQ8BQ)"

			keyboard := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL("📱 iPhone",
						"https://apps.apple.com/ru/app/v2raytun/id6476628951"),
					tgbotapi.NewInlineKeyboardButtonURL("🤖 Android",
						"https://play.google.com/store/apps/details?id=com.v2raytun.android&hl=ru"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("📋 Список команд", "help"),
				),
			)

			msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, text)
			msg.ParseMode = "Markdown"
			msg.ReplyMarkup = keyboard

			_, _ = bot.Send(msg)
		},
		"info_pc": func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
			deletePreview(update, bot)
			text := "💻 Инструкция для компьютера:\n" +
				"1. Установите v2RayN или другой клиент\n" +
				"2. Импортируйте токен в приложение\n" +
				"3. Подключитесь\n" +
				"Более гибкая настройка и объяснение что происходит по кнопке"

			keyboard := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL("V2RayN",
						"https://disk.yandex.ru/d/KmfPMvw42gMSYg"),
					tgbotapi.NewInlineKeyboardButtonURL("Что у вас здесь происходит",
						"https://disk.yandex.ru/i/fHv8u6gQ0hFKzg"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("📋 Список команд", "help"),
				),
			)

			msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, text)
			msg.ReplyMarkup = keyboard
			_, _ = bot.Send(msg)

		},
		"info_tv": func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
			_, _ = bot.Request(tgbotapi.NewDeleteMessage(
				update.CallbackQuery.Message.Chat.ID,
				update.CallbackQuery.Message.MessageID,
			))
			text := "📺 Инструкция для телевизора:\n" +
				"1. Установите VPN-приложение из магазина приложений v2RayTun\n" +
				"2. Пишите @KovshikGo"
			_, _ = bot.Send(tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, text))
		},
	}
}

func checkCallbackPermission(update tgbotapi.Update, bot *tgbotapi.BotAPI, adminId int64) error {
	if update.CallbackQuery.Message.Chat.ID != adminId {
		msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "У тебя нет доступа к этой команде")
		_, _ = bot.Send(msg)
		return errors.New("У тебя нет доступа к этой команде")
	}
	return nil
}

func deletePreview(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
	_, _ = bot.Request(tgbotapi.NewDeleteMessage(
		update.CallbackQuery.Message.Chat.ID,
		update.CallbackQuery.Message.MessageID,
	))
}
