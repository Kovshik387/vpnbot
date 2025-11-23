package usecases

import (
	"VpnBot/internal/domain/model"
	"VpnBot/internal/domain/ports/service"
	"VpnBot/internal/domain/repository"
	"fmt"
	"strings"
)

type UserUsecase struct {
	marzbanClient  service.MarzbanService
	yandexClient   service.YandexService
	userRepository *repository.UserRepository
	pollRepository *repository.PollRepository
}

func NewUserUsecase(marzbanClient service.MarzbanService, yandexService service.YandexService,
	userRepository *repository.UserRepository, pollRepository *repository.PollRepository) *UserUsecase {
	return &UserUsecase{marzbanClient: marzbanClient, userRepository: userRepository, pollRepository: pollRepository,
		yandexClient: yandexService}
}

func (u *UserUsecase) SavePollResults(poll *model.PollResult) error {
	return u.pollRepository.UpsertPollResults(
		poll.PollID,
		poll.Question,
		poll.IsAnonymous,
		poll.AllowsMultiple,
		poll.Options,
	)
}

func (u *UserUsecase) PollResult(pollID string) (string, error) {
	res, err := u.pollRepository.GetPollResults(pollID)
	if err != nil {
		return "", err
	}
	if res == nil {
		return "", nil
	}

	var sb strings.Builder
	sb.WriteString("Результаты опроса:\n")
	sb.WriteString(fmt.Sprintf("ID: `%s`\n", res.PollID))
	sb.WriteString(fmt.Sprintf("Вопрос: %s\n\n", res.Question))

	total := 0
	for _, o := range res.Options {
		total += o.Votes
	}

	for _, o := range res.Options {
		percent := 0
		if total > 0 {
			percent = int(float64(o.Votes) / float64(total) * 100)
		}
		sb.WriteString(fmt.Sprintf("%d) %s — %d голосов (%d%%)\n",
			o.OptionIndex+1, o.Text, o.Votes, percent))
	}

	return sb.String(), nil
}

func (u *UserUsecase) GetAllPolls() ([]model.PollResult, error) {
	return u.pollRepository.ListPolls()
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

func (u *UserUsecase) GetUserByUserId(uid int64) (string, error) {
	return u.userRepository.GetUsernameByUserID(uid)
}

func (u *UserUsecase) ListBlocked() ([]model.TgUserModel, error) {
	return u.userRepository.GetBlocked()
}

func (u *UserUsecase) ListActive() ([]model.TgUserModel, error) {
	return u.userRepository.GetActive()
}

func (u *UserUsecase) Skebob(url string) (string, error) {
	return u.yandexClient.GetYandexDirectLink(url)
}
