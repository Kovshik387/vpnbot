package admin

import (
	"VpnBot/internal/app/usecases"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"strings"
)

func SayCommandHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase) {
	if update.Message == nil {
		return
	}

	msg := update.Message
	var makeMsg func(chatID int64) tgbotapi.Chattable

	if msg.ReplyToMessage != nil {
		src := msg.ReplyToMessage

		switch {
		case len(src.Photo) > 0:
			photos := src.Photo
			last := photos[len(photos)-1]

			caption := src.Caption

			makeMsg = func(chatID int64) tgbotapi.Chattable {
				photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileID(last.FileID))
				photo.Caption = caption
				photo.DisableNotification = true
				return photo
			}

		case src.Poll != nil:
			poll := src.Poll
			options := make([]string, len(poll.Options))
			for i, o := range poll.Options {
				options[i] = o.Text
			}

			makeMsg = func(chatID int64) tgbotapi.Chattable {
				p := tgbotapi.NewPoll(chatID, poll.Question, options...)
				p.IsAnonymous = poll.IsAnonymous
				p.AllowsMultipleAnswers = poll.AllowsMultipleAnswers
				p.DisableNotification = true
				return p
			}

		case src.Text != "":
			text := src.Text

			makeMsg = func(chatID int64) tgbotapi.Chattable {
				m := tgbotapi.NewMessage(chatID, text)
				m.DisableNotification = true
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

		makeMsg = func(chatID int64) tgbotapi.Chattable {
			m := tgbotapi.NewMessage(chatID, text)
			m.DisableNotification = true
			return m
		}
	}

	// Общая рассылка
	err := broadcastToActive(bot, userUC, makeMsg)
	if err != nil {
		log.Println("broadcast error:", err)
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID,
			"Ошибка при получении списка пользователей ❌"))
		return
	}

	_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID,
		"Сообщение отправлено всем активным пользователям ✅"))
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

		msg := makeMsg(u.Uid)
		if msg == nil {
			continue
		}

		if _, sendErr := bot.Send(msg); sendErr != nil {
			log.Printf("Не удалось отправить сообщение пользователю %s (%d): %v",
				u.Username, u.Uid, sendErr)
		}
	}

	return nil
}
