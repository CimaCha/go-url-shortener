package apishortenbatch

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CimaCha/go-url-shortener/internal/handler/post-api-shorten-batch/mocks"
	"github.com/CimaCha/go-url-shortener/internal/model"
	"github.com/CimaCha/go-url-shortener/internal/service"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

var errBatchHandlerService = errors.New("service error")

type batchHandlerErrorReader struct{}

func (batchHandlerErrorReader) Read([]byte) (int, error) {
	return 0, errors.New("read error")
}

func TestAPIShortenBatchHandler(t *testing.T) {
	tests := []struct {
		name            string
		body            string
		readError       bool
		setup           func(*mocks.MockShortener, context.Context)
		wantStatus      int
		wantBody        string
		wantJSON        bool
		wantContentType string
	}{
		{
			name: "successful request",
			body: `[
				{"correlation_id":"first","original_url":"https://example.com/first"},
				{"correlation_id":"second","original_url":"https://example.com/second"}
			]`,
			setup: func(urlService *mocks.MockShortener, ctx context.Context) {
				urlService.EXPECT().ShortenBatch(ctx, []*model.OriginalURLRecord{
					{CorrelationId: "first", OriginalURL: "https://example.com/first"},
					{CorrelationId: "second", OriginalURL: "https://example.com/second"},
				}).Return([]*model.ShortURLRecord{
					{CorrelationId: "first", ShortURL: "short-first"},
					{CorrelationId: "second", ShortURL: "short-second"},
				}, nil)
			},
			wantStatus: http.StatusCreated,
			wantBody: `[
				{"correlation_id":"first","short_url":"http://localhost:8080/short-first"},
				{"correlation_id":"second","short_url":"http://localhost:8080/short-second"}
			]`,
			wantJSON:        true,
			wantContentType: "application/json",
		},
		{
			name: "empty batch",
			body: `[]`,
			setup: func(urlService *mocks.MockShortener, ctx context.Context) {
				urlService.EXPECT().ShortenBatch(ctx, []*model.OriginalURLRecord{}).Return(nil, service.ErrEmptyURLList)
			},
			wantStatus:      http.StatusBadRequest,
			wantBody:        "empty URL list\n",
			wantContentType: "text/plain; charset=utf-8",
		},
		{
			name: "service error",
			body: `[{"correlation_id":"first","original_url":"https://example.com/first"}]`,
			setup: func(urlService *mocks.MockShortener, ctx context.Context) {
				urlService.EXPECT().ShortenBatch(ctx, []*model.OriginalURLRecord{
					{CorrelationId: "first", OriginalURL: "https://example.com/first"},
				}).Return(nil, errBatchHandlerService)
			},
			wantStatus:      http.StatusInternalServerError,
			wantBody:        "Internal Server Error\n",
			wantContentType: "text/plain; charset=utf-8",
		},
		{
			name:            "body read error",
			readError:       true,
			wantStatus:      http.StatusInternalServerError,
			wantBody:        "Internal Server Error\n",
			wantContentType: "text/plain; charset=utf-8",
		},
		{
			name:            "invalid JSON",
			body:            `[`,
			wantStatus:      http.StatusBadRequest,
			wantBody:        "unexpected end of JSON input\n",
			wantContentType: "text/plain; charset=utf-8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			urlService := mocks.NewMockShortener(controller)
			var body io.Reader = strings.NewReader(tt.body)
			if tt.readError {
				body = batchHandlerErrorReader{}
			}
			request := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", body)
			if tt.setup != nil {
				tt.setup(urlService, request.Context())
			}
			handler := NewAPIShortenBatchHandler(*zap.NewNop(), urlService, "http://localhost:8080")
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			assert.Equal(t, tt.wantStatus, response.Code)
			if tt.wantJSON {
				assert.JSONEq(t, tt.wantBody, response.Body.String())
			} else {
				assert.Equal(t, tt.wantBody, response.Body.String())
			}
			assert.Equal(t, tt.wantContentType, response.Header().Get("Content-Type"))
		})
	}
}
