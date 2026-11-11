package audit

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
	"gopkg.in/h2non/gentleman.v2"
)

type benchmarkObserver struct{}

func (benchmarkObserver) update(Exposure) error { return nil }
func (benchmarkObserver) getID() string         { return "benchmark" }

func BenchmarkAuditMiddleware(b *testing.B) {
	for _, action := range []Action{Shorten, Follow} {
		b.Run(string(action), func(b *testing.B) {
			producer := Producer{}
			producer.Register(benchmarkObserver{})
			b.Cleanup(producer.Close)
			status := http.StatusCreated
			if action == Follow {
				status = http.StatusTemporaryRedirect
			}
			handler := AuditMiddleware(zap.NewNop(), producer, action)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if action == Follow {
					w.Header().Set("Location", "https://example.com/item")
				} else {
					RecordURL(r.Context(), "https://example.com/item")
				}
				w.WriteHeader(status)
			}))
			request := httptest.NewRequest(http.MethodPost, "/", nil)
			b.ReportAllocs()
			for b.Loop() {
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				if response.Code != status {
					b.Fatalf("unexpected status: %d", response.Code)
				}
			}
		})
	}
}

func BenchmarkFileWriteSubscriber(b *testing.B) {
	subscriber, err := NewFileWriteSubscriber("file", filepath.Join(b.TempDir(), "audit.json"))
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = subscriber.Close() })
	exposure := Exposure{Timestamp: 1, Action: Shorten, UserID: "user", URL: "https://example.com/item"}
	b.ReportAllocs()
	for b.Loop() {
		if err := subscriber.update(exposure); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkURLClientSubscriber(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	b.Cleanup(server.Close)
	subscriber := NewURLClientSubscriber("url", server.URL, gentleman.New())
	exposure := Exposure{Timestamp: 1, Action: Shorten, UserID: "user", URL: "https://example.com/item"}
	b.ReportAllocs()
	for b.Loop() {
		if err := subscriber.update(exposure); err != nil {
			b.Fatal(err)
		}
	}
}
