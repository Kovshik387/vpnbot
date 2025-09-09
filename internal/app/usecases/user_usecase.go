package usecases

import (
	"VpnBot/internal/domain/model"
	"VpnBot/internal/domain/ports/service"
)

type UserUsecase struct {
	marzbanClient service.MarzbanService
}

func NewUserUsecase(marzbanClient service.MarzbanService) *UserUsecase {
	return &UserUsecase{marzbanClient: marzbanClient}
}

func (u *UserUsecase) ListUsers() (model.UsersResponse, error) {
	return u.marzbanClient.GetUsers()
}

func (u *UserUsecase) SearchUser(username string) (model.User, error) {
	return u.marzbanClient.GetUser(username)
}

func (u *UserUsecase) AddUser(username string) (model.User, error) {
	return u.marzbanClient.AddUser(username)
}

func (u *UserUsecase) DeleteUser(username string) error {
	return u.marzbanClient.Delete(username)
}
