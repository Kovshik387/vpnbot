package service

import "VpnBot/internal/domain/model"

type MessageService interface {
	SendUserInfo(user model.User) (string, bool, error)
}
