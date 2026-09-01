package migrations

import (
	"context"
	"os"
	"strconv"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3/lock"
	"github.com/stretchr/testify/require"
)

func TestUpCreatesSchemaAndIsIdempotent(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	require.NoError(t, Up(ctx, pool))

	var urlsTableExists, versionTableExists bool
	require.NoError(t, pool.QueryRow(ctx, "SELECT to_regclass('urls') IS NOT NULL").Scan(&urlsTableExists))
	require.NoError(t, pool.QueryRow(ctx, "SELECT to_regclass('goose_db_version') IS NOT NULL").Scan(&versionTableExists))
	require.True(t, urlsTableExists)
	require.True(t, versionTableExists)

	_, err := pool.Exec(ctx, "INSERT INTO urls(short_url, full_url) VALUES($1, $2)", "short", "https://example.com")
	require.NoError(t, err)
	require.NoError(t, Up(ctx, pool))

	var fullURL string
	require.NoError(t, pool.QueryRow(ctx, "SELECT full_url FROM urls WHERE short_url = $1", "short").Scan(&fullURL))
	require.Equal(t, "https://example.com", fullURL)
}

func TestUpUsesMigrationLock(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	conn, err := pool.Acquire(ctx)
	require.NoError(t, err)
	t.Cleanup(conn.Release)

	_, err = conn.Exec(ctx, "SELECT pg_advisory_lock($1)", lock.DefaultLockID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, unlockErr := conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", lock.DefaultLockID)
		require.NoError(t, unlockErr)
	})

	lockedCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	err = up(lockedCtx, pool, fstest.MapFS{
		"000001_test.sql": {Data: []byte("-- +goose Up\nSELECT 1;\n")},
	})
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN is not set")
	}

	ctx := context.Background()
	adminPool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(adminPool.Close)

	schema := "migration_test_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	_, err = adminPool.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize())
	require.NoError(t, err)
	t.Cleanup(func() {
		_, dropErr := adminPool.Exec(context.Background(), "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
		require.NoError(t, dropErr)
	})

	config, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)
	config.ConnConfig.RuntimeParams["search_path"] = schema

	pool, err := pgxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)
	require.NoError(t, pool.Ping(ctx))
	t.Cleanup(pool.Close)

	return pool
}
