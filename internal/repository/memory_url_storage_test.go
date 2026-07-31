package repository

import (
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemoryURLStorage(t *testing.T) {
	tests := []struct {
		name       string
		writes     [][2]string
		shortURL   string
		want       string
		wantSetErr error
		wantGetErr error
	}{
		{
			name:     "stores URL",
			writes:   [][2]string{{"short", "https://example.com"}},
			shortURL: "short",
			want:     "https://example.com",
		},
		{
			name:       "missing URL",
			shortURL:   "missing",
			wantGetErr: ErrURLNotFound,
		},
		{
			name: "does not overwrite existing short URL",
			writes: [][2]string{
				{"short", "https://first.example.com"},
				{"short", "https://second.example.com"},
			},
			shortURL:   "short",
			want:       "https://first.example.com",
			wantSetErr: ErrShortURLExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewMemoryURLStorage()
			var setErr error
			for _, write := range tt.writes {
				setErr = storage.SetShortURL(write[0], write[1])
			}

			assert.ErrorIs(t, setErr, tt.wantSetErr)

			got, err := storage.GetFullURL(tt.shortURL)
			assert.ErrorIs(t, err, tt.wantGetErr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMemoryURLStorageConcurrentSet(t *testing.T) {
	tests := []struct {
		name    string
		workers int
	}{
		{name: "stores a short URL only once", workers: 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewMemoryURLStorage()
			start := make(chan struct{})
			successes := make(chan string, tt.workers)
			var wg sync.WaitGroup

			for worker := range tt.workers {
				wg.Add(1)
				go func() {
					defer wg.Done()
					<-start

					fullURL := "https://example.com/" + strconv.Itoa(worker)
					if storage.SetShortURL("short", fullURL) == nil {
						successes <- fullURL
					}
				}()
			}

			close(start)
			wg.Wait()
			close(successes)

			var successfulURLs []string
			for fullURL := range successes {
				successfulURLs = append(successfulURLs, fullURL)
			}

			if assert.Len(t, successfulURLs, 1) {
				got, err := storage.GetFullURL("short")
				assert.NoError(t, err)
				assert.Equal(t, successfulURLs[0], got)
			}
		})
	}
}
