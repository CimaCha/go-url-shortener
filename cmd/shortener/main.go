package main

import (
	"log"
	"net/http"

	"github.com/CimaCha/go-url-shortener/internal/config"
	"github.com/CimaCha/go-url-shortener/internal/handler/get-full-url"
	"github.com/CimaCha/go-url-shortener/internal/handler/post-shorten-url"
	"github.com/CimaCha/go-url-shortener/internal/repository"
	shortenerrouter "github.com/CimaCha/go-url-shortener/internal/router"
	"github.com/CimaCha/go-url-shortener/internal/service"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("address of service is: %s\n", cfg.Address)
	log.Printf("basic address for short url is: %s\n", cfg.BasicShortenAddress)

	storage := repository.NewMemoryURLStorage()
	shortenUrlService := service.NewService(storage)

	shortenUrlHandler := shortenurl.NewShortenURLHandler(shortenUrlService, cfg.BasicShortenAddress)
	getFullUrlHandler := fullurl.NewGetFullURLHandler(shortenUrlService)

	router := shortenerrouter.New(shortenUrlHandler, getFullUrlHandler)

	log.Fatal(http.ListenAndServe(cfg.Address, router))
}
