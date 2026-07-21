package service

import (
	"errors"
	"testing"

	"github.com/CimaCha/go-url-shortener/internal/repository"
	"github.com/CimaCha/go-url-shortener/internal/repository/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var errStorage = errors.New("storage error")

func TestShortenUrl(t *testing.T) {
	tests := []struct {
		name    string
		fullURL string
		want    string
	}{
		{name: "empty URL", want: "2jmj7l5rSw0yVb_vlWAYkK_YBwk"},
		{name: "full URL", fullURL: "https://example.com/path", want: "q8T575iSknB5NIL7Yf_g5s9Bnjk"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ShortenURL(tt.fullURL))
		})
	}
}

func TestServiceSetShortUrl(t *testing.T) {
	tests := []struct {
		name    string
		fullURL string
		setup   func(*mocks.MockUrlStorage)
		want    string
		wantErr error
	}{
		{name: "empty URL", wantErr: ErrEmptyURL},
		{
			name:    "stores URL",
			fullURL: "https://example.com/path",
			setup: func(storage *mocks.MockUrlStorage) {
				storage.EXPECT().SetShortUrl("q8T575iSknB5NIL7Yf_g5s9Bnjk", "https://example.com/path").Return(nil)
			},
			want: "q8T575iSknB5NIL7Yf_g5s9Bnjk",
		},
		{
			name:    "storage error",
			fullURL: "https://example.com/path",
			setup: func(storage *mocks.MockUrlStorage) {
				storage.EXPECT().SetShortUrl("q8T575iSknB5NIL7Yf_g5s9Bnjk", "https://example.com/path").Return(errStorage)
			},
			wantErr: errStorage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			storage := mocks.NewMockUrlStorage(controller)
			if tt.setup != nil {
				tt.setup(storage)
			}

			got, err := NewService(storage).SetShortURL(tt.fullURL)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestServiceGetFullUrl(t *testing.T) {
	tests := []struct {
		name     string
		shortURL string
		setup    func(*mocks.MockUrlStorage)
		want     string
		wantErr  error
	}{
		{name: "empty short URL", wantErr: ErrEmptyURL},
		{
			name:     "stored URL",
			shortURL: "short",
			setup: func(storage *mocks.MockUrlStorage) {
				storage.EXPECT().GetFullUrl("short").Return("https://example.com", nil)
			},
			want: "https://example.com",
		},
		{
			name:     "missing URL",
			shortURL: "missing",
			setup: func(storage *mocks.MockUrlStorage) {
				storage.EXPECT().GetFullUrl("missing").Return("", repository.ErrURLNotFound)
			},
			wantErr: repository.ErrURLNotFound,
		},
		{
			name:     "storage error",
			shortURL: "short",
			setup: func(storage *mocks.MockUrlStorage) {
				storage.EXPECT().GetFullUrl("short").Return("", errStorage)
			},
			wantErr: errStorage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			storage := mocks.NewMockUrlStorage(controller)
			if tt.setup != nil {
				tt.setup(storage)
			}

			got, err := NewService(storage).GetFullURL(tt.shortURL)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
