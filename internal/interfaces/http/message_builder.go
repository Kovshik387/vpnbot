package interfaces

import (
	"VpnBot/internal/domain/model"
	"VpnBot/internal/domain/ports/service"
	"fmt"
	"log"
	"strings"
	"time"
)

type messageBuilder struct {
}

func NewMessageBuilder() service.MessageService {
	return &messageBuilder{}
}

func (*messageBuilder) SendUserInfo(user model.User) (string, error) {
	now := time.Now().Add(3 * time.Hour)
	separator := "----------------------------------------\n"

	dateFlag := false

	onlineAt, err := time.Parse("2006-01-02T15:04:05.999999", user.OnlineAt)
	if err != nil {
		log.Printf("Не удалось распарсить время для пользователя %s: %v\n", user.Username, err)
		dateFlag = true
	}
	onlineAt = onlineAt.Add(3 * time.Hour)

	var statusStr string
	if now.Sub(onlineAt) <= 5*time.Minute {
		statusStr = "Активен сейчас"
	} else {
		duration := now.Sub(onlineAt)
		days := int(duration.Hours()) / 24
		hours := int(duration.Hours()) % 24
		minutes := int(duration.Minutes()) % 60
		statusStr = fmt.Sprintf("%d д %d ч %d мин назад", days, hours, minutes)
	}

	var onlineAtStr string
	if dateFlag {
		onlineAtStr = "Нет информации"
		statusStr = "Не подключен"
	} else {
		onlineAtStr = onlineAt.Format("02.01.2006 15:04")
	}

	log.Println(user.UsedTraffic)
	usage := float64(user.UsedTraffic) / (1024 * 1024 * 1024)

	response := fmt.Sprintf(
		"%s<b>Username:</b> %s\n"+
			"<b>Status:</b> %s\n"+
			"<b>OnlineAt:</b> %s\n"+
			"<b>Usage:</b> %.1f GB\n"+
			"<b>Sub URL:</b> <a href=\"%s\">ссылка</a>\n"+
			"<b>Link:</b> <tg-spoiler><code>%s</code></tg-spoiler>\n"+
			"%s",
		separator,
		htmlEscape(user.Username),
		htmlEscape(statusStr),
		htmlEscape(onlineAtStr),
		usage,
		user.SubscribedUrl,
		user.Links[0],
		separator,
	)

	return response, nil
}

func htmlEscape(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
	)
	return replacer.Replace(s)
}
