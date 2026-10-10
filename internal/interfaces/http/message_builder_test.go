package interfaces

import (
	"VpnBot/internal/domain/model"
	"strings"
	"testing"
)

func TestMessageBuilderMissingLinks(t *testing.T) {
	text, _, err := NewMessageBuilder().SendUserInfo(model.User{Username: "alice"}, 250)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Конфиг (ПК)</b> Пока недоступен") || !strings.Contains(text, "Подписка (телефон)</b> Пока недоступна") {
		t.Fatalf("missing connection data should have clear placeholders: %s", text)
	}
}

func TestMessageBuilderEscapesLinksAndSupportsRFC3339(t *testing.T) {
	user := model.User{
		Username:      "<alice>",
		SubscribedUrl: `https://example.test/sub?token="<&other=1`,
		OnlineAt:      "2026-04-10T09:00:00Z",
		Links:         []string{"", "vless://<token>&test"},
	}
	text, _, err := NewMessageBuilder().SendUserInfo(user, 250)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"&lt;alice&gt;", "token=&quot;&lt;&amp;other=1", "vless://&lt;token&gt;&amp;test", "10.04.2026 12:00"} {
		if !strings.Contains(text, expected) {
			t.Errorf("expected %q in message: %s", expected, text)
		}
	}
}
