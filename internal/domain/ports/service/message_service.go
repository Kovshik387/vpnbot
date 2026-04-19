package service

import "VpnBot/internal/domain/model"

type MessageService interface {
	SendUserInfo(user model.User, price float64) (string, bool, error)
}
