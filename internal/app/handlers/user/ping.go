package user

import (
	"io"
	"log"
	"net/http"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"VpnBot/internal/app/ui"
	"VpnBot/internal/domain/repository"
)

func PingHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, yBlockUrl string, adminId int64, pr *repository.PanelRepository) {
	_ = adminId
	var uid int64
	if update.Message != nil {
		uid = update.Message.From.ID
	} else if update.CallbackQuery != nil {
		_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
		uid = update.CallbackQuery.From.ID
	} else {
		return
	}

	client := &http.Client{
		Timeout: 1 * time.Second,
	}

	var result string
	resp, err := client.Get(yBlockUrl)
	if err != nil {
		log.Println(err)
		result = "🇷🇺 <b>RU-сервер:</b> не удалось достучаться (таймаут или сеть)."
	} else {
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				log.Println(err)
			}
		}(resp.Body)

		if resp.StatusCode != http.StatusOK {
			result = "🇷🇺 <b>RU-сервер:</b> недоступен (код " + resp.Status + ")"
		} else {
			result = "🇷🇺 <b>RU-сервер:</b> отвечает, соединение в порядке."
		}
	}

	kb := ui.HelpActionsKeyboard()
	EditPanelFromUpdate(bot, pr, update, uid, result, &kb, true)
}
