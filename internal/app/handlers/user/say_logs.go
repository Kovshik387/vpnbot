package user

import (
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"VpnBot/internal/app/ui"
	"VpnBot/internal/domain/repository"
	"VpnBot/internal/utils"
)

const sayLogPageSize = 5

func SayLogsPage(update tgbotapi.Update, bot *tgbotapi.BotAPI, sl *repository.SayLogRepository, pr *repository.PanelRepository, offset int) {
	if update.CallbackQuery == nil {
		return
	}
	_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))

	uid := update.CallbackQuery.From.ID
	entries, err := sl.ListDesc(offset, sayLogPageSize+1)
	if err != nil {
		log.Println("say_logs list:", err)
		kb := ui.HomeScreenKeyboard(false, false)
		EditPanelFromUpdate(bot, pr, update, uid, "Не удалось загрузить список объявлений.", &kb, true)
		return
	}
	hasMore := len(entries) > sayLogPageSize
	if hasMore {
		entries = entries[:sayLogPageSize]
	}
	total, _ := sl.Count()

	var b strings.Builder
	b.WriteString("<b>📣 Объявления</b>\n\n")
	if len(entries) == 0 {
		b.WriteString("Пока нет записей: после рассылки <code>/say</code> они появятся здесь.")
	} else {
		b.WriteString(fmt.Sprintf("Страница <b>%d</b> · всего записей: <b>%d</b>\n\n",
			offset/sayLogPageSize+1, total))
		b.WriteString("Нажмите кнопку ниже, чтобы открыть полный текст объявления.")
	}

	kb := sayLogsKeyboard(entries, offset, hasMore)
	markup := kb
	EditPanelFromUpdate(bot, pr, update, uid, b.String(), &markup, true)
}

func SayLogDetail(update tgbotapi.Update, bot *tgbotapi.BotAPI, sl *repository.SayLogRepository, pr *repository.PanelRepository, id int64) {
	if update.CallbackQuery == nil {
		return
	}
	_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))

	uid := update.CallbackQuery.From.ID
	e, ok := sl.GetByID(id)
	if !ok {
		t := "Запись не найдена или удалена."
		kb := ui.HomeScreenKeyboard(false, false)
		EditPanelFromUpdate(bot, pr, update, uid, t, &kb, true)
		return
	}
	body := utils.HtmlEscape(e.Body)
	if len(body) > 3500 {
		body = body[:3500] + "…"
	}
	text := fmt.Sprintf("<b>Объявление #%d</b>\n<i>%s</i> · <i>%s</i>\n\n%s",
		e.ID, utils.HtmlEscape(kindRu(e.Kind)), utils.HtmlEscape(e.CreatedAt), body)

	back := tgbotapi.NewInlineKeyboardButtonData("◀️ К списку", "sayl:0")
	row := tgbotapi.NewInlineKeyboardRow(back)
	m := ui.WithNavFooter(tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{row}})
	EditPanelFromUpdate(bot, pr, update, uid, text, &m, true)
}

func sayLogsKeyboard(entries []repository.SayLogEntry, offset int, hasMore bool) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, e := range entries {
		label := fmt.Sprintf("#%d · %s", e.ID, truncateRunes(strings.ReplaceAll(e.Body, "\n", " "), 28))
		if len([]rune(label)) > 60 {
			label = truncateRunes(label, 58)
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, fmt.Sprintf("sayd:%d", e.ID)),
		))
	}
	var nav []tgbotapi.InlineKeyboardButton
	if offset > 0 {
		prevOff := offset - sayLogPageSize
		if prevOff < 0 {
			prevOff = 0
		}
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("sayl:%d", prevOff)))
	}
	if hasMore {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("Вперёд ➡️", fmt.Sprintf("sayl:%d", offset+sayLogPageSize)))
	}
	if len(nav) > 0 {
		rows = append(rows, nav)
	}
	return ui.WithNavFooter(tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows})
}

func kindRu(k string) string {
	switch k {
	case "photo":
		return "фото"
	case "poll":
		return "опрос"
	default:
		return "текст"
	}
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
