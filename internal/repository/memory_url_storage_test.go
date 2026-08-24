package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemoryURLStorage(t *testing.T) {
	tests := []struct {
		name       string
		writes     [][2]string
		shortURL   string
		want       string
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			storage := NewMemoryURLStorage(make(map[string]string))
			var setErr error
			for _, write := range tt.writes {
				setErr = storage.SaveShortURL(ctx, write[0], write[1])
			}

			assert.ErrorIs(t, setErr, tt.wantSetErr)

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
			assert.NoError(t, storage.SaveShortURL(ctx, "short", "https://example.com"))

			snapshot := storage.Snapshot()
			snapshot["short"] = "changed"

			got, err := storage.FindFullURL(ctx, "short")
			assert.NoError(t, err)
			assert.Equal(t, "https://example.com", got)
		})
	}
}
