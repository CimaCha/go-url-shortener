package apideletebatch

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CimaCha/go-url-shortener/internal/authentication"
	authmocks "github.com/CimaCha/go-url-shortener/internal/authentication/mocks"
	"github.com/CimaCha/go-url-shortener/internal/handler/delete-url-batch/mocks"
	"github.com/CimaCha/go-url-shortener/internal/service"
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
			deleter.EXPECT().ProcessDeleteTasks(gomock.Any(), 100, time.Second, gomock.Any()).Do(func(_ func(error), batchSize int, timeout time.Duration, inputs ...<-chan service.DeleteTask) {
				for range inputs[0] {
					t.Error("invalid request was queued")
				}
			})
			handler := NewAPIDeleteBatchHandler(*zap.NewNop(), deleter, 100, time.Second)
			t.Cleanup(handler.Close)
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
		{name: "service error", serviceErr: errors.New("storage error"), wantStatus: http.StatusAccepted, wantLogs: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, logs := observer.New(zap.ErrorLevel)
			controller := gomock.NewController(t)
			deleter := mocks.NewMockDeleter(controller)
			request := httptest.NewRequest(http.MethodDelete, "/api/user/urls", strings.NewReader(`["first","second"]`))
			request.Header.Set("userID", "forged-user")
			request.AddCookie(&http.Cookie{Name: "jwt", Value: "test-token"})
			parser := authmocks.NewMockUserIDParser(controller)
			parser.EXPECT().GetUserID("test-token").Return("user-id", nil)
			builder := authmocks.NewMockTokenBuilder(controller)
			authentication.AuthMiddleware(zap.NewNop(), builder, parser)(http.HandlerFunc(func(_ http.ResponseWriter, authenticated *http.Request) {
				request = authenticated
			})).ServeHTTP(httptest.NewRecorder(), request)
			ctx, cancel := context.WithCancel(request.Context())
			request = request.WithContext(ctx)
			release := make(chan struct{})
			released := false
			tasks := make(chan service.DeleteTask, 1)
			deleter.EXPECT().ProcessDeleteTasks(gomock.Any(), 100, time.Second, gomock.Any()).Do(func(onError func(error), batchSize int, timeout time.Duration, inputs ...<-chan service.DeleteTask) {
				for task := range inputs[0] {
					<-release
					tasks <- task
					if tt.serviceErr != nil {
						onError(tt.serviceErr)
					}
				}
			})
			handler := NewAPIDeleteBatchHandler(*zap.New(core), deleter, 100, time.Second)
			response := httptest.NewRecorder()
			t.Cleanup(func() {
				cancel()
				if !released {
					close(release)
				}
				handler.Close()
			})

			returned := make(chan struct{})
			go func() {
				handler.ServeHTTP(response, request)
				close(returned)
			}()
			select {
			case <-returned:
			case <-time.After(time.Second):
				t.Fatal("handler waited for deletion")
			}
			cancel()
			close(release)
			released = true
			select {
			case task := <-tasks:
				require.NoError(t, task.Context.Err())
				require.Equal(t, "user-id", authentication.UserID(task.Context))
				require.Equal(t, "user-id", task.UserID)
				require.Equal(t, []string{"first", "second"}, task.URLs)
			case <-time.After(time.Second):
				t.Fatal("worker did not receive task")
			}
			require.Eventually(t, func() bool { return logs.Len() == tt.wantLogs }, time.Second, time.Millisecond)

			require.Equal(t, tt.wantStatus, response.Code)
			require.Len(t, logs.All(), tt.wantLogs)
		})
	}
}

func TestDeleteBatchHandlerRejectsFullQueue(t *testing.T) {
	handler := Handler{log: *zap.NewNop(), jobs: make(chan service.DeleteTask, 1)}
	handler.jobs <- service.DeleteTask{}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, "/api/user/urls", strings.NewReader(`["first"]`))
	handler.ServeHTTP(response, request)
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	require.Len(t, handler.jobs, 1)
}
