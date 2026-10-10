package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	BotToken    string
	AdminId     int64
	UsernameApi string
	PasswordApi string
	MarzbanUrl  string
	RussianUrl  string
	SkebobUrls  []string
}

func LoadConfig() (*Config, error) {
	botToken := os.Getenv("BOT_TOKEN")
	adminStr := os.Getenv("ADMIN_ID")

	if botToken == "" || adminStr == "" {
		return nil, errors.New("токен или id администратора отсутствуют")
	}

	usernameApi := os.Getenv("USERNAME_API")
	passwordApi := os.Getenv("PASSWORD_API")
	marzbanUrl := os.Getenv("MARZBAN_URL")
	russianUrl := os.Getenv("RUSSIAN_URL")
	rawSkebob := os.Getenv("SKEBOB_URLS")

	if usernameApi == "" || passwordApi == "" || marzbanUrl == "" {
		return nil, errors.New("ссылка, пользователь или пароль администратора отсутствуют")
	}

	if russianUrl == "" {
		return nil, errors.New("ссылка на ru сервер отсутствует")
	}

	adminId, err := strconv.ParseInt(adminStr, 10, 64)
	if err != nil {
		return nil, errors.New("ADMIN_ID должен быть положительным числовым Telegram ID")
	}
	if adminId <= 0 {
		return nil, errors.New("ADMIN_ID должен быть положительным числовым Telegram ID")
	}

	return &Config{
		BotToken:    botToken,
		AdminId:     adminId,
		UsernameApi: usernameApi,
		PasswordApi: passwordApi,
		MarzbanUrl:  marzbanUrl,
		RussianUrl:  russianUrl,
		SkebobUrls:  strings.Split(rawSkebob, ","),
	}, nil
}
