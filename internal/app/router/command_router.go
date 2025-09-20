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

		args, err := checkArgs(update, bot, "Использование: /say <text>")
		if err != nil {
			return
		}

		admin.SayHandler(update, bot, userUC, args)
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
