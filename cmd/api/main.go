package main

import (
	"VpnBot/config"
	"VpnBot/internal/app/handlers/admin"
	"VpnBot/internal/app/handlers/user"
	"VpnBot/internal/app/router"
	"VpnBot/internal/app/ui"
	"VpnBot/internal/app/usecases"
	"VpnBot/internal/domain/repository"
	interfaces "VpnBot/internal/interfaces/http"
	"VpnBot/internal/interfaces/jobs"
	"database/sql"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"log"
	_ "modernc.org/sqlite"
	"strings"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Ошибка загрузки .env файла")
		return
	}
	if err := config.SetupLog(); err != nil {
		log.Fatal(err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
		return
	}

	bot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Fatal(err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	marzbanClient := interfaces.NewMarzbanClient(cfg.MarzbanUrl, cfg.UsernameApi, cfg.PasswordApi)
	yandexClient := interfaces.NewYandexClient()

	db, err := sql.Open("sqlite", "data/bot.db") // путь к базе
	if err != nil {
		log.Fatalf("Ошибка подключения к SQLite: %v", err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(db)

	cr := repository.NewCooldownRepository(db)
	if err := cr.Init(); err != nil {
		log.Fatal(err)
	}

	ur := repository.NewUserRepository(db)
	if err := ur.EnsureSchema(); err != nil {
		log.Fatal(err)
	}

	pol := repository.NewPollRepository(db)
	if err := pol.Init(); err != nil {
		log.Fatal(err)
	}

	panelRepo := repository.NewPanelRepository(db)
	if err := panelRepo.Init(); err != nil {
		log.Fatalf("ui_panel init: %v", err)
	}
	sayLogRepo := repository.NewSayLogRepository(db)
	if err := sayLogRepo.Init(); err != nil {
		log.Fatalf("say_logs init: %v", err)
	}
	paymentReportRepo := repository.NewPaymentReportRepository(db)
	if err := paymentReportRepo.Init(); err != nil {
		log.Fatalf("payment_confirms init: %v", err)
	}

	userUC := usecases.NewUserUsecase(marzbanClient, yandexClient, ur, pol, paymentReportRepo)
	cooldownUC := usecases.NewCooldownUsecase(cr)

	reminderJob := jobs.NewReminderJob(userUC, bot, cfg, panelRepo)

	reminderJob.Start()

	uiR := &router.UIRepos{Panel: panelRepo, SayLog: sayLogRepo}
	commandRouter := router.NewCommandRouter(userUC, cfg, uiR)
	callbackRouter := router.NewCallbackRouter(userUC, cooldownUC, cfg, uiR)

	log.Print("Бот включился")

	for update := range updates {

		if update.Poll != nil || (update.Message != nil && update.Message.Poll != nil) {
			log.Print("Действие опроса")
			admin.HandlePollUpdate(update, userUC)
			continue
		}

		switch {

		case update.Message != nil && len(update.Message.Photo) > 0:
			if checkBlock(userUC, update.Message.Chat.ID) {
				blockUser(update, bot, true)
				continue
			}
			if user.HandlePaymentPhoto(update, bot, userUC, cfg.AdminId, panelRepo) {
				continue
			}
			continue

		case update.Message != nil && update.Message.Text != "" && !update.Message.IsCommand():
			if checkBlock(userUC, update.Message.Chat.ID) {
				blockUser(update, bot, true)
				continue
			}
			if paymentRevokedBlocksPlainText(userUC, update.Message) {
				sendPaymentRevokedNotice(bot, panelRepo, update.Message.From.ID)
				continue
			}

			switch update.Message.Text {
			case ui.BtnCheckSubscription:
				if handler, ok := commandRouter["subscribe"]; ok {
					handler(update, bot)
				}
			case ui.BtnShowPanel:
				user.StartHandler(update, bot, cfg.AdminId, panelRepo, userUC)
			case "./start":
				if handler, ok := commandRouter["start"]; ok {
					handler(update, bot)
				}
			}
			continue

		case update.Message != nil && update.Message.Text != "" && update.Message.IsCommand():
			if checkBlock(userUC, update.Message.Chat.ID) {
				blockUser(update, bot, true)
				continue
			}
			if paymentRevokedBlocksCommand(userUC, update.Message) {
				sendPaymentRevokedNotice(bot, panelRepo, update.Message.From.ID)
				continue
			}
			if handler, ok := commandRouter[update.Message.Command()]; ok {
				handler(update, bot)
			}

		case update.CallbackQuery != nil:
			if checkBlock(userUC, update.CallbackQuery.Message.Chat.ID) {
				blockUser(update, bot, false)
				continue
			}
			data := update.CallbackQuery.Data
			if paymentRevokedBlocksCallback(userUC, update.CallbackQuery.From.ID, data) {
				_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Нужна оплата"))
				sendPaymentRevokedNotice(bot, panelRepo, update.CallbackQuery.From.ID)
				continue
			}

			if handler, ok := callbackRouter[data]; ok {
				handler(update, bot)
				continue
			}

			found := false
			for prefix, handler := range callbackRouter {
				if strings.HasSuffix(prefix, ":") && strings.HasPrefix(data, prefix) {
					handler(update, bot)
					found = true
					break
				}
			}

			if !found {
				_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Действие не распознано"))
			}
		}
	}

	select {}
}

func checkBlock(uc *usecases.UserUsecase, id int64) bool {
	blocked, err := uc.CheckBlock(id)

	if err != nil {
		return false
	}

	return blocked
}

func paymentRevokedBlocksPlainText(uc *usecases.UserUsecase, m *tgbotapi.Message) bool {
	if m == nil {
		return false
	}
	revoked, err := uc.CheckPaymentRevoked(m.From.ID)
	if err != nil || !revoked {
		return false
	}
	if m.Text == "./start" || m.Text == ui.BtnShowPanel || m.Text == ui.BtnCheckSubscription {
		return false
	}
	return true
}

func paymentRevokedBlocksCommand(uc *usecases.UserUsecase, m *tgbotapi.Message) bool {
	if m == nil {
		return false
	}
	revoked, err := uc.CheckPaymentRevoked(m.From.ID)
	if err != nil || !revoked {
		return false
	}
	return m.Command() != "start"
}

func paymentRevokedBlocksCallback(uc *usecases.UserUsecase, fromUserID int64, data string) bool {
	revoked, err := uc.CheckPaymentRevoked(fromUserID)
	if err != nil || !revoked {
		return false
	}
	return !user.CallbackExemptWhenPaymentRevoked(data)
}

func sendPaymentRevokedNotice(bot *tgbotapi.BotAPI, panelRepo *repository.PanelRepository, userID int64) {
	text := "⏸️ <b>Доступ приостановлен</b> из-за просроченной оплаты.\n\n" +
		"Нажмите «Оплата» и отправьте скриншот перевода — после проверки срок продлят на месяц."
	kb := ui.PaymentRevokedKeyboard()
	user.SendNotificationHTMLForUser(bot, userID, text, &kb, true)
}

func blockUser(update tgbotapi.Update, bot *tgbotapi.BotAPI, isCommand bool) {
	var msg tgbotapi.MessageConfig
	if isCommand {
		msg = tgbotapi.NewMessage(
			update.Message.Chat.ID,
			"🚫 <b>Доступ ограничен</b>\n\n"+
				"По вопросам разблокировки напишите администратору: @KovshikGo",
		)
	} else {
		msg = tgbotapi.NewMessage(
			update.CallbackQuery.Message.Chat.ID,
			"🚫 <b>Доступ ограничен</b>\n\n"+
				"По вопросам разблокировки напишите администратору: @KovshikGo",
		)
	}

	msg.ParseMode = tgbotapi.ModeHTML
	_, _ = bot.Send(msg)
}
