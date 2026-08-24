package main

import (
	"context"
	"errors"
	"github.com/CimaCha/go-url-shortener/internal/repository"
	database_storage "github.com/CimaCha/go-url-shortener/internal/repository/database-storage"
	"github.com/CimaCha/go-url-shortener/internal/repository/file"
	"log"
	"net/http"
	"os"

	"github.com/CimaCha/go-url-shortener/internal/config"
	"github.com/CimaCha/go-url-shortener/internal/handler/get-full-url"
	getping "github.com/CimaCha/go-url-shortener/internal/handler/get-ping"
	apishortenurl "github.com/CimaCha/go-url-shortener/internal/handler/post-api-shorten-url"
	"github.com/CimaCha/go-url-shortener/internal/handler/post-shorten-url"
	"github.com/CimaCha/go-url-shortener/internal/logger"
	shortenerrouter "github.com/CimaCha/go-url-shortener/internal/router"
	"github.com/CimaCha/go-url-shortener/internal/service"
	"go.uber.org/zap"
)

var (
	initializedLogger *zap.Logger
	err               error
)

func main() {
	initializedLogger, err = logger.Initialize("debug")
	if err != nil {
		log.Fatal("logger initialization error", err.Error())
	}
	if err = run(); err != nil {
		initializedLogger.Error("application stopped", zap.Error(err))
		_ = initializedLogger.Sync()
		os.Exit(1)
	}
	_ = initializedLogger.Sync()
}

func run() error {
	ctx := context.Background()
	cfg, err := config.New()
	if err != nil {
		initializedLogger.Error("cannot parse config")
		return err
	}

	var storage service.URLStorage
	var pingHandler http.Handler = getping.NewDBConnectionPingHandler(
		getping.PingFunc(func(context.Context) error {
			return errors.New("database is not configured")
		}),
	)

	switch {
	case cfg.DatabaseURL != "":
		dbStorage, err := database_storage.NewDatabaseStorage(ctx, cfg.DatabaseURL)
		if err != nil {
			return err
		}
		defer dbStorage.Close()

		storage = dbStorage
		pingHandler = getping.NewDBConnectionPingHandler(dbStorage)

	case cfg.FilePath != "":
		fileStorage, err := file.NewFileStorage(cfg.FilePath)
		if err != nil {
			return err
		}
		storage = fileStorage

	default:
		storage = repository.NewMemoryURLStorage(make(map[string]string))
	}

	urlService := service.NewService(storage)

	shortenURLHandler := shortenurl.NewShortenURLHandler(urlService, cfg.BasicShortenAddress)
	apiShortenURLHandler := apishortenurl.NewAPIShortenURLHandler(urlService, cfg.BasicShortenAddress)
	getFullURLHandler := fullurl.NewGetFullURLHandler(urlService)

	router := shortenerrouter.New(
		initializedLogger.With(zap.String("layer", "router")),
		shortenURLHandler,
		apiShortenURLHandler,
		getFullURLHandler,
		pingHandler)

	err = http.ListenAndServe(cfg.Address, router)
	if err != nil {
		initializedLogger.Error("HTTP server stopped", zap.Error(err))
		return err
	}
	return nil
}
