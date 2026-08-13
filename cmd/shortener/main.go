package main

import (
	"fmt"
	"github.com/CimaCha/go-url-shortener/internal/config"
	"github.com/CimaCha/go-url-shortener/internal/handler/get-full-url"
	apishortenurl "github.com/CimaCha/go-url-shortener/internal/handler/post-api-shorten-url"
	"github.com/CimaCha/go-url-shortener/internal/handler/post-shorten-url"
	"github.com/CimaCha/go-url-shortener/internal/logger"
	"github.com/CimaCha/go-url-shortener/internal/repository/file"
	shortenerrouter "github.com/CimaCha/go-url-shortener/internal/router"
	"github.com/CimaCha/go-url-shortener/internal/service"
	"go.uber.org/zap"
	"net/http"
	"os"
)

func main() {
	if err := logger.Initialize("debug"); err != nil {
		panic(err)
	}
	if err := run(); err != nil {
		logger.Log.Error("application stopped", zap.Error(err))
		_ = logger.Log.Sync()
		os.Exit(1)
	}
	_ = logger.Log.Sync()
}

func run() error {
	cfg, err := config.New()
	if err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	logger.Log.Info("address of service", zap.String("address", cfg.Address))
	logger.Log.Info("basic address for short url", zap.String("basic url", cfg.BasicShortenAddress))
	logger.Log.Info("file storage path", zap.String("path", cfg.FilePath))

	fileStorage, err := file.NewFileStorage(cfg.FilePath)
	if err != nil {
		return fmt.Errorf("open file storage: %w", err)
	}
	shortenURLService := service.NewService(fileStorage)

	shortenURLHandler := shortenurl.NewShortenURLHandler(shortenURLService, cfg.BasicShortenAddress)
	apiShortenURLHandler := apishortenurl.NewApiShortenURLHandler(shortenURLService, cfg.BasicShortenAddress)
	getFullURLHandler := fullurl.NewGetFullURLHandler(shortenURLService)

	router := shortenerrouter.New(
		shortenURLHandler,
		apiShortenURLHandler,
		getFullURLHandler)

	err = http.ListenAndServe(cfg.Address, router)
	if err != nil {
		logger.Log.Error("HTTP server stopped", zap.Error(err))
		return err
	}
	return nil
}
