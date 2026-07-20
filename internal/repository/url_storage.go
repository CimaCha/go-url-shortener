package repository

//go:generate mockgen -source=url_storage.go -destination=mocks/mock_url_storage.go -package=mocks

type UrlStorage interface {
	SetShortUrl(shortURL string, fullURL string) error
	GetFullUrl(shortURL string) (string, error)
}
