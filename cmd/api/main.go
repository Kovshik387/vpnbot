package main

import (
	"VpnBot/config"
	"VpnBot/internal/app/router"
	"VpnBot/internal/app/usecases"
	interfaces "VpnBot/internal/interfaces/http"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"log"
)

func main() {
	log.Print("Бот включается")
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

	client := interfaces.NewMarzbanClient(cfg.MarzbanUrl, cfg.UsernameApi, cfg.PasswordApi)
	userUC := usecases.NewUserUsecase(client)

	commandRouter := router.NewCommandRouter(userUC, cfg)
	callbackRouter := router.NewCallbackRouter(userUC, cfg)

	log.Print("Бот включился")

	for update := range updates {
		switch {
		case update.Message != nil && update.Message.IsCommand():
			if handler, ok := commandRouter[update.Message.Command()]; ok {
				handler(update, bot)
			}

		case update.CallbackQuery != nil:
			if handler, ok := callbackRouter[update.CallbackQuery.Data]; ok {
				handler(update, bot)
			} else {
				_, _ = bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Неизвестная кнопка"))
			}
		}
	}

	select {}
}
