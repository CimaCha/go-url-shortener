package file

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/CimaCha/go-url-shortener/internal/model"
)

func BenchmarkFileStorageFindFullURL(b *testing.B) {
	storage, err := NewFileStorage(filepath.Join(b.TempDir(), "storage.json"))
	if err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
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

func BenchmarkFileStorageSaveShortURLBatch1000(b *testing.B) {
	path := filepath.Join(b.TempDir(), "storage.json")
	batch := make([]*model.URLRecord, 1000)
	for i := range batch {
		id := strconv.Itoa(i)
		batch[i] = &model.URLRecord{ShortURL: "short-" + id, OriginalURL: "https://example.com/item/" + id}
	}
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			b.Fatal(err)
		}
		storage, err := NewFileStorage(path)
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		if err := storage.SaveShortURLBatch(ctx, batch, "user"); err != nil {
			b.Fatal(err)
		}
	}
}
