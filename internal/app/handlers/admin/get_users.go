package admin

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"VpnBot/internal/app/handlers/user"
	"VpnBot/internal/app/ui"
	"VpnBot/internal/app/usecases"
	"VpnBot/internal/domain/model"
	"VpnBot/internal/domain/repository"
	interfaces "VpnBot/internal/interfaces/http"
	"VpnBot/internal/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const usersPageSize = 8

func UserListHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, uc *usecases.UserUsecase, pr *repository.PanelRepository, query string, activeOnly bool) {
	state := repository.UserListState{Query: strings.TrimSpace(query), ActiveOnly: activeOnly}
	showUsersPage(update, bot, uc, pr, state)
}

// Permission is checked by the callback router before this handler is called.
func UsersCallbackHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, uc *usecases.UserUsecase, pr *repository.PanelRepository) {
	q := update.CallbackQuery
	if q == nil || q.From == nil || q.Message == nil {
		return
	}
	if q.Data == "users:open" {
		_, _ = bot.Request(tgbotapi.NewCallback(q.ID, ""))
		UserListHandler(update, bot, uc, pr, "", false)
		return
	}
	state, err := pr.GetUserListState(q.From.ID)
	if err != nil {
		log.Println("user list state:", err)
		_, _ = bot.Request(tgbotapi.NewCallback(q.ID, "Не удалось загрузить список. Откройте /users."))
		return
	}
	parts := strings.Split(q.Data, ":")
	if len(parts) < 3 || len(parts) > 4 || state.Token == "" || parts[1] != state.Token {
		_, _ = bot.Request(tgbotapi.NewCallback(q.ID, "Этот список устарел. Откройте /users."))
		return
	}
	if parts[2] == "search" && len(parts) == 3 {
		answer := tgbotapi.NewCallback(q.ID, "Введите /users часть_логина\nНапример: /users f*** — логины, начинающиеся с f.\nБез * поиск находит совпадение в любой части логина.")
		answer.ShowAlert = true
		_, _ = bot.Request(answer)
		return
	}
	_, _ = bot.Request(tgbotapi.NewCallback(q.ID, ""))
	switch parts[2] {
	case "page":
		if len(parts) != 4 {
			return
		}
		offset, err := strconv.Atoi(parts[3])
		if err != nil || offset < 0 || offset%usersPageSize != 0 {
			return
		}
		state.Offset = offset
	case "filter":
		if len(parts) != 4 || (parts[3] != "all" && parts[3] != "active") {
			return
		}
		state.ActiveOnly = parts[3] == "active"
		state.Offset = 0
	case "clear":
		if len(parts) != 3 {
			return
		}
		state.Query, state.Offset = "", 0
	case "back", "refresh":
		if len(parts) != 3 {
			return
		}
	case "detail":
		if len(parts) != 4 {
			return
		}
		index, err := strconv.Atoi(parts[3])
		if err != nil || index < 0 || index >= len(state.Usernames) {
			return
		}
		showUserDetail(update, bot, uc, pr, state, state.Usernames[index])
		return
	default:
		return
	}
	showUsersPage(update, bot, uc, pr, state)
}

func showUsersPage(update tgbotapi.Update, bot *tgbotapi.BotAPI, uc *usecases.UserUsecase, pr *repository.PanelRepository, state repository.UserListState) {
	uid := usersPanelOwner(update)
	entries, total, err := uc.ListUserPage(state.Query, state.ActiveOnly, state.Offset, usersPageSize)
	if err != nil {
		log.Println("user list:", err)
		showUsersError(update, bot, pr, "Не удалось загрузить пользователей. Проверьте запрос и попробуйте ещё раз: /users или /users часть_логина.")
		return
	}
	if total == 0 {
		state.Offset = 0
	} else if state.Offset >= total {
		state.Offset = (total - 1) / usersPageSize * usersPageSize
	}
	var token [8]byte
	if _, err := rand.Read(token[:]); err != nil {
		log.Println("user list token:", err)
		showUsersError(update, bot, pr, "Не удалось открыть список. Попробуйте /users ещё раз.")
		return
	}
	state.Token = hex.EncodeToString(token[:])
	state.Usernames = make([]string, len(entries))
	callback := func(action string) string { return "users:" + state.Token + ":" + action }
	allLabel, activeLabel := "Все", "Активные"
	if state.ActiveOnly {
		activeLabel = "✓ " + activeLabel
	} else {
		allLabel = "✓ " + allLabel
	}
	rows := [][]tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(allLabel, callback("filter:all")),
		tgbotapi.NewInlineKeyboardButtonData(activeLabel, callback("filter:active")),
	)}
	text := fmt.Sprintf("<b>👥 Пользователи</b>\n\nНайдено: <b>%d</b> · страница %d из %d\n", total, state.Offset/usersPageSize+1, max(1, (total+usersPageSize-1)/usersPageSize))
	if state.Query != "" {
		text += "Поиск: <code>" + utils.HtmlEscape(state.Query) + "</code>\n"
	}
	if state.ActiveOnly {
		text += "Фильтр: подключались за последние 5 минут.\n"
	}
	text += "\n"
	now := time.Now()
	for i, entry := range entries {
		state.Usernames[i] = entry.Username
		number := state.Offset + i + 1
		text += fmt.Sprintf("%d. %s <b>%s</b> · %.1f ГБ\n", number, userStatusIcon(entry, now), utils.HtmlEscape(shortUsername(entry.Username, 40)), float64(entry.UsedTraffic)/(1024*1024*1024))
		button := tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%d. %s", number, shortUsername(entry.Username, 24)), callback("detail:"+strconv.Itoa(i)))
		if i%2 == 0 {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(button))
		} else {
			rows[len(rows)-1] = append(rows[len(rows)-1], button)
		}
	}
	if total == 0 {
		text += "Совпадений нет. Измените запрос или сбросьте поиск.\n"
	} else {
		text += "\n🟢 Онлайн · ⚪ Не в сети · 🔴 Доступ ограничен\nВыберите пользователя, чтобы открыть подробности.\n"
	}
	var nav []tgbotapi.InlineKeyboardButton
	if state.Offset > 0 {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", callback("page:"+strconv.Itoa(max(0, state.Offset-usersPageSize)))))
	}
	nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("🔄 Обновить", callback("refresh")))
	if state.Offset+len(entries) < total {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("Вперёд ➡️", callback("page:"+strconv.Itoa(state.Offset+usersPageSize))))
	}
	rows = append(rows, nav)
	searchRow := tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔎 Поиск", callback("search")))
	if state.Query != "" {
		searchRow = append(searchRow, tgbotapi.NewInlineKeyboardButtonData("✖ Сбросить поиск", callback("clear")))
	}
	rows = append(rows, searchRow)
	kb := ui.WithNavFooter(tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows})
	previousState, err := pr.GetUserListState(uid)
	if err != nil {
		log.Println("read user list state:", err)
		showUsersError(update, bot, pr, "Не удалось загрузить поиск. Попробуйте /users ещё раз.")
		return
	}
	if err := pr.SetUserListState(uid, state); err != nil {
		log.Println("save user list state:", err)
		showUsersError(update, bot, pr, "Не удалось сохранить поиск. Попробуйте /users ещё раз.")
		return
	}
	if !user.EditPanelFromUpdate(bot, pr, update, uid, text, &kb, true) {
		if err := pr.SetUserListState(uid, previousState); err != nil {
			log.Println("restore user list state:", err)
		}
		return
	}
	if update.Message != nil {
		_, _ = bot.Request(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID))
	}
}

func showUserDetail(update tgbotapi.Update, bot *tgbotapi.BotAPI, uc *usecases.UserUsecase, pr *repository.PanelRepository, state repository.UserListState, username string) {
	entry, err := uc.SearchUser(username)
	text := "Пользователь больше не доступен. Вернитесь к списку и обновите его."
	if err == nil && entry.Username != "" {
		price, priceErr := uc.GetPriceByUsername(username)
		if priceErr != nil {
			log.Println("user detail price:", priceErr)
		}
		text, _, err = interfaces.NewMessageBuilder().SendUserInfo(entry, price)
		if err == nil {
			text += "\n<b>Статус VPN:</b> " + utils.HtmlEscape(entry.Status)
			if paidUntil, dateErr := uc.GetPaymentDateByUsername(username); dateErr == nil && paidUntil != nil {
				text += "\n<b>Оплачен до:</b> " + paidUntil.Format("02.01.2006")
			}
		}
	} else if err != nil {
		log.Println("user detail:", err)
	}
	if err != nil {
		text = "Не удалось загрузить пользователя. Вернитесь к списку и попробуйте ещё раз."
	}
	kb := ui.WithNavFooter(tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("◀️ К результатам", "users:"+state.Token+":back"),
	)))
	user.EditPanelFromUpdate(bot, pr, update, usersPanelOwner(update), text, &kb, true)
}

func showUsersError(update tgbotapi.Update, bot *tgbotapi.BotAPI, pr *repository.PanelRepository, text string) {
	kb := ui.WithNavFooter(tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔄 Открыть список", "users:open"),
	)))
	user.EditPanelFromUpdate(bot, pr, update, usersPanelOwner(update), text, &kb, true)
}

func usersPanelOwner(update tgbotapi.Update) int64 {
	if update.CallbackQuery != nil && update.CallbackQuery.From != nil {
		return update.CallbackQuery.From.ID
	}
	if update.Message != nil && update.Message.From != nil {
		return update.Message.From.ID
	}
	return 0
}

func userStatusIcon(entry model.User, now time.Time) string {
	if entry.Status != "active" {
		return "🔴"
	}
	if usecases.UserIsOnline(entry, now) {
		return "🟢"
	}
	return "⚪"
}

func shortUsername(username string, limit int) string {
	runes := []rune(username)
	if len(runes) > limit {
		return string(runes[:limit-1]) + "…"
	}
	return username
}
