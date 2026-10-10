package router

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"VpnBot/config"
	"VpnBot/internal/app/usecases"
	"VpnBot/internal/domain/repository"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "modernc.org/sqlite"
)

type proxyBotClient struct {
	forms []url.Values
}

func (c *proxyBotClient) Do(req *http.Request) (*http.Response, error) {
	if err := req.ParseForm(); err != nil {
		return nil, err
	}
	c.forms = append(c.forms, req.Form)
	result := `{"message_id":100,"chat":{"id":42,"type":"private"}}`
	if strings.HasSuffix(req.URL.Path, "/getMe") {
		result = `{"id":999,"is_bot":true,"first_name":"test"}`
	}
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"ok":true,"result":` + result + `}`)), Header: make(http.Header)}, nil
}

func proxyCommandUpdate(uid int64, text string) tgbotapi.Update {
	command := strings.Fields(text)[0]
	return tgbotapi.Update{Message: &tgbotapi.Message{
		From: &tgbotapi.User{ID: uid}, Chat: &tgbotapi.Chat{ID: uid, Type: "private"}, Text: text,
		Entities: []tgbotapi.MessageEntity{{Type: "bot_command", Offset: 0, Length: len(command)}},
	}}
}

func setupProxyRouter(t *testing.T) (*usecases.ProxyUsecase, *UIRepos, *tgbotapi.BotAPI, *proxyBotClient) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	repo := repository.NewProxyRepository(db)
	if err := repo.Init(); err != nil {
		t.Fatal(err)
	}
	panel := repository.NewPanelRepository(db)
	if err := panel.Init(); err != nil {
		t.Fatal(err)
	}
	uc := usecases.NewProxyUsecase(repo)
	client := &proxyBotClient{}
	bot, err := tgbotapi.NewBotAPIWithClient("test", "https://fake.invalid/%s/%s", client)
	if err != nil {
		t.Fatal(err)
	}
	return uc, &UIRepos{Panel: panel, Proxy: uc}, bot, client
}

func TestProxyCommandsPermissionAndCRUD(t *testing.T) {
	uc, repos, bot, _ := setupProxyRouter(t)
	routes := NewCommandRouter(nil, &config.Config{AdminId: 42}, repos)
	commands := []string{
		"/addproxy Европа | socks5://proxy.example.com:1080",
		"/editproxy 1 Новый | https://proxy.example.com:8443",
		"/deleteproxy 1",
		"/proxylist",
	}
	for _, command := range commands {
		update := proxyCommandUpdate(7, command)
		routes[update.Message.Command()](update, bot)
	}
	if count, err := uc.Count(); err != nil || count != 0 {
		t.Fatalf("non-admin mutated catalog: count=%d err=%v", count, err)
	}
	for _, command := range commands[:2] {
		update := proxyCommandUpdate(42, command)
		routes[update.Message.Command()](update, bot)
	}
	p, err := uc.Get(1)
	if err != nil || p.Name != "Новый" || p.URL != "https://proxy.example.com:8443" {
		t.Fatalf("edit failed: proxy=%+v err=%v", p, err)
	}
	update := proxyCommandUpdate(42, commands[2])
	routes["deleteproxy"](update, bot)
	if count, err := uc.Count(); err != nil || count != 0 {
		t.Fatalf("delete failed: count=%d err=%v", count, err)
	}
}

func TestProxyCallbacksPaginationAndDeletedCard(t *testing.T) {
	uc, repos, bot, client := setupProxyRouter(t)
	for i := 0; i < 6; i++ {
		if _, err := uc.Add(fmt.Sprintf("Прокси %d", i), "http://proxy.example.com:8080"); err != nil {
			t.Fatal(err)
		}
	}
	routes := NewCallbackRouter(nil, nil, &config.Config{AdminId: 42}, repos)
	update := tgbotapi.Update{CallbackQuery: &tgbotapi.CallbackQuery{
		ID: "callback", From: &tgbotapi.User{ID: 42},
		Message: &tgbotapi.Message{MessageID: 100, Chat: &tgbotapi.Chat{ID: 42, Type: "private"}},
	}}
	update.CallbackQuery.Data = "proxyl:0"
	routes["proxyl:"](update, bot)
	form := client.forms[len(client.forms)-1]
	if !strings.Contains(form.Get("reply_markup"), "proxyl:5") || !strings.Contains(form.Get("text"), "Страница 1") {
		t.Fatalf("pagination missing: %v", form)
	}
	if err := uc.Delete(6); err != nil {
		t.Fatal(err)
	}
	update.CallbackQuery.Data = "proxyd:6:5"
	routes["proxyd:"](update, bot)
	form = client.forms[len(client.forms)-1]
	if !strings.Contains(form.Get("text"), "удалён") || !strings.Contains(form.Get("reply_markup"), "proxyl:5") {
		t.Fatalf("deleted card did not retain back navigation: %v", form)
	}
	update.CallbackQuery.Data = "proxyl:5"
	routes["proxyl:"](update, bot)
	form = client.forms[len(client.forms)-1]
	if !strings.Contains(form.Get("text"), "Страница 1") {
		t.Fatalf("stale page not clamped: %v", form)
	}
}

func TestCallbackAdminPermissionUsesSender(t *testing.T) {
	_, _, bot, _ := setupProxyRouter(t)
	update := tgbotapi.Update{CallbackQuery: &tgbotapi.CallbackQuery{
		ID: "callback", From: &tgbotapi.User{ID: 7},
		Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 42, Type: "private"}},
	}}
	if err := checkCallbackPermission(update, bot, 42); err == nil {
		t.Fatal("chat ID incorrectly granted admin rights to another sender")
	}
	update.CallbackQuery.From.ID = 42
	if err := checkCallbackPermission(update, bot, 42); err != nil {
		t.Fatal(err)
	}
	update.CallbackQuery.Message.Chat.Type = "group"
	if err := checkCallbackPermission(update, bot, 42); err == nil {
		t.Fatal("admin action allowed outside a private chat")
	}
	if err := checkCallbackPermission(tgbotapi.Update{}, bot, 42); err == nil {
		t.Fatal("invalid update accepted")
	}
}
