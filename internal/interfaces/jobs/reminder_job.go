package jobs

import (
	"VpnBot/internal/app/usecases"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/robfig/cron/v3"
	"log"
)

type ReminderJob struct {
	uc *usecases.ReminderUsecase
	tg *tgbotapi.BotAPI
}

func NewReminderJob(uc *usecases.ReminderUsecase, tg *tgbotapi.BotAPI) *ReminderJob {
	return &ReminderJob{uc: uc, tg: tg}
}

const msg = "Привет 👋  \nПора пополнить сервер, чтобы он продолжал работать.  \nСумма к оплате: 130 рублей 💳"

func (job *ReminderJob) Start() {
	cr := cron.New()

	_, err := cr.AddFunc("0 20 21 * *", func() {
		u, err := job.uc.InitReminder()
		if err != nil {
			log.Println("Ошибка при отправке напоминаний:", err)
		}

		for _, user := range u {
			_, _ = job.tg.Send(tgbotapi.NewMessage(user.Uid, msg))
		}

	})

	if err != nil {
		log.Println(err)
	}

	cr.Start()
}
