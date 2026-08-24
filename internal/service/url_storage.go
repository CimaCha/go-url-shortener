package service

//go:generate mockgen -source=url_storage.go -destination=mocks/mock_url_storage.gen.go -package=mocks

type URLStorage interface {
	SaveShortURL(shortURL string, fullURL string) error
	FindFullURL(shortURL string) (string, error)
}
