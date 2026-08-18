package logger

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestInitialize(t *testing.T) {
	tests := []struct {
		name      string
		level     string
		wantErr   bool
		wantDebug bool
		wantInfo  bool
		wantError bool
	}{
		{name: "debug level", level: "debug", wantDebug: true, wantInfo: true, wantError: true},
		{name: "error level", level: "error", wantError: true},
		{name: "unknown level", level: "unknown", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLog, err := Initialize(tt.level)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, gotLog)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, gotLog)
			t.Cleanup(func() { _ = gotLog.Sync() })
			assert.Equal(t, tt.wantDebug, gotLog.Core().Enabled(zap.DebugLevel))
			assert.Equal(t, tt.wantInfo, gotLog.Core().Enabled(zap.InfoLevel))
			assert.Equal(t, tt.wantError, gotLog.Core().Enabled(zap.ErrorLevel))
		})
	}
}

func TestRequestLogger(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		target     string
		status     int
		body       string
		wantStatus int
	}{
		{name: "explicit success status", method: http.MethodPost, target: "/short?source=test", status: http.StatusCreated, body: "hello", wantStatus: http.StatusCreated},
		{name: "implicit success status", method: http.MethodGet, target: "/health", body: "ok", wantStatus: http.StatusOK},
		{name: "error without body", method: http.MethodGet, target: "/short", status: http.StatusInternalServerError, wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, logs := observer.New(zap.InfoLevel)
			log := zap.New(core)

			handler := RequestLogger(log)(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				if tt.status != 0 {
					writer.WriteHeader(tt.status)
				}
				if tt.body != "" {
					_, _ = writer.Write([]byte(tt.body))
				}
			}))
			request := httptest.NewRequest(tt.method, tt.target, nil)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			assert.Equal(t, tt.wantStatus, response.Code)
			assert.Equal(t, tt.body, response.Body.String())
			require.Equal(t, 1, logs.Len())

			entry := logs.All()[0]
			assert.Equal(t, "handled HTTP request", entry.Message)
			assert.Equal(t, zap.InfoLevel, entry.Level)
			fields := entry.ContextMap()
			require.Len(t, fields, 3)
			assert.Equal(t, tt.method, fields["method"])
			assert.Equal(t, tt.target, fields["uri"])
			duration, ok := fields["duration"].(time.Duration)
			require.True(t, ok)
			assert.GreaterOrEqual(t, duration, time.Duration(0))
		})
	}
}
