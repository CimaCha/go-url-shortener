package main

import (
	apishortenurl "github.com/CimaCha/go-url-shortener/internal/handler/post-api-shorten-url"
	"net/http"
	"os"

	"github.com/CimaCha/go-url-shortener/internal/config"
	"github.com/CimaCha/go-url-shortener/internal/handler/get-full-url"
	"github.com/CimaCha/go-url-shortener/internal/handler/post-shorten-url"
	"github.com/CimaCha/go-url-shortener/internal/logger"
	"github.com/CimaCha/go-url-shortener/internal/repository"
	shortenerrouter "github.com/CimaCha/go-url-shortener/internal/router"
	"github.com/CimaCha/go-url-shortener/internal/service"
	"go.uber.org/zap"
)

func main() {
	if err := logger.Initialize("debug"); err != nil {
		panic(err)
	}
	defer func() {
		_ = logger.Log.Sync()
	}()

	cfg, err := config.New()
	if err != nil {
		logger.Log.Error("Error to parse config", zap.Error(err))
		_ = logger.Log.Sync()
		os.Exit(1)
	}

	logger.Log.Info("address of service", zap.String("address", cfg.Address))
	logger.Log.Info("basic address for short url", zap.String("basic url", cfg.BasicShortenAddress))

	storage := repository.NewMemoryURLStorage()
	shortenUrlService := service.NewService(storage)

	shortenUrlHandler := shortenurl.NewShortenURLHandler(shortenUrlService, cfg.BasicShortenAddress)
	apiShortenUrlHandler := apishortenurl.NewApiShortenURLHandler(shortenUrlService, cfg.BasicShortenAddress)
	getFullUrlHandler := fullurl.NewGetFullURLHandler(shortenUrlService)

	router := shortenerrouter.New(
		shortenUrlHandler,
		apiShortenUrlHandler,
		getFullUrlHandler)

	err = http.ListenAndServe(cfg.Address, router)
	if err != nil {
		logger.Log.Error("HTTP server stopped", zap.Error(err))
		os.Exit(1)
	}
}
