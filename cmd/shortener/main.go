package main

import (
	"github.com/CimaCha/go-url-shortener/internal/handler/get_full_url"
	"github.com/CimaCha/go-url-shortener/internal/handler/post_shorten_url"
	"github.com/CimaCha/go-url-shortener/internal/service"
	"net/http"
	"sync"
)

func main() {
	storage := sync.Map{}
	shortenUrlService := service.NewService(&storage)

	shortenUrlHandler := post_shorten_url.NewShortenUrlHandler(shortenUrlService)
	getFullUrlHandler := get_full_url.NewGetFullUrlHandler(shortenUrlService)

	mux := http.NewServeMux()
	mux.Handle(`/`, shortenUrlHandler)
	mux.Handle(`/{id}`, getFullUrlHandler)

	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
}
