package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/CimaCha/go-url-shortener/internal/model"
	"github.com/CimaCha/go-url-shortener/internal/repository"
	"github.com/CimaCha/go-url-shortener/migrations"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrPingDataBase  = errors.New("failed to ping database")
	ErrCreateNewPool = errors.New("failed to create new pool")
)

type Storage struct {
	Pool *pgxpool.Pool
}

func NewDatabaseStorage(ctx context.Context, databaseURL string) (*Storage, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreateNewPool, err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err = pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("%w: %w", ErrPingDataBase, err)
	}
	if err = migrations.Up(ctx, pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("migrate database: %w", err)
	}
	return &Storage{Pool: pool}, nil
}

func (s Storage) SaveShortURL(ctx context.Context, shortURL, fullURL string) error {
	_, err := s.Pool.Exec(ctx, "INSERT INTO urls(short_url, full_url) VALUES($1,$2)", shortURL, fullURL)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return repository.ErrFullURLExists
	}

	return err
}

func (s Storage) FindFullURL(ctx context.Context, shortURL string) (string, error) {
	var fullURL string
	err := s.Pool.QueryRow(ctx, "SELECT full_url FROM urls WHERE short_url = $1", shortURL).Scan(&fullURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", repository.ErrURLNotFound
	}
	if err != nil {
		return "", err
	}
	return fullURL, nil
}

func (s Storage) Close() {
	s.Pool.Close()
}

func (s Storage) Ping(ctx context.Context) error {
	return s.Pool.Ping(ctx)
}

func (s Storage) SaveShortUrlBatch(ctx context.Context, URLRecords []*model.URLRecord) error {
	// начинаем транзакцию
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	var rows [][]interface{}
	for _, urls := range URLRecords {
		rows = append(rows, []interface{}{urls.ShortURL, urls.OriginalURL})
	}

	_, err = tx.CopyFrom(
		ctx,
		pgx.Identifier{"urls"},
		[]string{"short_url", "full_url"},
		pgx.CopyFromRows(rows),
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return repository.ErrShortURLExists
		}
		return err
	}

	// завершаем транзакцию
	return tx.Commit(ctx)
}

func (s Storage) FindShortURL(ctx context.Context, fullURL string) (string, error) {
	var shortURL string
	err := s.Pool.QueryRow(ctx, "SELECT short_url FROM urls WHERE full_url = $1", fullURL).Scan(&shortURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", repository.ErrURLNotFound
	}
	if err != nil {
		return "", err
	}
	return shortURL, nil
}
