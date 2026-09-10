package main

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/CimaCha/go-url-shortener/internal/authentication"
	"github.com/CimaCha/go-url-shortener/internal/config"
	apideletebatch "github.com/CimaCha/go-url-shortener/internal/handler/delete-url-batch"
	userurls "github.com/CimaCha/go-url-shortener/internal/handler/get-api-user-urls"
	"github.com/CimaCha/go-url-shortener/internal/handler/get-full-url"
	getping "github.com/CimaCha/go-url-shortener/internal/handler/get-ping"
	apishortenbatch "github.com/CimaCha/go-url-shortener/internal/handler/post-api-shorten-batch"
	apishortenurl "github.com/CimaCha/go-url-shortener/internal/handler/post-api-shorten-url"
	"github.com/CimaCha/go-url-shortener/internal/handler/post-shorten-url"
	"github.com/CimaCha/go-url-shortener/internal/logger"
	"github.com/CimaCha/go-url-shortener/internal/repository"
	databasestorage "github.com/CimaCha/go-url-shortener/internal/repository/database-storage"
	"github.com/CimaCha/go-url-shortener/internal/repository/file"
	shortenerrouter "github.com/CimaCha/go-url-shortener/internal/router"
	"github.com/CimaCha/go-url-shortener/internal/service"
	"go.uber.org/zap"
)

func main() {
	initializedLogger, err := logger.Initialize("debug")
	if err != nil {
		log.Fatal("logger initialization error", err.Error())
	}
	defer initializedLogger.Sync()

	if err = run(*initializedLogger); err != nil {
		initializedLogger.Fatal("application stopped", zap.Error(err))
	}
}

func run(log zap.Logger) error {
	ctx := context.Background()
	cfg, err := config.New()
	if err != nil {
		log.Error("cannot parse config")
		return err
	}

	var storage service.URLStorage
	var pingHandler http.Handler = getping.NewDBConnectionPingHandler(log,
		getping.PingFunc(func(context.Context) error {
			return errors.New("database is not configured")
		}),
	)

	switch {
	case cfg.DatabaseURL != "":
		dbStorage, err := databasestorage.NewDatabaseStorage(ctx, cfg.DatabaseURL)
		if err != nil {
			return err
		}
		defer dbStorage.Close()

		storage = dbStorage
		pingHandler = getping.NewDBConnectionPingHandler(*log.With(zap.String("handler", "ping to db")), dbStorage)

	case cfg.FilePath != "":
		fileStorage, err := file.NewFileStorage(cfg.FilePath)
		if err != nil {
			return err
		}
		storage = fileStorage

	default:
		storage = repository.NewMemoryURLStorage(make(map[string]repository.URLData))
	}

	urlService := service.NewService(storage)

	shortenURLHandler := shortenurl.NewShortenURLHandler(*log.With(zap.String("handler", "shorten URL")), urlService, cfg.BasicShortenAddress)
	apiShortenURLHandler := apishortenurl.NewAPIShortenURLHandler(*log.With(zap.String("handler", "api shorten URL")), urlService, cfg.BasicShortenAddress)
	getFullURLHandler := fullurl.NewGetFullURLHandler(*log.With(zap.String("handler", "get full URL")), urlService)
	apiShortenBatchHandler := apishortenbatch.NewAPIShortenBatchHandler(*log.With(zap.String("handler", "api shorten batch")), urlService, cfg.BasicShortenAddress)
	userURLsHandler := userurls.NewHandler(*log.With(zap.String("handler", "get user URLs")), urlService, cfg.BasicShortenAddress)
	deleteURLsHandler := apideletebatch.NewAPIDeleteBatchHandler(*log.With(zap.String("handler", "delete URLs")), urlService)

	jwtBuilder := authentication.NewJWTBuilder([]byte(cfg.SecretKey))

	router := shortenerrouter.New(
		log.With(zap.String("layer", "router")),
		*jwtBuilder,
		shortenURLHandler,
		apiShortenURLHandler,
		getFullURLHandler,
		pingHandler,
		apiShortenBatchHandler,
		userURLsHandler,
		deleteURLsHandler)

	err = http.ListenAndServe(cfg.Address, router)
	if err != nil {
		log.Error("HTTP server stopped", zap.Error(err))
		return err
	}
	return nil
}
