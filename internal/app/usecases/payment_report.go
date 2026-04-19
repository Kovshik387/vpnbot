package usecases

import (
	"VpnBot/internal/domain/repository"
	"fmt"
	"strings"
	"time"
)

func (u *UserUsecase) LogConfirmedPayment(userID int64) error {
	if u.paymentReportRepo == nil {
		return nil
	}
	username, err := u.userRepository.GetUsernameByUserID(userID)
	if err != nil {
		return err
	}
	amount, err := u.userRepository.GetPriceByUserID(userID)
	if err != nil {
		return err
	}
	if username == "" {
		username = fmt.Sprintf("id_%d", userID)
	}
	return u.paymentReportRepo.Add(userID, username, amount)
}

func (u *UserUsecase) PaymentMonthlyReport(month string) (string, error) {
	if u.paymentReportRepo == nil {
		return "Сервис отчётности не подключён.", nil
	}
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	if _, err := time.Parse("2006-01", month); err != nil {
		return "", fmt.Errorf("месяц должен быть в формате YYYY-MM")
	}

	items, err := u.paymentReportRepo.Monthly(month)
	if err != nil {
		return "", err
	}
	if len(items) == 0 {
		return fmt.Sprintf("📊 Подтверждённых оплат за %s нет.", month), nil
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("<b>📊 Оплаты за %s</b>\n\n", month))
	total := 0.0
	for i, e := range items {
		b.WriteString(fmt.Sprintf("%d) <b>%s</b> · ID <code>%d</code>\n", i+1, e.Username, e.UserID))
		b.WriteString(fmt.Sprintf("   💰 %.2f\n", e.Amount))
		b.WriteString(fmt.Sprintf("   🗓 %s\n", e.ConfirmedAt))
		total += e.Amount
	}
	b.WriteString(fmt.Sprintf("\n<b>Итого</b>\n• Подтверждений: %d\n• Сумма: <b>%.2f</b>", len(items), total))
	return b.String(), nil
}

func normalizeMonthArg(arg string) string {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return ""
	}
	return arg
}

func monthOrNow(arg string) string {
	m := normalizeMonthArg(arg)
	if m == "" {
		return time.Now().Format("2006-01")
	}
	return m
}

func parseMonthArg(arg string) (string, error) {
	month := monthOrNow(arg)
	if _, err := time.Parse("2006-01", month); err != nil {
		return "", fmt.Errorf("месяц должен быть в формате YYYY-MM")
	}
	return month, nil
}

func (u *UserUsecase) PaymentMonthlyReportByArg(arg string) (string, error) {
	month, err := parseMonthArg(arg)
	if err != nil {
		return "", err
	}
	return u.PaymentMonthlyReport(month)
}

func (u *UserUsecase) PaymentReportEntries(month string) ([]repository.PaymentConfirmEntry, error) {
	if u.paymentReportRepo == nil {
		return nil, nil
	}
	return u.paymentReportRepo.Monthly(month)
}
