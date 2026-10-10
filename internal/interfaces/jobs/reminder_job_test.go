package jobs

import (
	"VpnBot/config"
	"VpnBot/internal/app/usecases"
	"VpnBot/internal/domain/repository"
	"database/sql"
	"io"
	"net/http"
	"path"
	"strings"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "modernc.org/sqlite"
)

type fakeTelegramClient struct {
	userMessages  int
	adminMessages int
}

func (f *fakeTelegramClient) Do(req *http.Request) (*http.Response, error) {
	body := `{"ok":true,"result":{"message_id":77}}`
	if path.Base(req.URL.Path) == "getMe" {
		body = `{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"Test"}}`
	} else if path.Base(req.URL.Path) == "sendMessage" {
		if err := req.ParseForm(); err != nil {
			return nil, err
		}
		if req.PostForm.Get("chat_id") == "999" {
			f.adminMessages++
		} else {
			f.userMessages++
		}
	}
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
}

func TestPeriodicBillingRunDoesNotRepeatRemindersOrAdminReports(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(1)
	repo := repository.NewUserRepository(db)
	if err := repo.EnsureSchema(); err != nil {
		t.Fatal(err)
	}
	if err := repo.Insert("payer", 101, time.Now()); err != nil {
		t.Fatal(err)
	}
	client := &fakeTelegramClient{}
	bot, err := tgbotapi.NewBotAPIWithClient("test", "https://telegram.invalid/bot%s/%s", client)
	if err != nil {
		t.Fatal(err)
	}
	uc := usecases.NewUserUsecase(nil, nil, repo, nil, nil)
	job := NewReminderJob(uc, bot, &config.Config{AdminId: 999}, nil)
	job.run(false)
	job.run(false)
	if client.userMessages != 1 || client.adminMessages != 0 {
		t.Fatalf("periodic checks spammed messages: users=%d admin=%d", client.userMessages, client.adminMessages)
	}
	job.run(true)
	if client.userMessages != 1 || client.adminMessages != 1 {
		t.Fatalf("daily report should be the only extra message: users=%d admin=%d", client.userMessages, client.adminMessages)
	}
}
