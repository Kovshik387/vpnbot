package service

type YandexService interface {
	GetYandexDirectLink(publicLink string) (string, error)
}
