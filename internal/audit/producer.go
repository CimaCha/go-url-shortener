package audit

import (
	"context"
	"net/http"
	"time"

	"github.com/CimaCha/go-url-shortener/internal/authentication"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// Producer holds the observers notified by AuditMiddleware.
type Producer struct {
	observers map[string]Observer
}

type urlKey struct{}

// RecordURL attaches the original URL to the current audited request.
func RecordURL(ctx context.Context, url string) {
	if value, ok := ctx.Value(urlKey{}).(*string); ok {
		*value = url
	}
}

// AuditMiddleware sends events after successful shortening or redirect responses.
func AuditMiddleware(log *zap.Logger, producer Producer, action Action) func(http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			response := middleware.NewWrapResponseWriter(writer, request.ProtoMajor)
			recordedURL := new(string)
			handler.ServeHTTP(response, request.WithContext(context.WithValue(request.Context(), urlKey{}, recordedURL)))
			if !((action == Shorten && response.Status() == http.StatusCreated) ||
				(action == Follow && response.Status() == http.StatusTemporaryRedirect)) {
				return
			}
			url := *recordedURL
			if action == Follow {
				url = response.Header().Get("Location")
			}
			if url == "" {
				log.Error("audit URL missing", zap.String("action", string(action)))
				return
			}
			exposure := Exposure{
				Timestamp: time.Now().UnixMilli(),
				Action:    action,
				UserID:    authentication.UserID(request.Context()),
				URL:       url,
			}
			for _, observer := range producer.observers {
				err := observer.update(exposure)
				if err != nil {
					log.Error("error updating observer", zap.Error(err))
				}
			}

		})
	}
}

// Register adds or replaces an observer by its ID.
func (e *Producer) Register(o Observer) {
	if e.observers == nil {
		e.observers = make(map[string]Observer)
	}
	e.observers[o.getID()] = o
}

// Deregister removes an observer by its ID.
func (e *Producer) Deregister(o Observer) {
	delete(e.observers, o.getID())
}
