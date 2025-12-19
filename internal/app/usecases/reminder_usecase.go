package usecases

import (
	"VpnBot/internal/domain/model"
	"VpnBot/internal/domain/repository"
)

type ReminderUsecase struct {
	userRepository *repository.UserRepository
}

func NewReminderUsecase(rep *repository.UserRepository) *ReminderUsecase {
	return &ReminderUsecase{
		rep,
	}
}

func (uc *ReminderUsecase) InitReminder() ([]model.TgUserModel, error) {
	return uc.userRepository.GetPayment()
}
