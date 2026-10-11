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
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Fatalf("Ошибка загрузки .env файла: %v", err)
	}
	if err := config.SetupLog(); err != nil {
		log.Fatal(err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
		return
	}

	bot, err := tgbotapi.NewBotAPIWithClient(cfg.BotToken, tgbotapi.APIEndpoint, &http.Client{Timeout: 75 * time.Second})
	if err != nil {
		log.Fatal(err)
	}

	commands := tgbotapi.NewSetMyCommandsWithScope(tgbotapi.NewBotCommandScopeAllPrivateChats(),
		tgbotapi.BotCommand{Command: "menu", Description: "Открыть меню в конце чата"},
		tgbotapi.BotCommand{Command: "start", Description: "Главный экран"},
		tgbotapi.BotCommand{Command: "subscribe", Description: "Моя подписка"},
		tgbotapi.BotCommand{Command: "proxies", Description: "Прокси"},
		tgbotapi.BotCommand{Command: "info", Description: "Настройка VPN"},
		tgbotapi.BotCommand{Command: "help", Description: "Помощь"},
	)
	if _, err := bot.Request(commands); err != nil {
		log.Println("register bot commands:", err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	u.AllowedUpdates = []string{"message", "callback_query", "poll"}
	updates := bot.GetUpdatesChan(u)

	marzbanClient := interfaces.NewMarzbanClient(cfg.MarzbanUrl, cfg.UsernameApi, cfg.PasswordApi)
	yandexClient := interfaces.NewYandexClient()

	if err := os.MkdirAll("data", 0700); err != nil {
		log.Fatalf("Ошибка создания каталога данных: %v", err)
	}
	db, err := sql.Open("sqlite", "data/bot.db")
	if err != nil {
		log.Fatalf("Ошибка подключения к SQLite: %v", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		log.Fatalf("Ошибка настройки SQLite: %v", err)
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
	proxyRepo := repository.NewProxyRepository(db)
	if err := proxyRepo.Init(); err != nil {
		log.Fatalf("proxies init: %v", err)
	}
	proxyUC := usecases.NewProxyUsecase(proxyRepo)

	userUC := usecases.NewUserUsecase(marzbanClient, yandexClient, ur, pol, paymentReportRepo)
	cooldownUC := usecases.NewCooldownUsecase(cr)

	reminderJob := jobs.NewReminderJob(userUC, bot, cfg, panelRepo)

	reminderJob.Start()

	uiR := &router.UIRepos{Panel: panelRepo, SayLog: sayLogRepo, Proxy: proxyUC}
	commandRouter := router.NewCommandRouter(userUC, cfg, uiR)
	callbackRouter := router.NewCallbackRouter(userUC, cooldownUC, cfg, uiR)

	log.Print("Бот включился")

	for update := range updates {
		// Panels, subscription links, and proxy credentials belong in private chats.
		if update.Message != nil && !privateMessage(update.Message) {
			if update.Message.Chat != nil && update.Message.IsCommand() {
				_, _ = bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Откройте личный чат с ботом, чтобы пользоваться панелью."))
			}
			continue
		}
		if update.CallbackQuery != nil && !privateCallback(update.CallbackQuery) {
			_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Откройте личный чат с ботом."))
			continue
		}
		if update.Message != nil && !user.HasUserInput(update.Message) {
			continue
		}

		if update.Poll != nil {
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
			user.UnsupportedMessageHandler(update, bot, userUC, cfg.AdminId, panelRepo)
			continue

		case update.Message != nil && update.Message.Text != "" && !update.Message.IsCommand():
			if checkBlock(userUC, update.Message.Chat.ID) {
				blockUser(update, bot, true)
				continue
			}
			switch update.Message.Text {
			case ui.BtnCheckSubscription:
				if handler, ok := commandRouter["subscribe"]; ok {
					handler(update, bot)
				}
			case ui.BtnShowPanel:
				user.MenuHandler(update, bot, cfg.AdminId, panelRepo, userUC)
			case "./start":
				if handler, ok := commandRouter["start"]; ok {
					handler(update, bot)
				}
			default:
				user.UnsupportedMessageHandler(update, bot, userUC, cfg.AdminId, panelRepo)
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
			} else {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Команда не найдена. Откройте /help или вернитесь в панель.")
				msg.ReplyMarkup = ui.PanelShortcutKeyboard()
				_, _ = bot.Send(msg)
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

		case update.Message != nil:
			if checkBlock(userUC, update.Message.Chat.ID) {
				blockUser(update, bot, true)
				continue
			}
			user.UnsupportedMessageHandler(update, bot, userUC, cfg.AdminId, panelRepo)
		}
	}

	select {}
}

func privateMessage(m *tgbotapi.Message) bool {
	return m != nil && m.From != nil && m.Chat != nil && m.Chat.IsPrivate()
}

func privateCallback(q *tgbotapi.CallbackQuery) bool {
	return q != nil && q.From != nil && q.Message != nil && q.Message.Chat != nil &&
		q.Message.Chat.IsPrivate() && q.Message.Chat.ID == q.From.ID
}

func checkBlock(uc *usecases.UserUsecase, id int64) bool {
	blocked, err := uc.CheckBlock(id)

	if err != nil {
		return false
	}

	return blocked
}

func paymentRevokedBlocksCommand(uc *usecases.UserUsecase, m *tgbotapi.Message) bool {
	if m == nil || m.From == nil {
		return false
	}
	revoked, err := uc.CheckPaymentRevoked(m.From.ID)
	if err != nil || !revoked {
		return false
	}
	switch m.Command() {
	case "start", "menu", "subscribe", "help":
		return false
	default:
		return true
	}
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
		_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Доступ ограничен"))
		msg = tgbotapi.NewMessage(
			update.CallbackQuery.Message.Chat.ID,
			"🚫 <b>Доступ ограничен</b>\n\n"+
				"По вопросам разблокировки напишите администратору: @KovshikGo",
		)
	}

	msg.ParseMode = tgbotapi.ModeHTML
	_, _ = bot.Send(msg)
}
