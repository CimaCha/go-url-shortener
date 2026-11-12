package service

import (
	"context"
	"strconv"
	"testing"

	"github.com/CimaCha/go-url-shortener/internal/model"
)

type benchmarkStorage struct{ URLStorage }

func (benchmarkStorage) SaveShortURL(context.Context, string, string, string) (string, error) {
	return "", nil
}

func (benchmarkStorage) SaveShortURLBatch(context.Context, []*model.URLRecord, string) error {
	return nil
}

func (benchmarkStorage) GetShortURLData(context.Context, string) (*model.StorageRecord, error) {
	return &model.StorageRecord{UserID: "user"}, nil
}

func (benchmarkStorage) DeleteURLsBatch(context.Context, []string, string) error {
	return nil
}

func BenchmarkServiceShorten(b *testing.B) {
	service := NewService(benchmarkStorage{}, 5, 1)
	b.Cleanup(service.Close)
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		if _, err := service.Shorten(ctx, "https://example.com/item", "user"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkServiceShortenBatch1000(b *testing.B) {
	batch := make([]*model.OriginalURLRecord, 1000)
	for i := range batch {
		batch[i] = &model.OriginalURLRecord{
			CorrelationID: strconv.Itoa(i),
			OriginalURL:   "https://example.com/item/" + strconv.Itoa(i),
		}
	}
	service := NewService(benchmarkStorage{}, 5, 1)
	b.Cleanup(service.Close)
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		if _, err := service.ShortenBatch(ctx, batch, "user"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkServiceDeleteBatch1000(b *testing.B) {
	urls := make([]string, 1000)
	for i := range urls {
		urls[i] = strconv.Itoa(i)
	}
	service := NewService(benchmarkStorage{}, 5, 10)
	b.Cleanup(service.Close)
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		if err := service.DeleteBatch(ctx, urls, "user"); err != nil {
			b.Fatal(err)
		}
	}
}
