package interfaces

import (
	"VpnBot/internal/domain/model"
	"VpnBot/internal/domain/ports/service"
	"VpnBot/internal/utils"
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

func (*messageBuilder) SendUserInfo(user model.User, price float64) (string, bool, error) {
	now := time.Now()
	separator := "──────────────\n"

	dateFlag := false
	isActive := false
	onlineAt, err := time.Parse(time.RFC3339Nano, user.OnlineAt)
	if err != nil {
		onlineAt, err = time.Parse("2006-01-02T15:04:05.999999", user.OnlineAt)
	}
	if err != nil {
		log.Printf("Не удалось распарсить время для пользователя %s: %v\n", user.Username, err)
		dateFlag = true
	}
	onlineAt = onlineAt.In(time.FixedZone("MSK", 3*60*60))

	var statusStr string
	if now.Sub(onlineAt) <= 5*time.Minute {
		statusStr = "Активен сейчас"
		isActive = true
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

	priceLine := ""
	if price > 0 {
		priceLine = fmt.Sprintf("<b>Оплата</b> %.2f\n", price)
	}
	subscriptionLine := "<b>Подписка (телефон)</b> Пока недоступна\n"
	if strings.TrimSpace(user.SubscribedUrl) != "" {
		subscriptionLine = fmt.Sprintf("<b>Подписка (телефон)</b> <a href=\"%s\">открыть</a>\n", utils.HtmlEscape(user.SubscribedUrl))
	}
	configLine := "<b>Конфиг (ПК)</b> Пока недоступен\n"
	for _, link := range user.Links {
		if strings.TrimSpace(link) != "" {
			configLine = fmt.Sprintf("<b>Конфиг (ПК)</b> <code>%s</code>\n", utils.HtmlEscape(link))
			break
		}
	}

	response := fmt.Sprintf(
		"%s<b>Логин</b> %s\n"+
			"<b>Статус</b> %s\n"+
			"<b>Последняя активность</b> %s\n"+
			"<b>Трафик</b> %.1f ГБ\n"+
			"%s"+
			"%s"+
			"%s"+
			"%s",
		separator,
		utils.HtmlEscape(user.Username),
		utils.HtmlEscape(statusStr),
		utils.HtmlEscape(onlineAtStr),
		usage,
		priceLine,
		subscriptionLine,
		configLine,
		separator,
	)

	return response, isActive, nil
}
