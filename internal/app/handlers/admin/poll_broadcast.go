package admin

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"VpnBot/internal/app/ui"
	"VpnBot/internal/app/usecases"
	"VpnBot/internal/domain/repository"
	"VpnBot/internal/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func broadcastPoll(update tgbotapi.Update, bot *tgbotapi.BotAPI, uc *usecases.UserUsecase, sl *repository.SayLogRepository) {
	m := update.Message
	source := m.ReplyToMessage.Poll
	fail := func(text string) { _, _ = bot.Send(tgbotapi.NewMessage(m.Chat.ID, text)) }
	if !source.IsAnonymous {
		fail("Для общей рассылки нужен анонимный опрос. Создайте его в Telegram и ответьте на него командой /say.")
		return
	}
	if source.Type != "regular" && source.Type != "" {
		fail("В рассылке поддерживаются обычные опросы с одним или несколькими ответами. Викторины пока не поддерживаются.")
		return
	}
	published, err := uc.FindPublishedPoll(source.ID)
	if err != nil {
		log.Println("find published poll:", err)
		fail("Не удалось загрузить опрос. Рассылка не началась; попробуйте позже.")
		return
	}
	if (published == nil && source.IsClosed) || (published != nil && published.IsClosed) ||
		(source.CloseDate > 0 && int64(source.CloseDate) <= time.Now().Unix()) {
		fail("Этот опрос уже закрыт. Создайте новый и ответьте на него командой /say.")
		return
	}
	created := published == nil
	if created {
		pollMessage, err := sendSharedPoll(bot, m.Chat.ID, source)
		if err != nil {
			log.Println("create shared poll:", err)
			fail("Не удалось создать общий опрос. Рассылка не началась; попробуйте позже.")
			return
		}
		published = pollSnapshot(pollMessage.Poll)
		published.ChatID, published.MessageID = m.Chat.ID, pollMessage.MessageID
		published.SourcePollID = source.ID
		if err := uc.SavePollResults(published); err != nil {
			log.Println("save shared poll:", err)
			_, _ = bot.Request(tgbotapi.NewStopPoll(published.ChatID, published.MessageID))
			_, _ = bot.Request(tgbotapi.NewDeleteMessage(published.ChatID, published.MessageID))
			fail("Не удалось сохранить общий опрос. Он не разослан; попробуйте позже.")
			return
		}
	}

	archiveNote := ""
	if created && sl != nil {
		var options []string
		for _, option := range published.Options {
			options = append(options, option.Text)
		}
		body := published.Question + "\n" + strings.Join(options, " · ") + "\nID: " + published.PollID
		if err := sl.Insert("poll", body); err != nil {
			log.Println("poll say log:", err)
			archiveNote = "\nНе удалось сохранить запись в объявлениях."
		}
	}
	// Forward the same poll, so every chat votes in one Telegram poll ID.
	stats, err := broadcastToActive(bot, uc, func(chatID int64) tgbotapi.Chattable {
		if chatID == published.ChatID {
			return nil // The administrator already has the original bot poll.
		}
		return tgbotapi.NewForward(chatID, published.ChatID, published.MessageID)
	})
	if err != nil {
		log.Println("poll broadcast recipients:", err)
		fail("Не удалось получить список получателей. Общий опрос сохранён; повторите /say в ответ на исходный опрос.")
		return
	}
	report := fmt.Sprintf("<b>Опрос разослан</b>\nID: <code>%s</code>\nДоставлено: %d\nНе доставлено: %d\n\nВсе получатели голосуют в одном общем опросе.%s",
		utils.HtmlEscape(published.PollID), stats.Sent, stats.Failed, archiveNote)
	if stats.Sent == 0 {
		report = fmt.Sprintf("<b>Опрос сохранён, но пользователям не доставлен</b>\nID: <code>%s</code>\nНе доставлено: %d\n\nПовторите /say в ответ на исходный опрос, чтобы повторить рассылку.%s",
			utils.HtmlEscape(published.PollID), stats.Failed, archiveNote)
	}
	reply := tgbotapi.NewMessage(m.Chat.ID, report)
	reply.ParseMode = tgbotapi.ModeHTML
	reply.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("📊 Результаты", "polls:r:0:"+published.PollID),
	))
	if _, err := bot.Send(reply); err != nil {
		log.Println("poll broadcast report:", err)
	}
}

func sendSharedPoll(bot *tgbotapi.BotAPI, chatID int64, source *tgbotapi.Poll) (tgbotapi.Message, error) {
	options := make([]struct {
		Text string `json:"text"`
	}, len(source.Options))
	for i, option := range source.Options {
		options[i].Text = option.Text
	}
	// The pinned SDK sends string options; the current Bot API uses InputPollOption objects.
	params := tgbotapi.Params{
		"chat_id": strconv.FormatInt(chatID, 10), "question": source.Question,
		"is_anonymous": "true", "type": "regular",
		"allows_multiple_answers": strconv.FormatBool(source.AllowsMultipleAnswers),
	}
	if err := params.AddInterface("options", options); err != nil {
		return tgbotapi.Message{}, err
	}
	if err := params.AddInterface("reply_markup", ui.PanelShortcutKeyboard()); err != nil {
		return tgbotapi.Message{}, err
	}
	if source.CloseDate > 0 {
		params["close_date"] = strconv.Itoa(source.CloseDate)
	} else if source.OpenPeriod > 0 {
		params["open_period"] = strconv.Itoa(source.OpenPeriod)
	}
	response, err := bot.MakeRequest("sendPoll", params)
	if err != nil {
		return tgbotapi.Message{}, err
	}
	var message tgbotapi.Message
	if err := json.Unmarshal(response.Result, &message); err != nil {
		return message, err
	}
	if message.Poll == nil || message.Poll.ID == "" || message.MessageID == 0 {
		return message, fmt.Errorf("sendPoll returned no poll message")
	}
	return message, nil
}
