package usecases

import (
	"VpnBot/internal/domain/model"
	"VpnBot/internal/domain/ports/service"
	"VpnBot/internal/domain/repository"
)

type UserUsecase struct {
	marzbanClient  service.MarzbanService
	userRepository *repository.UserRepository
}

func NewUserUsecase(marzbanClient service.MarzbanService, userRepository *repository.UserRepository) *UserUsecase {
	return &UserUsecase{marzbanClient: marzbanClient, userRepository: userRepository}
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

func (u *UserUsecase) Insert(username string, uid int64) error {
	return u.userRepository.Insert(username, uid)
}

func (u *UserUsecase) Block(uid int64, block bool) error {
	return u.userRepository.Block(uid, block)
}

func (u *UserUsecase) CheckBlock(uid int64) (bool, error) {
	return u.userRepository.CheckBlock(uid)
}

func (u *UserUsecase) UserExist(uid int64) (bool, error) {
	return u.userRepository.UserExist(uid)
}
func (u *UserUsecase) ListBlocked() ([]model.TgUserModel, error) {
	return u.userRepository.GetBlocked()
}
