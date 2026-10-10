package service

import (
	"VpnBot/internal/domain/model"
	"errors"
)

var ErrUserNotFound = errors.New("пользователь Marzban не найден")

type MarzbanService interface {
	Login() error
	GetUsers() (model.UsersResponse, error)
	GetUser(username string) (model.User, error)
	AddUser(username string) (model.User, error)
	Delete(username string) error
	SetUserStatus(username string, status string) error
}
