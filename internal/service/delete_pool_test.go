package service

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/CimaCha/go-url-shortener/internal/model"
	"github.com/CimaCha/go-url-shortener/internal/service/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDeleteBatchSharesPoolAndLimitsWriters(t *testing.T) {
	const workers = 2
	storage := mocks.NewMockURLStorage(gomock.NewController(t))
	started := make(chan string, 10)
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	ctx, cancel := context.WithCancel(context.Background())
	var requests sync.WaitGroup
	storage.EXPECT().GetShortURLData(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, url string) (*model.StorageRecord, error) {
		started <- url
		<-release
		return &model.StorageRecord{UserID: url}, nil
	}).Times(5)
	storage.EXPECT().DeleteURLsBatch(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, urls []string, userID string) error {
		if len(urls) != 1 || urls[0] != userID {
			return fmt.Errorf("mixed batch results: %v, %s", urls, userID)
		}
		return nil
	}).Times(5)
	service := NewService(storage, 5, workers)
	results := make(chan error, 5)
	t.Cleanup(func() {
		cancel()
		unblock()
		done := make(chan struct{})
		go func() { requests.Wait(); close(done) }()
		select {
		case <-done:
			service.Close()
		case <-time.After(time.Second):
			t.Error("deletion requests did not stop")
		}
	})
	for i := range 4 {
		url := fmt.Sprintf("url-%d", i)
		requests.Add(1)
		go func() { defer requests.Done(); results <- service.DeleteBatch(ctx, []string{url}, url) }()
	}
	for range workers {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("pool did not start")
		}
	}
	require.Len(t, service.deleteWriterSlots, workers)
	select {
	case <-started:
		t.Fatal("concurrent requests exceeded the shared pool limit")
	case <-time.After(30 * time.Millisecond):
	}
	unblock()
	for range 4 {
		select {
		case err := <-results:
			require.NoError(t, err)
		case <-time.After(time.Second):
			t.Fatal("deletion did not finish")
		}
	}
	requests.Add(1)
	go func() { defer requests.Done(); results <- service.DeleteBatch(ctx, []string{"last"}, "last") }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("pool was not reused")
	}
	require.NoError(t, awaitDeletion(t, results))
}

func TestDeleteBatchCancelsWhileWaitingForWriter(t *testing.T) {
	storage := mocks.NewMockURLStorage(gomock.NewController(t))
	service := NewService(storage, 5, 1)
	defer service.Close()
	service.deleteWriterSlots <- struct{}{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := service.DeleteBatch(ctx, []string{"url"}, "user")
	require.ErrorIs(t, err, context.Canceled)
	<-service.deleteWriterSlots
}

func TestDeleteBatchPoolSurvivesLookupError(t *testing.T) {
	storage := mocks.NewMockURLStorage(gomock.NewController(t))
	storage.EXPECT().GetShortURLData(gomock.Any(), "bad").Return(nil, errStorage)
	storage.EXPECT().GetShortURLData(gomock.Any(), "good").Return(&model.StorageRecord{UserID: "user"}, nil)
	storage.EXPECT().DeleteURLsBatch(gomock.Any(), []string{"good"}, "user").Return(nil)
	service := newTestService(t, storage, 5, 1)
	require.ErrorIs(t, service.DeleteBatch(context.Background(), []string{"bad"}, "user"), errStorage)
	require.NoError(t, service.DeleteBatch(context.Background(), []string{"good"}, "user"))
}
