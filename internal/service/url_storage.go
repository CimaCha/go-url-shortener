package service

import (
	"context"
	"github.com/CimaCha/go-url-shortener/internal/model"
)

//go:generate mockgen -source=url_storage.go -destination=mocks/mock_url_storage.gen.go -package=mocks

type URLStorage interface {
	SaveShortURL(ctx context.Context, shortURL, fullURL string) error
	FindFullURL(ctx context.Context, shortURL string) (string, error)
	SaveShortUrlBatch(ctx context.Context, URLRecords []*model.URLRecord) error

	FindShortURL(ctx context.Context, fullURL string) (string, error)
}
