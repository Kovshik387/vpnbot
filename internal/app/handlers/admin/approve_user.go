package admin

import (
	interfaces "VpnBot/internal/interfaces/http"
	"log"
	"strconv"
	"strings"

	"VpnBot/internal/app/usecases"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func ApproveHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase, adminId int64) {
	param := strings.TrimPrefix(update.CallbackQuery.Data, "approve:")
	id := strings.Split(param, " ")
	targetID, err := strconv.ParseInt(id[0], 10, 64)
	if err != nil {
		_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Ошибка: некорректный id"))
		return
	}

	parts := strings.Split(update.CallbackQuery.Data, ":")
	if len(parts) != 2 {
		return
	}

	query := strings.Split(parts[1], " ")

	userId, _ := strconv.Atoi(query[0])

	user, err := userUC.AddUser(query[1])
	if err != nil {
		msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Ошибка при выдаче ключа: "+err.Error())
		_, _ = bot.Send(tgbotapi.NewMessage(targetID, "Ошибка при выдаче ключа, возможно ключ уже выдан"))
		_, _ = bot.Send(msg)
		return
	}

	if flag, _ := userUC.UserExist(targetID); !flag {
		err = userUC.Insert(user.Username, int64(userId))
		if err != nil {
			msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Ошибка при выдаче ключа: "+err.Error())
			_, _ = bot.Send(tgbotapi.NewMessage(targetID, "Ошибка при выдаче ключа, возможно ключ уже выдан"))
			_, _ = bot.Send(msg)
			log.Println("Не удалось добавить пользователя в бд", err)
			return
		}
	}

	info, err := interfaces.NewMessageBuilder().SendUserInfo(user)
	if err != nil {
		return
	}

	callback := tgbotapi.NewMessage(int64(userId), "✅ Ключ выдан")
	_, _ = bot.Request(callback)

	msgToUser := tgbotapi.NewMessage(int64(userId), info)
	msgToUser.ParseMode = "Html"
	_, _ = bot.Send(msgToUser)

	delMsg := tgbotapi.NewDeleteMessage(adminId, update.CallbackQuery.Message.MessageID)
	_, _ = bot.Request(delMsg)

	_, _ = bot.Request(tgbotapi.NewMessage(adminId, "Пользователь одобрен: "+user.Username+" ✅"))
	_, _ = bot.Send(tgbotapi.NewEditMessageReplyMarkup(adminId, update.CallbackQuery.Message.MessageID,
		tgbotapi.InlineKeyboardMarkup{}))
}
