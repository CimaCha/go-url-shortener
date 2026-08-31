package repository

import (
	"context"
	"testing"

	"github.com/CimaCha/go-url-shortener/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestMemoryURLStorage(t *testing.T) {
	tests := []struct {
		name       string
		writes     [][2]string
		shortURL   string
		want       string
		wantStored string
		wantSetErr error
		wantGetErr error
	}{
		{
			name:     "stores URL",
			writes:   [][2]string{{"short", "https://example.com"}},
			shortURL: "short",
			want:     "https://example.com",
		},
		{
			name:       "missing URL",
			shortURL:   "missing",
			wantGetErr: ErrURLNotFound,
		},
		{
			name: "does not overwrite existing short URL",
			writes: [][2]string{
				{"short", "https://first.example.com"},
				{"short", "https://second.example.com"},
			},
			shortURL:   "short",
			want:       "https://first.example.com",
			wantSetErr: ErrShortURLExists,
		},
		{
			name: "returns existing short URL for duplicate full URL",
			writes: [][2]string{
				{"first", "https://example.com"},
				{"second", "https://example.com"},
			},
			shortURL:   "first",
			want:       "https://example.com",
			wantStored: "first",
			wantSetErr: ErrFullURLExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			storage := NewMemoryURLStorage(make(map[string]string))
			var storedShortURL string
			var setErr error
			for _, write := range tt.writes {
				storedShortURL, setErr = storage.SaveShortURL(ctx, write[0], write[1])
			}

			assert.ErrorIs(t, setErr, tt.wantSetErr)
			assert.Equal(t, tt.wantStored, storedShortURL)

			got, err := storage.FindFullURL(ctx, tt.shortURL)
			assert.ErrorIs(t, err, tt.wantGetErr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMemoryURLStorageSnapshot(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "returns independent copy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			storage := NewMemoryURLStorage(make(map[string]string))
			_, err := storage.SaveShortURL(ctx, "short", "https://example.com")
			assert.NoError(t, err)

			snapshot := storage.Snapshot()
			snapshot["short"] = "changed"

			got, err := storage.FindFullURL(ctx, "short")
			assert.NoError(t, err)
			assert.Equal(t, "https://example.com", got)
		})
	}
}

func TestMemoryURLStorageBatchCollisionIsAtomic(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryURLStorage(map[string]string{
		"existing": "https://example.com/existing",
	})

	err := storage.SaveShortUrlBatch(ctx, []*model.URLRecord{
		{ShortURL: "new", OriginalURL: "https://example.com/new"},
		{ShortURL: "existing", OriginalURL: "https://example.com/collision"},
	})

	assert.ErrorIs(t, err, ErrShortURLExists)
	assert.Equal(t, map[string]string{
		"existing": "https://example.com/existing",
	}, storage.Snapshot())
}

func TestMemoryURLStorageBatchRejectsInternalDuplicate(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryURLStorage(make(map[string]string))

	err := storage.SaveShortUrlBatch(ctx, []*model.URLRecord{
		{ShortURL: "duplicate", OriginalURL: "https://example.com/first"},
		{ShortURL: "duplicate", OriginalURL: "https://example.com/second"},
	})

	assert.ErrorIs(t, err, ErrShortURLExists)
	assert.Empty(t, storage.Snapshot())
}
