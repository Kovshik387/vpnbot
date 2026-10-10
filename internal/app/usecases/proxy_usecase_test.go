package usecases

import (
	"VpnBot/internal/domain/repository"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/url"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func setupProxyUsecase(t *testing.T) *ProxyUsecase {
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
	return NewProxyUsecase(repo)
}

func TestProxyUsecaseValidURLs(t *testing.T) {
	secret := "000102030405060708090a0b0c0d0e0f"
	tlsSecret := "ee" + secret + hex.EncodeToString([]byte("example.com"))
	secretBytes, _ := hex.DecodeString(secret)
	paddedSecret := append([]byte{0xdd}, secretBytes...)
	tlsSecretBytes, _ := hex.DecodeString(tlsSecret)
	urls := []string{
		"tg://proxy?server=example.com&port=443&secret=" + secret,
		"https://t.me/proxy?server=192.0.2.1&port=443&secret=dd" + secret,
		"tg://proxy?server=example.com&port=443&secret=" + strings.ToUpper(tlsSecret),
		"https://t.me/proxy?server=example.com&port=443&secret=" + base64.RawURLEncoding.EncodeToString(tlsSecretBytes),
		"tg://proxy?server=example.com&port=443&secret=" + url.QueryEscape(base64.URLEncoding.EncodeToString(paddedSecret)),
		"tg://proxy?server=example.com&port=443&secret=" + base64.RawURLEncoding.EncodeToString(secretBytes),
		"tg://proxy?server=example.com&port=443&secret=" + url.QueryEscape(base64.StdEncoding.EncodeToString(bytesForStandardBase64Test())),
		"tg://socks?server=example.com&port=1080",
		"https://t.me/socks?server=example.com&port=1080&user=alice&pass=p%26ss%2Bword",
		"tg://socks?server=%3A%3A1&port=1080",
		"socks5://example.com:1080",
		"socks5h://alice:p%40ss@example.com:1080",
		"socks5://[2001:db8::1]:1080",
		"http://example.com:8080",
		"https://alice:secret@example.com:8443",
		"http://[2001:db8::1]:8080",
		"https://пример.рф:443",
	}
	u := setupProxyUsecase(t)
	for _, proxyURL := range urls {
		t.Run(proxyURL, func(t *testing.T) {
			proxy, err := u.Add("  Москва 🐈  ", "  "+proxyURL+"  ")
			if err != nil {
				t.Fatal(err)
			}
			if proxy.Name != "Москва 🐈" || proxy.URL != proxyURL || proxy.ID <= 0 {
				t.Fatalf("unexpected normalized proxy: %+v", proxy)
			}
		})
	}
}

func bytesForStandardBase64Test() []byte {
	// Ensures the standard base64 alphabet includes + and /, which must be escaped in a query.
	return []byte{0xfb, 0xff, 0xff, 0xfb, 0xff, 0xff, 0xfb, 0xff, 0xff, 0xfb, 0xff, 0xff, 0xfb, 0xff, 0xff, 0xfb}
}

func TestProxyUsecaseRejectsInvalidURLsWithoutWriting(t *testing.T) {
	secret := "000102030405060708090a0b0c0d0e0f"
	urls := []string{
		"",
		"javascript:alert(1)",
		"https://example.com/",
		"https://t.me/channel",
		"ftp://example.com:1080",
		"tg://resolve?domain=channel",
		"tg://proxy?server=example.com&port=443",
		"tg://proxy?server=example.com&port=443&secret=invalid",
		"tg://proxy?server=example.com&port=443&secret=aa" + secret,
		"tg://proxy?server=example.com&port=443&secret=ee" + secret,
		"tg://proxy?server=example.com&port=443&secret=ee" + secret + hex.EncodeToString([]byte("bad/path")),
		"tg://proxy?server=example.com&port=443&secret=" + secret + "00",
		"tg://proxy?server=example.com&port=443&secret=" + secret + "#fragment",
		"tg://socks?server=example.com&port=0",
		"tg://socks?server=example.com&port=65536",
		"tg://socks?server=example.com&port=+1080",
		"tg://socks?server=example.com&port=1080&port=443",
		"tg://socks?server=bad..example&port=1080",
		"tg://socks?server=999.999.999.999&port=1080",
		"tg://socks?server=example.com%2Fpath&port=1080",
		"tg://socks?server=example.com&port=1080&pass=%00",
		"https://alice@t.me/socks?server=example.com&port=1080",
		"socks5://example.com",
		"socks5://example.com:1080/path",
		"socks5://example.com:1080?query=value",
		"socks5://example.com:1080?",
		"socks5://example.com:1080#",
		"socks5://[invalid]:1080",
		"socks5://2001:db8::1:1080",
		"socks5://example.com:port",
		"socks5://alice:p%00ss@example.com:1080",
		"socks5://" + strings.Repeat("a", 256) + ":pass@example.com:1080",
		"http://example.com",
		"https://example.com:443/",
		"http://example.com:8080/admin",
		"https://example.com:443?redirect=https://t.me/proxy",
		"https://example.com:65536",
		"http://-bad.example:8080",
		"https://example.com:443#fragment",
		"http://example.com:8080\nmalformed",
		"tg://socks?server=example.com&port=1080&extra=" + strings.Repeat("a", 1024),
	}
	u := setupProxyUsecase(t)
	for _, proxyURL := range urls {
		t.Run(proxyURL, func(t *testing.T) {
			if _, err := u.Add("Proxy", proxyURL); err == nil {
				t.Fatalf("accepted invalid URL %q", proxyURL)
			}
		})
	}
	if count, err := u.Count(); err != nil || count != 0 {
		t.Fatalf("invalid records were written: count=%d, %v", count, err)
	}
}

func TestProxyUsecaseNamesAndUpdate(t *testing.T) {
	u := setupProxyUsecase(t)
	proxyURL := "socks5://example.com:1080"
	for _, name := range []string{"", "   ", strings.Repeat("я", 81), "name\nsecond line", "name\x00", string([]byte{0xff})} {
		if _, err := u.Add(name, proxyURL); err == nil {
			t.Fatalf("accepted invalid name %q", name)
		}
	}
	proxy, err := u.Add(strings.Repeat("я", 80), proxyURL)
	if err != nil {
		t.Fatal(err)
	}
	if err := u.Update(proxy.ID, " New name ", " https://example.com:443 "); err != nil {
		t.Fatal(err)
	}
	if err := u.Update(proxy.ID, "Oops", "https://bad.example/arbitrary"); err == nil {
		t.Fatal("invalid update accepted")
	}
	got, err := u.Get(proxy.ID)
	if err != nil || got.Name != "New name" || got.URL != "https://example.com:443" {
		t.Fatalf("invalid update altered stored proxy: %+v, %v", got, err)
	}
	if err := u.Update(proxy.ID+1, "Missing", proxyURL); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing update should return sql.ErrNoRows, got %v", err)
	}
	if err := u.Delete(0); err == nil {
		t.Fatal("non-positive ID accepted")
	}
	if err := u.Delete(proxy.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := u.Get(proxy.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing get should return sql.ErrNoRows, got %v", err)
	}
	if err := u.Delete(proxy.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing delete should return sql.ErrNoRows, got %v", err)
	}
}
