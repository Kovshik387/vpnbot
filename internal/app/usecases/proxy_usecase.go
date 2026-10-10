package usecases

import (
	"VpnBot/internal/domain/model"
	"VpnBot/internal/domain/repository"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type ProxyUsecase struct {
	repo *repository.ProxyRepository
}

func NewProxyUsecase(repo *repository.ProxyRepository) *ProxyUsecase {
	return &ProxyUsecase{repo: repo}
}

func (u *ProxyUsecase) Add(name, proxyURL string) (model.Proxy, error) {
	name, proxyURL, err := normalizeProxy(name, proxyURL)
	if err != nil {
		return model.Proxy{}, err
	}
	return u.repo.Create(name, proxyURL)
}

func (u *ProxyUsecase) Update(id int64, name, proxyURL string) error {
	if id <= 0 {
		return fmt.Errorf("ID прокси должен быть положительным числом")
	}
	name, proxyURL, err := normalizeProxy(name, proxyURL)
	if err != nil {
		return err
	}
	return u.repo.Update(id, name, proxyURL)
}

func (u *ProxyUsecase) Delete(id int64) error {
	if id <= 0 {
		return fmt.Errorf("ID прокси должен быть положительным числом")
	}
	return u.repo.Delete(id)
}

func (u *ProxyUsecase) Get(id int64) (model.Proxy, error) {
	if id <= 0 {
		return model.Proxy{}, fmt.Errorf("ID прокси должен быть положительным числом")
	}
	return u.repo.GetByID(id)
}

func (u *ProxyUsecase) List(offset, limit int) ([]model.Proxy, error) {
	return u.repo.ListPage(offset, limit)
}

func (u *ProxyUsecase) Count() (int64, error) {
	return u.repo.Count()
}

func normalizeProxy(name, proxyURL string) (string, string, error) {
	name = strings.TrimSpace(name)
	if !utf8.ValidString(name) || utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 80 {
		return "", "", fmt.Errorf("название прокси должно содержать от 1 до 80 символов")
	}
	if strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return "", "", fmt.Errorf("название прокси не должно содержать управляющие символы")
	}
	proxyURL = strings.TrimSpace(proxyURL)
	if len(proxyURL) == 0 || len(proxyURL) > 1024 || !utf8.ValidString(proxyURL) {
		return "", "", fmt.Errorf("ссылка прокси должна содержать от 1 до 1024 байт")
	}
	if strings.IndexFunc(proxyURL, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) >= 0 || strings.Contains(proxyURL, "#") {
		return "", "", fmt.Errorf("ссылка прокси не должна содержать пробелы, управляющие символы или фрагмент #")
	}
	parsed, err := url.Parse(proxyURL)
	if err != nil || parsed.Opaque != "" {
		return "", "", fmt.Errorf("неверный формат ссылки прокси")
	}
	switch strings.ToLower(parsed.Scheme) {
	case "tg":
		if parsed.User != nil || parsed.Port() != "" || parsed.Path != "" || parsed.RawPath != "" {
			return "", "", fmt.Errorf("ожидается tg://proxy или tg://socks с параметрами подключения")
		}
		if err := validateTelegramProxy(parsed, strings.ToLower(parsed.Host)); err != nil {
			return "", "", err
		}
	case "https":
		if strings.EqualFold(parsed.Hostname(), "t.me") && (parsed.Path == "/proxy" || parsed.Path == "/socks") {
			if parsed.User != nil || parsed.Port() != "" || parsed.RawPath != "" {
				return "", "", fmt.Errorf("неверный формат Telegram-ссылки прокси")
			}
			if err := validateTelegramProxy(parsed, strings.TrimPrefix(parsed.Path, "/")); err != nil {
				return "", "", err
			}
			break
		}
		fallthrough
	case "http", "socks5", "socks5h":
		if err := validateProxyEndpoint(parsed); err != nil {
			return "", "", err
		}
	default:
		return "", "", fmt.Errorf("поддерживаются Telegram MTProto/SOCKS5, socks5://, socks5h://, http:// и https:// с адресом и портом")
	}
	return name, proxyURL, nil
}

func validateTelegramProxy(parsed *url.URL, kind string) error {
	if kind != "proxy" && kind != "socks" {
		return fmt.Errorf("поддерживаются только Telegram-ссылки tg://proxy и tg://socks")
	}
	query, err := url.ParseQuery(parsed.RawQuery)
	if err != nil {
		return fmt.Errorf("неверные параметры Telegram-ссылки")
	}
	for key, values := range query {
		if len(values) != 1 {
			return fmt.Errorf("параметр %s не должен повторяться", key)
		}
		if strings.IndexFunc(values[0], unicode.IsControl) >= 0 {
			return fmt.Errorf("параметры ссылки не должны содержать управляющие символы")
		}
	}
	if !validProxyHost(query.Get("server")) {
		return fmt.Errorf("server должен содержать корректный IP-адрес или имя сервера")
	}
	if !validProxyPort(query.Get("port")) {
		return fmt.Errorf("port должен быть числом от 1 до 65535")
	}
	if kind == "proxy" && !validMTProtoSecret(query.Get("secret")) {
		return fmt.Errorf("secret должен быть корректным MTProto-секретом в hex или base64")
	}
	if kind == "socks" {
		if len(query.Get("user")) > 255 || len(query.Get("pass")) > 255 {
			return fmt.Errorf("логин и пароль SOCKS5 не должны превышать 255 байт")
		}
	}
	return nil
}

func validateProxyEndpoint(parsed *url.URL) error {
	if parsed.Path != "" || parsed.RawPath != "" || parsed.RawQuery != "" || parsed.ForceQuery {
		return fmt.Errorf("адрес прокси должен иметь формат схема://[логин:пароль@]сервер:порт без пути и параметров")
	}
	if !validProxyHost(parsed.Hostname()) || !validProxyPort(parsed.Port()) {
		return fmt.Errorf("укажите корректный сервер и явный порт от 1 до 65535")
	}
	// A colon in the host must belong to a bracketed IPv6 literal.
	if strings.Contains(parsed.Hostname(), ":") && !strings.HasPrefix(parsed.Host, "[") {
		return fmt.Errorf("IPv6-адрес в ссылке должен быть заключён в квадратные скобки")
	}
	if parsed.User != nil {
		username := parsed.User.Username()
		password, _ := parsed.User.Password()
		if strings.IndexFunc(username+password, unicode.IsControl) >= 0 {
			return fmt.Errorf("логин и пароль не должны содержать управляющие символы")
		}
		if (strings.EqualFold(parsed.Scheme, "socks5") || strings.EqualFold(parsed.Scheme, "socks5h")) && (len(username) > 255 || len(password) > 255) {
			return fmt.Errorf("логин и пароль SOCKS5 не должны превышать 255 байт")
		}
	}
	return nil
}

func validProxyPort(port string) bool {
	if port == "" || strings.IndexFunc(port, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
		return false
	}
	n, err := strconv.Atoi(port)
	return err == nil && n >= 1 && n <= 65535
}

func validProxyHost(host string) bool {
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		return net.ParseIP(host[1:len(host)-1]) != nil
	}
	if net.ParseIP(host) != nil {
		return true
	}
	if len(host) == 0 || len(host) > 253 || strings.ContainsAny(host, ":/@?#%[]\\") {
		return false
	}
	host = strings.TrimSuffix(host, ".")
	onlyDigitsAndDots := true
	for _, label := range strings.Split(host, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, r := range label {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' {
				return false
			}
			if r < '0' || r > '9' {
				onlyDigitsAndDots = false
			}
		}
	}
	return !onlyDigitsAndDots
}

func validMTProtoSecret(secret string) bool {
	if decoded, err := hex.DecodeString(secret); err == nil && len(decoded) >= 16 {
		return validMTProtoSecretBytes(decoded)
	}
	for _, encoding := range []*base64.Encoding{base64.RawURLEncoding, base64.URLEncoding, base64.RawStdEncoding, base64.StdEncoding} {
		if decoded, err := encoding.DecodeString(secret); err == nil && validMTProtoSecretBytes(decoded) {
			return true
		}
	}
	return false
}

func validMTProtoSecretBytes(secret []byte) bool {
	switch {
	case len(secret) == 16:
		return true
	case len(secret) == 17 && secret[0] == 0xdd:
		return true
	case len(secret) >= 21 && secret[0] == 0xee:
		return validProxyHost(string(secret[17:]))
	default:
		return false
	}
}
