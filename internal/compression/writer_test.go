package compression

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var errOptionalOperation = errors.New("optional operation error")

type optionalResponseWriter struct {
	*httptest.ResponseRecorder
	flushed          bool
	hijacked         bool
	pushedPath       string
	writeHeaderCodes []int
}

func (w *optionalResponseWriter) WriteHeader(statusCode int) {
	w.writeHeaderCodes = append(w.writeHeaderCodes, statusCode)
	w.ResponseRecorder.WriteHeader(statusCode)
}

func (w *optionalResponseWriter) Flush() {
	w.flushed = true
}

func (w *optionalResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.hijacked = true
	return nil, nil, errOptionalOperation
}

func (w *optionalResponseWriter) Push(target string, _ *http.PushOptions) error {
	w.pushedPath = target
	return errOptionalOperation
}

type errorResponseWriter struct {
	header http.Header
}

func (w *errorResponseWriter) Header() http.Header {
	return w.header
}

func (*errorResponseWriter) Write([]byte) (int, error) {
	return 0, errors.New("write error")
}

func (*errorResponseWriter) WriteHeader(int) {}

func readGzipBody(t *testing.T, body []byte) string {
	t.Helper()

	zr, err := gzip.NewReader(bytes.NewReader(body))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, zr.Close())
	}()

	decoded, err := io.ReadAll(zr)
	require.NoError(t, err)

	return string(decoded)
}

func TestGzipMiddlewareResponse(t *testing.T) {
	tests := []struct {
		name                    string
		method                  string
		acceptEncoding          string
		contentType             string
		contentEncoding         string
		contentLength           string
		vary                    string
		status                  int
		explicitWriteHeader     bool
		body                    string
		wantCompressed          bool
		wantBody                string
		wantVary                string
		wantContentLength       string
		wantDetectedContentType string
	}{
		{
			name:                "JSON response is compressed",
			method:              http.MethodPost,
			acceptEncoding:      "gzip",
			contentType:         "application/json",
			contentLength:       "128",
			status:              http.StatusCreated,
			explicitWriteHeader: true,
			body:                `{"result":"http://localhost:8080/short"}`,
			wantCompressed:      true,
			wantBody:            `{"result":"http://localhost:8080/short"}`,
			wantVary:            "Accept-Encoding",
		},
		{
			name:                "HTML response with charset is compressed",
			method:              http.MethodGet,
			acceptEncoding:      "br, gzip;q=0.5",
			contentType:         "text/html; charset=utf-8",
			status:              http.StatusOK,
			explicitWriteHeader: true,
			body:                "<html><body>short URL</body></html>",
			wantCompressed:      true,
			wantBody:            "<html><body>short URL</body></html>",
			wantVary:            "Accept-Encoding",
		},
		{
			name:                    "implicit OK detects HTML and compresses it",
			method:                  http.MethodGet,
			acceptEncoding:          "gzip",
			body:                    "<html><body>detected</body></html>",
			wantCompressed:          true,
			wantBody:                "<html><body>detected</body></html>",
			wantVary:                "Accept-Encoding",
			wantDetectedContentType: "text/html; charset=utf-8",
		},
		{
			name:                "text response is not compressed",
			method:              http.MethodPost,
			acceptEncoding:      "gzip",
			contentType:         "text/plain",
			status:              http.StatusCreated,
			explicitWriteHeader: true,
			body:                "http://localhost:8080/short",
			wantBody:            "http://localhost:8080/short",
		},
		{
			name:                "gzip denial returns identity JSON",
			method:              http.MethodPost,
			acceptEncoding:      "gzip;q=0",
			contentType:         "application/json",
			status:              http.StatusCreated,
			explicitWriteHeader: true,
			body:                `{"result":"short"}`,
			wantBody:            `{"result":"short"}`,
			wantVary:            "Accept-Encoding",
		},
		{
			name:                "redirect is not compressed",
			method:              http.MethodGet,
			acceptEncoding:      "gzip",
			contentType:         "text/html",
			status:              http.StatusTemporaryRedirect,
			explicitWriteHeader: true,
			body:                "redirect",
			wantBody:            "redirect",
		},
		{
			name:                "multiple choices boundary is not compressed",
			method:              http.MethodGet,
			acceptEncoding:      "gzip",
			contentType:         "text/html",
			status:              http.StatusMultipleChoices,
			explicitWriteHeader: true,
			body:                "choices",
			wantBody:            "choices",
		},
		{
			name:                "error is not compressed",
			method:              http.MethodPost,
			acceptEncoding:      "gzip",
			contentType:         "application/json",
			status:              http.StatusBadRequest,
			explicitWriteHeader: true,
			body:                `{"error":"bad request"}`,
			wantBody:            `{"error":"bad request"}`,
		},
		{
			name:                "no content response has no gzip stream",
			method:              http.MethodPost,
			acceptEncoding:      "gzip",
			contentType:         "application/json",
			status:              http.StatusNoContent,
			explicitWriteHeader: true,
		},
		{
			name:                "HEAD response is not compressed",
			method:              http.MethodHead,
			acceptEncoding:      "gzip",
			contentType:         "application/json",
			status:              http.StatusOK,
			explicitWriteHeader: true,
		},
		{
			name:                "existing content encoding prevents double compression",
			method:              http.MethodGet,
			acceptEncoding:      "gzip",
			contentType:         "text/html",
			contentEncoding:     "br",
			status:              http.StatusOK,
			explicitWriteHeader: true,
			body:                "already encoded",
			wantBody:            "already encoded",
		},
		{
			name:                "invalid media type is not compressed",
			method:              http.MethodPost,
			acceptEncoding:      "gzip",
			contentType:         "application/json; charset",
			status:              http.StatusOK,
			explicitWriteHeader: true,
			body:                `{}`,
			wantBody:            `{}`,
		},
		{
			name:                "existing Vary value is preserved without duplicate",
			method:              http.MethodPost,
			acceptEncoding:      "gzip",
			contentType:         "application/json",
			vary:                "Origin, Accept-Encoding",
			status:              http.StatusOK,
			explicitWriteHeader: true,
			body:                `{}`,
			wantCompressed:      true,
			wantBody:            `{}`,
			wantVary:            "Origin, Accept-Encoding",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				if tt.contentType != "" {
					writer.Header().Set("Content-Type", tt.contentType)
				}
				if tt.contentEncoding != "" {
					writer.Header().Set("Content-Encoding", tt.contentEncoding)
				}
				if tt.contentLength != "" {
					writer.Header().Set("Content-Length", tt.contentLength)
				}
				if tt.vary != "" {
					writer.Header().Set("Vary", tt.vary)
				}
				if tt.explicitWriteHeader {
					writer.WriteHeader(tt.status)
				}
				if tt.body != "" {
					_, err := writer.Write([]byte(tt.body))
					require.NoError(t, err)
				}
			})
			request := httptest.NewRequest(tt.method, "/resource", nil)
			if tt.acceptEncoding != "" {
				request.Header.Set("Accept-Encoding", tt.acceptEncoding)
			}
			response := httptest.NewRecorder()

			GzipMiddleware(zap.NewNop())(handler).ServeHTTP(response, request)

			wantStatus := tt.status
			if !tt.explicitWriteHeader {
				wantStatus = http.StatusOK
			}
			assert.Equal(t, wantStatus, response.Code)
			assert.Equal(t, tt.wantCompressed, response.Header().Get("Content-Encoding") == "gzip")
			if tt.wantCompressed {
				assert.Equal(t, tt.wantBody, readGzipBody(t, response.Body.Bytes()))
			} else {
				assert.Equal(t, tt.wantBody, response.Body.String())
				assert.Equal(t, tt.contentEncoding, response.Header().Get("Content-Encoding"))
			}
			var wantVaryValues []string
			if tt.wantVary != "" {
				wantVaryValues = []string{tt.wantVary}
			}
			assert.Equal(t, wantVaryValues, response.Header().Values("Vary"))
			assert.Equal(t, tt.wantContentLength, response.Header().Get("Content-Length"))
			if tt.wantDetectedContentType != "" {
				assert.Equal(t, tt.wantDetectedContentType, response.Header().Get("Content-Type"))
			}
		})
	}
}

func TestCompressWriterOptionalInterfaces(t *testing.T) {
	tests := []struct {
		name string
		run  func(*testing.T, *compressWriter, *optionalResponseWriter)
	}{
		{
			name: "unwrap returns underlying writer",
			run: func(t *testing.T, writer *compressWriter, underlying *optionalResponseWriter) {
				assert.Same(t, underlying, writer.Unwrap())
			},
		},
		{
			name: "flush is delegated",
			run: func(t *testing.T, writer *compressWriter, underlying *optionalResponseWriter) {
				writer.Flush()
				assert.True(t, underlying.flushed)
				assert.Equal(t, []int{http.StatusOK}, underlying.writeHeaderCodes)
			},
		},
		{
			name: "repeated response status is ignored",
			run: func(t *testing.T, writer *compressWriter, underlying *optionalResponseWriter) {
				writer.WriteHeader(http.StatusCreated)
				writer.WriteHeader(http.StatusAccepted)
				assert.Equal(t, []int{http.StatusCreated}, underlying.writeHeaderCodes)
			},
		},
		{
			name: "hijack is delegated",
			run: func(t *testing.T, writer *compressWriter, underlying *optionalResponseWriter) {
				_, _, err := writer.Hijack()
				assert.ErrorIs(t, err, errOptionalOperation)
				assert.True(t, underlying.hijacked)
			},
		},
		{
			name: "HTTP2 push is delegated",
			run: func(t *testing.T, writer *compressWriter, underlying *optionalResponseWriter) {
				err := writer.Push("/asset.css", nil)
				assert.ErrorIs(t, err, errOptionalOperation)
				assert.Equal(t, "/asset.css", underlying.pushedPath)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			underlying := &optionalResponseWriter{ResponseRecorder: httptest.NewRecorder()}
			writer := newCompressWriter(underlying, http.MethodGet, true)

			tt.run(t, writer, underlying)
		})
	}
}

func TestCompressWriterUnsupportedInterfaces(t *testing.T) {
	tests := []struct {
		name string
		run  func(*compressWriter) error
	}{
		{
			name: "hijack returns not supported",
			run: func(writer *compressWriter) error {
				_, _, err := writer.Hijack()
				return err
			},
		},
		{
			name: "HTTP2 push returns not supported",
			run: func(writer *compressWriter) error {
				return writer.Push("/asset.css", nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := newCompressWriter(httptest.NewRecorder(), http.MethodGet, true)

			assert.NotPanics(t, func() {
				assert.ErrorIs(t, tt.run(writer), http.ErrNotSupported)
			})
		})
	}
}

func TestGzipMiddlewareCloseErrorDoesNotPanic(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "gzip close error is logged without panic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(http.StatusOK)
				_, _ = writer.Write([]byte(`{"result":"short"}`))
			})
			request := httptest.NewRequest(http.MethodGet, "/resource", nil)
			request.Header.Set("Accept-Encoding", "gzip")
			response := &errorResponseWriter{header: make(http.Header)}

			assert.NotPanics(t, func() {
				GzipMiddleware(zap.NewNop())(handler).ServeHTTP(response, request)
			})
		})
	}
}
