package router

import (
	"VpnBot/config"
	"VpnBot/internal/app/handlers/admin"
	"VpnBot/internal/app/handlers/user"
	"VpnBot/internal/app/usecases"
	"errors"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"strconv"
	"strings"
)

type CommandHandler func(update tgbotapi.Update, bot *tgbotapi.BotAPI)

func NewCommandRouter(userUC *usecases.UserUsecase, config *config.Config) map[string]CommandHandler {
	var baseHandlers = make(map[string]CommandHandler)

	baseHandlers["start"] = func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
		user.StartHandler(update, bot, config.AdminId)
	}

	baseHandlers["ping"] = func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
		user.PingHandler(update, bot, config.RussianUrl, config.AdminId)
	}

	baseHandlers["help"] = func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
		user.HelpHandler(update, bot, config.AdminId)
	}

	baseHandlers["info"] = func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
		user.InfoHandler(update, bot)
	}

	baseHandlers["adduser"] = func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
		err := checkPermission(update, bot, config.AdminId)
		if err != nil {
			log.Println(err)
			return
		}

		args, err := checkArgs(update, bot, "Использование: /adduser <username>")
		if err != nil {
			return
		}

		admin.AddUserHandler(update, bot, userUC, args)
	}

	baseHandlers["users"] = func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
		err := checkPermission(update, bot, config.AdminId)
		if err != nil {
			log.Println(err)
			return
		}

		args := strings.Fields(update.Message.Text)

		if len(args) > 1 {
			username := args[1]
			admin.SearchUserHandler(update, bot, userUC, username)
		} else {
			admin.UserListHandler(update, bot, userUC, false)
		}
	}

	baseHandlers["activity"] = func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
		err := checkPermission(update, bot, config.AdminId)
		if err != nil {
			log.Println(err)
			return
		}

		admin.UserListHandler(update, bot, userUC, true)
	}

	baseHandlers["status"] = func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
		err := checkPermission(update, bot, config.AdminId)
		if err != nil {
			log.Println(err)
			return
		}

		admin.ServerStatHandler(update, bot)
	}

	baseHandlers["count"] = func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
		err := checkPermission(update, bot, config.AdminId)
		if err != nil {
			log.Println(err)
			return
		}

		admin.UserActivityCount(update, bot, userUC)
	}

	baseHandlers["deleteuser"] = func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
		err := checkPermission(update, bot, config.AdminId)
		if err != nil {
			log.Println(err)
			return
		}

		args, err := checkArgs(update, bot, "Использование: /deleteuser <username>")
		if err != nil {
			return
		}

		admin.DeleteUserHandler(update, bot, userUC, args)
	}
	baseHandlers["say"] = func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
		err := checkPermission(update, bot, config.AdminId)
		if err != nil {
			log.Println(err)
			return
		}

		admin.SayCommandHandler(update, bot, userUC)
	}
	baseHandlers["poll_result"] = func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
		if err := checkPermission(update, bot, config.AdminId); err != nil {
			log.Println(err)
			return
		}

		pollID, err := checkArgs(update, bot, "Использование: /poll_result <poll_id>")
		if err != nil {
			return
		}

		admin.PollResultHandler(update, bot, userUC, pollID)
	}
	baseHandlers["poll_list"] = func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
		if err := checkPermission(update, bot, config.AdminId); err != nil {
			log.Println(err)
			return
		}

		admin.PollListHandler(update, bot, userUC)
	}

	baseHandlers["skebob"] = func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
		log.Printf("Скебоб)")
		user.Skebob(update, bot, userUC, config.SkebobUrls)
	}

	baseHandlers["unblock"] = func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
		err := checkPermission(update, bot, config.AdminId)
		if err != nil {
			log.Println(err)
			return
		}

		args, err := checkArgs(update, bot, "Использование: /unblock <ID>")
		if err != nil {
			return
		}

		admin.UnblockUserHandler(update, bot, userUC, args)
	}

	baseHandlers["blocked"] = func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
		err := checkPermission(update, bot, config.AdminId)
		if err != nil {
			log.Println(err)
			return
		}

		admin.UserBlockedHandler(update, bot, userUC)
	}

	baseHandlers["setprice"] = func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
		err := checkPermission(update, bot, config.AdminId)
		if err != nil {
			log.Println(err)
			return
		}

		args, err := checkArgs(update, bot,
			"Использование: /setprice username цена\n"+
				"Пример: /setprice vasya 500.00")
		if err != nil {
			return
		}

		admin.SetPriceHandler(update, bot, userUC, args)
	}

	baseHandlers["setdate"] = func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
		err := checkPermission(update, bot, config.AdminId)
		if err != nil {
			log.Println(err)
			return
		}

		args, err := checkArgs(update, bot,
			"Использование: /setdate username YYYY-MM-DD\n"+
				"Пример: /setdate vasya 2024-12-25")
		if err != nil {
			return
		}

		admin.SetPaymentDateHandler(update, bot, userUC, args)
	}

	baseHandlers["setfree"] = func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
		err := checkPermission(update, bot, config.AdminId)
		if err != nil {
			log.Println(err)
			return
		}

		args, err := checkArgs(update, bot,
			"Использование: /setfree username true/false\n")
		if err != nil {
			return
		}

		admin.UpdateTypePaymentHandler(update, bot, userUC, args)
	}

	baseHandlers["subscribe"] = func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
		user.GetSubscribeHandler(update, bot, userUC)
	}

	return baseHandlers
}

func checkArgs(update tgbotapi.Update, bot *tgbotapi.BotAPI, str string) (string, error) {
	args := update.Message.CommandArguments()
	if args == "" {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, str)
		_, _ = bot.Send(msg)
		return "", errors.New("")
	}

	return args, nil
}

func checkPermission(update tgbotapi.Update, bot *tgbotapi.BotAPI, adminId int64) error {
	if update.Message.From.ID != adminId {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "У тебя нет доступа к этой команде")
		_, _ = bot.Send(msg)
		return errors.New("У тебя нет доступа к этой команде" + strconv.Itoa(int(update.Message.From.ID)))
	}
	return nil
}
