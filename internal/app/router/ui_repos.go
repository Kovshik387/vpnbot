package router

import "VpnBot/internal/domain/repository"

type UIRepos struct {
	Panel  *repository.PanelRepository
	SayLog *repository.SayLogRepository
}
