package usecases

import (
	"VpnBot/internal/domain/model"
	"VpnBot/internal/domain/ports/service"
	"VpnBot/internal/domain/repository"
	"VpnBot/internal/utils"
	"fmt"
	"strings"
	"sync"
	"time"
)

type UserUsecase struct {
	billingMu         sync.Mutex
	marzbanClient     service.MarzbanService
	yandexClient      service.YandexService
	userRepository    *repository.UserRepository
	pollRepository    *repository.PollRepository
	paymentReportRepo *repository.PaymentReportRepository
}

func NewUserUsecase(marzbanClient service.MarzbanService, yandexService service.YandexService,
	userRepository *repository.UserRepository, pollRepository *repository.PollRepository, paymentReportRepo *repository.PaymentReportRepository) *UserUsecase {
	return &UserUsecase{
		marzbanClient:     marzbanClient,
		userRepository:    userRepository,
		pollRepository:    pollRepository,
		yandexClient:      yandexService,
		paymentReportRepo: paymentReportRepo,
	}
}

func (u *UserUsecase) SavePollResults(poll *model.PollResult) error {
	return u.pollRepository.UpsertPollResults(poll)
}

func (u *UserUsecase) UpdatePollResults(poll *model.PollResult) error {
	return u.pollRepository.UpdatePollResults(poll)
}

func (u *UserUsecase) FindPublishedPoll(sourceID string) (*model.PollResult, error) {
	return u.pollRepository.FindPublishedPoll(sourceID)
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
	sb.WriteString("<b>📊 Результаты опроса</b>\n\n")
	sb.WriteString(fmt.Sprintf("ID: <code>%s</code>\n", utils.HtmlEscape(res.PollID)))
	sb.WriteString(fmt.Sprintf("<b>%s</b>\n", utils.HtmlEscape(res.Question)))
	if res.IsClosed {
		sb.WriteString("Статус: закрыт\n")
	} else if res.MessageID > 0 {
		sb.WriteString("Статус: открыт\n")
	}

	total := 0
	for _, o := range res.Options {
		total += o.Votes
	}

	voters := res.TotalVoterCount
	if voters < 0 && !res.AllowsMultiple {
		voters = total
	}
	if voters >= 0 {
		sb.WriteString(fmt.Sprintf("Участников: <b>%d</b>\n\n", voters))
	} else {
		sb.WriteString("Число участников для старого опроса неизвестно.\n\n")
	}
	for _, o := range res.Options {
		percent := ""
		if voters >= 0 {
			value := 0.0
			if voters > 0 {
				value = float64(o.Votes) / float64(voters) * 100
			}
			percent = fmt.Sprintf(" (%.1f%%)", value)
		}
		sb.WriteString(fmt.Sprintf("%d) %s — %d голосов%s\n",
			o.OptionIndex+1, utils.HtmlEscape(o.Text), o.Votes, percent))
	}
	if res.AllowsMultiple && voters >= 0 {
		sb.WriteString("\nМожно выбрать несколько ответов: сумма процентов может превышать 100%.\n")
	}
	if res.MessageID == 0 {
		sb.WriteString("\nОпрос из прежней версии: общей статистики по разосланным копиям нет.\n")
	}

	return sb.String(), nil
}

func (u *UserUsecase) GetPollsPage(offset, limit int) ([]model.PollResult, error) {
	return u.pollRepository.ListPolls(offset, limit)
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
	return u.userRepository.Insert(username, uid, paymentDateOneMonthFrom(time.Now()))
}

func (u *UserUsecase) Block(uid int64, block bool) error {
	return u.userRepository.Block(uid, block)
}

func (u *UserUsecase) UpdatePrice(username string, price float64) error {
	return u.userRepository.UpdatePrice(username, price)
}

func (u *UserUsecase) UpdateTypePayment(username string, isFree bool) error {
	u.billingMu.Lock()
	defer u.billingMu.Unlock()
	if err := u.userRepository.UpdateTypePayment(username, isFree); err != nil {
		return err
	}
	return u.restorePaymentAccess(time.Now(), username)
}

func (u *UserUsecase) UpdatePaymentDate(username string, date time.Time) error {
	u.billingMu.Lock()
	defer u.billingMu.Unlock()
	if err := u.userRepository.UpdatePaymentDate(username, date); err != nil {
		return err
	}
	return u.restorePaymentAccess(time.Now(), username)
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

func (u *UserUsecase) GetPriceByUserID(uid int64) (float64, error) {
	return u.userRepository.GetPriceByUserID(uid)
}

func (u *UserUsecase) GetPriceByUsername(username string) (float64, error) {
	return u.userRepository.GetPriceByUsername(username)
}

func (u *UserUsecase) GetPaymentDateByUserID(uid int64) (*time.Time, error) {
	return u.userRepository.GetPaymentDateByUserID(uid)
}

func (u *UserUsecase) GetPaymentDateByUsername(username string) (*time.Time, error) {
	return u.userRepository.GetPaymentDateByUsername(username)
}

func (u *UserUsecase) ListBlocked() ([]model.TgUserModel, error) {
	return u.userRepository.GetBlocked()
}

func (u *UserUsecase) ListActive() ([]model.TgUserModel, error) {
	return u.userRepository.GetActive()
}

func (u *UserUsecase) ListPayment() ([]model.TgUserModel, error) {
	return u.userRepository.GetPayment()
}

func (u *UserUsecase) OverrideDate(date time.Time, dateOverride time.Time) error {
	u.billingMu.Lock()
	defer u.billingMu.Unlock()
	if err := u.userRepository.OverrideDate(date, dateOverride); err != nil {
		return err
	}
	return u.restorePaymentAccess(time.Now(), "")
}

func (u *UserUsecase) Skebob(url string) (string, error) {
	return u.yandexClient.GetYandexDirectLink(url)
}

func (u *UserUsecase) AddCompensationDays(daysCount int) error {
	if daysCount <= 0 {
		return fmt.Errorf("число дней компенсации должно быть положительным")
	}
	u.billingMu.Lock()
	defer u.billingMu.Unlock()
	if err := u.userRepository.AddCompensationDays(daysCount); err != nil {
		return err
	}
	return u.restorePaymentAccess(time.Now(), "")
}

func (u *UserUsecase) NormalizePaymentDatesToCurrentMonth(now time.Time) (int64, error) {
	u.billingMu.Lock()
	defer u.billingMu.Unlock()
	affected, err := u.userRepository.NormalizePaymentDatesToCurrentMonth(now)
	if err != nil {
		return affected, err
	}
	return affected, u.restorePaymentAccess(now, "")
}
