package user

import (
	"VpnBot/internal/app/usecases"
	"VpnBot/internal/domain/model"
	"VpnBot/internal/domain/repository"
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "modernc.org/sqlite"
)

type telegramCall struct {
	method string
	params url.Values
}

type fakeTelegram struct {
	calls []telegramCall
	fail  func(string, url.Values) string
}

func (f *fakeTelegram) Do(req *http.Request) (*http.Response, error) {
	if err := req.ParseForm(); err != nil {
		return nil, err
	}
	method := path.Base(req.URL.Path)
	f.calls = append(f.calls, telegramCall{method, req.PostForm})
	body := `{"ok":true,"result":{"message_id":77}}`
	if method == "getMe" {
		body = `{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"Test"}}`
	}
	if f.fail != nil {
		if reason := f.fail(method, req.PostForm); reason != "" {
			body = fmt.Sprintf(`{"ok":false,"error_code":400,"description":%q}`, reason)
		}
	}
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
}

func newTestBot(t *testing.T) (*tgbotapi.BotAPI, *fakeTelegram) {
	t.Helper()
	f := &fakeTelegram{}
	bot, err := tgbotapi.NewBotAPIWithClient("test", "https://telegram.invalid/bot%s/%s", f)
	if err != nil {
		t.Fatal(err)
	}
	f.calls = nil
	return bot, f
}

func newTestDB(t *testing.T) (*sql.DB, *repository.PanelRepository) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	pr := repository.NewPanelRepository(db)
	if err := pr.Init(); err != nil {
		t.Fatal(err)
	}
	return db, pr
}

func callbackUpdate(messageID int, data string) tgbotapi.Update {
	return tgbotapi.Update{CallbackQuery: &tgbotapi.CallbackQuery{
		ID: "callback", Data: data, From: &tgbotapi.User{ID: 101, UserName: "renamed"},
		Message: &tgbotapi.Message{MessageID: messageID, Chat: &tgbotapi.Chat{ID: 101}},
	}}
}

func (f *fakeTelegram) find(method string) []telegramCall {
	var calls []telegramCall
	for _, call := range f.calls {
		if call.method == method {
			calls = append(calls, call)
		}
	}
	return calls
}

func TestCallbackUpdatesSavedPanelBeforeRemovingNotification(t *testing.T) {
	_, pr := newTestDB(t)
	if err := pr.Set(101, 101, 7); err != nil {
		t.Fatal(err)
	}
	bot, api := newTestBot(t)
	EditPanelFromUpdate(bot, pr, callbackUpdate(9, "info"), 101, "Настройки", nil, true)
	if len(api.calls) != 2 || api.calls[0].method != "editMessageText" || api.calls[0].params.Get("message_id") != "7" ||
		api.calls[1].method != "deleteMessage" || api.calls[1].params.Get("message_id") != "9" {
		t.Fatalf("expected saved panel edit followed by notification deletion, got %+v", api.calls)
	}
	_, mid, ok := pr.Get(101)
	if !ok || mid != 7 {
		t.Fatalf("saved panel changed: message=%d ok=%v", mid, ok)
	}
}

func TestPanelReplacementFailurePreservesMessages(t *testing.T) {
	_, pr := newTestDB(t)
	if err := pr.Set(101, 101, 7); err != nil {
		t.Fatal(err)
	}
	bot, api := newTestBot(t)
	api.fail = func(method string, _ url.Values) string {
		switch method {
		case "editMessageText":
			return "Bad Request: message to edit not found"
		case "sendMessage":
			return "Bad Request: temporary failure"
		}
		return ""
	}
	EditPanelFromUpdate(bot, pr, callbackUpdate(9, "panel_home"), 101, "Главная", nil, true)
	if len(api.find("deleteMessage")) != 0 {
		t.Fatalf("messages deleted after failed replacement: %+v", api.calls)
	}
	_, mid, ok := pr.Get(101)
	if !ok || mid != 7 {
		t.Fatalf("saved panel lost after failed replacement: message=%d ok=%v", mid, ok)
	}
}

func TestNewPanelPreservesPreviewSetting(t *testing.T) {
	_, pr := newTestDB(t)
	bot, api := newTestBot(t)
	update := tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 101}}}
	EditPanelFromUpdate(bot, pr, update, 101, "<a href=\"https://example.com\">Ссылка</a>", nil, true)
	sends := api.find("sendMessage")
	if len(sends) != 1 || sends[0].params.Get("disable_web_page_preview") != "true" {
		t.Fatalf("preview setting lost: %+v", sends)
	}
}

type fakeSubscriptionService struct {
	lookups []string
}

func (f *fakeSubscriptionService) Login() error { return nil }
func (f *fakeSubscriptionService) GetUsers() (model.UsersResponse, error) {
	return model.UsersResponse{}, nil
}
func (f *fakeSubscriptionService) GetUser(username string) (model.User, error) {
	f.lookups = append(f.lookups, username)
	return model.User{Username: username, OnlineAt: "2026-01-01T00:00:00.000000", SubscribedUrl: "https://example.com/sub", Links: []string{"vless://test"}}, nil
}
func (f *fakeSubscriptionService) AddUser(string) (model.User, error) { return model.User{}, nil }
func (f *fakeSubscriptionService) Delete(string) error                { return nil }
func (f *fakeSubscriptionService) SetUserStatus(string, string) error { return nil }

func newTestUserUsecase(t *testing.T, db *sql.DB) (*usecases.UserUsecase, *repository.UserRepository, *fakeSubscriptionService) {
	t.Helper()
	ur := repository.NewUserRepository(db)
	if err := ur.EnsureSchema(); err != nil {
		t.Fatal(err)
	}
	service := &fakeSubscriptionService{}
	return usecases.NewUserUsecase(service, nil, ur, nil, nil), ur, service
}

func TestSubscriptionWithoutMappingDoesNotQueryTelegramUsername(t *testing.T) {
	db, pr := newTestDB(t)
	uc, _, service := newTestUserUsecase(t, db)
	bot, api := newTestBot(t)
	GetSubscribeHandler(callbackUpdate(7, "subscribe"), bot, uc, pr)
	if len(service.lookups) != 0 {
		t.Fatalf("queried unlinked Telegram username: %v", service.lookups)
	}
	edits := api.find("editMessageText")
	if len(edits) != 1 || !strings.Contains(edits[0].params.Get("text"), "пока нет выданного ключа") ||
		!strings.Contains(edits[0].params.Get("reply_markup"), "request_key") {
		t.Fatalf("missing onboarding action: %+v", edits)
	}
}

func TestSubscriptionUsesStoredMappingAfterTelegramRename(t *testing.T) {
	db, pr := newTestDB(t)
	uc, ur, service := newTestUserUsecase(t, db)
	if err := ur.Insert("assigned", 101, time.Now()); err != nil {
		t.Fatal(err)
	}
	bot, _ := newTestBot(t)
	GetSubscribeHandler(callbackUpdate(7, "subscribe"), bot, uc, pr)
	if len(service.lookups) != 1 || service.lookups[0] != "assigned" {
		t.Fatalf("queried wrong subscription: %v", service.lookups)
	}
}

func TestRevokedSubscriptionShowsPaymentInsteadOfVPNConfig(t *testing.T) {
	db, pr := newTestDB(t)
	uc, ur, service := newTestUserUsecase(t, db)
	if err := ur.Insert("assigned", 101, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := ur.SetPaymentAccessRevoked(101, true); err != nil {
		t.Fatal(err)
	}
	bot, api := newTestBot(t)
	GetSubscribeHandler(callbackUpdate(7, "subscribe"), bot, uc, pr)
	if len(service.lookups) != 0 {
		t.Fatalf("revoked subscription queried VPN config: %v", service.lookups)
	}
	edits := api.find("editMessageText")
	if len(edits) != 1 || !strings.Contains(edits[0].params.Get("text"), "Доступ приостановлен") ||
		!strings.Contains(edits[0].params.Get("reply_markup"), "payment_flow") {
		t.Fatalf("missing payment status: %+v", edits)
	}
}

func TestFreeSubscriptionDoesNotDisplayPaymentAmountOrDate(t *testing.T) {
	db, pr := newTestDB(t)
	uc, ur, _ := newTestUserUsecase(t, db)
	if err := ur.Insert("assigned", 101, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := ur.UpdateTypePayment("assigned", true); err != nil {
		t.Fatal(err)
	}
	bot, api := newTestBot(t)
	GetSubscribeHandler(callbackUpdate(7, "subscribe"), bot, uc, pr)
	edits := api.find("editMessageText")
	if len(edits) != 1 {
		t.Fatalf("missing subscription panel: %+v", edits)
	}
	text := edits[0].params.Get("text")
	if !strings.Contains(text, "бесплатный") || strings.Contains(text, "Оплачен до") || strings.Contains(text, "<b>Оплата</b>") {
		t.Fatalf("wrong free tier display: %s", text)
	}
}

func TestAnnouncementDetailKeepsHTMLAndUTF8Valid(t *testing.T) {
	db, pr := newTestDB(t)
	sl := repository.NewSayLogRepository(db)
	if err := sl.Init(); err != nil {
		t.Fatal(err)
	}
	body := strings.Repeat("Я&😀<", 1000)
	if err := sl.Insert("text", body); err != nil {
		t.Fatal(err)
	}
	bot, api := newTestBot(t)
	SayLogDetail(callbackUpdate(7, "sayd:1"), bot, sl, pr, 1)
	edits := api.find("editMessageText")
	if len(edits) != 1 {
		t.Fatalf("missing announcement: %+v", edits)
	}
	text := edits[0].params.Get("text")
	if !utf8.ValidString(text) {
		t.Fatal("announcement contains invalid UTF-8")
	}
	decoder := xml.NewDecoder(strings.NewReader("<div>" + text + "</div>"))
	for {
		_, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("invalid HTML entity or tag: %v", err)
		}
	}
}

func TestTruncateMessageTextUsesUTF16Boundaries(t *testing.T) {
	for _, input := range []string{strings.Repeat("Я", 3500), strings.Repeat("Я", 3501), strings.Repeat("😀", 1800)} {
		got := truncateMessageText(input, 3500)
		if !utf8.ValidString(got) || len(utf16.Encode([]rune(got))) > 3500 {
			t.Fatalf("invalid truncated message (%d UTF-16 units)", len(utf16.Encode([]rune(got))))
		}
		if len(utf16.Encode([]rune(input))) <= 3500 && got != input {
			t.Fatal("message within limit was truncated")
		}
	}
}

func TestKeyRequestOnlyConfirmsSuccessfulDeliveryAndSetsCooldown(t *testing.T) {
	for _, failSend := range []bool{true, false} {
		t.Run(strconv.FormatBool(failSend), func(t *testing.T) {
			db, pr := newTestDB(t)
			uc, _, _ := newTestUserUsecase(t, db)
			cr := repository.NewCooldownRepository(db)
			if err := cr.Init(); err != nil {
				t.Fatal(err)
			}
			cd := usecases.NewCooldownUsecase(cr)
			bot, api := newTestBot(t)
			api.fail = func(method string, params url.Values) string {
				if failSend && method == "sendMessage" && params.Get("chat_id") == "202" {
					return "Bad Request: unavailable"
				}
				return ""
			}
			GetKeyHandler(callbackUpdate(7, "request_key"), bot, cd, 202, pr, uc)
			callbacks := api.find("answerCallbackQuery")
			if len(callbacks) != 1 {
				t.Fatalf("expected one callback response, got %+v", callbacks)
			}
			confirmed := callbacks[0].params.Get("text") == "Отправлено"
			onCooldown, _, err := cd.IsOnCooldown(101)
			if err != nil || confirmed == failSend || onCooldown == failSend {
				t.Fatalf("delivery=%v cooldown=%v sendFailure=%v err=%v", confirmed, onCooldown, failSend, err)
			}
		})
	}
}
