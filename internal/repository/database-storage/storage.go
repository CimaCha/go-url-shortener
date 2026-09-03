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

func (s Storage) SaveShortURL(ctx context.Context, shortURL, fullURL, userId string) (string, error) {
	var storedFullURL string
	err := s.Pool.QueryRow(ctx, "INSERT INTO urls(short_url, full_url, user_id) VALUES ($1,$2, $3) ON CONFLICT (full_url) DO UPDATE SET full_url=urls.full_url RETURNING urls.short_url;", shortURL, fullURL, userId).Scan(&storedFullURL)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return "", repository.ErrShortURLExists
	}
	if storedFullURL != shortURL {
		return storedFullURL, repository.ErrFullURLExists
	}

	return "", err
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

func (s Storage) SaveShortUrlBatch(ctx context.Context, URLRecords []*model.URLRecord, userId string) error {
	// начинаем транзакцию
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	var rows [][]interface{}
	for _, urls := range URLRecords {
		rows = append(rows, []interface{}{urls.ShortURL, urls.OriginalURL, userId})
	}

	_, err = tx.CopyFrom(
		ctx,
		pgx.Identifier{"urls"},
		[]string{"short_url", "full_url", "user_id"},
		pgx.CopyFromRows(rows),
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if pgErr.ConstraintName == "urls_pkey" {
				return repository.ErrShortURLExists
			}
			return repository.ErrFullURLExists
		}
		return err
	}

	// завершаем транзакцию
	return tx.Commit(ctx)
}

func (s Storage) GetUserURLs(ctx context.Context, userId string) ([]*model.UserRecord, error) {
	var userRecords []*model.UserRecord
	rows, err := s.Pool.Query(ctx, "SELECT full_url, short_url FROM urls WHERE user_id = $1", userId)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, repository.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var userRecord model.UserRecord
		err = rows.Scan(&userRecord.OriginalURL, &userRecord.ShortURL)
		if err != nil {
			return nil, err
		}

		userRecords = append(userRecords, &userRecord)
	}
	return userRecords, nil
}
