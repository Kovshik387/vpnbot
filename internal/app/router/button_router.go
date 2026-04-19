package router

import (
	"strconv"
	"strings"

	"VpnBot/config"
	"VpnBot/internal/app/handlers/admin"
	"VpnBot/internal/app/handlers/user"
	"VpnBot/internal/app/ui"
	"VpnBot/internal/app/usecases"
	"errors"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CallbackHandler func(update tgbotapi.Update, bot *tgbotapi.BotAPI)

func NewCallbackRouter(userUC *usecases.UserUsecase, cdUC *usecases.CooldownUsecase, config *config.Config, uir *UIRepos) map[string]CallbackHandler {
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
			user.GetKeyHandler(update, bot, cdUC, config.AdminId, uir.Panel, userUC)
		},
		"block:": func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
			err := checkCallbackPermission(update, bot, config.AdminId)
			if err != nil {
				return
			}
			admin.BlockUserHandler(update, bot, userUC, config.AdminId)
		},
		"ping_server": func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
			user.PingHandler(update, bot, config.RussianUrl, config.AdminId, uir.Panel)
		},
		"help": func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
			user.HelpHandler(update, bot, config.AdminId, uir.Panel)
		},
		"subscribe": func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
			user.GetSubscribeHandler(update, bot, userUC, uir.Panel)
		},
		"info": func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
			user.InfoHandler(update, bot, uir.Panel)
		},
		"panel_home": func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
			user.HomePanel(update, bot, config.AdminId, uir.Panel, userUC)
		},
		"pay_seen": func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
			user.PaymentSeenHandler(update, bot)
		},
		"payment_flow": func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
			user.PaymentFlowHandler(update, bot, userUC, uir.Panel)
		},
		"pc:": func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
			err := checkCallbackPermission(update, bot, config.AdminId)
			if err != nil {
				return
			}
			idStr := strings.TrimPrefix(update.CallbackQuery.Data, "pc:")
			targetID, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Некорректный id"))
				return
			}
			admin.PaymentProofConfirmHandler(update, bot, userUC, config.AdminId, targetID, uir.Panel)
		},
		"pd:": func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
			err := checkCallbackPermission(update, bot, config.AdminId)
			if err != nil {
				return
			}
			idStr := strings.TrimPrefix(update.CallbackQuery.Data, "pd:")
			targetID, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Некорректный id"))
				return
			}
			admin.PaymentProofDenyHandler(update, bot, userUC, config.AdminId, targetID, uir.Panel)
		},
		"sayl:": func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
			offStr := strings.TrimPrefix(update.CallbackQuery.Data, "sayl:")
			off, _ := strconv.Atoi(offStr)
			user.SayLogsPage(update, bot, uir.SayLog, uir.Panel, off)
		},
		"sayd:": func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
			idStr := strings.TrimPrefix(update.CallbackQuery.Data, "sayd:")
			id, _ := strconv.ParseInt(idStr, 10, 64)
			user.SayLogDetail(update, bot, uir.SayLog, uir.Panel, id)
		},
		"info_phone": func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
			_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			uid := update.CallbackQuery.From.ID
			text := "<b>📱 Телефон</b>\n\n" +
				"1. Установите клиент (кнопки ниже).\n" +
				"2. Импортируйте выданный токен или ссылку подписки.\n" +
				"3. Выберите профиль и подключитесь.\n\n" +
				"<b>Подписка (SUB)</b>\n" +
				"Если в регионе режут доступ, удобнее работать через подписку: скопируйте SUB URL из раздела «Моя подписка» и в клиенте выберите «Добавить из буфера» / импорт по ссылке.\n\n" +
				"📎 <a href=\"" + ui.URLPhoneSub + "\">Подробная инструкция на Яндекс.Диске</a>"
			kb := ui.InfoPhoneKeyboard()
			user.EditPanelFromUpdate(bot, uir.Panel, update, uid, text, &kb, true)
		},
		"info_pc": func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
			_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			uid := update.CallbackQuery.From.ID
			text := "<b>💻 Компьютер</b>\n\n" +
				"1. Установите <b>V2RayN</b> (или совместимый клиент).\n" +
				"2. Импортируйте токен / ссылку из бота.\n" +
				"3. Активируйте профиль и подключитесь.\n\n" +
				"Дополнительно — разбор настроек и типичных вопросов по кнопке «Обзор настроек»."
			kb := ui.InfoPCKeyboard()
			user.EditPanelFromUpdate(bot, uir.Panel, update, uid, text, &kb, true)
		},
		"info_tv": func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
			_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			uid := update.CallbackQuery.From.ID
			text := "<b>📺 Телевизор</b>\n\n" +
				"1. Установите поддерживаемый клиент из магазина приложений ТВ (часто ищут по названию <b>v2RayTun</b> или аналог для вашей платформы).\n" +
				"2. Импортируйте токен так же, как на телефоне.\n" +
				"3. Если модель редкая или клиент не ставится — напишите в поддержку, подскажем вариант."
			kb := ui.InfoTVKeyboard()
			user.EditPanelFromUpdate(bot, uir.Panel, update, uid, text, &kb, true)
		},
	}
}

func checkCallbackPermission(update tgbotapi.Update, bot *tgbotapi.BotAPI, adminId int64) error {
	if update.CallbackQuery.Message.Chat.ID != adminId {
		msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID,
			"🔒 Это действие только для администратора.")
		_, _ = bot.Send(msg)
		return errors.New("У тебя нет доступа к этой команде")
	}
	return nil
}
