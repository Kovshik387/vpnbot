package admin

import (
	"VpnBot/internal/app/usecases"
	"VpnBot/internal/domain/model"
	"VpnBot/internal/domain/repository"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "modernc.org/sqlite"
)

type requestTestCall struct {
	method string
	params url.Values
}

type requestTestAPI struct {
	calls []requestTestCall
	fail  func(string, url.Values) string
}

func (f *requestTestAPI) Do(req *http.Request) (*http.Response, error) {
	if err := req.ParseForm(); err != nil {
		return nil, err
	}
	method := path.Base(req.URL.Path)
	f.calls = append(f.calls, requestTestCall{method, req.PostForm})
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

func (f *requestTestAPI) find(method string) []requestTestCall {
	var found []requestTestCall
	for _, call := range f.calls {
		if call.method == method {
			found = append(found, call)
		}
	}
	return found
}

type requestTestMarzban struct {
	added   []string
	deleted []string
	addErr  error
}

func (f *requestTestMarzban) Login() error { return nil }
func (f *requestTestMarzban) GetUsers() (model.UsersResponse, error) {
	return model.UsersResponse{}, nil
}
func (f *requestTestMarzban) GetUser(string) (model.User, error) { return model.User{}, nil }
func (f *requestTestMarzban) AddUser(username string) (model.User, error) {
	f.added = append(f.added, username)
	return model.User{Username: username, OnlineAt: "2026-01-01T00:00:00.000000", SubscribedUrl: "https://example.com/sub", Links: []string{"vless://test"}}, f.addErr
}
func (f *requestTestMarzban) Delete(username string) error {
	f.deleted = append(f.deleted, username)
	return nil
}
func (f *requestTestMarzban) SetUserStatus(string, string) error { return nil }

type requestFixture struct {
	db   *sql.DB
	ur   *repository.UserRepository
	uc   *usecases.UserUsecase
	cd   *usecases.CooldownUsecase
	marz *requestTestMarzban
	bot  *tgbotapi.BotAPI
	api  *requestTestAPI
}

func newRequestFixture(t *testing.T) requestFixture {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	ur := repository.NewUserRepository(db)
	if err := ur.EnsureSchema(); err != nil {
		t.Fatal(err)
	}
	cr := repository.NewCooldownRepository(db)
	if err := cr.Init(); err != nil {
		t.Fatal(err)
	}
	marz := &requestTestMarzban{}
	api := &requestTestAPI{}
	bot, err := tgbotapi.NewBotAPIWithClient("test", "https://telegram.invalid/bot%s/%s", api)
	if err != nil {
		t.Fatal(err)
	}
	api.calls = nil
	return requestFixture{db, ur, usecases.NewUserUsecase(marz, nil, ur, nil, nil), usecases.NewCooldownUsecase(cr), marz, bot, api}
}

func requestUpdate(data string) tgbotapi.Update {
	return tgbotapi.Update{CallbackQuery: &tgbotapi.CallbackQuery{
		ID: "callback", Data: data, From: &tgbotapi.User{ID: 202},
		Message: &tgbotapi.Message{MessageID: 33, Chat: &tgbotapi.Chat{ID: 202}},
	}}
}

func assertRequestAnswer(t *testing.T, api *requestTestAPI, contains string) {
	t.Helper()
	answers := api.find("answerCallbackQuery")
	if len(answers) != 1 || !strings.Contains(answers[0].params.Get("text"), contains) {
		t.Fatalf("expected one callback answer containing %q, got %+v", contains, answers)
	}
}

func TestRequestCallbacksRejectMalformedPayloads(t *testing.T) {
	for _, prefix := range []string{"approve:", "deny:", "block:"} {
		for _, suffix := range []string{"", "101", "abc user", "0 user", "-1 user", "9223372036854775808 user", "101 user extra", "101 user:extra"} {
			t.Run(prefix+suffix, func(t *testing.T) {
				f := newRequestFixture(t)
				update := requestUpdate(prefix + suffix)
				switch prefix {
				case "approve:":
					ApproveHandler(update, f.bot, f.uc, 202)
				case "deny:":
					DenyHandler(update, f.bot, f.cd, 202)
				case "block:":
					BlockUserHandler(update, f.bot, f.uc, 202)
				}
				assertRequestAnswer(t, f.api, "некорректная заявка")
				if len(f.marz.added) != 0 || len(f.api.find("sendMessage")) != 0 || len(f.api.find("deleteMessage")) != 0 {
					t.Fatalf("malformed callback caused action: added=%v calls=%+v", f.marz.added, f.api.calls)
				}
			})
		}
	}
}

func TestRequestCallbacksIgnoreMissingCallback(t *testing.T) {
	f := newRequestFixture(t)
	ApproveHandler(tgbotapi.Update{}, f.bot, f.uc, 202)
	DenyHandler(tgbotapi.Update{}, f.bot, f.cd, 202)
	BlockUserHandler(tgbotapi.Update{}, f.bot, f.uc, 202)
	if len(f.api.calls) != 0 {
		t.Fatalf("missing callback caused action: %+v", f.api.calls)
	}
}

func TestApprovalDoesNotRecreateMappedUser(t *testing.T) {
	f := newRequestFixture(t)
	if err := f.ur.Insert("assigned", 101, time.Now()); err != nil {
		t.Fatal(err)
	}
	ApproveHandler(requestUpdate("approve:101 renamed"), f.bot, f.uc, 202)
	assertRequestAnswer(t, f.api, "уже есть в базе")
	if len(f.marz.added) != 0 || len(f.marz.deleted) != 0 {
		t.Fatalf("existing mapping recreated or deleted: %+v", f.marz)
	}
	username, err := f.ur.GetUsernameByUserID(101)
	if err != nil || username != "assigned" {
		t.Fatalf("existing mapping changed: username=%q err=%v", username, err)
	}
}

func TestApprovalRollsBackOnlyNewRemoteUserAfterInsertFailure(t *testing.T) {
	f := newRequestFixture(t)
	if _, err := f.db.Exec(`create trigger reject_user_insert before insert on users begin select raise(abort, 'insert failed'); end`); err != nil {
		t.Fatal(err)
	}
	ApproveHandler(requestUpdate("approve:101 new_user"), f.bot, f.uc, 202)
	assertRequestAnswer(t, f.api, "Не удалось сохранить")
	if len(f.marz.added) != 1 || f.marz.added[0] != "new_user" || len(f.marz.deleted) != 1 || f.marz.deleted[0] != "new_user" {
		t.Fatalf("wrong rollback target: %+v", f.marz)
	}
	if len(f.api.find("sendMessage")) != 0 || len(f.api.find("deleteMessage")) != 0 {
		t.Fatalf("failed approval reported success or removed request: %+v", f.api.calls)
	}
}

func TestApprovalDoesNotDeleteAfterRemoteCreationFailure(t *testing.T) {
	f := newRequestFixture(t)
	f.marz.addErr = errors.New("existing remote account")
	ApproveHandler(requestUpdate("approve:101 new_user"), f.bot, f.uc, 202)
	assertRequestAnswer(t, f.api, "Не удалось создать ключ")
	if len(f.marz.deleted) != 0 {
		t.Fatalf("deleted account without creating it: %v", f.marz.deleted)
	}
}

func TestApprovalReportsUndeliveredMessageWithoutUndoingAccess(t *testing.T) {
	f := newRequestFixture(t)
	f.api.fail = func(method string, params url.Values) string {
		if method == "sendMessage" && params.Get("chat_id") == "101" {
			return "Bad Request: user blocked bot"
		}
		return ""
	}
	ApproveHandler(requestUpdate("approve:101 new_user"), f.bot, f.uc, 202)
	assertRequestAnswer(t, f.api, "сообщение не доставлено")
	username, err := f.ur.GetUsernameByUserID(101)
	if err != nil || username != "new_user" || len(f.marz.deleted) != 0 || len(f.api.find("deleteMessage")) != 0 {
		t.Fatalf("undelivered message lost access or removed request: user=%q deleted=%v calls=%+v err=%v", username, f.marz.deleted, f.api.calls, err)
	}
}

func TestApprovalSuccessfulDeliveryAcknowledgesAndRemovesRequest(t *testing.T) {
	f := newRequestFixture(t)
	ApproveHandler(requestUpdate("approve:101 new_user"), f.bot, f.uc, 202)
	assertRequestAnswer(t, f.api, "Ключ выдан")
	deletes := f.api.find("deleteMessage")
	if len(deletes) != 1 || deletes[0].params.Get("message_id") != "33" || len(f.api.find("editMessageReplyMarkup")) != 0 {
		t.Fatalf("wrong request cleanup: %+v", f.api.calls)
	}
}

func TestDenialDatabaseFailureDoesNotNotifyOrRemoveRequest(t *testing.T) {
	f := newRequestFixture(t)
	if err := f.db.Close(); err != nil {
		t.Fatal(err)
	}
	DenyHandler(requestUpdate("deny:101 new_user"), f.bot, f.cd, 202)
	assertRequestAnswer(t, f.api, "Не удалось отклонить")
	if len(f.api.find("sendMessage")) != 0 || len(f.api.find("deleteMessage")) != 0 {
		t.Fatalf("failed denial caused success actions: %+v", f.api.calls)
	}
}

func TestBlockDatabaseFailureDoesNotNotifyOrRemoveRequest(t *testing.T) {
	f := newRequestFixture(t)
	if err := f.ur.Insert("assigned", 101, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(`create trigger reject_user_block before update of is_block on users begin select raise(abort, 'block failed'); end`); err != nil {
		t.Fatal(err)
	}
	BlockUserHandler(requestUpdate("block:101 assigned"), f.bot, f.uc, 202)
	assertRequestAnswer(t, f.api, "Не удалось заблокировать")
	if len(f.api.find("sendMessage")) != 0 || len(f.api.find("deleteMessage")) != 0 {
		t.Fatalf("failed block caused success actions: %+v", f.api.calls)
	}
	blocked, err := f.ur.CheckBlock(101)
	if err != nil || blocked {
		t.Fatalf("failed block changed state: blocked=%v err=%v", blocked, err)
	}
}

func TestBlockSuccessPersistsBeforeNotification(t *testing.T) {
	f := newRequestFixture(t)
	BlockUserHandler(requestUpdate("block:101 new_user"), f.bot, f.uc, 202)
	assertRequestAnswer(t, f.api, "Пользователь заблокирован")
	blocked, err := f.ur.CheckBlock(101)
	if err != nil || !blocked || len(f.marz.added) != 0 {
		t.Fatalf("wrong block state: blocked=%v remote=%v err=%v", blocked, f.marz.added, err)
	}
}
