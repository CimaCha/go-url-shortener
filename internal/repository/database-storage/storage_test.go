package postgres

import (
	"context"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/CimaCha/go-url-shortener/internal/model"
	"github.com/CimaCha/go-url-shortener/internal/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestStorage(t *testing.T) {
	tests := []struct {
		name string
		run  func(*testing.T, context.Context, *Storage)
	}{
		{
			name: "creates schema",
			run: func(t *testing.T, ctx context.Context, storage *Storage) {
				var exists bool
				require.NoError(t, storage.Pool.QueryRow(ctx, "SELECT to_regclass('urls') IS NOT NULL").Scan(&exists))
				require.True(t, exists)
			},
		},
		{
			name: "saves and finds URL",
			run: func(t *testing.T, ctx context.Context, storage *Storage) {
				storedShortURL, err := storage.SaveShortURL(ctx, "short", "https://example.com", "")
				require.NoError(t, err)
				require.Empty(t, storedShortURL)
				fullURL, err := storage.FindFullURL(ctx, "short")
				require.NoError(t, err)
				require.Equal(t, "https://example.com", fullURL)
			},
		},
		{
			name: "maps not found error",
			run: func(t *testing.T, ctx context.Context, storage *Storage) {
				_, err := storage.FindFullURL(ctx, "missing")
				require.ErrorIs(t, err, repository.ErrURLNotFound)
			},
		},
		{
			name: "maps duplicate error",
			run: func(t *testing.T, ctx context.Context, storage *Storage) {
				_, err := storage.SaveShortURL(ctx, "short", "https://example.com", "")
				require.NoError(t, err)
				_, err = storage.SaveShortURL(ctx, "short", "https://other.example.com", "")
				require.ErrorIs(t, err, repository.ErrShortURLExists)
			},
		},
		{
			name: "returns existing short URL for duplicate full URL",
			run: func(t *testing.T, ctx context.Context, storage *Storage) {
				_, err := storage.SaveShortURL(ctx, "first", "https://example.com", "")
				require.NoError(t, err)
				storedShortURL, err := storage.SaveShortURL(ctx, "second", "https://example.com", "")
				require.ErrorIs(t, err, repository.ErrFullURLExists)
				require.Equal(t, "first", storedShortURL)
			},
		},
		{
			name: "rolls back batch and maps duplicate short URL",
			run: func(t *testing.T, ctx context.Context, storage *Storage) {
				_, err := storage.SaveShortURL(ctx, "existing", "https://example.com/existing", "")
				require.NoError(t, err)
				err = storage.SaveShortURLBatch(ctx, []*model.URLRecord{
					{ShortURL: "new", OriginalURL: "https://example.com/new"},
					{ShortURL: "existing", OriginalURL: "https://example.com/collision"},
				}, "")
				require.ErrorIs(t, err, repository.ErrShortURLExists)
				_, err = storage.FindFullURL(ctx, "new")
				require.ErrorIs(t, err, repository.ErrURLNotFound)
			},
		},
		{
			name: "rolls back batch and maps duplicate full URL",
			run: func(t *testing.T, ctx context.Context, storage *Storage) {
				_, err := storage.SaveShortURL(ctx, "existing", "https://example.com/existing", "")
				require.NoError(t, err)
				err = storage.SaveShortURLBatch(ctx, []*model.URLRecord{
					{ShortURL: "new", OriginalURL: "https://example.com/new"},
					{ShortURL: "other", OriginalURL: "https://example.com/existing"},
				}, "")
				require.ErrorIs(t, err, repository.ErrFullURLExists)
				_, err = storage.FindFullURL(ctx, "new")
				require.ErrorIs(t, err, repository.ErrURLNotFound)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			storage, err := NewDatabaseStorage(ctx, newTestDSN(t))
			require.NoError(t, err)
			t.Cleanup(storage.Close)

			tt.run(t, ctx, storage)
		})
	}
}

func newTestDSN(t *testing.T) string {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN is not set")
	}

	ctx := context.Background()
	adminPool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(adminPool.Close)

	schema := "storage_test_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	_, err = adminPool.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize())
	require.NoError(t, err)
	t.Cleanup(func() {
		_, dropErr := adminPool.Exec(context.Background(), "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
		require.NoError(t, dropErr)
	})

	databaseURL, err := url.Parse(dsn)
	require.NoError(t, err)
	query := databaseURL.Query()
	query.Set("search_path", schema)
	databaseURL.RawQuery = query.Encode()
	return databaseURL.String()
}
