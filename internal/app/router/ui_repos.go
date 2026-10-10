package router

import (
	"VpnBot/internal/app/usecases"
	"VpnBot/internal/domain/repository"
)

type UIRepos struct {
	Panel  *repository.PanelRepository
	SayLog *repository.SayLogRepository
	Proxy  *usecases.ProxyUsecase
}
