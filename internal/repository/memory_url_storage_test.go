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
			storage := NewMemoryURLStorage(make(map[string]URLData))
			var storedShortURL string
			var setErr error
			for _, write := range tt.writes {
				storedShortURL, setErr = storage.SaveShortURL(ctx, write[0], write[1], "")
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
			storage := NewMemoryURLStorage(make(map[string]URLData))
			_, err := storage.SaveShortURL(ctx, "short", "https://example.com", "")
			assert.NoError(t, err)

			snapshot := storage.Snapshot()
			snapshot["short"] = URLData{OriginalURL: "changed"}

			got, err := storage.FindFullURL(ctx, "short")
			assert.NoError(t, err)
			assert.Equal(t, "https://example.com", got)
		})
	}
}

func TestMemoryURLStorageBatchCollisionIsAtomic(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryURLStorage(map[string]URLData{
		"existing": {OriginalURL: "https://example.com/existing"},
	})

	err := storage.SaveShortURLBatch(ctx, []*model.URLRecord{
		{ShortURL: "new", OriginalURL: "https://example.com/new"},
		{ShortURL: "existing", OriginalURL: "https://example.com/collision"},
	}, "")

	assert.ErrorIs(t, err, ErrShortURLExists)
	assert.Equal(t, map[string]URLData{
		"existing": {OriginalURL: "https://example.com/existing"},
	}, storage.Snapshot())
	userURLs, getErr := storage.GetUserURLs(ctx, "")
	assert.NoError(t, getErr)
	assert.Equal(t, []*model.UserRecord{{ShortURL: "existing", OriginalURL: "https://example.com/existing"}}, userURLs)
}

func TestMemoryURLStorageDeletesOnlyOwnedURLs(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryURLStorage(map[string]URLData{
		"own":     {UserID: "user-id", OriginalURL: "https://example.com/own"},
		"foreign": {UserID: "other-user", OriginalURL: "https://example.com/foreign"},
	})

	err := storage.DeleteURLsBatch(ctx, []string{"own", "foreign", "missing"}, "user-id")

	assert.NoError(t, err)
	_, err = storage.FindFullURL(ctx, "own")
	assert.ErrorIs(t, err, ErrURLHasGone)
	got, err := storage.FindFullURL(ctx, "foreign")
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com/foreign", got)
	assert.NotContains(t, storage.Snapshot(), "missing")
}

func TestMemoryURLStorageGetShortURLData(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryURLStorage(map[string]URLData{
		"active":  {UserID: "user-id", OriginalURL: "https://example.com/active"},
		"deleted": {UserID: "user-id", OriginalURL: "https://example.com/deleted", DeletedFlag: true},
	})

	active, err := storage.GetShortURLData(ctx, "active")
	assert.NoError(t, err)
	assert.Equal(t, &model.StorageRecord{UserID: "user-id", OriginalURL: "https://example.com/active"}, active)
	deleted, err := storage.GetShortURLData(ctx, "deleted")
	assert.NoError(t, err)
	assert.Equal(t, &model.StorageRecord{UserID: "user-id", OriginalURL: "https://example.com/deleted", DeletedFlag: true}, deleted)
	missing, err := storage.GetShortURLData(ctx, "missing")
	assert.ErrorIs(t, err, ErrURLNotFound)
	assert.Nil(t, missing)
}

func TestMemoryURLStorageBatchRejectsInternalDuplicate(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryURLStorage(make(map[string]URLData))

	err := storage.SaveShortURLBatch(ctx, []*model.URLRecord{
		{ShortURL: "duplicate", OriginalURL: "https://example.com/first"},
		{ShortURL: "duplicate", OriginalURL: "https://example.com/second"},
	}, "")

	assert.ErrorIs(t, err, ErrShortURLExists)
	assert.Empty(t, storage.Snapshot())
}
