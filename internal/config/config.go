package config

import (
	"flag"

	configflag "github.com/CimaCha/go-url-shortener/internal/config/flag"
	"github.com/caarlos0/env/v11"
)

type Config struct {
	Address             string `env:"SERVER_ADDRESS"`
	BasicShortenAddress string `env:"BASE_URL"`
}

func New() (*Config, error) {

	netAddress := configflag.NewNetAddressFlag("a", "address of service", "localhost:8080")
	basicShortenAddress := configflag.NewNetAddressFlag("b", "basic address for short url", "http://localhost:8080")
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

	return &config, nil
}
