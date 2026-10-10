package admin

import (
	"VpnBot/internal/app/ui"
	interfaces "VpnBot/internal/interfaces/http"
	"fmt"
	"log"
	"strconv"
	"strings"

	"VpnBot/internal/app/usecases"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func ApproveHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase, adminId int64) {
	if update.CallbackQuery == nil {
		return
	}
	targetID, username, err := requestCallbackTarget(update.CallbackQuery, "approve:")
	if err != nil {
		answerRequestCallback(bot, update.CallbackQuery, "Ошибка: некорректная заявка")
		return
	}
	exists, err := userUC.UserExist(targetID)
	if err != nil {
		log.Println("approve user lookup:", err)
		answerRequestCallback(bot, update.CallbackQuery, "Не удалось проверить пользователя. Попробуйте позже.")
		return
	}
	if exists {
		answerRequestCallback(bot, update.CallbackQuery, "Пользователь уже есть в базе. Проверьте /users.")
		_, _ = bot.Request(tgbotapi.NewDeleteMessage(adminId, update.CallbackQuery.Message.MessageID))
		return
	}
	user, err := userUC.AddUser(username)
	if err != nil {
		log.Println("approve create user:", err)
		answerRequestCallback(bot, update.CallbackQuery, "Не удалось создать ключ. Попробуйте позже.")
		return
	}
	if err := userUC.Insert(user.Username, targetID); err != nil {
		log.Println("approve persist user:", err)
		// AddUser успешно создал эту запись в текущем запросе; убираем её при сбое локальной привязки.
		if rollbackErr := userUC.DeleteUser(user.Username); rollbackErr != nil {
			log.Println("approve rollback created user:", rollbackErr)
			answerRequestCallback(bot, update.CallbackQuery, "Ошибка сохранения. Проверьте пользователя на VPN-сервере.")
			return
		}
		answerRequestCallback(bot, update.CallbackQuery, "Не удалось сохранить пользователя. Попробуйте позже.")
		return
	}

	price, err := userUC.GetPriceByUserID(targetID)
	if err != nil {
		log.Println("approve price:", err)
		price = 0
	}

	info, _, err := interfaces.NewMessageBuilder().SendUserInfo(user, price)
	if err != nil {
		log.Println("approve build info:", err)
		answerRequestCallback(bot, update.CallbackQuery, "Ключ создан, но данные не удалось подготовить. Проверьте подписку.")
		return
	}

	kb := ui.PanelShortcutKeyboard()
	msgToUser := tgbotapi.NewMessage(targetID, "✅ Ключ выдан\n\n"+info)
	msgToUser.ParseMode = tgbotapi.ModeHTML
	msgToUser.ReplyMarkup = kb
	msgToUser.DisableWebPagePreview = true
	if _, err := bot.Send(msgToUser); err != nil {
		log.Println("approve send info:", err)
		answerRequestCallback(bot, update.CallbackQuery, "Ключ создан, но сообщение не доставлено. Пользователь может открыть /subscribe.")
		return
	}

	delMsg := tgbotapi.NewDeleteMessage(adminId, update.CallbackQuery.Message.MessageID)
	_, _ = bot.Request(delMsg)

	_, _ = bot.Request(tgbotapi.NewMessage(adminId, "Пользователь одобрен: "+user.Username+" ✅"))
	answerRequestCallback(bot, update.CallbackQuery, "Ключ выдан")
}

func requestCallbackTarget(query *tgbotapi.CallbackQuery, prefix string) (int64, string, error) {
	if query == nil || query.Message == nil || !strings.HasPrefix(query.Data, prefix) {
		return 0, "", fmt.Errorf("invalid callback")
	}
	parts := strings.Fields(strings.TrimPrefix(query.Data, prefix))
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("expected id and username")
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id <= 0 {
		return 0, "", fmt.Errorf("invalid user id")
	}
	for _, r := range parts[1] {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
			return 0, "", fmt.Errorf("invalid username")
		}
	}
	return id, parts[1], nil
}

func answerRequestCallback(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery, text string) {
	_, _ = bot.Request(tgbotapi.NewCallback(query.ID, text))
}
