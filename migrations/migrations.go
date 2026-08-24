package migrations

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

//go:embed *.sql
var files embed.FS

func Up(ctx context.Context, pool *pgxpool.Pool) (err error) {
	return up(ctx, pool, files)
}

func up(ctx context.Context, pool *pgxpool.Pool, migrationFiles fs.FS) (err error) {
	db := stdlib.OpenDBFromPool(pool)

	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("create migration lock: %w", err)
	}

	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		db,
		migrationFiles,
		goose.WithSessionLocker(locker),
	)
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("create migration provider: %w", err)
	}
	defer func() {
		err = errors.Join(err, provider.Close())
	}()

	if _, err = provider.Up(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
