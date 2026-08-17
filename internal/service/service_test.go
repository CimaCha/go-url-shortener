package service

import (
	"errors"
	"testing"

	"github.com/CimaCha/go-url-shortener/internal/repository"
	"github.com/CimaCha/go-url-shortener/internal/service/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

var errStorage = errors.New("storage error")

func TestServiceSetShortURL(t *testing.T) {
	tests := []struct {
		name    string
		fullURL string
		setup   func(*mocks.MockURLStorage, *string)
		wantErr error
	}{
		{name: "empty URL", wantErr: ErrEmptyURL},
		{
			name:    "stores URL",
			fullURL: "https://example.com/path",
			setup: func(storage *mocks.MockURLStorage, storedShortURL *string) {
				storage.EXPECT().
					SetShortURL(gomock.Any(), "https://example.com/path").
					DoAndReturn(func(shortURL string, _ string) error {
						*storedShortURL = shortURL
						return nil
					})
			},
		},
		{
			name:    "storage error",
			fullURL: "https://example.com/path",
			setup: func(storage *mocks.MockURLStorage, _ *string) {
				storage.EXPECT().
					SetShortURL(gomock.Any(), "https://example.com/path").
					Return(errStorage)
			},
			wantErr: ErrRepository,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			storage := mocks.NewMockURLStorage(controller)
			var storedShortURL string
			if tt.setup != nil {
				tt.setup(storage, &storedShortURL)
			}

			got, err := NewService(zap.NewNop(), storage).SetShortURL(tt.fullURL)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, got)
				assert.Equal(t, storedShortURL, got)
			}
		})
	}
}

func TestServiceGetFullURL(t *testing.T) {
	tests := []struct {
		name     string
		shortURL string
		setup    func(*mocks.MockURLStorage)
		want     string
		wantErr  error
	}{
		{name: "empty short URL", wantErr: ErrEmptyURL},
		{
			name:     "stored URL",
			shortURL: "short",
			setup: func(storage *mocks.MockURLStorage) {
				storage.EXPECT().GetFullURL("short").Return("https://example.com", nil)
			},
			want: "https://example.com",
		},
		{
			name:     "missing URL",
			shortURL: "missing",
			setup: func(storage *mocks.MockURLStorage) {
				storage.EXPECT().GetFullURL("missing").Return("", repository.ErrURLNotFound)
			},
			wantErr: ErrURLNotFound,
		},
		{
			name:     "storage error",
			shortURL: "short",
			setup: func(storage *mocks.MockURLStorage) {
				storage.EXPECT().GetFullURL("short").Return("", errStorage)
			},
			wantErr: ErrRepository,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			storage := mocks.NewMockURLStorage(controller)
			if tt.setup != nil {
				tt.setup(storage)
			}

			got, err := NewService(zap.NewNop(), storage).GetFullURL(tt.shortURL)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
