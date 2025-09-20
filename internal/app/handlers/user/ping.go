package user

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"io"
	"log"
	"net/http"
	"time"
)

func PingHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, yBlockUrl string, adminId int64) {
	var chatId int64

	if update.Message != nil {
		chatId = update.Message.Chat.ID
	} else if update.CallbackQuery != nil {
		chatId = update.CallbackQuery.Message.Chat.ID
	}

	client := &http.Client{
		Timeout: 1 * time.Second,
	}

	resp, err := client.Get(yBlockUrl)
	if err != nil {
		log.Println(err)
		sendMessage(bot, chatId, "🇷🇺 Сервер недоступен ❌")
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println(err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		sendMessage(bot, chatId, "🇷🇺 Сервер недоступен ❌")
	}

	sendMessage(bot, chatId, "🇦🇱 Сервер доступен ✅")
	HelpHandler(update, bot, adminId)
}

func sendMessage(bot *tgbotapi.BotAPI, chatId int64, mes string) {
	msg := tgbotapi.NewMessage(chatId, mes)
	_, _ = bot.Send(msg)
}
