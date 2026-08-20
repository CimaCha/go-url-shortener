package config

import (
	"flag"
	"github.com/CimaCha/go-url-shortener/internal/config/db"

	configflag "github.com/CimaCha/go-url-shortener/internal/config/flag"
	"github.com/caarlos0/env/v11"
)

type Config struct {
	Address             string `env:"SERVER_ADDRESS"`
	BasicShortenAddress string `env:"BASE_URL"`
	FilePath            string `env:"FILE_STORAGE_PATH"`
	DatabaseURL         string `env:"DATABASE_DSN"`
}

func New() (*Config, error) {

	netAddress := configflag.NewNetAddressFlag("a", "address of service", "localhost:8080")
	basicShortenAddress := configflag.NewNetAddressFlag("b", "basic address for short url", "http://localhost:8080")
	filePath := configflag.NewFilePathFlag("f", "path to the storage file", "./storage.json")
	databaseURL := db.NewDatabaseURLFlag("d", "database url", "postgres://username:password@localhost:5432/database_name")
	flag.Parse()

	var config Config
	if err := env.Parse(&config); err != nil {
		return nil, err
	}

	if config.Address == "" {
		config.Address = netAddress.String()
	}
	if config.BasicShortenAddress == "" {
		config.BasicShortenAddress = basicShortenAddress.String()
	}
	if config.FilePath == "" {
		config.FilePath = filePath.String()
	}
	if config.DatabaseURL == "" {
		config.DatabaseURL = databaseURL.String()
	}

	return &config, nil
}
