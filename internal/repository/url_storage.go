package repository

//go:generate mockgen -source=url_storage.go -destination=mocks/mock_url_storage.go -package=mocks

type URLStorage interface {
	SetShortURL(shortURL string, fullURL string)
	GetFullURL(shortURL string) (string, error)
}
