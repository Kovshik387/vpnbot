package user

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"VpnBot/internal/app/usecases"
	"VpnBot/internal/domain/repository"
	interfaces "VpnBot/internal/interfaces/http"
)

func GetSubscribeHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI, userUC *usecases.UserUsecase, pr *repository.PanelRepository) {
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
		t := "Не удалось загрузить подписку. Попробуйте позже или напишите администратору."
		kb := MainMenuKeyboard(userUC, uid, false)
		EditPanelFromUpdate(bot, pr, update, uid, t, &kb, true)
		return
	}

	effectiveUsername := storedUsername
	if effectiveUsername == "" {
		effectiveUsername = currentUsername
	}

	user, err := userUC.SearchUser(effectiveUsername)
	if err != nil {
		t := "Не удалось получить данные подписки. Попробуйте позже или напишите администратору."
		kb := MainMenuKeyboard(userUC, uid, false)
		EditPanelFromUpdate(bot, pr, update, uid, t, &kb, true)
		return
	}

	mb := interfaces.NewMessageBuilder()
	price, err := userUC.GetPriceByUserID(uid)
	if err != nil {
		log.Println("failed to get user price:", err)
		price = 0
	}
	response, _, err := mb.SendUserInfo(user, price)
	if err != nil {
		log.Println("Ошибка при формировании сообщения:", err)
		t := "Данные подписки получены, но не удалось оформить сообщение. Обратитесь к администратору."
		kb := MainMenuKeyboard(userUC, uid, false)
		EditPanelFromUpdate(bot, pr, update, uid, t, &kb, true)
		return
	}
	if paidUntil, err := userUC.GetPaymentDateByUserID(uid); err == nil && paidUntil != nil {
		response += fmt.Sprintf("\n<b>Оплачен до</b> %s", paidUntil.Format("02.01.2006"))
	}

	kb := MainMenuKeyboard(userUC, uid, false)
	EditPanelFromUpdate(bot, pr, update, uid, response, &kb, true)
}
