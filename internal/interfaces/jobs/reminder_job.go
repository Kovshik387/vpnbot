package jobs

import (
	"VpnBot/config"
	"VpnBot/internal/app/handlers/user"
	"VpnBot/internal/app/ui"
	"VpnBot/internal/app/usecases"
	"VpnBot/internal/domain/model"
	"VpnBot/internal/domain/repository"
	"VpnBot/internal/utils"
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/robfig/cron/v3"
)

type ReminderJob struct {
	userUC *usecases.UserUsecase
	tg     *tgbotapi.BotAPI
	cfg    *config.Config
	panel  *repository.PanelRepository
}

func NewReminderJob(userUC *usecases.UserUsecase, tg *tgbotapi.BotAPI, cfg *config.Config, panel *repository.PanelRepository) *ReminderJob {
	return &ReminderJob{userUC: userUC, tg: tg, cfg: cfg, panel: panel}
}

func (job *ReminderJob) Start() {
	cr := cron.New()

	//_, err := cr.AddFunc("*/1 * * * *", func() --debug
	_, err := cr.AddFunc("20 18 * * *", func() {
		dueToday, outs, err := job.userUC.ProcessBillingReminders(time.Now())
		if err != nil {
			log.Println("Ошибка при отправке напоминаний:", err)
		}

		job.sendAdminReport(dueToday)

		for _, out := range outs {
			u := out.User
			var text string
			switch out.Kind {
			case usecases.BillingRemind2d:
				text = fmt.Sprintf(
					"👋 Здравствуйте, <b>%s</b>\n\n"+
						"Напоминание: оплатите VPN <b>через 2 дня</b> (до %s), чтобы доступ не прерывался.\n\n"+
						"💳 Сумма: <b>%.2f</b>",
					utils.HtmlEscape(u.Username),
					formatDueDate(u.PaymentDate),
					u.Price,
				)
			case usecases.BillingRemind1d:
				text = fmt.Sprintf(
					"👋 Здравствуйте, <b>%s</b>\n\n"+
						"Напоминание: оплатите VPN <b>завтра</b> (до %s).\n\n"+
						"💳 Сумма: <b>%.2f</b>",
					utils.HtmlEscape(u.Username),
					formatDueDate(u.PaymentDate),
					u.Price,
				)
			case usecases.BillingRemindDue:
				text = fmt.Sprintf(
					"👋 Здравствуйте, <b>%s</b>\n\n"+
						"Сегодня — срок оплаты VPN (до %s). После полуночи доступ будет приостановлен до подтверждения оплаты.\n\n"+
						"💳 Сумма: <b>%.2f</b>",
					utils.HtmlEscape(u.Username),
					formatDueDate(u.PaymentDate),
					u.Price,
				)
			case usecases.BillingNewlyRevoked:
				text = fmt.Sprintf(
					"👋 Здравствуйте, <b>%s</b>\n\n"+
						"Срок оплаты истёк. Доступ к разделам бота приостановлен.\n"+
						"Нажмите «Оплата», пришлите скриншот перевода — после проверки администратором срок продлится на месяц.",
					utils.HtmlEscape(u.Username),
				)
			default:
				continue
			}

			var kb tgbotapi.InlineKeyboardMarkup
			if out.Kind == usecases.BillingNewlyRevoked {
				kb = ui.PaymentRevokedKeyboard()
			} else {
				kb = ui.PaymentReminderKeyboard()
			}
			if user.SendNotificationHTMLForUser(job.tg, u.Uid, text, &kb, true) {
				if err := job.userUC.CommitBillingReminderStage(u.Uid, out.Kind); err != nil {
					log.Printf("reminder stage uid=%d: %v", u.Uid, err)
				}
			}
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
			"✅ Сегодня нет пользователей с напоминанием об оплате.")
		msg.ParseMode = tgbotapi.ModeHTML
		_, err := job.tg.Send(msg)
		if err != nil {
			log.Printf("❌ Ошибка отправки админу: %v", err)
		}
		return
	}

	var report strings.Builder
	report.WriteString("<b>📅 Оплаты на сегодня</b>\n\n")

	totalAmount := 0.0
	paidUsers := 0

	for i, userModel := range users {
		report.WriteString(fmt.Sprintf("%d. 👤 <b>%s</b>\n", i+1, utils.HtmlEscape(userModel.Username)))
		report.WriteString(fmt.Sprintf("   📱 ID: <code>%d</code>\n", userModel.Uid))
		report.WriteString(fmt.Sprintf("   💰 Сумма: <b>%.2f</b>\n", userModel.Price))
		report.WriteString(fmt.Sprintf("   📨 Текст пользователю: сумма <b>%.2f</b>\n", userModel.Price))
		totalAmount += userModel.Price
		paidUsers++

		report.WriteString("   ─────\n")
	}

	report.WriteString(fmt.Sprintf("\n<b>📊 Итого</b>\n"))
	report.WriteString(fmt.Sprintf("   • В списке: %d\n", len(users)))
	report.WriteString(fmt.Sprintf("   • Уведомлений отправлено: %d\n", paidUsers))
	report.WriteString(fmt.Sprintf("   • Сумма: <b>%.2f</b>\n", totalAmount))
	report.WriteString(fmt.Sprintf("⏰ <b>Проверка</b> %s",
		time.Now().Format("15:04 02.01.2006")))

	msg := tgbotapi.NewMessage(job.cfg.AdminId, report.String())
	msg.ParseMode = tgbotapi.ModeHTML

	_, err := job.tg.Send(msg)
	if err != nil {
		log.Printf("❌ Ошибка отправки отчета админу: %v", err)
	}
}

func formatDueDate(t *time.Time) string {
	if t == nil {
		return "—"
	}
	return t.Format("02.01.2006")
}
