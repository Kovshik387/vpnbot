package admin

import (
	"VpnBot/internal/app/handlers/user"
	"VpnBot/internal/app/ui"
	"VpnBot/internal/app/usecases"
	"VpnBot/internal/domain/repository"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"strconv"
	"strings"
)

const pollPageSize = 6

func PollListHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase, pr *repository.PanelRepository, offset int) {
	if update.CallbackQuery != nil {
		_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
	}
	if offset < 0 {
		offset = 0
	}
	list, err := userUC.GetPollsPage(offset, pollPageSize+1)
	if err != nil {
		log.Println("poll list:", err)
		showPollPanel(update, bot, pr, "Не удалось загрузить опросы. Попробуйте позже.", pollListBackKeyboard(offset))
		return
	}
	hasMore := len(list) > pollPageSize
	if hasMore {
		list = list[:pollPageSize]
	}
	text := fmt.Sprintf("<b>📋 Опросы</b>\n\nСтраница %d\n\nВыберите опрос, чтобы посмотреть результаты.\n"+
		"Для рассылки создайте обычный анонимный опрос в Telegram и ответьте на него /say.", offset/pollPageSize+1)
	if len(list) == 0 {
		text = "<b>📋 Опросы</b>\n\nНа этой странице опросов нет.\n" +
			"Создайте обычный анонимный опрос в Telegram и ответьте на него /say."
	}
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, p := range list {
		label := []rune(strings.ReplaceAll(p.Question, "\n", " "))
		if len(label) > 36 {
			label = append(label[:36], '…')
		}
		status := "🗳️"
		if p.IsClosed {
			status = "🔒"
		} else if p.MessageID == 0 {
			status = "📁"
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(
			status+" "+string(label), fmt.Sprintf("polls:r:%d:%s", offset, p.PollID),
		)))
	}
	var navigation []tgbotapi.InlineKeyboardButton
	if offset > 0 {
		previous := offset - pollPageSize
		if previous < 0 {
			previous = 0
		}
		navigation = append(navigation, tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("polls:l:%d", previous)))
	}
	if hasMore {
		navigation = append(navigation, tgbotapi.NewInlineKeyboardButtonData("Вперёд ➡️", fmt.Sprintf("polls:l:%d", offset+pollPageSize)))
	}
	if len(navigation) > 0 {
		rows = append(rows, navigation)
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔄 Обновить", fmt.Sprintf("polls:l:%d", offset)),
	))
	showPollPanel(update, bot, pr, text, ui.WithNavFooter(tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}))
}

func PollsCallbackHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, uc *usecases.UserUsecase, pr *repository.PanelRepository) {
	q := update.CallbackQuery
	if q == nil || q.Message == nil {
		return
	}
	parts := strings.Split(q.Data, ":")
	if len(parts) >= 3 && parts[0] == "polls" {
		offset, err := strconv.Atoi(parts[2])
		if err == nil && offset >= 0 {
			switch {
			case parts[1] == "l" && len(parts) == 3:
				PollListHandler(update, bot, uc, pr, offset)
				return
			case parts[1] == "r" && len(parts) == 4 && validPollID(parts[3]):
				showPollResult(update, bot, uc, pr, parts[3], offset)
				return
			case (parts[1] == "s" || parts[1] == "c") && len(parts) == 4 && validPollID(parts[3]):
				closePoll(update, bot, uc, pr, parts[3], offset, parts[1] == "c")
				return
			}
		}
	}
	_, _ = bot.Request(tgbotapi.NewCallback(q.ID, "Некорректный опрос. Откройте /poll_list."))
}

func validPollID(id string) bool {
	if len(id) == 0 || len(id) > 32 {
		return false
	}
	for _, r := range id {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func pollListBackKeyboard(offset int) tgbotapi.InlineKeyboardMarkup {
	return ui.WithNavFooter(tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("◀️ К опросам", fmt.Sprintf("polls:l:%d", offset)),
	)))
}

func showPollPanel(update tgbotapi.Update, bot *tgbotapi.BotAPI, pr *repository.PanelRepository, text string, kb tgbotapi.InlineKeyboardMarkup) {
	var uid int64
	if update.CallbackQuery != nil && update.CallbackQuery.From != nil {
		uid = update.CallbackQuery.From.ID
	} else if update.Message != nil && update.Message.From != nil {
		uid = update.Message.From.ID
	} else {
		return
	}
	if user.EditPanelFromUpdate(bot, pr, update, uid, text, &kb, true) && update.Message != nil && update.Message.IsCommand() {
		_, _ = bot.Request(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID))
	}
}
