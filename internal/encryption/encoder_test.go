package encryption

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gzipBody(t *testing.T, body string) []byte {
	t.Helper()

	var compressed bytes.Buffer
	zw := gzip.NewWriter(&compressed)
	_, err := zw.Write([]byte(body))
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	return compressed.Bytes()
}

func TestGzipMiddlewareRequest(t *testing.T) {
	tests := []struct {
		name                string
		contentEncoding     string
		body                func(*testing.T) []byte
		wantStatus          int
		wantResponseBody    string
		wantAcceptEncoding  string
		wantHandlerCalled   bool
		wantRequestBody     string
		wantContentEncoding string
		wantContentLength   int64
	}{
		{
			name:              "request without content encoding passes through",
			body:              func(*testing.T) []byte { return []byte(`{"url":"https://example.com"}`) },
			wantStatus:        http.StatusNoContent,
			wantHandlerCalled: true,
			wantRequestBody:   `{"url":"https://example.com"}`,
			wantContentLength: int64(len(`{"url":"https://example.com"}`)),
		},
		{
			name:                "identity request passes through",
			contentEncoding:     "identity",
			body:                func(*testing.T) []byte { return []byte("plain body") },
			wantStatus:          http.StatusNoContent,
			wantHandlerCalled:   true,
			wantRequestBody:     "plain body",
			wantContentEncoding: "identity",
			wantContentLength:   int64(len("plain body")),
		},
		{
			name:              "gzip request is decoded",
			contentEncoding:   "gzip",
			body:              func(t *testing.T) []byte { return gzipBody(t, `{"url":"https://example.com"}`) },
			wantStatus:        http.StatusNoContent,
			wantHandlerCalled: true,
			wantRequestBody:   `{"url":"https://example.com"}`,
			wantContentLength: -1,
		},
		{
			name:              "gzip content encoding is case insensitive",
			contentEncoding:   "GZIP",
			body:              func(t *testing.T) []byte { return gzipBody(t, "case insensitive") },
			wantStatus:        http.StatusNoContent,
			wantHandlerCalled: true,
			wantRequestBody:   "case insensitive",
			wantContentLength: -1,
		},
		{
			name:              "invalid gzip request is rejected",
			contentEncoding:   "gzip",
			body:              func(*testing.T) []byte { return []byte("not gzip") },
			wantStatus:        http.StatusBadRequest,
			wantResponseBody:  "Bad Request\n",
			wantHandlerCalled: false,
			wantContentLength: -2,
		},
		{
			name:               "unsupported content encoding is rejected",
			contentEncoding:    "br",
			body:               func(*testing.T) []byte { return []byte("compressed elsewhere") },
			wantStatus:         http.StatusUnsupportedMediaType,
			wantResponseBody:   "Unsupported Media Type\n",
			wantAcceptEncoding: "gzip",
			wantHandlerCalled:  false,
			wantContentLength:  -2,
		},
		{
			name:               "encoding chain is rejected",
			contentEncoding:    "gzip, br",
			body:               func(t *testing.T) []byte { return gzipBody(t, "encoded body") },
			wantStatus:         http.StatusUnsupportedMediaType,
			wantResponseBody:   "Unsupported Media Type\n",
			wantAcceptEncoding: "gzip",
			wantHandlerCalled:  false,
			wantContentLength:  -2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestBody := tt.body(t)
			request := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(requestBody))
			if tt.contentEncoding != "" {
				request.Header.Set("Content-Encoding", tt.contentEncoding)
			}
			request.Header.Set("Content-Length", strconv.Itoa(len(requestBody)))

			var handlerCalled bool
			var gotRequestBody string
			var gotContentEncoding string
			var gotContentLengthHeader string
			var gotContentLength int64 = -2
			handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				handlerCalled = true
				body, err := io.ReadAll(request.Body)
				require.NoError(t, err)
				gotRequestBody = string(body)
				gotContentEncoding = request.Header.Get("Content-Encoding")
				gotContentLengthHeader = request.Header.Get("Content-Length")
				gotContentLength = request.ContentLength
				writer.WriteHeader(http.StatusNoContent)
			})
			response := httptest.NewRecorder()

			GzipMiddleware(handler).ServeHTTP(response, request)

			assert.Equal(t, tt.wantStatus, response.Code)
			assert.Equal(t, tt.wantResponseBody, response.Body.String())
			assert.Equal(t, tt.wantAcceptEncoding, response.Header().Get("Accept-Encoding"))
			assert.Equal(t, tt.wantHandlerCalled, handlerCalled)
			assert.Equal(t, tt.wantRequestBody, gotRequestBody)
			assert.Equal(t, tt.wantContentEncoding, gotContentEncoding)
			assert.Equal(t, tt.wantContentLength, gotContentLength)
			wantContentLengthHeader := ""
			if tt.wantHandlerCalled && tt.wantContentLength >= 0 {
				wantContentLengthHeader = strconv.Itoa(len(requestBody))
			}
			assert.Equal(t, wantContentLengthHeader, gotContentLengthHeader)
		})
	}
}

func TestAcceptsGzip(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		want   bool
	}{
		{name: "missing header", want: false},
		{name: "gzip", values: []string{"gzip"}, want: true},
		{name: "case insensitive gzip", values: []string{"GZIP"}, want: true},
		{name: "gzip among other encodings", values: []string{"br, gzip"}, want: true},
		{name: "positive gzip quality", values: []string{"gzip;q=0.5"}, want: true},
		{name: "zero gzip quality", values: []string{"gzip;q=0"}, want: false},
		{name: "zero decimal gzip quality", values: []string{"gzip; q=0.000"}, want: false},
		{name: "wildcard allows gzip", values: []string{"br, *;q=0.5"}, want: true},
		{name: "zero wildcard quality", values: []string{"*;q=0"}, want: false},
		{name: "invalid wildcard quality", values: []string{"*;q=invalid"}, want: false},
		{name: "explicit gzip denial overrides wildcard", values: []string{"*;q=1", "gzip;q=0"}, want: false},
		{name: "substring is not gzip", values: []string{"xgzip"}, want: false},
		{name: "invalid quality is rejected", values: []string{"gzip;q=invalid"}, want: false},
		{name: "quality above one is rejected", values: []string{"gzip;q=2"}, want: false},
		{name: "negative quality is rejected", values: []string{"gzip;q=-1"}, want: false},
		{name: "unknown parameter is rejected", values: []string{"gzip;level=1"}, want: false},
		{name: "multiple header lines", values: []string{"br", "gzip;q=1"}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := make(http.Header)
			for _, value := range tt.values {
				header.Add("Accept-Encoding", value)
			}

			assert.Equal(t, tt.want, acceptsGzip(header))
		})
	}
}
