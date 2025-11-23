package main

import (
	"VpnBot/config"
	"VpnBot/internal/app/handlers/admin"
	"VpnBot/internal/app/router"
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
	if err := ur.Init(); err != nil {
		log.Fatal(err)
	}

	pol := repository.NewPollRepository(db)
	if err := pol.Init(); err != nil {
		log.Fatal(err)
	}

	userUC := usecases.NewUserUsecase(marzbanClient, yandexClient, ur, pol)
	cooldownUC := usecases.NewCooldownUsecase(cr)
	reminderUC := usecases.NewReminderUsecase(ur)

	reminderJob := jobs.NewReminderJob(reminderUC, bot)

	reminderJob.Start()

	commandRouter := router.NewCommandRouter(userUC, cfg)
	callbackRouter := router.NewCallbackRouter(userUC, cooldownUC, cfg)

	log.Print("Бот включился")

	for update := range updates {

		if update.Poll != nil || (update.Message != nil && update.Message.Poll != nil) {
			log.Print("Действие опроса")
			admin.HandlePollUpdate(update, userUC)
			continue
		}

		switch {
		case update.Message != nil && update.Message.IsCommand():
			if checkBlock(userUC, update.Message.Chat.ID) {
				blockUser(update, bot, true)
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
				_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Неизвестная кнопка"))
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

func blockUser(update tgbotapi.Update, bot *tgbotapi.BotAPI, isCommand bool) {
	var msg tgbotapi.MessageConfig
	if isCommand {
		msg = tgbotapi.NewMessage(
			update.Message.Chat.ID,
			"🚫 Доступ заблокирован.\nЕсли у вас есть вопросы — обратитесь к администратору: @KovshikGo",
		)
	} else {
		msg = tgbotapi.NewMessage(
			update.CallbackQuery.Message.Chat.ID,
			"🚫 Доступ заблокирован.\nЕсли у вас есть вопросы — обратитесь к администратору: @KovshikGo",
		)
	}

	msg.ParseMode = "Markdown"
	_, _ = bot.Send(msg)
}
