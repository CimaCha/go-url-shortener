package db

import (
	"flag"
	"fmt"
)

type DatabaseURL struct {
	URL string
}

func NewDatabaseURLFlag(flagName string, flagUsage string, defaultURL string) *DatabaseURL {
	databaseURL := DatabaseURL{
		URL: defaultURL,
	}
	flag.Var(&databaseURL, flagName, flagUsage)
	return &databaseURL
}

func (dbURL *DatabaseURL) String() string {
	return dbURL.URL
}

func (dbURL *DatabaseURL) Set(value string) error {
	if value == "" {
		return fmt.Errorf("database URL should not be empty")
	}
	dbURL.URL = value
	return nil
}
