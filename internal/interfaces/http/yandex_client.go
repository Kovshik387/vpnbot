package interfaces

import (
	"VpnBot/internal/domain/ports/service"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type yandexClient struct {
	client  *http.Client
	baseURL string
}

type yandexResponse struct {
	File string `json:"file"`
}

func NewYandexClient() service.YandexService {
	return &yandexClient{client: &http.Client{Timeout: 10 * time.Second}, baseURL: "https://cloud-api.yandex.net/v1/disk/public/resources"}
}

func (y yandexClient) GetYandexDirectLink(publicLink string) (string, error) {
	query := url.Values{}
	query.Set("public_key", publicLink)
	apiURL := y.baseURL + "?" + query.Encode()

	resp, err := y.client.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("ошибка запроса к API Яндекса: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("не удалось получить ссылку, код ответа: %d", resp.StatusCode)
	}

	var data yandexResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	if data.File == "" {
		return "", fmt.Errorf("прямая ссылка не найдена")
	}

	return data.File, nil
}
