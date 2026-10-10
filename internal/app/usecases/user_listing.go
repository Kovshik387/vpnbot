package usecases

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"VpnBot/internal/domain/model"
)

// ListUserPage searches VPN users, including users without a bot account.
func (u *UserUsecase) ListUserPage(query string, activeOnly bool, offset, limit int) ([]model.User, int, error) {
	query = strings.TrimSpace(query)
	if !utf8.ValidString(query) || utf8.RuneCountInString(query) > 80 || strings.IndexFunc(query, unicode.IsControl) >= 0 {
		return nil, 0, fmt.Errorf("поисковый запрос должен быть не длиннее 80 символов и не содержать управляющих символов")
	}
	if offset < 0 || limit < 1 || limit > 50 {
		return nil, 0, fmt.Errorf("некорректная страница пользователей")
	}
	query = strings.ToLower(strings.TrimPrefix(query, "@"))
	var pattern *regexp.Regexp
	if strings.Contains(query, "*") {
		pattern = regexp.MustCompile("^" + strings.ReplaceAll(regexp.QuoteMeta(query), `\*`, ".*") + "$")
	}
	response, err := u.ListUsers()
	if err != nil {
		return nil, 0, err
	}
	now := time.Now()
	users := make([]model.User, 0, len(response.Users))
	for _, entry := range response.Users {
		name := strings.ToLower(entry.Username)
		matches := strings.Contains(name, query)
		if pattern != nil {
			matches = pattern.MatchString(name)
		}
		if matches && (!activeOnly || UserIsOnline(entry, now)) {
			users = append(users, entry)
		}
	}
	sort.Slice(users, func(i, j int) bool {
		a, b := strings.ToLower(users[i].Username), strings.ToLower(users[j].Username)
		if a == b {
			return users[i].Username < users[j].Username
		}
		return a < b
	})
	total := len(users)
	if total == 0 {
		return nil, 0, nil
	}
	if offset >= total {
		offset = (total - 1) / limit * limit
	}
	return users[offset:min(offset+limit, total)], total, nil
}

func UserIsOnline(entry model.User, now time.Time) bool {
	if entry.Status != "active" {
		return false
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.999999"} {
		if onlineAt, err := time.Parse(layout, entry.OnlineAt); err == nil {
			elapsed := now.Sub(onlineAt)
			return elapsed >= 0 && elapsed <= 5*time.Minute
		}
	}
	return false
}
