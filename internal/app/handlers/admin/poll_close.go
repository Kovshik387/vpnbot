package admin

import (
	"encoding/json"
	"fmt"
	"log"

	"VpnBot/internal/app/ui"
	"VpnBot/internal/app/usecases"
	"VpnBot/internal/domain/repository"
	"VpnBot/internal/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func closePoll(update tgbotapi.Update, bot *tgbotapi.BotAPI, uc *usecases.UserUsecase, pr *repository.PanelRepository, id string, offset int, confirm bool) {
	_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
	back := fmt.Sprintf("polls:r:%d:%s", offset, id)
	kb := ui.WithNavFooter(tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("◀️ К результатам", back),
	)))
	published, err := uc.FindPublishedPoll(id)
	if err != nil {
		log.Println("poll close lookup:", err)
		showPollPanel(update, bot, pr, "Не удалось загрузить опрос. Попробуйте позже.", kb)
		return
	}
	if published == nil {
		showPollPanel(update, bot, pr, "Закрыть можно только общий опрос, созданный новой версией бота.", kb)
		return
	}
	if published.IsClosed {
		renderPollResult(update, bot, uc, pr, published.PollID, offset)
		return
	}
	if !confirm {
		text := "<b>Закрыть опрос?</b>\n\n" + utils.HtmlEscape(published.Question) +
			"\n\nГолосование завершится у всех получателей. Результаты сохранятся, открыть этот опрос заново нельзя."
		kb.InlineKeyboard = append([][]tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏹ Да, закрыть", fmt.Sprintf("polls:c:%d:%s", offset, published.PollID)),
		)}, kb.InlineKeyboard...)
		showPollPanel(update, bot, pr, text, kb)
		return
	}
	response, err := bot.Request(tgbotapi.NewStopPoll(published.ChatID, published.MessageID))
	if err != nil {
		log.Println("stop poll:", err)
		showPollPanel(update, bot, pr, "Не удалось закрыть опрос. Обновите результаты и попробуйте ещё раз.", kb)
		return
	}
	var poll tgbotapi.Poll
	if err := json.Unmarshal(response.Result, &poll); err != nil || poll.ID != published.PollID {
		log.Printf("stop poll %s response: %v", published.PollID, err)
		showPollPanel(update, bot, pr, "Опрос закрыт, но не удалось получить итоговые результаты. Попробуйте обновить их позже.", kb)
		return
	}
	if err := uc.UpdatePollResults(pollSnapshot(&poll)); err != nil {
		log.Println("save closed poll:", err)
		showPollPanel(update, bot, pr, "Опрос закрыт, но результаты не удалось сохранить. Попробуйте обновить их позже.", kb)
		return
	}
	renderPollResult(update, bot, uc, pr, published.PollID, offset)
}
