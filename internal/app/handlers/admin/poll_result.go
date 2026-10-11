package admin

import (
	"VpnBot/internal/app/usecases"
	"VpnBot/internal/domain/repository"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"strings"
)

func PollResultHandler(
	update tgbotapi.Update,
	bot *tgbotapi.BotAPI,
	userUC *usecases.UserUsecase,
	pollID string,
	pr *repository.PanelRepository,
) {
	showPollResult(update, bot, userUC, pr, strings.TrimSpace(pollID), 0)
}

func showPollResult(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase, pr *repository.PanelRepository, pollID string, offset int) {
	if update.CallbackQuery != nil {
		_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
	}
	renderPollResult(update, bot, userUC, pr, pollID, offset)
}

func renderPollResult(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase, pr *repository.PanelRepository, pollID string, offset int) {
	kb := pollListBackKeyboard(offset)
	if !validPollID(pollID) {
		showPollPanel(update, bot, pr, "Укажите ID опроса из /poll_list: <code>/poll_result ID</code>.", kb)
		return
	}
	text, err := userUC.PollResult(pollID)
	if err != nil {
		log.Println("PollResult error:", err)
		showPollPanel(update, bot, pr, "Не удалось загрузить результаты. Попробуйте позже.", kb)
		return
	}

	if text == "" {
		showPollPanel(update, bot, pr, "Опрос с таким ID не найден. Откройте /poll_list и выберите опрос.", kb)
		return
	}

	kb.InlineKeyboard = append([][]tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔄 Обновить результаты", fmt.Sprintf("polls:r:%d:%s", offset, pollID)),
	)}, kb.InlineKeyboard...)
	published, err := userUC.FindPublishedPoll(pollID)
	if err != nil {
		log.Println("poll controls:", err)
	} else if published != nil && !published.IsClosed {
		kb.InlineKeyboard = append([][]tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏹ Закрыть опрос", fmt.Sprintf("polls:s:%d:%s", offset, pollID)),
		)}, kb.InlineKeyboard...)
	}
	showPollPanel(update, bot, pr, text, kb)
}
