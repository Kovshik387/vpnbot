package service

import (
	"VpnBot/internal/domain/model"
)

type MarzbanService interface {
	Login() error
	GetUsers() (model.UsersResponse, error)
	GetUser(username string) (model.User, error)
	AddUser(username string) (model.User, error)
	Delete(username string) error
}
