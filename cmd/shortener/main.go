package main

import (
	"context"
	"fmt"
	database_storage "github.com/CimaCha/go-url-shortener/internal/repository/database-storage"
	"log"
	"net/http"
	"os"

	"github.com/CimaCha/go-url-shortener/internal/config"
	"github.com/CimaCha/go-url-shortener/internal/handler/get-full-url"
	getping "github.com/CimaCha/go-url-shortener/internal/handler/get-ping"
	apishortenurl "github.com/CimaCha/go-url-shortener/internal/handler/post-api-shorten-url"
	"github.com/CimaCha/go-url-shortener/internal/handler/post-shorten-url"
	"github.com/CimaCha/go-url-shortener/internal/logger"
	"github.com/CimaCha/go-url-shortener/internal/repository/file"
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
		return fmt.Errorf("parse config: %w", err)
	}

	fileStorage, err := file.NewFileStorage(cfg.FilePath)
	if err != nil {
		initializedLogger.Error("new file storage error", zap.Error(err))
		return fmt.Errorf("open file storage: %w", err)
	}

	storage, err := database_storage.NewDatabaseStorage(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer storage.Db.Close()
	shortenURLService := service.NewService(fileStorage)

	shortenURLHandler := shortenurl.NewShortenURLHandler(shortenURLService, cfg.BasicShortenAddress)
	apiShortenURLHandler := apishortenurl.NewAPIShortenURLHandler(shortenURLService, cfg.BasicShortenAddress)
	getFullURLHandler := fullurl.NewGetFullURLHandler(shortenURLService)
	pingConnectionHandler := getping.NewDBConnectionPingHandler(storage.Db)

	router := shortenerrouter.New(
		initializedLogger.With(zap.String("layer", "router")),
		shortenURLHandler,
		apiShortenURLHandler,
		getFullURLHandler,
		pingConnectionHandler)

	err = http.ListenAndServe(cfg.Address, router)
	if err != nil {
		initializedLogger.Error("HTTP server stopped", zap.Error(err))
		return err
	}
	return nil
}
