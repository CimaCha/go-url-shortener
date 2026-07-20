package repository

type UrlStorage interface {
	SetShortUrl(shortURL string, fullURL string) error
	GetFullUrl(shortURL string) (string, error)
}
