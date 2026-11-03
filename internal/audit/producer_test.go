package audit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CimaCha/go-url-shortener/internal/authentication"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingObserver struct {
	exposures []Exposure
}

func (o *recordingObserver) update(exposure Exposure) error {
	o.exposures = append(o.exposures, exposure)
	return nil
}

func (*recordingObserver) getID() string { return "test" }

func TestAuditMiddlewareOnlyReportsSuccessfulActions(t *testing.T) {
	tests := []struct {
		name   string
		action Action
		status int
		want   bool
	}{
		{name: "shortened", action: Shorten, status: http.StatusCreated, want: true},
		{name: "redirected", action: Follow, status: http.StatusTemporaryRedirect, want: true},
		{name: "bad request", action: Shorten, status: http.StatusBadRequest},
		{name: "not found", action: Follow, status: http.StatusNotFound},
		{name: "existing URL", action: Shorten, status: http.StatusConflict},
		{name: "server error", action: Shorten, status: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			observer := &recordingObserver{}
			producer := Producer{}
			producer.Register(observer)
			handler := authentication.AuthMiddleware(zap.NewNop(), authentication.JWTBuilder{SecretKey: []byte("test-secret")}, authentication.NewUserIDParser([]byte("test-secret")))(AuditMiddleware(zap.NewNop(), producer, tt.action)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				RecordURL(r.Context(), "https://example.com/path")
				if tt.action == Follow {
					w.Header().Set("Location", "https://example.com/path")
				}
				w.WriteHeader(tt.status)
			})))
			request := httptest.NewRequest(http.MethodGet, "/short", nil)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			require.Equal(t, tt.status, response.Code)
			if !tt.want {
				require.Empty(t, observer.exposures)
				return
			}
			require.Len(t, observer.exposures, 1)
			require.NotEmpty(t, observer.exposures[0].UserID)
			require.Equal(t, tt.action, observer.exposures[0].Action)
			require.Equal(t, "https://example.com/path", observer.exposures[0].URL)
		})
	}
}

func TestAuditMiddlewareDoesNotReportRejectedAuthentication(t *testing.T) {
	observer := &recordingObserver{}
	producer := Producer{}
	producer.Register(observer)
	handler := authentication.AuthMiddleware(zap.NewNop(), authentication.JWTBuilder{SecretKey: []byte("test-secret")}, authentication.NewUserIDParser([]byte("test-secret")))(AuditMiddleware(zap.NewNop(), producer, Shorten)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})))
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.AddCookie(&http.Cookie{Name: "jwt", Value: "invalid"})
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	require.Equal(t, http.StatusUnauthorized, response.Code)
	require.Empty(t, observer.exposures)
}
