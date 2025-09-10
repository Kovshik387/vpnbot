package usecases

import (
	"VpnBot/internal/domain/repository"
	"time"
)

const duration = 30 * time.Minute

type CooldownUsecase struct {
	repo *repository.CooldownRepository
}

func NewCooldownUsecase(repo *repository.CooldownRepository) *CooldownUsecase {
	return &CooldownUsecase{repo: repo}
}

func (uc *CooldownUsecase) SetCooldown(userID int64) error {
	return uc.repo.SetCooldown(userID, duration)
}

func (uc *CooldownUsecase) IsOnCooldown(userID int64) (bool, time.Duration, error) {
	cd, err := uc.repo.GetCooldown(userID)
	if err != nil {
		return false, 0, err
	}
	if cd == nil {
		return false, 0, nil
	}

	if time.Now().Before(cd.ExpiresAt) {
		return true, time.Until(cd.ExpiresAt), nil
	}

	_ = uc.repo.ClearExpired()
	return false, 0, nil
}
