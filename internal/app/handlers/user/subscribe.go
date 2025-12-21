package user

import (
	"VpnBot/internal/app/ui"
	"VpnBot/internal/app/usecases"
	interfaces "VpnBot/internal/interfaces/http"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

func GetSubscribeHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase) {
	var (
		uid             int64
		currentUsername string
	)

	switch {
	case update.CallbackQuery != nil:
		uid = update.CallbackQuery.From.ID
		currentUsername = update.CallbackQuery.From.UserName

		_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))

	case update.Message != nil:
		uid = update.Message.From.ID
		currentUsername = update.Message.From.UserName

	default:
		return
	}

	if currentUsername == "" {
		currentUsername = fmt.Sprintf("id_%d", uid)
	}

	storedUsername, err := userUC.GetUserByUserId(uid)
	if err != nil {
		log.Println("failed to get stored username by user id:", err)
		msg := tgbotapi.NewMessage(uid, "Ошибка при получении подписки")
		msg.ReplyMarkup = ui.MainKeyboard()
		_, _ = bot.Send(msg)
		return
	}

	effectiveUsername := storedUsername
	if effectiveUsername == "" {
		effectiveUsername = currentUsername
	}

	user, err := userUC.SearchUser(effectiveUsername)
	if err != nil {
		msg := tgbotapi.NewMessage(uid, "Ошибка при получении подписки")
		msg.ReplyMarkup = ui.MainKeyboard()
		_, _ = bot.Send(msg)
		return
	}

	mb := interfaces.NewMessageBuilder()
	response, _, err := mb.SendUserInfo(user)
	if err != nil {
		log.Println("Ошибка при формировании сообщения:", err)
		msg := tgbotapi.NewMessage(uid, "Ошибка при формировании сообщения")
		msg.ReplyMarkup = ui.MainKeyboard()
		_, _ = bot.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(uid, response)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = ui.MainKeyboard()
	_, _ = bot.Send(msg)

	InfoHandler(update, bot)
}
