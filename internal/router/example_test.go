package router

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/CimaCha/go-url-shortener/internal/authentication"
	apideletebatch "github.com/CimaCha/go-url-shortener/internal/handler/delete-url-batch"
	userurls "github.com/CimaCha/go-url-shortener/internal/handler/get-api-user-urls"
	fullurl "github.com/CimaCha/go-url-shortener/internal/handler/get-full-url"
	ping "github.com/CimaCha/go-url-shortener/internal/handler/get-ping"
	apishortenbatch "github.com/CimaCha/go-url-shortener/internal/handler/post-api-shorten-batch"
	apishortenurl "github.com/CimaCha/go-url-shortener/internal/handler/post-api-shorten-url"
	shortenurl "github.com/CimaCha/go-url-shortener/internal/handler/post-shorten-url"
	"github.com/CimaCha/go-url-shortener/internal/model"
	"github.com/CimaCha/go-url-shortener/internal/repository"
	"github.com/CimaCha/go-url-shortener/internal/service"
	"go.uber.org/zap"
)

func exampleRouter() (http.Handler, func()) {
	log := *zap.NewNop()
	urls := service.NewService(repository.NewMemoryURLStorage(nil), 5, 2)
	deleteHandler := apideletebatch.NewAPIDeleteBatchHandler(log, urls, 1, time.Second)
	r := New(zap.NewNop(), authentication.NewJWTBuilder([]byte("example-secret")),
		authentication.NewUserIDParser([]byte("example-secret")),
		shortenurl.NewShortenURLHandler(log, urls, "http://short"),
		apishortenurl.NewAPIShortenURLHandler(log, urls, "http://short"),
		fullurl.NewGetFullURLHandler(log, urls),
		ping.NewDBConnectionPingHandler(log, ping.PingFunc(func(_ context.Context) error { return nil })),
		apishortenbatch.NewAPIShortenBatchHandler(log, urls, "http://short"),
		userurls.NewHandler(log, urls, "http://short"), deleteHandler)
	return r, func() {
		deleteHandler.Close()
		urls.Close()
	}
}

func exampleRequest(r http.Handler, method, path, contentType, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	if cookie != nil {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	r.ServeHTTP(response, request)
	return response
}

func ExampleNew_singleURL() {
	r, closeRouter := exampleRouter()
	defer closeRouter()

	plain := exampleRequest(r, http.MethodPost, "/", "text/plain", "https://example.com/plain", nil)
	cookie := plain.Result().Cookies()[0] // Reuse this cookie for requests by the same user.
	shortURL := plain.Body.String()
	fmt.Println("POST /:", plain.Code, strings.HasPrefix(shortURL, "http://short/"))

	jsonResponse := exampleRequest(r, http.MethodPost, "/api/shorten", "application/json",
		`{"url":"https://example.com/json"}`, cookie)
	var result model.ShortenURLResponse
	if err := json.Unmarshal(jsonResponse.Body.Bytes(), &result); err != nil {
		panic(err)
	}
	fmt.Println("POST /api/shorten:", jsonResponse.Code, strings.HasPrefix(result.Result, "http://short/"))

	redirect := exampleRequest(r, http.MethodGet, "/"+strings.TrimPrefix(shortURL, "http://short/"), "", "", cookie)
	fmt.Println("GET /{id}:", redirect.Code, redirect.Header().Get("Location"))

	list := exampleRequest(r, http.MethodGet, "/api/user/urls", "", "", cookie)
	var userURLs model.UserURLsResponse
	if err := json.Unmarshal(list.Body.Bytes(), &userURLs); err != nil {
		panic(err)
	}
	fmt.Println("GET /api/user/urls:", list.Code, len(userURLs))

	check := exampleRequest(r, http.MethodGet, "/ping", "", "", cookie)
	fmt.Println("GET /ping:", check.Code)
	// Output:
	// POST /: 201 true
	// POST /api/shorten: 201 true
	// GET /{id}: 307 https://example.com/plain
	// GET /api/user/urls: 200 2
	// GET /ping: 200
}

func ExampleNew_batchAndDelete() {
	r, closeRouter := exampleRouter()
	defer closeRouter()

	batch := exampleRequest(r, http.MethodPost, "/api/shorten/batch", "application/json",
		`[{"correlation_id":"first","original_url":"https://example.com/first"}]`, nil)
	cookie := batch.Result().Cookies()[0]
	var shortened model.ShortenBatchResponse
	if err := json.Unmarshal(batch.Body.Bytes(), &shortened); err != nil {
		panic(err)
	}
	fmt.Println("POST /api/shorten/batch:", batch.Code, shortened[0].CorrelationID,
		strings.HasPrefix(shortened[0].ShortURL, "http://short/"))

	// The deletion endpoint accepts short IDs and queues them for later processing.
	urls, err := json.Marshal([]string{strings.TrimPrefix(shortened[0].ShortURL, "http://short/")})
	if err != nil {
		panic(err)
	}
	deleted := exampleRequest(r, http.MethodDelete, "/api/user/urls", "application/json", string(urls), cookie)
	fmt.Println("DELETE /api/user/urls:", deleted.Code)
	// Output:
	// POST /api/shorten/batch: 201 first true
	// DELETE /api/user/urls: 202
}
