package main

import (
	"github.com/CimaCha/go-url-shortener/internal/handler/get-full-url"
	"github.com/CimaCha/go-url-shortener/internal/handler/post-shorten-url"
	"github.com/CimaCha/go-url-shortener/internal/repository"
	"github.com/CimaCha/go-url-shortener/internal/service"
	"net/http"
)

func main() {
	storage := repository.NewMemoryURLStorage()
	shortenUrlService := service.NewService(storage)

	shortenUrlHandler := post_shorten_url.NewShortenURLHandler(shortenUrlService)
	getFullUrlHandler := get_full_url.NewGetFullURLHandler(shortenUrlService)

	mux := http.NewServeMux()
	mux.Handle(`/`, shortenUrlHandler)
	mux.Handle(`/{id}`, getFullUrlHandler)

	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
}
