package admin

import (
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"VpnBot/internal/app/ui"
	"VpnBot/internal/app/usecases"
	"VpnBot/internal/domain/repository"
)

func SayCommandHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase, sayLog *repository.SayLogRepository) {
	if update.Message == nil {
		return
	}

	msg := update.Message
	var makeMsg func(chatID int64) tgbotapi.Chattable
	var logKind, logBody string

	if msg.ReplyToMessage != nil {
		src := msg.ReplyToMessage

		switch {
		case len(src.Photo) > 0:
			photos := src.Photo
			last := photos[len(photos)-1]

			caption := strings.TrimSpace(src.Caption)
			logKind = "photo"
			if caption == "" {
				logBody = "[Фото без подписи]"
			} else {
				logBody = "[Фото] " + caption
			}

			makeMsg = func(chatID int64) tgbotapi.Chattable {
				photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileID(last.FileID))
				photo.Caption = src.Caption
				kb := ui.PanelShortcutKeyboard()
				photo.ReplyMarkup = kb
				return photo
			}

		case src.Poll != nil:
			poll := src.Poll
			options := make([]string, len(poll.Options))
			for i, o := range poll.Options {
				options[i] = o.Text
			}
			logKind = "poll"
			logBody = poll.Question + "\n" + strings.Join(options, " · ")

			makeMsg = func(chatID int64) tgbotapi.Chattable {
				p := tgbotapi.NewPoll(chatID, poll.Question, options...)
				p.IsAnonymous = poll.IsAnonymous
				p.AllowsMultipleAnswers = poll.AllowsMultipleAnswers
				return p
			}

		case src.Text != "":
			text := src.Text
			logKind = "text"
			logBody = text

			makeMsg = func(chatID int64) tgbotapi.Chattable {
				m := tgbotapi.NewMessage(chatID, text)
				kb := ui.PanelShortcutKeyboard()
				m.ReplyMarkup = kb
				return m
			}

		default:
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID,
				"Не знаю, как разослать этот тип сообщения 😕\nПоддерживаются текст, фото и опросы."))
			return
		}
	} else {
		text := strings.TrimSpace(strings.TrimPrefix(msg.Text, "/say"))
		if text == "" {
			_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID,
				"Использование:\n"+
					"1) Ответь командой /say на сообщение (текст/фото/опрос), чтобы разослать его.\n"+
					"2) Или напиши: /say текст_для_рассылки"))
			return
		}

		logKind = "text"
		logBody = text

		makeMsg = func(chatID int64) tgbotapi.Chattable {
			m := tgbotapi.NewMessage(chatID, text)
			m.DisableNotification = true
			kb := ui.PanelShortcutKeyboard()
			m.ReplyMarkup = kb
			return m
		}
	}

	err := broadcastToActive(bot, userUC, makeMsg)
	if err != nil {
		log.Println("broadcast error:", err)
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID,
			"Ошибка при получении списка пользователей ❌"))
		return
	}

	if sayLog != nil && logBody != "" {
		if insErr := sayLog.Insert(logKind, logBody); insErr != nil {
			log.Println("say log insert:", insErr)
		}
	}

	_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID,
		fmt.Sprintf("Сообщение отправлено всем активным пользователям ✅\nВ лог объявлений добавлена запись (%s).", logKind)))
}

func broadcastToActive(
	bot *tgbotapi.BotAPI,
	userUC *usecases.UserUsecase,
	makeMsg func(chatID int64) tgbotapi.Chattable,
) error {
	users, err := userUC.ListActive()
	if err != nil {
		return err
	}

	for _, u := range users {
		if u.Uid == 0 {
			continue
		}

		out := makeMsg(u.Uid)
		if out == nil {
			continue
		}

		if _, sendErr := bot.Send(out); sendErr != nil {
			log.Printf("Не удалось отправить сообщение пользователю %s (%d): %v",
				u.Username, u.Uid, sendErr)
		}
	}

	return nil
}
