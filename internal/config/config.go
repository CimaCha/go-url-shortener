package config

import (
	"flag"
	"github.com/caarlos0/env/v11"
)

type Config struct {
	Address             string `env:"SERVER_ADDRESS"`
	BasicShortenAddress string `env:"BASE_URL"`
	FilePath            string `env:"FILE_STORAGE_PATH"`
	DatabaseURL         string `env:"DATABASE_DSN"`
}

func New() (*Config, error) {
	address := flag.String("a", "localhost:8080", "address of service")
	baseURL := flag.String("b", "http://localhost:8080", "basic address for short url")
	filePath := flag.String("f", "", "path to the storage file")
	databaseURL := flag.String("d", "", "database url")
	flag.Parse()

	config := Config{
		Address:             *address,
		BasicShortenAddress: *baseURL,
		FilePath:            *filePath,
		DatabaseURL:         *databaseURL,
	}
	if err := env.Parse(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
