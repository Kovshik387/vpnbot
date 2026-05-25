package ui

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// WithNavFooter добавляет строку «Главная» и «Объявления».
func WithNavFooter(m tgbotapi.InlineKeyboardMarkup) tgbotapi.InlineKeyboardMarkup {
	rows := append([][]tgbotapi.InlineKeyboardButton{}, m.InlineKeyboard...)
	rows = append(rows, NavFooterRow())
	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func NavFooterRow() []tgbotapi.InlineKeyboardButton {
	return tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🏠 Главная", "panel_home"),
		tgbotapi.NewInlineKeyboardButtonData("📣 Объявления", "sayl:0"),
	)
}

// PanelShortcutKeyboard — кнопка открытия панели из рассылок и уведомлений.
func PanelShortcutKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(panelShowRow())
}

func panelShowRow() []tgbotapi.InlineKeyboardButton {
	return tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🏠 Показать панель", "panel_home"),
	)
}

// HomeScreenKeyboard — главный экран после /start и кнопки «Главная».
func HomeScreenKeyboard(showRequestKey, showPaymentRow bool) tgbotapi.InlineKeyboardMarkup {
	rows := [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройка VPN", "info"),
			tgbotapi.NewInlineKeyboardButtonData("📋 Моя подписка", "subscribe"),
		),
	}
	if showRequestKey {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔑 Запросить ключ", "request_key"),
		))
	}
	if showPaymentRow {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💳 Оплата", "payment_flow"),
		))
	}
	return WithNavFooter(tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows})
}

// PaymentReminderKeyboard — push-уведомления за 2 / 1 день и в день оплаты.
func PaymentReminderKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💳 Оплата", "payment_flow"),
		),
		panelShowRow(),
	)
}

// PaymentRevokedKeyboard — только оплата и возврат к панели (без «Объявления»).
func PaymentRevokedKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💳 Оплата", "payment_flow"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Панель", "panel_home"),
		),
	)
}

// Действия внизу экрана «Помощь».
func HelpActionsKeyboard() tgbotapi.InlineKeyboardMarkup {
	return WithNavFooter(tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройка VPN", "info"),
			tgbotapi.NewInlineKeyboardButtonData("📋 Моя подписка", "subscribe"),
		),
	))
}

// Корневое меню /info: выбор устройства и ссылки.
func InfoRootKeyboard() tgbotapi.InlineKeyboardMarkup {
	return WithNavFooter(tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📱 Телефон", "info_phone"),
			tgbotapi.NewInlineKeyboardButtonData("💻 Компьютер", "info_pc"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📺 ТВ", "info_tv"),
			tgbotapi.NewInlineKeyboardButtonURL("💬 Поддержка", URLSupport),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("📖 Wiki", URLWiki),
		),
	))
}

func infoBackRow() []tgbotapi.InlineKeyboardButton {
	return tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("◀️ Назад к устройствам", "info"),
	)
}

// После выбора «Телефон».
func InfoPhoneKeyboard() tgbotapi.InlineKeyboardMarkup {
	return WithNavFooter(tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("App Store (iPhone)", URLiPhoneApp),
			tgbotapi.NewInlineKeyboardButtonURL("Google Play (Android)", URLAndroid),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("Инструкция по подписке", URLPhoneSub),
		),
		infoBackRow(),
	))
}

// После выбора «Компьютер».
func InfoPCKeyboard() tgbotapi.InlineKeyboardMarkup {
	return WithNavFooter(tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("Скачать V2RayN", URLV2RayN),
			tgbotapi.NewInlineKeyboardButtonURL("Обзор настроек", URLPCGuide),
		),
		infoBackRow(),
	))
}

// После выбора «ТВ».
func InfoTVKeyboard() tgbotapi.InlineKeyboardMarkup {
	return WithNavFooter(tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("Написать в поддержку", URLSupport),
		),
		infoBackRow(),
	))
}
