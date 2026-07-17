package main

import (
	"context"
	"github.com/CimaCha/go-url-shortener/internal/handler/post_shorten_url"
	"github.com/CimaCha/go-url-shortener/internal/repository"
	"github.com/CimaCha/go-url-shortener/internal/service"
	"net/http"
)

func main() {
	ctx := context.Background()

	redisClient := repository.NewRedisClient(`localhost:6379`)

	shortenUrlService := service.NewService(redisClient)

	shortenUrlHandler := post_shorten_url.NewShortenUrlHandler(ctx, shortenUrlService)

	mux := http.NewServeMux()
	mux.Handle(`/`, shortenUrlHandler)

	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
}
