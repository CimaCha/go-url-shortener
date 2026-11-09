package repository

import (
	"context"
	"strconv"
	"testing"

	"github.com/CimaCha/go-url-shortener/internal/model"
)

func BenchmarkMemoryStorageFindFullURL(b *testing.B) {
	storage := NewMemoryURLStorage(map[string]URLData{
		"short": {UserID: "user", OriginalURL: "https://example.com/item"},
	})
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		if _, err := storage.FindFullURL(ctx, "short"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMemoryStorageSaveShortURLBatch1000(b *testing.B) {
	const size = 1000
	existing := make(map[string]URLData, size)
	batch := make([]*model.URLRecord, size)
	for i := range batch {
		id := strconv.Itoa(i)
		existing["old-"+id] = URLData{UserID: "user", OriginalURL: "https://example.com/old/" + id}
		batch[i] = &model.URLRecord{ShortURL: "new-" + id, OriginalURL: "https://example.com/new/" + id}
	}
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		storage := NewMemoryURLStorage(existing)
		b.StartTimer()
		if err := storage.SaveShortURLBatch(ctx, batch, "user"); err != nil {
			b.Fatal(err)
		}
	}
}
