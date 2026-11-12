package audit

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/CimaCha/go-url-shortener/internal/authentication"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// Producer holds the observers notified by AuditMiddleware.
type Producer struct {
	state *producerState
}

type producerState struct {
	mu        sync.Mutex
	observers map[string]Observer
	events    chan auditEvent
	done      chan struct{}
	closed    bool
}

type auditEvent struct {
	exposure Exposure
	log      *zap.Logger
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
			producer.enqueue(auditEvent{exposure: exposure, log: log})
		})
	}
}

func (e Producer) enqueue(event auditEvent) {
	if e.state == nil {
		return
	}
	e.state.mu.Lock()
	queued := false
	if !e.state.closed {
		// shortcut: the bounded queue drops events under sustained overload; use a durable queue if every event must survive.
		select {
		case e.state.events <- event:
			queued = true
		default:
		}
	}
	e.state.mu.Unlock()
	if !queued {
		event.log.Error("audit queue full or closed; event dropped")
	}
}

// Register adds or replaces an observer by its ID.
func (e *Producer) Register(o Observer) {
	if e.state == nil {
		e.state = &producerState{
			observers: make(map[string]Observer),
			events:    make(chan auditEvent, 256),
			done:      make(chan struct{}),
		}
		go e.run()
	}
	e.state.mu.Lock()
	defer e.state.mu.Unlock()
	if !e.state.closed {
		e.state.observers[o.getID()] = o
	}
}

// Deregister removes an observer by its ID.
func (e *Producer) Deregister(o Observer) {
	if e.state == nil {
		return
	}
	e.state.mu.Lock()
	delete(e.state.observers, o.getID())
	e.state.mu.Unlock()
}

// Close drains queued audit events and stops the background worker.
func (e *Producer) Close() {
	if e.state == nil {
		return
	}
	e.state.mu.Lock()
	if !e.state.closed {
		e.state.closed = true
		close(e.state.events)
	}
	e.state.mu.Unlock()
	<-e.state.done
}

func (e *Producer) run() {
	defer close(e.state.done)
	for event := range e.state.events {
		e.state.mu.Lock()
		observers := make([]Observer, 0, len(e.state.observers))
		for _, observer := range e.state.observers {
			observers = append(observers, observer)
		}
		e.state.mu.Unlock()
		for _, observer := range observers {
			if err := observer.update(event.exposure); err != nil {
				event.log.Error("error updating observer", zap.Error(err))
			}
		}
	}
}
