package admin

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	defaultLogLines   = 50
	maxLogLines       = 1000
	maxLogBodyChars   = 3800
	logCallbackPrefix = "logt:"
)

var logLinePresets = []int{20, 50, 100, 200, 500, 1000}

// LogsHandler — /logs [строк] [файл№]. Без аргументов — меню выбора.
func LogsHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
	if update.Message == nil {
		return
	}
	chatID := update.Message.Chat.ID
	paths := LogPathsFromEnv()
	if len(paths) == 0 {
		sendLogSetupHelp(bot, chatID)
		return
	}

	fields := strings.Fields(update.Message.Text)
	// /logs [lines] [fileIndex 1-based]
	if len(fields) >= 2 {
		lines, err := strconv.Atoi(fields[1])
		if err != nil || lines < 1 {
			sendLogUsage(bot, chatID, paths)
			return
		}
		lines = clampLogLines(lines)
		fileIdx := 0
		if len(fields) >= 3 {
			n, err := strconv.Atoi(fields[2])
			if err != nil || n < 1 || n > len(paths) {
				sendLogUsage(bot, chatID, paths)
				return
			}
			fileIdx = n - 1
		}
		if len(paths) == 1 {
			sendLogTail(bot, chatID, 0, lines)
			return
		}
		sendLogTail(bot, chatID, fileIdx, lines)
		return
	}

	sendLogPicker(bot, chatID, paths, defaultLogLines)
}

// LogsTailCallback — callback logt:<fileIdx>:<lines>
func LogsTailCallback(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
	if update.CallbackQuery == nil || update.CallbackQuery.Message == nil {
		return
	}
	_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Читаю лог…"))

	paths := LogPathsFromEnv()
	if len(paths) == 0 {
		sendLogSetupHelp(bot, update.CallbackQuery.Message.Chat.ID)
		return
	}

	data := strings.TrimPrefix(update.CallbackQuery.Data, logCallbackPrefix)
	parts := strings.Split(data, ":")
	if len(parts) != 2 {
		return
	}
	fileIdx, err1 := strconv.Atoi(parts[0])
	lines, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || fileIdx < 0 || fileIdx >= len(paths) {
		return
	}
	sendLogTail(bot, update.CallbackQuery.Message.Chat.ID, fileIdx, clampLogLines(lines))
}

func LogPathsFromEnv() []string {
	raw := strings.TrimSpace(os.Getenv("BOT_LOG_PATHS"))
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("BOT_LOG_PATH"))
	}
	if raw == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func sendLogPicker(bot *tgbotapi.BotAPI, chatID int64, paths []string, presetLines int) {
	var b strings.Builder
	b.WriteString("<b>📋 Логи из файла</b>\n\n")
	b.WriteString("Выберите число строк")
	if len(paths) > 1 {
		b.WriteString(" и файл")
	}
	b.WriteString(".\n\n")
	b.WriteString(fmt.Sprintf("Или команда: <code>/logs %d</code>", presetLines))
	if len(paths) > 1 {
		b.WriteString(fmt.Sprintf(" <code>1</code>…<code>%d</code>", len(paths)))
	}
	b.WriteString("\n\n<b>Файлы:</b>\n")
	for i, p := range paths {
		b.WriteString(fmt.Sprintf("%d. <code>%s</code>\n", i+1, escapeForPre(filepath.Base(p))))
	}

	msg := tgbotapi.NewMessage(chatID, b.String())
	msg.ParseMode = tgbotapi.ModeHTML
	msg.DisableWebPagePreview = true
	msg.ReplyMarkup = logPickerKeyboard(paths, presetLines)
	_, _ = bot.Send(msg)
}

func logPickerKeyboard(paths []string, defaultLines int) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	if len(paths) > 1 {
		for i, p := range paths {
			label := fmt.Sprintf("📄 %s", truncateLabel(filepath.Base(p), 28))
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(label, logCallbackData(i, defaultLines)),
			))
		}
	}

	for i := 0; i < len(logLinePresets); i += 3 {
		var row []tgbotapi.InlineKeyboardButton
		for j := i; j < i+3 && j < len(logLinePresets); j++ {
			n := logLinePresets[j]
			label := fmt.Sprintf("%d строк", n)
			if len(paths) == 1 {
				row = append(row, tgbotapi.NewInlineKeyboardButtonData(label, logCallbackData(0, n)))
			} else {
				row = append(row, tgbotapi.NewInlineKeyboardButtonData(label, fmt.Sprintf("logpick:%d", n)))
			}
		}
		rows = append(rows, row)
	}

	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// LogsPickLinesCallback — logpick:<lines> — выбор строк при нескольких файлах.
func LogsPickLinesCallback(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
	if update.CallbackQuery == nil || update.CallbackQuery.Message == nil {
		return
	}
	paths := LogPathsFromEnv()
	if len(paths) <= 1 {
		return
	}
	linesStr := strings.TrimPrefix(update.CallbackQuery.Data, "logpick:")
	lines, err := strconv.Atoi(linesStr)
	if err != nil {
		return
	}
	lines = clampLogLines(lines)
	_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))

	var rows [][]tgbotapi.InlineKeyboardButton
	for i, p := range paths {
		label := fmt.Sprintf("📄 %s", truncateLabel(filepath.Base(p), 24))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, logCallbackData(i, lines)),
		))
	}

	text := fmt.Sprintf("<b>%d строк</b> — выберите файл:", lines)
	edit := tgbotapi.NewEditMessageText(
		update.CallbackQuery.Message.Chat.ID,
		update.CallbackQuery.Message.MessageID,
		text,
	)
	edit.ParseMode = tgbotapi.ModeHTML
	edit.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
	_, _ = bot.Request(edit)
}

func logCallbackData(fileIdx, lines int) string {
	return fmt.Sprintf("%s%d:%d", logCallbackPrefix, fileIdx, lines)
}

func sendLogTail(bot *tgbotapi.BotAPI, chatID int64, fileIdx int, lines int) {
	paths := LogPathsFromEnv()
	if fileIdx < 0 || fileIdx >= len(paths) {
		_, _ = bot.Send(tgbotapi.NewMessage(chatID, "❌ Неверный номер файла."))
		return
	}
	path := paths[fileIdx]

	tail, err := tailFile(path, lines)
	if err != nil {
		_, _ = bot.Send(tgbotapi.NewMessage(chatID,
			fmt.Sprintf("❌ Не удалось прочитать <code>%s</code>: %v", escapeForPre(path), err)))
		return
	}

	body := strings.Join(tail, "\n")
	if body == "" {
		body = "(файл пуст или нет строк)"
	}
	truncated := false
	if len(body) > maxLogBodyChars {
		body = body[len(body)-maxLogBodyChars:]
		body = "…\n" + body
		truncated = true
	}

	title := filepath.Base(path)
	if len(paths) > 1 {
		title = fmt.Sprintf("%d. %s", fileIdx+1, title)
	}
	note := ""
	if truncated {
		note = "\n<i>(обрезано по лимиту Telegram)</i>"
	}
	text := fmt.Sprintf("<b>📄 %s</b>\nПоследние <b>%d</b> строк%s\n\n<pre>%s</pre>",
		escapeForPre(title), lines, note, escapeForPre(body))

	if len(text) > 4096 {
		sendLogAsDocument(bot, chatID, path, tail, lines)
		return
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.DisableWebPagePreview = true
	_, _ = bot.Send(msg)
}

func sendLogAsDocument(bot *tgbotapi.BotAPI, chatID int64, path string, lines []string, count int) {
	content := strings.Join(lines, "\n")
	if content == "" {
		content = "(пусто)"
	}
	doc := tgbotapi.NewDocument(chatID, tgbotapi.FileBytes{
		Name:  filepath.Base(path) + fmt.Sprintf(".tail%d.txt", count),
		Bytes: []byte(content),
	})
	doc.Caption = fmt.Sprintf("Последние %d строк: %s", count, filepath.Base(path))
	_, err := bot.Send(doc)
	if err != nil {
		_, _ = bot.Send(tgbotapi.NewMessage(chatID, "❌ Не удалось отправить файл: "+err.Error()))
	}
}

func sendLogUsage(bot *tgbotapi.BotAPI, chatID int64, paths []string) {
	text := "<b>Использование</b>\n\n" +
		"<code>/logs</code> — меню с кнопками\n" +
		"<code>/logs 100</code> — последние 100 строк"
	if len(paths) > 1 {
		text += fmt.Sprintf("\n<code>/logs 100 2</code> — файл №2 из %d", len(paths))
	}
	text += fmt.Sprintf("\n\nМаксимум строк: <b>%d</b>", maxLogLines)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	_, _ = bot.Send(msg)
}

func sendLogSetupHelp(bot *tgbotapi.BotAPI, chatID int64) {
	msg := tgbotapi.NewMessage(chatID,
		"<b>Логи из файла</b>\n\n"+
			"Укажите в <code>.env</code> один или несколько файлов через запятую:\n"+
			"<code>BOT_LOG_PATH=/var/log/vpnbot.log</code>\n"+
			"<code>BOT_LOG_PATHS=/var/log/vpnbot.log,/var/log/vpnbot.err.log</code>\n\n"+
			"Затем: <code>/logs</code> или <code>/logs 50</code>")
	msg.ParseMode = tgbotapi.ModeHTML
	msg.DisableWebPagePreview = true
	_, _ = bot.Send(msg)
}

func tailFile(path string, n int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var ring []string
	sc := bufio.NewScanner(f)
	const maxScan = 1024 * 1024
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, maxScan)
	for sc.Scan() {
		ring = append(ring, sc.Text())
		if len(ring) > n {
			ring = ring[1:]
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return ring, nil
}

func clampLogLines(n int) int {
	if n < 1 {
		return 1
	}
	if n > maxLogLines {
		return maxLogLines
	}
	return n
}

func truncateLabel(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func escapeForPre(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	return s
}
