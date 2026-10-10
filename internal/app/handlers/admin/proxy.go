package admin

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"VpnBot/internal/app/usecases"
	"VpnBot/internal/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const proxyCommandHelp = "Формат: /addproxy название | ссылка\n" +
	"Изменить: /editproxy ID название | ссылка\n" +
	"Удалить: /deleteproxy ID\nСписок ID: /proxylist\n\n" +
	"Поддерживаются Telegram MTProto, Telegram WEB, SOCKS5 и HTTP/HTTPS.\n" +
	"Для MTProto, SOCKS5 и HTTP/HTTPS нужен порт; Telegram WEB — без порта (HTTPS, 443)."

func parseProxyCommand(args string, withID bool) (int64, string, string, error) {
	name, rawURL, ok := strings.Cut(args, "|")
	if !ok || strings.TrimSpace(name) == "" || strings.TrimSpace(rawURL) == "" {
		return 0, "", "", errors.New(proxyCommandHelp)
	}
	name = strings.TrimSpace(name)
	var id int64
	if withID {
		index := strings.IndexAny(name, " \t\n")
		if index < 0 {
			return 0, "", "", errors.New(proxyCommandHelp)
		}
		var err error
		id, err = strconv.ParseInt(name[:index], 10, 64)
		if err != nil || id <= 0 {
			return 0, "", "", errors.New("ID прокси должен быть положительным числом.\n\n" + proxyCommandHelp)
		}
		name = strings.TrimSpace(name[index:])
		if name == "" {
			return 0, "", "", errors.New(proxyCommandHelp)
		}
	}
	return id, name, strings.TrimSpace(rawURL), nil
}

func ProxyAddHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, uc *usecases.ProxyUsecase) {
	_, name, rawURL, err := parseProxyCommand(update.Message.CommandArguments(), false)
	if err != nil {
		sendProxyResult(update, bot, err)
		return
	}
	p, err := uc.Add(name, rawURL)
	if err != nil {
		sendProxyResult(update, bot, err)
		return
	}
	sendProxyText(update, bot, fmt.Sprintf("✅ Прокси #%d «%s» добавлен. Откройте /proxies для проверки.", p.ID, p.Name))
}

func ProxyEditHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, uc *usecases.ProxyUsecase) {
	id, name, rawURL, err := parseProxyCommand(update.Message.CommandArguments(), true)
	if err == nil {
		err = uc.Update(id, name, rawURL)
	}
	if err != nil {
		sendProxyResult(update, bot, err)
		return
	}
	sendProxyText(update, bot, fmt.Sprintf("✅ Прокси #%d обновлён. Откройте /proxies для проверки.", id))
}

func ProxyDeleteHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, uc *usecases.ProxyUsecase) {
	id, err := strconv.ParseInt(strings.TrimSpace(update.Message.CommandArguments()), 10, 64)
	if err != nil || id <= 0 {
		sendProxyText(update, bot, "Использование: /deleteproxy ID\nПосмотреть ID: /proxylist")
		return
	}
	if err := uc.Delete(id); err != nil {
		sendProxyResult(update, bot, err)
		return
	}
	sendProxyText(update, bot, fmt.Sprintf("✅ Прокси #%d удалён.", id))
}

func ProxyListHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, uc *usecases.ProxyUsecase) {
	entries, err := uc.List(0, 50)
	if err != nil {
		sendProxyResult(update, bot, err)
		return
	}
	text := "<b>🌐 Прокси: ID для управления</b>\n\n"
	if len(entries) == 0 {
		text += "Пока нет прокси.\n"
	}
	// Send each page separately so a large catalog cannot exceed Telegram's limit.
	for i, p := range entries {
		text += fmt.Sprintf("<code>%d</code> · %s\n", p.ID, utils.HtmlEscape(p.Name))
		if (i+1)%20 == 0 {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
			msg.ParseMode = tgbotapi.ModeHTML
			_, _ = bot.Send(msg)
			text = ""
		}
	}
	total, err := uc.Count()
	if err == nil && total > int64(len(entries)) {
		text += fmt.Sprintf("\nПоказаны первые %d из %d. Остальные ID — в карточках /proxies.\n", len(entries), total)
	}
	text += "\n" + utils.HtmlEscape(proxyCommandHelp)
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	_, _ = bot.Send(msg)
}

func sendProxyResult(update tgbotapi.Update, bot *tgbotapi.BotAPI, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		sendProxyText(update, bot, "Прокси с таким ID не найден. Посмотрите /proxylist.")
		return
	}
	sendProxyText(update, bot, "❌ "+err.Error())
}

func sendProxyText(update tgbotapi.Update, bot *tgbotapi.BotAPI, text string) {
	_, _ = bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, text))
}
