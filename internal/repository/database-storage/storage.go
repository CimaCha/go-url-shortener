package database_storage

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrParseConfig   = errors.New("failed to parse config")
	ErrCreateNewPool = errors.New("failed to create new pool")
)

type Storage struct {
	Db *pgxpool.Pool
}

func NewDatabaseStorage(ctx context.Context, databaseURL string) (*Storage, error) {
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, ErrParseConfig
	}
	db, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, ErrCreateNewPool
	}
	return &Storage{Db: db}, nil
}
