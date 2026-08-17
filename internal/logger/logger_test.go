package logger

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
			if (err != nil) != tt.wantErr {
				t.Fatalf("Initialize(%q) error = %v, wantErr %v", tt.level, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			t.Cleanup(func() { _ = gotLog.Sync() })
			if got := gotLog.Core().Enabled(zap.DebugLevel); got != tt.wantDebug {
				t.Errorf("debug enabled = %v, want %v", got, tt.wantDebug)
			}
			if got := gotLog.Core().Enabled(zap.InfoLevel); got != tt.wantInfo {
				t.Errorf("info enabled = %v, want %v", got, tt.wantInfo)
			}
			if got := gotLog.Core().Enabled(zap.ErrorLevel); got != tt.wantError {
				t.Errorf("error enabled = %v, want %v", got, tt.wantError)
			}
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
			var output bytes.Buffer
			encoder := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
				MessageKey:     "message",
				EncodeDuration: zapcore.NanosDurationEncoder,
			})
			log := zap.New(zapcore.NewCore(encoder, zapcore.AddSync(&output), zap.InfoLevel))

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

			if response.Code != tt.wantStatus {
				t.Errorf("response status = %d, want %d", response.Code, tt.wantStatus)
			}
			if response.Body.String() != tt.body {
				t.Errorf("response body = %q, want %q", response.Body.String(), tt.body)
			}

			var entry map[string]any
			if err := json.NewDecoder(&output).Decode(&entry); err != nil {
				t.Fatalf("decode log: %v", err)
			}
			wantFields := map[string]any{
				"message": "handled HTTP request",
				"method":  tt.method,
				"uri":     tt.target,
				"status":  float64(tt.wantStatus),
				"size":    float64(len(tt.body)),
			}
			for field, want := range wantFields {
				if got := entry[field]; got != want {
					t.Errorf("%s = %v, want %v", field, got, want)
				}
			}
			if duration, ok := entry["duration"].(float64); !ok || duration < 0 {
				t.Errorf("duration = %v, want non-negative number", entry["duration"])
			}
		})
	}
}
