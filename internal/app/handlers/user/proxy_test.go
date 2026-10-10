package user

import (
	"net/url"
	"strings"
	"testing"

	"VpnBot/internal/domain/model"
)

func TestProxyCardConnectionDetails(t *testing.T) {
	cases := []struct {
		name, rawURL, kind, connectPath string
	}{
		{"MTProto", "tg://proxy?server=proxy.example.com&port=443&secret=0123456789abcdef0123456789abcdef", "Telegram MTProto", "/proxy"},
		{"uppercase MTProto", "tg://PROXY?server=proxy.example.com&port=443&secret=0123456789abcdef0123456789abcdef", "Telegram MTProto", "/proxy"},
		{"Telegram SOCKS", "https://t.me/socks?server=proxy.example.com&port=1080&user=user&pass=p%26ss", "SOCKS5", "/socks"},
		{"SOCKS", "socks5://user:p%26ss@proxy.example.com:1080", "SOCKS5", "/socks"},
		{"IPv6", "socks5h://[2001:db8::1]:1080", "SOCKS5", "/socks"},
		{"HTTP", "http://user:p%26ss@proxy.example.com:8080", "HTTP", ""},
		{"HTTPS", "https://proxy.example.com:8443", "HTTPS", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			text, connect := proxyCard(model.Proxy{ID: 1, Name: "A < B & C", URL: tc.rawURL})
			if strings.Contains(text, "A < B") || !strings.Contains(text, "A &lt; B &amp; C") || !strings.Contains(text, tc.kind+" · #1") {
				t.Fatalf("card not escaped or missing type: %s", text)
			}
			if strings.Contains(text, "<tg-spoiler><code>") {
				t.Fatal("Telegram does not support nesting code inside spoiler")
			}
			if tc.connectPath == "" {
				if connect != "" || !strings.Contains(text, "настройках прокси браузера") {
					t.Fatalf("browser proxy has wrong action: %s", connect)
				}
				return
			}
			u, err := url.Parse(connect)
			if err != nil || u.Hostname() != "t.me" || u.Path != tc.connectPath || u.Query().Get("server") == "" {
				t.Fatalf("invalid Telegram connect URL: %s", connect)
			}
			if strings.Contains(tc.rawURL, "p%26ss") && (u.Query().Get("pass") != "p&ss" || !strings.Contains(text, "p&amp;ss")) {
				t.Fatalf("credentials not decoded or escaped: %s", text)
			}
		})
	}
}
