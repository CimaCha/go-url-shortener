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
	"github.com/jackc/pgx/v5/pgconn"
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
			name: "saves multiple URLs and a batch for one user",
			run: func(t *testing.T, ctx context.Context, storage *Storage) {
				for _, id := range []string{"first", "second"} {
					_, err := storage.SaveShortURL(ctx, id, "https://example.com/"+id, "user")
					require.NoError(t, err)
				}
				require.NoError(t, storage.SaveShortURLBatch(ctx, []*model.URLRecord{
					{ShortURL: "third", OriginalURL: "https://example.com/third"},
					{ShortURL: "fourth", OriginalURL: "https://example.com/fourth"},
				}, "user"))
				urls, err := storage.GetUserURLs(ctx, "user")
				require.NoError(t, err)
				require.Len(t, urls, 4)
			},
		},
		{
			name: "preserves other unique constraint errors",
			run: func(t *testing.T, ctx context.Context, storage *Storage) {
				_, err := storage.Pool.Exec(ctx, "CREATE UNIQUE INDEX test_user_unique ON urls(user_id)")
				require.NoError(t, err)
				_, err = storage.SaveShortURL(ctx, "first", "https://example.com/first", "user")
				require.NoError(t, err)
				_, err = storage.SaveShortURL(ctx, "second", "https://example.com/second", "user")
				var pgErr *pgconn.PgError
				require.ErrorAs(t, err, &pgErr)
				require.Equal(t, "test_user_unique", pgErr.ConstraintName)
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
		{
			name: "deletes only URLs owned by user",
			run: func(t *testing.T, ctx context.Context, storage *Storage) {
				_, err := storage.SaveShortURL(ctx, "own", "https://example.com/own", "user-id")
				require.NoError(t, err)
				_, err = storage.SaveShortURL(ctx, "foreign", "https://example.com/foreign", "other-user")
				require.NoError(t, err)

				require.NoError(t, storage.DeleteURLsBatch(ctx, []string{"own", "foreign", "missing"}, "user-id"))
				data, err := storage.GetShortURLData(ctx, "own")
				require.NoError(t, err)
				require.Equal(t, &model.StorageRecord{UserID: "user-id", OriginalURL: "https://example.com/own", DeletedFlag: true}, data)
				_, err = storage.FindFullURL(ctx, "own")
				require.ErrorIs(t, err, repository.ErrURLHasGone)
				fullURL, err := storage.FindFullURL(ctx, "foreign")
				require.NoError(t, err)
				require.Equal(t, "https://example.com/foreign", fullURL)
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

func TestSaveShortURLPreservesCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	pool, err := pgxpool.New(ctx, "postgres://test:test@127.0.0.1:1/test?sslmode=disable")
	require.NoError(t, err)
	defer pool.Close()
	stored, err := (Storage{Pool: pool}).SaveShortURL(ctx, "short", "https://example.com", "user")
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, stored)
}
