package interfaces

import (
	"VpnBot/internal/domain/ports/service"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestMarzbanConcurrentRequestsShareToken(t *testing.T) {
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"exp": time.Now().Add(time.Hour).Unix()}).SignedString([]byte("test-only"))
	if err != nil {
		t.Fatal(err)
	}
	var loginCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/admin/token" {
			loginCalls.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]string{"access_token": token, "token_type": "bearer"})
			return
		}
		if r.Header.Get("Authorization") != "bearer "+token {
			t.Error("request sent with an incomplete authorization header")
		}
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()
	client := NewMarzbanClient(server.URL, "test", "test")
	var wg sync.WaitGroup
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := client.GetUser("alice"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if loginCalls.Load() != 1 {
		t.Fatalf("expected a single token refresh, got %d", loginCalls.Load())
	}
}

func TestMarzbanMissingUserIsDistinguishableFromOutage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"detail":"User not found"}`)
	}))
	defer server.Close()
	client := &marzbanClient{baseUrl: server.URL, client: server.Client(), token: "cached", tokenType: "bearer", expires: time.Now().Add(time.Hour)}
	if err := client.SetUserStatus("missing", "disabled"); !errors.Is(err, service.ErrUserNotFound) {
		t.Fatalf("expected missing-user error, got %v", err)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type recordingBody struct {
	io.Reader
	closed bool
}

func (b *recordingBody) Close() error {
	b.closed = true
	return nil
}

func TestMarzbanDeleteClosesResponseBody(t *testing.T) {
	body := &recordingBody{Reader: strings.NewReader(`{}`)}
	client := &marzbanClient{
		baseUrl: "http://example.test", token: "cached", tokenType: "bearer", expires: time.Now().Add(time.Hour),
		client: &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
		})},
	}
	if err := client.Delete("alice"); err != nil {
		t.Fatal(err)
	}
	if !body.closed {
		t.Fatal("delete leaked its response body")
	}
}
