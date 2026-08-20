package router

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRouter(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		path        string
		contentType string
		wantStatus  int
		wantHandler string
		wantID      string
	}{
		{name: "shorten URL", method: http.MethodPost, path: "/", contentType: "text/plain", wantStatus: http.StatusCreated, wantHandler: "shorten"},
		{name: "unsupported content type", method: http.MethodPost, path: "/", contentType: "application/json", wantStatus: http.StatusUnsupportedMediaType},
		{name: "shorten URL through API", method: http.MethodPost, path: "/api/shorten", contentType: "application/json", wantStatus: http.StatusCreated, wantHandler: "api-shorten"},
		{name: "unsupported API content type", method: http.MethodPost, path: "/api/shorten", contentType: "text/plain", wantStatus: http.StatusUnsupportedMediaType},
		{name: "get full URL", method: http.MethodGet, path: "/short", wantStatus: http.StatusTemporaryRedirect, wantHandler: "full", wantID: "short"},
		{name: "ping database", method: http.MethodGet, path: "/ping", wantStatus: http.StatusOK, wantHandler: "ping"},
		{name: "method not allowed", method: http.MethodPut, path: "/", wantStatus: http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotHandler string
			var gotID string
			shortenURLHandler := http.HandlerFunc(func(res http.ResponseWriter, _ *http.Request) {
				gotHandler = "shorten"
				res.WriteHeader(http.StatusCreated)
			})
			apiShortenURLHandler := http.HandlerFunc(func(res http.ResponseWriter, _ *http.Request) {
				gotHandler = "api-shorten"
				res.WriteHeader(http.StatusCreated)
			})
			getFullURLHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
				gotHandler = "full"
				gotID = chi.URLParam(req, "id")
				res.WriteHeader(http.StatusTemporaryRedirect)
			})
			pingHandler := http.HandlerFunc(func(res http.ResponseWriter, _ *http.Request) {
				gotHandler = "ping"
				res.WriteHeader(http.StatusOK)
			})
			router := New(zap.NewNop(), shortenURLHandler, apiShortenURLHandler, getFullURLHandler, pingHandler)
			request := httptest.NewRequest(tt.method, tt.path, strings.NewReader("https://example.com"))
			request.Header.Set("Content-Type", tt.contentType)
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			assert.Equal(t, tt.wantStatus, response.Code)
			assert.Equal(t, tt.wantHandler, gotHandler)
			assert.Equal(t, tt.wantID, gotID)
		})
	}
}

func gzipRequestBody(t *testing.T, body string) []byte {
	t.Helper()

	var compressed bytes.Buffer
	zw := gzip.NewWriter(&compressed)
	_, err := zw.Write([]byte(body))
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	return compressed.Bytes()
}

func decodeResponseBody(t *testing.T, compressed bool, body []byte) string {
	t.Helper()
	if !compressed {
		return string(body)
	}

	zr, err := gzip.NewReader(bytes.NewReader(body))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, zr.Close())
	}()
	decoded, err := io.ReadAll(zr)
	require.NoError(t, err)

	return string(decoded)
}

func TestRouterGzipMiddleware(t *testing.T) {
	tests := []struct {
		name               string
		path               string
		contentType        string
		acceptEncoding     string
		contentEncoding    string
		body               func(*testing.T) []byte
		wantStatus         int
		wantHandler        string
		wantRequestBody    string
		wantResponseBody   string
		wantCompressed     bool
		wantAcceptEncoding string
	}{
		{
			name:             "API JSON response is compressed",
			path:             "/api/shorten",
			contentType:      "application/json",
			acceptEncoding:   "gzip",
			body:             func(*testing.T) []byte { return []byte(`{"url":"https://example.com"}`) },
			wantStatus:       http.StatusCreated,
			wantHandler:      "api-shorten",
			wantRequestBody:  `{"url":"https://example.com"}`,
			wantResponseBody: `{"result":"short"}`,
			wantCompressed:   true,
		},
		{
			name:             "API JSON response stays plain without negotiation",
			path:             "/api/shorten",
			contentType:      "application/json",
			body:             func(*testing.T) []byte { return []byte(`{"url":"https://example.com"}`) },
			wantStatus:       http.StatusCreated,
			wantHandler:      "api-shorten",
			wantRequestBody:  `{"url":"https://example.com"}`,
			wantResponseBody: `{"result":"short"}`,
		},
		{
			name:             "plain text response is not compressed",
			path:             "/",
			contentType:      "text/plain",
			acceptEncoding:   "gzip",
			body:             func(*testing.T) []byte { return []byte("https://example.com") },
			wantStatus:       http.StatusCreated,
			wantHandler:      "shorten",
			wantRequestBody:  "https://example.com",
			wantResponseBody: "short",
		},
		{
			name:             "compressed API request is decoded",
			path:             "/api/shorten",
			contentType:      "application/json",
			contentEncoding:  "gzip",
			body:             func(t *testing.T) []byte { return gzipRequestBody(t, `{"url":"https://example.com"}`) },
			wantStatus:       http.StatusCreated,
			wantHandler:      "api-shorten",
			wantRequestBody:  `{"url":"https://example.com"}`,
			wantResponseBody: `{"result":"short"}`,
		},
		{
			name:             "invalid gzip request is rejected before route handler",
			path:             "/api/shorten",
			contentType:      "application/json",
			acceptEncoding:   "gzip",
			contentEncoding:  "gzip",
			body:             func(*testing.T) []byte { return []byte("not gzip") },
			wantStatus:       http.StatusBadRequest,
			wantResponseBody: "Bad Request\n",
		},
		{
			name:               "unsupported request encoding is rejected",
			path:               "/api/shorten",
			contentType:        "application/json",
			contentEncoding:    "br",
			body:               func(*testing.T) []byte { return []byte("brotli bytes") },
			wantStatus:         http.StatusUnsupportedMediaType,
			wantResponseBody:   "Unsupported Media Type\n",
			wantAcceptEncoding: "gzip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotHandler string
			var gotRequestBody string
			shortenURLHandler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				gotHandler = "shorten"
				body, err := io.ReadAll(request.Body)
				require.NoError(t, err)
				gotRequestBody = string(body)
				writer.Header().Set("Content-Type", "text/plain")
				writer.WriteHeader(http.StatusCreated)
				_, err = writer.Write([]byte("short"))
				require.NoError(t, err)
			})
			apiShortenURLHandler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				gotHandler = "api-shorten"
				body, err := io.ReadAll(request.Body)
				require.NoError(t, err)
				gotRequestBody = string(body)
				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(http.StatusCreated)
				_, err = writer.Write([]byte(`{"result":"short"}`))
				require.NoError(t, err)
			})
			getFullURLHandler := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				gotHandler = "full"
				writer.WriteHeader(http.StatusTemporaryRedirect)
			})
			router := New(zap.NewNop(), shortenURLHandler, apiShortenURLHandler, getFullURLHandler, http.NotFoundHandler())
			request := httptest.NewRequest(http.MethodPost, tt.path, bytes.NewReader(tt.body(t)))
			request.Header.Set("Content-Type", tt.contentType)
			if tt.acceptEncoding != "" {
				request.Header.Set("Accept-Encoding", tt.acceptEncoding)
			}
			if tt.contentEncoding != "" {
				request.Header.Set("Content-Encoding", tt.contentEncoding)
			}
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			gotCompressed := response.Header().Get("Content-Encoding") == "gzip"
			assert.Equal(t, tt.wantStatus, response.Code)
			assert.Equal(t, tt.wantHandler, gotHandler)
			assert.Equal(t, tt.wantRequestBody, gotRequestBody)
			assert.Equal(t, tt.wantCompressed, gotCompressed)
			assert.Equal(t, tt.wantResponseBody, decodeResponseBody(t, gotCompressed, response.Body.Bytes()))
			assert.Equal(t, tt.wantAcceptEncoding, response.Header().Get("Accept-Encoding"))
		})
	}
}
