package user

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"net"
	"time"
)

func Ping(update tgbotapi.Update, bot *tgbotapi.BotAPI, yBlockUrl string) {
	timeout := 2 * time.Second
	conn, err := net.DialTimeout("tcp", yBlockUrl, timeout)
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "🇷🇺 Сервер недоступен ❌")
		_, _ = bot.Send(msg)
		return
	}
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(conn)

	fmt.Println("✅ Сервер доступен:", yBlockUrl)

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "🇷🇺 Сервер доступен ✅")
	_, _ = bot.Send(msg)
}
