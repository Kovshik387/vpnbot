package jobs

import (
	"VpnBot/config"
	"VpnBot/internal/app/usecases"
	"VpnBot/internal/domain/model"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/robfig/cron/v3"
	"log"
	"strings"
	"time"
)

type ReminderJob struct {
	uc  *usecases.ReminderUsecase
	tg  *tgbotapi.BotAPI
	cfg *config.Config
}

func NewReminderJob(uc *usecases.ReminderUsecase, tg *tgbotapi.BotAPI, cfg *config.Config) *ReminderJob {
	return &ReminderJob{uc: uc, tg: tg, cfg: cfg}
}

func (job *ReminderJob) Start() {
	cr := cron.New()

	log.Println("Pepe")
	//_, err := cr.AddFunc("*/1 * * * *", func()
	_, err := cr.AddFunc("* 20 * * *", func() {
		u, err := job.uc.InitReminder()
		if err != nil {
			log.Println("Ошибка при отправке напоминаний:", err)
		}

		job.sendAdminReport(u)

		for _, user := range u {

			msg := tgbotapi.NewMessage(
				user.Uid,
				fmt.Sprintf(
					"👋 Привет, %s!\n\n"+
						"Пора пополнить сервер, чтобы он продолжал работать 💳.\n\n"+
						"*Сумма: %.2f*",
					user.Username,
					user.Price,
				),
			)

			msg.ParseMode = tgbotapi.ModeMarkdown

			_, _ = job.tg.Send(msg)
		}

	})

	if err != nil {
		log.Println(err)
	}

	cr.Start()
}

func (job *ReminderJob) sendAdminReport(users []model.TgUserModel) {
	if len(users) == 0 {
		msg := tgbotapi.NewMessage(job.cfg.AdminId,
			"✅ *Сегодня нет пользователей с истекающей подпиской*")
		msg.ParseMode = "Markdown"
		_, err := job.tg.Send(msg)
		if err != nil {
			log.Printf("❌ Ошибка отправки админу: %v", err)
		}
		return
	}

	var report strings.Builder
	report.WriteString("📅 *Список пользователей на оплату сегодня:*\n\n")

	totalAmount := 0.0
	paidUsers := 0

	for i, user := range users {
		report.WriteString(fmt.Sprintf("%d. 👤 *%s*\n", i+1, user.Username))
		report.WriteString(fmt.Sprintf("   📱 ID: `%d`\n", user.Uid))
		report.WriteString(fmt.Sprintf("   💰 Сумма: *%.2f*\n", user.Price))
		report.WriteString(fmt.Sprintf("   📨 Получит: \"Сумма к оплате: %.2f\"\n", user.Price))
		totalAmount += user.Price
		paidUsers++

		report.WriteString("   ─────\n")
	}

	report.WriteString(fmt.Sprintf("\n📊 *Статистика:*\n"))
	report.WriteString(fmt.Sprintf("   • Всего в списке: %d пользователей\n", len(users)))
	report.WriteString(fmt.Sprintf("   • Получат уведомление: %d\n", paidUsers))
	report.WriteString(fmt.Sprintf("   • Общая сумма: *%.2f*\n", totalAmount))
	report.WriteString(fmt.Sprintf("⏰ *Время проверки:* %s",
		time.Now().Format("15:04 02.01.2006")))

	msg := tgbotapi.NewMessage(job.cfg.AdminId, report.String())
	msg.ParseMode = "Markdown"

	_, err := job.tg.Send(msg)
	if err != nil {
		log.Printf("❌ Ошибка отправки отчета админу: %v", err)
	}
}
