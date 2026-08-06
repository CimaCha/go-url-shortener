package service

//go:generate mockgen -source=url_storage.go -destination=mocks/mock_url_storage.gen.go -package=mocks

type URLStorage interface {
	SetShortURL(shortURL string, fullURL string) error
	GetFullURL(shortURL string) (string, error)
}
