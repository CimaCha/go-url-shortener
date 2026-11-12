package audit

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

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

type blockingObserver struct {
	once    sync.Once
	started chan struct{}
	release chan struct{}
}

func (o *blockingObserver) update(Exposure) error {
	o.once.Do(func() { close(o.started) })
	<-o.release
	return nil
}

func (*blockingObserver) getID() string { return "blocking" }

func TestAuditMiddlewareDoesNotWaitForObserver(t *testing.T) {
	observer := &blockingObserver{started: make(chan struct{}), release: make(chan struct{})}
	producer := Producer{}
	producer.Register(observer)
	defer func() {
		close(observer.release)
		producer.Close()
	}()
	handler := AuditMiddleware(zap.NewNop(), producer, Shorten)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		RecordURL(r.Context(), "https://example.com")
		w.WriteHeader(http.StatusCreated)
	}))
	completed := make(chan struct{})
	go func() {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil))
		close(completed)
	}()

	select {
	case <-observer.started:
	case <-time.After(time.Second):
		t.Fatal("audit observer was not called")
	}
	select {
	case <-completed:
	case <-time.After(time.Second):
		t.Fatal("HTTP handler waited for audit observer")
	}
	for range cap(producer.state.events) + 1 {
		completed := make(chan struct{})
		go func() {
			handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil))
			close(completed)
		}()
		select {
		case <-completed:
		case <-time.After(time.Second):
			t.Fatal("HTTP handler waited for a full audit queue")
		}
	}
}

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
			producer.Close()

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
	producer.Close()

	require.Equal(t, http.StatusUnauthorized, response.Code)
	require.Empty(t, observer.exposures)
}
