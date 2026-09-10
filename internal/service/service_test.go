package service

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CimaCha/go-url-shortener/internal/model"
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
					SaveShortURL(ctx, gomock.Any(), "https://example.com/path", "user-id").
					DoAndReturn(func(_ context.Context, shortURL string, _, _ string) (string, error) {
						*storedShortURL = shortURL
						return "", nil
					})
			},
		},
		{
			name:    "returns stored short URL for duplicate full URL",
			fullURL: "https://example.com/path",
			setup: func(storage *mocks.MockURLStorage, ctx context.Context, storedShortURL *string) {
				*storedShortURL = "stored"
				storage.EXPECT().
					SaveShortURL(ctx, gomock.Any(), "https://example.com/path", "user-id").
					Return("stored", repository.ErrFullURLExists)
			},
			wantErr: ErrFullURLExists,
		},
		{
			name:    "retries after collision",
			fullURL: "https://example.com/path",
			setup: func(storage *mocks.MockURLStorage, ctx context.Context, storedShortURL *string) {
				first := storage.EXPECT().
					SaveShortURL(ctx, gomock.Any(), "https://example.com/path", "user-id").
					Return("", repository.ErrShortURLExists)
				second := storage.EXPECT().
					SaveShortURL(ctx, gomock.Any(), "https://example.com/path", "user-id").
					DoAndReturn(func(_ context.Context, shortURL string, _, _ string) (string, error) {
						*storedShortURL = shortURL
						return "", nil
					})
				gomock.InOrder(first, second)
			},
		},
		{
			name:    "returns error after maximum collisions",
			fullURL: "https://example.com/path",
			setup: func(storage *mocks.MockURLStorage, ctx context.Context, _ *string) {
				storage.EXPECT().
					SaveShortURL(ctx, gomock.Any(), "https://example.com/path", "user-id").
					Return("", repository.ErrShortURLExists).
					Times(maxShortURLAttempts)
			},
			wantErr: ErrUniqueShortURL,
		},
		{
			name:    "storage error",
			fullURL: "https://example.com/path",
			setup: func(storage *mocks.MockURLStorage, ctx context.Context, _ *string) {
				storage.EXPECT().
					SaveShortURL(ctx, gomock.Any(), "https://example.com/path", "user-id").
					Return("", errStorage)
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

			got, err := NewService(storage).Shorten(ctx, tt.fullURL, "user-id")
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				if errors.Is(tt.wantErr, ErrFullURLExists) {
					assert.Equal(t, storedShortURL, got)
				}
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

func TestServiceShortenBatchReturnsStoredURLs(t *testing.T) {
	ctx := context.Background()
	controller := gomock.NewController(t)
	storage := mocks.NewMockURLStorage(controller)
	var stored []*model.URLRecord
	storage.EXPECT().
		SaveShortURLBatch(ctx, gomock.Any(), "user-id").
		DoAndReturn(func(_ context.Context, records []*model.URLRecord, _ string) error {
			stored = records
			return nil
		})

	got, err := NewService(storage).ShortenBatch(ctx, []*model.OriginalURLRecord{
		{CorrelationID: "first", OriginalURL: "https://example.com/first"},
		{CorrelationID: "second", OriginalURL: "https://example.com/second"},
	}, "user-id")

	assert.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Len(t, stored, 2)
	for i := range got {
		assert.Equal(t, stored[i].CorrelationID, got[i].CorrelationID)
		assert.Equal(t, stored[i].ShortURL, got[i].ShortURL)
	}
}

func TestServiceShortenBatchRegeneratesURLsAfterCollision(t *testing.T) {
	ctx := context.Background()
	controller := gomock.NewController(t)
	storage := mocks.NewMockURLStorage(controller)
	attempts := make([][]string, 0, 2)
	storage.EXPECT().
		SaveShortURLBatch(ctx, gomock.Any(), "user-id").
		DoAndReturn(func(_ context.Context, records []*model.URLRecord, _ string) error {
			shortURLs := make([]string, len(records))
			for i, record := range records {
				shortURLs[i] = record.ShortURL
			}
			attempts = append(attempts, shortURLs)
			if len(attempts) == 1 {
				return repository.ErrShortURLExists
			}
			return nil
		}).
		Times(2)

	got, err := NewService(storage).ShortenBatch(ctx, []*model.OriginalURLRecord{
		{CorrelationID: "first", OriginalURL: "https://example.com/first"},
		{CorrelationID: "second", OriginalURL: "https://example.com/second"},
	}, "user-id")

	assert.NoError(t, err)
	assert.Len(t, attempts, 2)
	assert.NotEqual(t, attempts[0], attempts[1])
	assert.Equal(t, attempts[1][0], got[0].ShortURL)
	assert.Equal(t, attempts[1][1], got[1].ShortURL)
}

func TestServiceShortenBatchReturnsErrorAfterMaximumCollisions(t *testing.T) {
	ctx := context.Background()
	controller := gomock.NewController(t)
	storage := mocks.NewMockURLStorage(controller)
	storage.EXPECT().
		SaveShortURLBatch(ctx, gomock.Any(), "user-id").
		Return(repository.ErrShortURLExists).
		Times(maxShortURLAttempts)

	got, err := NewService(storage).ShortenBatch(ctx, []*model.OriginalURLRecord{
		{CorrelationID: "first", OriginalURL: "https://example.com/first"},
	}, "user-id")

	assert.ErrorIs(t, err, ErrUniqueShortURL)
	assert.Nil(t, got)
}

func TestServiceShortenBatchReturnsRepositoryError(t *testing.T) {
	ctx := context.Background()
	controller := gomock.NewController(t)
	storage := mocks.NewMockURLStorage(controller)
	storage.EXPECT().SaveShortURLBatch(ctx, gomock.Any(), "user-id").Return(errStorage)

	got, err := NewService(storage).ShortenBatch(ctx, []*model.OriginalURLRecord{
		{CorrelationID: "first", OriginalURL: "https://example.com/first"},
	}, "user-id")

	assert.ErrorIs(t, err, ErrRepository)
	assert.ErrorIs(t, err, errStorage)
	assert.Nil(t, got)
}

func TestServiceShortenBatchRejectsEmptyList(t *testing.T) {
	controller := gomock.NewController(t)
	storage := mocks.NewMockURLStorage(controller)

	got, err := NewService(storage).ShortenBatch(context.Background(), nil, "user-id")

	assert.ErrorIs(t, err, ErrEmptyURLList)
	assert.Nil(t, got)
}

func TestServiceShortenBatchRejectsEmptyRecord(t *testing.T) {
	controller := gomock.NewController(t)
	storage := mocks.NewMockURLStorage(controller)

	tests := []struct {
		name  string
		batch []*model.OriginalURLRecord
	}{
		{name: "nil record", batch: []*model.OriginalURLRecord{nil}},
		{name: "empty original URL", batch: []*model.OriginalURLRecord{{CorrelationID: "first"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewService(storage).ShortenBatch(context.Background(), tt.batch, "user-id")
			assert.ErrorIs(t, err, ErrEmptyURL)
			assert.Nil(t, got)
		})
	}
}

func TestServiceDeleteBatch(t *testing.T) {
	ctx := context.Background()
	controller := gomock.NewController(t)
	storage := mocks.NewMockURLStorage(controller)
	storage.EXPECT().GetShortURLData(gomock.Any(), "own").Return(&model.StorageRecord{UserID: "user-id"}, nil)
	storage.EXPECT().GetShortURLData(gomock.Any(), "foreign").Return(&model.StorageRecord{UserID: "other-user"}, nil)
	storage.EXPECT().GetShortURLData(gomock.Any(), "deleted").Return(&model.StorageRecord{UserID: "user-id", DeletedFlag: true}, nil)
	storage.EXPECT().GetShortURLData(gomock.Any(), "missing").Return(nil, repository.ErrURLNotFound)
	storage.EXPECT().DeleteURLsBatch(ctx, []string{"own"}, "user-id").Return(nil)

	err := NewService(storage).DeleteBatch(ctx, []string{"own", "foreign", "deleted", "missing"}, "user-id")

	assert.NoError(t, err)
}

func TestServiceDeleteBatchWrapsLookupError(t *testing.T) {
	ctx := context.Background()
	controller := gomock.NewController(t)
	storage := mocks.NewMockURLStorage(controller)
	storage.EXPECT().GetShortURLData(gomock.Any(), "first").Return(nil, errStorage)

	err := NewService(storage).DeleteBatch(ctx, []string{"first"}, "user-id")

	assert.ErrorIs(t, err, ErrRepository)
	assert.ErrorIs(t, err, errStorage)
}

func TestServiceDeleteBatchWrapsDeleteError(t *testing.T) {
	ctx := context.Background()
	controller := gomock.NewController(t)
	storage := mocks.NewMockURLStorage(controller)
	storage.EXPECT().GetShortURLData(gomock.Any(), "first").Return(&model.StorageRecord{UserID: "user-id"}, nil)
	storage.EXPECT().DeleteURLsBatch(ctx, []string{"first"}, "user-id").Return(errStorage)

	err := NewService(storage).DeleteBatch(ctx, []string{"first"}, "user-id")

	assert.ErrorIs(t, err, ErrRepository)
	assert.ErrorIs(t, err, errStorage)
}

func TestServiceDeleteBatchRejectsNilMetadata(t *testing.T) {
	controller := gomock.NewController(t)
	storage := mocks.NewMockURLStorage(controller)
	storage.EXPECT().GetShortURLData(gomock.Any(), "first").Return(nil, nil)

	err := NewService(storage).DeleteBatch(context.Background(), []string{"first"}, "user-id")

	assert.ErrorIs(t, err, ErrRepository)
}

func TestServiceDeleteBatchHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	controller := gomock.NewController(t)
	storage := mocks.NewMockURLStorage(controller)
	storage.EXPECT().
		GetShortURLData(gomock.Any(), "first").
		DoAndReturn(func(ctx context.Context, _ string) (*model.StorageRecord, error) {
			return nil, ctx.Err()
		}).
		AnyTimes()

	err := NewService(storage).DeleteBatch(ctx, []string{"first"}, "user-id")

	assert.ErrorIs(t, err, ErrRepository)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestServiceDeleteBatchRejectsEmptyList(t *testing.T) {
	controller := gomock.NewController(t)
	storage := mocks.NewMockURLStorage(controller)

	err := NewService(storage).DeleteBatch(context.Background(), nil, "user-id")

	assert.ErrorIs(t, err, ErrEmptyURLList)
}

func TestServiceDeleteBatchSkipsDeleteWhenNoOwnedURLs(t *testing.T) {
	controller := gomock.NewController(t)
	storage := mocks.NewMockURLStorage(controller)
	storage.EXPECT().GetShortURLData(gomock.Any(), "foreign").Return(&model.StorageRecord{UserID: "other-user"}, nil)

	err := NewService(storage).DeleteBatch(context.Background(), []string{"foreign"}, "user-id")

	assert.NoError(t, err)
}

func TestServiceDeleteBatchLooksUpURLsConcurrently(t *testing.T) {
	ctx := context.Background()
	controller := gomock.NewController(t)
	storage := mocks.NewMockURLStorage(controller)
	started := make(chan string, 2)
	release := make(chan struct{})
	for _, shortURL := range []string{"first", "second"} {
		shortURL := shortURL
		storage.EXPECT().
			GetShortURLData(gomock.Any(), shortURL).
			DoAndReturn(func(context.Context, string) (*model.StorageRecord, error) {
				started <- shortURL
				<-release
				return &model.StorageRecord{UserID: "user-id"}, nil
			})
	}
	storage.EXPECT().
		DeleteURLsBatch(ctx, gomock.InAnyOrder([]string{"first", "second"}), "user-id").
		Return(nil)
	errCh := make(chan error, 1)
	go func() {
		errCh <- NewService(storage).DeleteBatch(ctx, []string{"first", "second"}, "user-id")
	}()

	for range 2 {
		select {
		case <-started:
		case <-time.After(time.Second):
			close(release)
			<-errCh
			t.Fatal("URL lookups did not run concurrently")
		}
	}
	close(release)
	assert.NoError(t, <-errCh)
}

func TestServiceDeleteBatchLimitsConcurrentLookups(t *testing.T) {
	const maxConcurrentLookups = 10

	ctx := context.Background()
	controller := gomock.NewController(t)
	storage := mocks.NewMockURLStorage(controller)
	shortURLs := make([]string, maxConcurrentLookups+1)
	started := make(chan struct{}, len(shortURLs))
	release := make(chan struct{})
	var activeLookups int32
	var peakLookups int32

	for i := range shortURLs {
		shortURL := fmt.Sprintf("short-%d", i)
		shortURLs[i] = shortURL
		storage.EXPECT().
			GetShortURLData(gomock.Any(), shortURL).
			DoAndReturn(func(context.Context, string) (*model.StorageRecord, error) {
				active := atomic.AddInt32(&activeLookups, 1)
				for peak := atomic.LoadInt32(&peakLookups); active > peak; peak = atomic.LoadInt32(&peakLookups) {
					if atomic.CompareAndSwapInt32(&peakLookups, peak, active) {
						break
					}
				}
				started <- struct{}{}
				<-release
				atomic.AddInt32(&activeLookups, -1)
				return &model.StorageRecord{UserID: "user-id"}, nil
			})
	}
	storage.EXPECT().
		DeleteURLsBatch(ctx, gomock.InAnyOrder(shortURLs), "user-id").
		Return(nil)

	errCh := make(chan error, 1)
	go func() {
		errCh <- NewService(storage).DeleteBatch(ctx, shortURLs, "user-id")
	}()

	for range maxConcurrentLookups {
		select {
		case <-started:
		case <-time.After(time.Second):
			close(release)
			<-errCh
			t.Fatal("ten URL lookups did not start")
		}
	}
	var exceeded bool
	select {
	case <-started:
		exceeded = true
	case <-time.After(100 * time.Millisecond):
	}
	close(release)

	assert.NoError(t, <-errCh)
	assert.False(t, exceeded)
	assert.Equal(t, int32(maxConcurrentLookups), atomic.LoadInt32(&peakLookups))
}

func TestServiceGetUserURLs(t *testing.T) {
	ctx := context.Background()
	controller := gomock.NewController(t)
	storage := mocks.NewMockURLStorage(controller)
	want := []*model.UserRecord{{ShortURL: "short", OriginalURL: "https://example.com"}}
	storage.EXPECT().GetUserURLs(ctx, "user-id").Return(want, nil)

	got, err := NewService(storage).GetUserURLs(ctx, "user-id")

	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestServiceGetUserURLsMapsNotFound(t *testing.T) {
	ctx := context.Background()
	controller := gomock.NewController(t)
	storage := mocks.NewMockURLStorage(controller)
	storage.EXPECT().GetUserURLs(ctx, "user-id").Return(nil, repository.ErrUserNotFound)

	got, err := NewService(storage).GetUserURLs(ctx, "user-id")

	assert.ErrorIs(t, err, ErrUserNotFound)
	assert.Nil(t, got)
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
			name:     "deleted URL",
			shortURL: "deleted",
			setup: func(storage *mocks.MockURLStorage, ctx context.Context) {
				storage.EXPECT().FindFullURL(ctx, "deleted").Return("", repository.ErrURLHasGone)
			},
			wantErr: ErrURLHasGone,
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
