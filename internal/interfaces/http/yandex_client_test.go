package interfaces

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestYandexPublicLinkIsEncodedAsOneQueryParameter(t *testing.T) {
	publicLink := "https://disk.yandex.ru/d/id?x=a&y=b#fragment"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("public_key"); got != publicLink {
			t.Errorf("public key altered during request: got %q", got)
		}
		if len(r.URL.Query()) != 1 {
			t.Error("public link leaked into separate query parameters")
		}
		_, _ = io.WriteString(w, `{"file":"https://example.test/download"}`)
	}))
	defer server.Close()
	client := yandexClient{client: server.Client(), baseURL: server.URL}
	if link, err := client.GetYandexDirectLink(publicLink); err != nil || link != "https://example.test/download" {
		t.Fatalf("unexpected download response: link=%q err=%v", link, err)
	}
}
