package main

import (
	"flag"
	"log"
	"net/http"

	flag2 "github.com/CimaCha/go-url-shortener/internal/config/flag"
	"github.com/CimaCha/go-url-shortener/internal/handler/get-full-url"
	"github.com/CimaCha/go-url-shortener/internal/handler/post-shorten-url"
	"github.com/CimaCha/go-url-shortener/internal/repository"
	shortenerrouter "github.com/CimaCha/go-url-shortener/internal/router"
	"github.com/CimaCha/go-url-shortener/internal/service"
)

func main() {

	netAddress := flag2.NewNetAddressFlag("a", "address of service")
	basicShortAddress := flag2.NewNetAddressFlag("b", "basic address for short url")

	flag.Parse()

	log.Printf("address of service is: %s\n", netAddress.String())
	log.Printf("basic address for short url is: %s\n", basicShortAddress.String())
	storage := repository.NewMemoryURLStorage()
	shortenUrlService := service.NewService(storage)

	shortenUrlHandler := shortenurl.NewShortenURLHandler(shortenUrlService, *basicShortAddress)
	getFullUrlHandler := fullurl.NewGetFullURLHandler(shortenUrlService)

	router := shortenerrouter.New(shortenUrlHandler, getFullUrlHandler)

	log.Fatal(http.ListenAndServe(netAddress.String(), router))
}
