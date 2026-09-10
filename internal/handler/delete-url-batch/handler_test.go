package apideletebatch

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CimaCha/go-url-shortener/internal/handler/delete-url-batch/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestDeleteBatchHandlerRejectsInvalidBody(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "malformed JSON", body: "["},
		{name: "empty list", body: "[]"},
		{name: "null list", body: "null"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			deleter := mocks.NewMockDeleter(controller)
			handler := NewAPIDeleteBatchHandler(*zap.NewNop(), deleter)
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodDelete, "/api/user/urls", strings.NewReader(tt.body))

			handler.ServeHTTP(response, request)

			require.Equal(t, http.StatusBadRequest, response.Code)
		})
	}
}

func TestDeleteBatchHandlerHandlesDeletionResult(t *testing.T) {
	tests := []struct {
		name       string
		serviceErr error
		wantStatus int
		wantLogs   int
	}{
		{name: "accepted", wantStatus: http.StatusAccepted},
		{name: "service error", serviceErr: errors.New("storage error"), wantStatus: http.StatusInternalServerError, wantLogs: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, logs := observer.New(zap.ErrorLevel)
			controller := gomock.NewController(t)
			deleter := mocks.NewMockDeleter(controller)
			request := httptest.NewRequest(http.MethodDelete, "/api/user/urls", strings.NewReader(`["first","second"]`))
			request.Header.Set("userID", "user-id")
			deleter.EXPECT().
				DeleteBatch(request.Context(), []string{"first", "second"}, "user-id").
				Return(tt.serviceErr)
			handler := NewAPIDeleteBatchHandler(*zap.New(core), deleter)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			require.Equal(t, tt.wantStatus, response.Code)
			require.Len(t, logs.All(), tt.wantLogs)
		})
	}
}
