package repository

import (
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemoryURLStorage(t *testing.T) {
	tests := []struct {
		name     string
		writes   [][2]string
		shortURL string
		want     string
		wantErr  error
	}{
		{name: "stored URL", writes: [][2]string{{"short", "https://example.com"}}, shortURL: "short", want: "https://example.com"},
		{name: "overwritten URL", writes: [][2]string{{"short", "https://old.example.com"}, {"short", "https://example.com"}}, shortURL: "short", want: "https://example.com"},
		{name: "missing URL", shortURL: "missing", wantErr: ErrURLNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewMemoryURLStorage()
			for _, write := range tt.writes {
				storage.SetShortURL(write[0], write[1])
			}

			got, err := storage.GetFullURL(tt.shortURL)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMemoryURLStorageConcurrentAccess(t *testing.T) {
	storage := NewMemoryURLStorage()
	start := make(chan struct{})
	var wg sync.WaitGroup

	for worker := range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start

			key := strconv.Itoa(worker)
			for range 100 {
				if err := storage.SetShortURL(key, key); err != nil {
					t.Errorf("SetShortURL() error = %v", err)
					return
				}
				if got, err := storage.GetFullURL(key); err != nil || got != key {
					t.Errorf("GetFullURL() = %q, %v; want %q, nil", got, err, key)
					return
				}
			}
		}()
	}

	close(start)
	wg.Wait()
}
