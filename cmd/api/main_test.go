package main

import (
	"database/sql"
	"testing"
	"time"

	"VpnBot/internal/app/usecases"
	"VpnBot/internal/domain/repository"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "modernc.org/sqlite"
)

func TestPrivateUpdateGuards(t *testing.T) {
	m := &tgbotapi.Message{From: &tgbotapi.User{ID: 7}, Chat: &tgbotapi.Chat{ID: 7, Type: "private"}}
	if !privateMessage(m) {
		t.Fatal("valid private message rejected")
	}
	q := &tgbotapi.CallbackQuery{From: m.From, Message: m}
	if !privateCallback(q) {
		t.Fatal("valid private callback rejected")
	}
	m.Chat.ID = 42
	if privateCallback(q) {
		t.Fatal("callback from another user accepted")
	}
	m.Chat.Type = "group"
	if privateMessage(m) || privateCallback(q) {
		t.Fatal("group update accepted")
	}
	if privateMessage(nil) || privateMessage(&tgbotapi.Message{}) || privateCallback(nil) || privateCallback(&tgbotapi.CallbackQuery{}) {
		t.Fatal("incomplete update accepted")
	}
}

func TestRevokedAccessRetainsStatusHelpAndPayment(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer func() { _ = db.Close() }()
	repo := repository.NewUserRepository(db)
	if err := repo.EnsureSchema(); err != nil {
		t.Fatal(err)
	}
	if err := repo.Insert("alice", 7, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetPaymentAccessRevoked(7, true); err != nil {
		t.Fatal(err)
	}
	uc := usecases.NewUserUsecase(nil, nil, repo, nil, nil)
	for _, command := range []string{"start", "subscribe", "help", "proxies"} {
		m := &tgbotapi.Message{From: &tgbotapi.User{ID: 7}, Text: "/" + command,
			Entities: []tgbotapi.MessageEntity{{Type: "bot_command", Length: len(command) + 1}}}
		if got, want := paymentRevokedBlocksCommand(uc, m), command == "proxies"; got != want {
			t.Errorf("command %s blocked=%v, want %v", command, got, want)
		}
	}
	for _, data := range []string{"payment_flow", "panel_home", "help", "subscribe", "proxyl:0", "info"} {
		want := data == "proxyl:0" || data == "info"
		if got := paymentRevokedBlocksCallback(uc, 7, data); got != want {
			t.Errorf("callback %s blocked=%v, want %v", data, got, want)
		}
	}
}
