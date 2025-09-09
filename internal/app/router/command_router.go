package router

import (
	"VpnBot/config"
	"VpnBot/internal/app/handlers/admin"
	"VpnBot/internal/app/handlers/user"
	"VpnBot/internal/app/usecases"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"strings"
)

type CommandHandler func(update tgbotapi.Update, bot *tgbotapi.BotAPI)

func NewCommandRouter(userUC *usecases.UserUsecase, config *config.Config) map[string]CommandHandler {
	baseHandlers := map[string]CommandHandler{
		"start": user.Start,
	}

	baseHandlers["ping"] = func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
		user.Ping(update, bot, config.RussianUrl)
	}

	baseHandlers["help"] = func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
		user.Help(update, bot, config.AdminId)
	}

	baseHandlers["adduser"] = func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
		CheckPermission(update, bot, config.AdminId)
		args := update.Message.CommandArguments()
		if args == "" {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Использование: /adduser <username>")
			_, _ = bot.Send(msg)
			return
		}
		admin.AddUserHandler(update, bot, userUC, args)
	}

	baseHandlers["users"] = func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
		CheckPermission(update, bot, config.AdminId)
		args := strings.Fields(update.Message.Text)

		if len(args) > 1 {
			username := args[1]
			admin.SearchUser(update, bot, userUC, username)
		} else {
			admin.UserList(update, bot, userUC)
		}
	}

	baseHandlers["deleteuser"] = func(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
		CheckPermission(update, bot, config.AdminId)
		args := update.Message.CommandArguments()

		if args == "" {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Использование: /deleteuser <username>")
			_, _ = bot.Send(msg)
			return
		}

		admin.DeleteUserHandler(update, bot, userUC, args)
	}

	return baseHandlers
}

func CheckPermission(update tgbotapi.Update, bot *tgbotapi.BotAPI, adminId int64) {
	if update.Message.From.ID != adminId {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "У тебя нет доступа к этой команде")
		_, _ = bot.Send(msg)
		return
	}
}
