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
			broadcastPoll(update, bot, userUC, sayLog)
			return

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
		text := strings.TrimSpace(msg.CommandArguments())
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

	stats, err := broadcastToActive(bot, userUC, makeMsg)
	if err != nil {
		log.Println("broadcast error:", err)
		_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID,
			"Ошибка при получении списка пользователей ❌"))
		return
	}

	archived := false
	if sayLog != nil && logBody != "" && stats.Sent > 0 {
		if insErr := sayLog.Insert(logKind, logBody); insErr != nil {
			log.Println("say log insert:", insErr)
		} else {
			archived = true
		}
	}

	report := fmt.Sprintf("Рассылка завершена.\nДоставлено: %d\nНе доставлено: %d", stats.Sent, stats.Failed)
	if archived {
		report += "\nДобавлено в объявления."
	} else if stats.Sent > 0 {
		report += "\nНе удалось сохранить запись в объявлениях."
	}
	_, _ = bot.Send(tgbotapi.NewMessage(msg.Chat.ID, report))
}

type broadcastStats struct {
	Sent   int
	Failed int
}

func broadcastToActive(
	bot *tgbotapi.BotAPI,
	userUC *usecases.UserUsecase,
	makeMsg func(chatID int64) tgbotapi.Chattable,
) (broadcastStats, error) {
	var stats broadcastStats
	users, err := userUC.ListActive()
	if err != nil {
		return stats, err
	}

	seen := make(map[int64]bool)
	for _, u := range users {
		if u.Uid == 0 || seen[u.Uid] {
			continue
		}
		seen[u.Uid] = true

		out := makeMsg(u.Uid)
		if out == nil {
			continue
		}

		if _, sendErr := bot.Send(out); sendErr != nil {
			stats.Failed++
			log.Printf("Не удалось отправить сообщение пользователю %s (%d): %v",
				u.Username, u.Uid, sendErr)
		} else {
			stats.Sent++
		}
	}

	return stats, nil
}
