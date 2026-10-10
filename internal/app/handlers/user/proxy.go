package user

import (
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"

	"VpnBot/internal/app/ui"
	"VpnBot/internal/app/usecases"
	"VpnBot/internal/domain/model"
	"VpnBot/internal/domain/repository"
	"VpnBot/internal/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const proxyPageSize = 5

func ProxyPageHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, uc *usecases.ProxyUsecase, pr *repository.PanelRepository, offset int) {
	uid, ok := proxyUser(update, bot)
	if !ok {
		return
	}
	if offset < 0 {
		offset = 0
	}
	total, err := uc.Count()
	if err != nil {
		proxyLoadError(update, bot, pr, uid, err)
		return
	}
	if total == 0 {
		offset = 0
	} else if int64(offset) >= total {
		offset = int((total - 1) / proxyPageSize * proxyPageSize)
	}
	entries, err := uc.List(offset, proxyPageSize+1)
	if err != nil {
		proxyLoadError(update, bot, pr, uid, err)
		return
	}
	hasMore := len(entries) > proxyPageSize
	if hasMore {
		entries = entries[:proxyPageSize]
	}
	text := "<b>🌐 Прокси</b>\n\n"
	if len(entries) == 0 {
		text += "Прокси пока не добавлены. Они появятся здесь после настройки администратором."
	} else {
		text += fmt.Sprintf("Страница %d · всего: %d\n\n", offset/proxyPageSize+1, total) +
			"Выберите прокси. В карточке доступны ссылка подключения или настройки для браузера."
	}
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, p := range entries {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(truncateRunes(p.Name, 45), fmt.Sprintf("proxyd:%d:%d", p.ID, offset)),
		))
	}
	var nav []tgbotapi.InlineKeyboardButton
	if offset > 0 {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("proxyl:%d", max(0, offset-proxyPageSize))))
	}
	if hasMore {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("Вперёд ➡️", fmt.Sprintf("proxyl:%d", offset+proxyPageSize)))
	}
	if len(nav) > 0 {
		rows = append(rows, nav)
	}
	kb := ui.WithNavFooter(tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows})
	EditPanelFromUpdate(bot, pr, update, uid, text, &kb, true)
}

func ProxyDetailHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, uc *usecases.ProxyUsecase, pr *repository.PanelRepository, id int64, offset int) {
	uid, ok := proxyUser(update, bot)
	if !ok {
		return
	}
	p, err := uc.Get(id)
	var text, connectURL string
	if errors.Is(err, sql.ErrNoRows) {
		text = "Этот прокси уже удалён. Вернитесь к списку и выберите другой."
	} else if err != nil {
		proxyLoadError(update, bot, pr, uid, err)
		return
	} else {
		text, connectURL = proxyCard(p)
	}
	var rows [][]tgbotapi.InlineKeyboardButton
	if connectURL != "" {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("🔗 Подключить в Telegram", connectURL),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("◀️ К списку прокси", fmt.Sprintf("proxyl:%d", max(0, offset))),
	))
	kb := ui.WithNavFooter(tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows})
	EditPanelFromUpdate(bot, pr, update, uid, text, &kb, true)
}

func proxyCard(p model.Proxy) (text, connectURL string) {
	u, err := url.Parse(p.URL)
	if err != nil {
		return "Не удалось прочитать настройки прокси. Обратитесь в поддержку.", ""
	}
	var kind, host, port, username, password, secret string
	scheme := strings.ToLower(u.Scheme)
	path := strings.ToLower(strings.TrimPrefix(u.Path, "/"))
	if scheme == "tg" {
		path = strings.ToLower(u.Host)
	}
	telegramPath := path == "proxy" || path == "socks" || path == "webproxy"
	if telegramPath && (scheme == "tg" || (scheme == "https" && strings.EqualFold(u.Hostname(), "t.me"))) {
		q := u.Query()
		host, port = q.Get("server"), q.Get("port")
		username, password, secret = q.Get("user"), q.Get("pass"), q.Get("secret")
		connectURL = "https://t.me/" + path + "?" + q.Encode()
		switch path {
		case "proxy":
			kind = "Telegram MTProto"
		case "webproxy":
			kind = "Telegram WEB"
			// Uppercase hex can resemble a marked base64url secret to the client.
			if decoded, err := hex.DecodeString(secret); err == nil && (len(decoded) == 16 || (len(decoded) == 17 && decoded[0] == 0xdd)) {
				q.Set("secret", strings.ToLower(secret))
			}
			connectURL = "tg://webproxy?" + q.Encode()
		default:
			kind = "SOCKS5"
		}
	} else {
		host, port = u.Hostname(), u.Port()
		if u.User != nil {
			username = u.User.Username()
			password, _ = u.User.Password()
		}
		switch scheme {
		case "socks5", "socks5h":
			kind = "SOCKS5"
			q := url.Values{"server": {host}, "port": {port}}
			if username != "" {
				q.Set("user", username)
			}
			if password != "" {
				q.Set("pass", password)
			}
			connectURL = "https://t.me/socks?" + q.Encode()
		case "http", "https":
			kind = strings.ToUpper(scheme)
		default:
			return "Формат прокси не поддерживается. Обратитесь в поддержку.", ""
		}
	}
	text = fmt.Sprintf("<b>🌐 %s</b>\n<i>%s · #%d</i>\n\n<b>Адрес:</b> <code>%s</code>\n",
		utils.HtmlEscape(p.Name), kind, p.ID, utils.HtmlEscape(host))
	if kind != "Telegram WEB" {
		text += "<b>Порт:</b> <code>" + utils.HtmlEscape(port) + "</code>\n"
	}
	if username != "" {
		text += "<b>Логин:</b> <code>" + utils.HtmlEscape(username) + "</code>\n"
	}
	if password != "" {
		text += "<b>Пароль:</b> <code>" + utils.HtmlEscape(password) + "</code>\n"
	}
	if secret != "" {
		label := "Секрет"
		if kind == "Telegram WEB" {
			label = "Секрет ссылки"
		}
		text += "<b>" + label + ":</b> <code>" + utils.HtmlEscape(secret) + "</code>\n"
	}
	if connectURL != "" {
		text += "\nНажмите «Подключить в Telegram» и подтвердите подключение.\n"
		if kind == "Telegram WEB" {
			text += "Нужен клиент Telegram с поддержкой WEB-прокси. Соединение использует HTTPS на порту 443.\n"
		}
	} else {
		text += "\nУкажите тип, адрес и порт в настройках прокси браузера или системы. Если требуется авторизация, используйте логин и пароль выше.\n"
	}
	text += "\n<b>Ссылка для копирования:</b>\n<code>" + utils.HtmlEscape(p.URL) + "</code>"
	return text, connectURL
}

func proxyUser(update tgbotapi.Update, bot *tgbotapi.BotAPI) (int64, bool) {
	if update.CallbackQuery != nil && update.CallbackQuery.From != nil && update.CallbackQuery.Message != nil {
		_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
		return update.CallbackQuery.From.ID, true
	}
	if update.Message != nil && update.Message.From != nil {
		return update.Message.From.ID, true
	}
	return 0, false
}

func proxyLoadError(update tgbotapi.Update, bot *tgbotapi.BotAPI, pr *repository.PanelRepository, uid int64, err error) {
	log.Println("proxy catalog:", err)
	kb := ui.WithNavFooter(tgbotapi.NewInlineKeyboardMarkup())
	EditPanelFromUpdate(bot, pr, update, uid, "Не удалось загрузить прокси. Попробуйте позже или напишите в поддержку.", &kb, true)
}
