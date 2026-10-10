package admin

import (
	"VpnBot/internal/app/usecases"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

func BlockUserHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase, adminId int64) {
	if update.CallbackQuery == nil {
		return
	}
	targetID, username, err := requestCallbackTarget(update.CallbackQuery, "block:")
	if err != nil {
		answerRequestCallback(bot, update.CallbackQuery, "Ошибка: некорректная заявка")
		return
	}
	exists, err := userUC.UserExist(targetID)
	if err != nil {
		log.Println("block user lookup:", err)
		answerRequestCallback(bot, update.CallbackQuery, "Не удалось проверить пользователя. Попробуйте позже.")
		return
	}
	if !exists {
		err = userUC.Insert(username, targetID)
		if err != nil {
			log.Println(err)
			answerRequestCallback(bot, update.CallbackQuery, "Не удалось сохранить пользователя. Попробуйте позже.")
			return
		}
	}

	err = userUC.Block(targetID, true)
	if err != nil {
		log.Println(err)
		answerRequestCallback(bot, update.CallbackQuery, "Не удалось заблокировать пользователя. Попробуйте позже.")
		return
	}
	if _, err := bot.Send(tgbotapi.NewMessage(targetID, "❌ Ваша заявка отклонена")); err != nil {
		log.Println("block notify:", err)
		answerRequestCallback(bot, update.CallbackQuery, "Пользователь заблокирован, но уведомление не доставлено")
		return
	}

	delMsg := tgbotapi.NewDeleteMessage(adminId, update.CallbackQuery.Message.MessageID)
	_, _ = bot.Request(delMsg)

	answerRequestCallback(bot, update.CallbackQuery, "Пользователь заблокирован")
}
