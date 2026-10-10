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
	var uid int64

	switch {
	case update.CallbackQuery != nil:
		uid = update.CallbackQuery.From.ID

		_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))

	case update.Message != nil:
		uid = update.Message.From.ID

	default:
		return
	}

	revoked, err := userUC.CheckPaymentRevoked(uid)
	if err != nil {
		log.Println("failed to get subscription access:", err)
		kb := MainMenuKeyboard(userUC, uid, false)
		EditPanelFromUpdate(bot, pr, update, uid, "Не удалось загрузить подписку. Попробуйте позже или напишите администратору.", &kb, true)
		return
	}
	if revoked {
		text := "<b>Моя подписка</b>\n\nДоступ приостановлен из-за просроченной оплаты. Нажмите «Оплата» и отправьте чек администратору."
		if paidUntil, err := userUC.GetPaymentDateByUserID(uid); err == nil && paidUntil != nil {
			text += fmt.Sprintf("\n\n<b>Дата оплаты:</b> %s", paidUntil.Format("02.01.2006"))
		}
		kb := MainMenuKeyboard(userUC, uid, false)
		EditPanelFromUpdate(bot, pr, update, uid, text, &kb, true)
		return
	}

	storedUsername, err := userUC.GetUserByUserId(uid)
	if err != nil {
		log.Println("failed to get stored username by user id:", err)
		t := "Не удалось загрузить подписку. Попробуйте позже или напишите администратору."
		kb := MainMenuKeyboard(userUC, uid, false)
		EditPanelFromUpdate(bot, pr, update, uid, t, &kb, true)
		return
	}

	if storedUsername == "" {
		t := "<b>Моя подписка</b>\n\nУ вас пока нет выданного ключа. Нажмите «Запросить ключ» — администратор рассмотрит заявку."
		kb := MainMenuKeyboard(userUC, uid, true)
		EditPanelFromUpdate(bot, pr, update, uid, t, &kb, true)
		return
	}

	user, err := userUC.SearchUser(storedUsername)
	if err != nil {
		t := "Не удалось получить данные подписки. Попробуйте позже или напишите администратору."
		kb := MainMenuKeyboard(userUC, uid, false)
		EditPanelFromUpdate(bot, pr, update, uid, t, &kb, true)
		return
	}

	mb := interfaces.NewMessageBuilder()
	paid, err := userUC.IsPaidSubscription(uid)
	if err != nil {
		log.Println("failed to get subscription payment type:", err)
		kb := MainMenuKeyboard(userUC, uid, false)
		EditPanelFromUpdate(bot, pr, update, uid, "Не удалось загрузить условия подписки. Попробуйте позже или напишите администратору.", &kb, true)
		return
	}
	price := 0.0
	if paid {
		price, err = userUC.GetPriceByUserID(uid)
		if err != nil {
			log.Println("failed to get user price:", err)
			price = 0
		}
	}
	response, _, err := mb.SendUserInfo(user, price)
	if err != nil {
		log.Println("Ошибка при формировании сообщения:", err)
		t := "Данные подписки получены, но не удалось оформить сообщение. Обратитесь к администратору."
		kb := MainMenuKeyboard(userUC, uid, false)
		EditPanelFromUpdate(bot, pr, update, uid, t, &kb, true)
		return
	}
	if paid {
		if paidUntil, err := userUC.GetPaymentDateByUserID(uid); err == nil && paidUntil != nil {
			response += fmt.Sprintf("\n<b>Оплачен до</b> %s", paidUntil.Format("02.01.2006"))
		}
	} else {
		response += "\n<b>Тариф:</b> бесплатный"
	}

	kb := MainMenuKeyboard(userUC, uid, false)
	EditPanelFromUpdate(bot, pr, update, uid, response, &kb, true)
}
