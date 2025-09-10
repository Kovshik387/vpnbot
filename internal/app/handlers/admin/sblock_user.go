package admin

import (
	"VpnBot/internal/app/usecases"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"strconv"
	"strings"
)

func BlockUserHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase, adminId int64) {
	param := strings.TrimPrefix(update.CallbackQuery.Data, "block:")
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

	_, _ = bot.Send(tgbotapi.NewMessage(targetID, "❌ Ваша заявка отклонена"))
	username := strings.Split(parts[1], " ")[1]

	if flag, _ := userUC.UserExist(targetID); !flag {
		err = userUC.Insert(username, targetID)
		if err != nil {
			log.Println(err)
			return
		}
	}

	err = userUC.Block(targetID, true)
	if err != nil {
		log.Println(err)
		return
	}

	delMsg := tgbotapi.NewDeleteMessage(adminId, update.CallbackQuery.Message.MessageID)
	_, _ = bot.Request(delMsg)

	_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Заявка отклонена"))
	_, _ = bot.Send(tgbotapi.NewEditMessageReplyMarkup(adminId, update.CallbackQuery.Message.MessageID,
		tgbotapi.InlineKeyboardMarkup{}))
}
