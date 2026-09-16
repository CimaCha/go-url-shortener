package config

import (
	"flag"
	"fmt"
	"github.com/caarlos0/env/v11"
	"math"
	"os"
	"time"
)

type Config struct {
	Address                string `env:"SERVER_ADDRESS"`
	BasicShortenAddress    string `env:"BASE_URL"`
	FilePath               string `env:"FILE_STORAGE_PATH"`
	DatabaseURL            string `env:"DATABASE_DSN"`
	SecretKey              string `env:"SECRET_KEY"`
	MaxShortURLsAttempts   int    `env:"MAX_SHORTEN_ATTEMPTS"`
	DeleteBatchConcurrency int    `env:"DELETE_BATCH_CONCURRENCY"`
	DeleteBatchSize        int    `env:"DELETE_BATCH_SIZE"`
	DeletionTimeout        int    `env:"DELETION_TIMEOUT"`
}

func New() (*Config, error) {
	address := flag.String("a", "localhost:8080", "address of service")
	baseURL := flag.String("b", "http://localhost:8080", "basic address for short url")
	filePath := flag.String("f", "", "path to the storage file")
	databaseURL := flag.String("d", "", "database url")
	secretKey := flag.String("k", "secret_key", "secret key for jwt")
	maxShortURLsAttempts := flag.Int("m", 5, "max attempts to generate unique short url")
	deleteBatchConcurrency := flag.Int("c", 10, "delete batch concurrency")
	deleteBatchSize := flag.Int("db", 100, "delete batch size")
	deletionTimeout := flag.Int("dt", 10, "deletion timeout in seconds")
	if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
		return nil, err
	}

	config := Config{
		Address:                *address,
		BasicShortenAddress:    *baseURL,
		FilePath:               *filePath,
		DatabaseURL:            *databaseURL,
		SecretKey:              *secretKey,
		MaxShortURLsAttempts:   *maxShortURLsAttempts,
		DeleteBatchConcurrency: *deleteBatchConcurrency,
		DeleteBatchSize:        *deleteBatchSize,
		DeletionTimeout:        *deletionTimeout,
	}
	if err := env.Parse(&config); err != nil {
		return nil, err
	}

	for _, setting := range []struct {
		name  string
		value int
	}{
		{"MAX_SHORTEN_ATTEMPTS", config.MaxShortURLsAttempts},
		{"DELETE_BATCH_CONCURRENCY", config.DeleteBatchConcurrency},
		{"DELETE_BATCH_SIZE", config.DeleteBatchSize},
		{"DELETION_TIMEOUT", config.DeletionTimeout},
	} {
		if setting.value <= 0 {
			return nil, fmt.Errorf("%s must be positive", setting.name)
		}
	}
	if int64(config.DeletionTimeout) > math.MaxInt64/int64(time.Second) {
		return nil, fmt.Errorf("DELETION_TIMEOUT exceeds time.Duration range")
	}
	return &config, nil
}
