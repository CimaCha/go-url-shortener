package service

import (
	"context"
	"errors"
	"testing"

	"github.com/CimaCha/go-url-shortener/internal/repository"
	"github.com/CimaCha/go-url-shortener/internal/service/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var errStorage = errors.New("storage error")

func TestServiceShorten(t *testing.T) {
	tests := []struct {
		name      string
		fullURL   string
		setup     func(*mocks.MockURLStorage, context.Context, *string)
		wantErr   error
		wantCause error
	}{
		{name: "empty URL", wantErr: ErrEmptyURL},
		{
			name:    "stores URL",
			fullURL: "https://example.com/path",
			setup: func(storage *mocks.MockURLStorage, ctx context.Context, storedShortURL *string) {
				storage.EXPECT().
					SaveShortURL(ctx, gomock.Any(), "https://example.com/path").
					DoAndReturn(func(_ context.Context, shortURL string, _ string) error {
						*storedShortURL = shortURL
						return nil
					})
			},
		},
		{
			name:    "retries after collision",
			fullURL: "https://example.com/path",
			setup: func(storage *mocks.MockURLStorage, ctx context.Context, storedShortURL *string) {
				first := storage.EXPECT().
					SaveShortURL(ctx, gomock.Any(), "https://example.com/path").
					Return(repository.ErrShortURLExists)
				second := storage.EXPECT().
					SaveShortURL(ctx, gomock.Any(), "https://example.com/path").
					DoAndReturn(func(_ context.Context, shortURL string, _ string) error {
						*storedShortURL = shortURL
						return nil
					})
				gomock.InOrder(first, second)
			},
		},
		{
			name:    "returns error after maximum collisions",
			fullURL: "https://example.com/path",
			setup: func(storage *mocks.MockURLStorage, ctx context.Context, _ *string) {
				storage.EXPECT().
					SaveShortURL(ctx, gomock.Any(), "https://example.com/path").
					Return(repository.ErrShortURLExists).
					Times(maxShortURLAttempts)
			},
			wantErr: ErrUniqueShortURL,
		},
		{
			name:    "storage error",
			fullURL: "https://example.com/path",
			setup: func(storage *mocks.MockURLStorage, ctx context.Context, _ *string) {
				storage.EXPECT().
					SaveShortURL(ctx, gomock.Any(), "https://example.com/path").
					Return(errStorage)
			},
			wantErr:   ErrRepository,
			wantCause: errStorage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			controller := gomock.NewController(t)
			storage := mocks.NewMockURLStorage(controller)
			var storedShortURL string
			if tt.setup != nil {
				tt.setup(storage, ctx, &storedShortURL)
			}

			got, err := NewService(storage).Shorten(ctx, tt.fullURL)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, got)
				assert.Equal(t, storedShortURL, got)
			}
			if tt.wantCause != nil {
				assert.ErrorIs(t, err, tt.wantCause)
			}
		})
	}
}

func TestServiceResolve(t *testing.T) {
	tests := []struct {
		name     string
		shortURL string
		setup    func(*mocks.MockURLStorage, context.Context)
		want     string
		wantErr  error
	}{
		{name: "empty short URL", wantErr: ErrEmptyURL},
		{
			name:     "stored URL",
			shortURL: "short",
			setup: func(storage *mocks.MockURLStorage, ctx context.Context) {
				storage.EXPECT().FindFullURL(ctx, "short").Return("https://example.com", nil)
			},
			want: "https://example.com",
		},
		{
			name:     "missing URL",
			shortURL: "missing",
			setup: func(storage *mocks.MockURLStorage, ctx context.Context) {
				storage.EXPECT().FindFullURL(ctx, "missing").Return("", repository.ErrURLNotFound)
			},
			wantErr: ErrURLNotFound,
		},
		{
			name:     "storage error",
			shortURL: "short",
			setup: func(storage *mocks.MockURLStorage, ctx context.Context) {
				storage.EXPECT().FindFullURL(ctx, "short").Return("", errStorage)
			},
			wantErr: ErrRepository,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			controller := gomock.NewController(t)
			storage := mocks.NewMockURLStorage(controller)
			if tt.setup != nil {
				tt.setup(storage, ctx)
			}

			got, err := NewService(storage).Resolve(ctx, tt.shortURL)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
