package postgres

import (
	"context"
	"strconv"
	"testing"

	"github.com/CimaCha/go-url-shortener/internal/model"
)

func BenchmarkDatabaseStorageFindFullURL(b *testing.B) {
	ctx := context.Background()
	storage, err := NewDatabaseStorage(ctx, newTestDSN(b))
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(storage.Close)
	if _, err := storage.SaveShortURL(ctx, "short", "https://example.com/item", "user"); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := storage.FindFullURL(ctx, "short"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDatabaseStorageSaveShortURLBatch1000(b *testing.B) {
	ctx := context.Background()
	storage, err := NewDatabaseStorage(ctx, newTestDSN(b))
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(storage.Close)
	batch := make([]*model.URLRecord, 1000)
	for i := range batch {
		id := strconv.Itoa(i)
		batch[i] = &model.URLRecord{ShortURL: "short-" + id, OriginalURL: "https://example.com/item/" + id}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		if _, err := storage.Pool.Exec(ctx, "TRUNCATE TABLE urls"); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		if err := storage.SaveShortURLBatch(ctx, batch, "user"); err != nil {
			b.Fatal(err)
		}
	}
}
