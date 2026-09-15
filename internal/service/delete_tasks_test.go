package service

import (
	"context"
	"testing"
	"time"

	"github.com/CimaCha/go-url-shortener/internal/model"
	"github.com/CimaCha/go-url-shortener/internal/service/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type deletionCall struct {
	ctx    context.Context
	urls   []string
	userID string
}

func TestProcessDeleteTasksFlush(t *testing.T) {
	tests := []struct {
		name        string
		tasks       [][]string
		interval    time.Duration
		closeFirst  bool
		idleFor     time.Duration
		beforeClose []int
		afterClose  []int
	}{
		{name: "threshold combines requests", tasks: [][]string{{"a"}, {"b"}}, interval: time.Hour, beforeClose: []int{2}},
		{name: "timeout flushes partial buffer", tasks: [][]string{{"a"}}, interval: 10 * time.Millisecond, beforeClose: []int{1}},
		{name: "close flushes remainder", tasks: [][]string{{"a"}}, interval: time.Hour, closeFirst: true, beforeClose: []int{1}},
		{name: "large batch splits at threshold", tasks: [][]string{{"a", "b", "c", "d", "e"}}, interval: time.Hour, beforeClose: []int{2, 2}, afterClose: []int{1}},
		{name: "empty input does not delete", interval: time.Hour, closeFirst: true},
		{name: "empty ticks do not delete", interval: 5 * time.Millisecond, idleFor: 30 * time.Millisecond},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := mocks.NewMockURLStorage(gomock.NewController(t))
			calls := make(chan deletionCall, 10)
			storage.EXPECT().GetShortURLData(gomock.Any(), gomock.Any()).Return(&model.StorageRecord{UserID: "user"}, nil).AnyTimes()
			storage.EXPECT().DeleteURLsBatch(gomock.Any(), gomock.Any(), "user").DoAndReturn(func(ctx context.Context, urls []string, id string) error {
				calls <- deletionCall{ctx: ctx, urls: append([]string(nil), urls...), userID: id}
				return nil
			}).AnyTimes()
			jobs := make(chan DeleteTask, 10)
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			var expected []string
			for _, urls := range tt.tasks {
				jobs <- DeleteTask{Context: ctx, URLs: urls, UserID: "user"}
				expected = append(expected, urls...)
			}
			closed := tt.closeFirst
			if closed {
				close(jobs)
			}
			s := newTestService(t, storage, 5, 10)
			done := make(chan struct{})
			errors := make(chan error, 10)
			go func() {
				s.processDeleteTasks(2, tt.interval, func(err error) { errors <- err }, jobs)
				close(done)
			}()
			t.Cleanup(func() {
				if !closed {
					close(jobs)
				}
				select {
				case <-done:
				case <-time.After(time.Second):
					t.Error("worker did not stop")
				}
			})
			var deleted []string
			receive := func(sizes []int) {
				for _, size := range sizes {
					select {
					case call := <-calls:
						require.Len(t, call.urls, size)
						require.NoError(t, call.ctx.Err())
						deleted = append(deleted, call.urls...)
					case <-time.After(time.Second):
						t.Fatal("buffer was not flushed")
					}
				}
			}
			if tt.idleFor > 0 {
				select {
				case <-calls:
					t.Fatal("empty buffer was deleted")
				case <-time.After(tt.idleFor):
				}
			}
			receive(tt.beforeClose)
			if !closed {
				close(jobs)
				closed = true
			}
			receive(tt.afterClose)
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("worker did not stop")
			}
			require.ElementsMatch(t, expected, deleted)
			require.Empty(t, calls)
			require.Empty(t, errors)
		})
	}
}

func TestProcessDeleteTasksGroupsUsersAndContinuesAfterError(t *testing.T) {
	storage := mocks.NewMockURLStorage(gomock.NewController(t))
	ctx := context.WithValue(context.Background(), struct{}{}, "request-value")
	storage.EXPECT().GetShortURLData(gomock.Any(), "a").Return(&model.StorageRecord{UserID: "alice"}, nil)
	storage.EXPECT().GetShortURLData(gomock.Any(), "b").Return(&model.StorageRecord{UserID: "bob"}, nil)
	storage.EXPECT().DeleteURLsBatch(gomock.Any(), []string{"a"}, "alice").Return(errStorage)
	storage.EXPECT().DeleteURLsBatch(gomock.Any(), []string{"b"}, "bob").DoAndReturn(func(got context.Context, urls []string, id string) error {
		require.Equal(t, "request-value", got.Value(struct{}{}))
		return nil
	})
	jobs := make(chan DeleteTask, 2)
	jobs <- DeleteTask{Context: ctx, URLs: []string{"a"}, UserID: "alice"}
	jobs <- DeleteTask{Context: ctx, URLs: []string{"b"}, UserID: "bob"}
	close(jobs)
	var errors []error
	newTestService(t, storage, 5, 10).processDeleteTasks(2, time.Hour, func(err error) { errors = append(errors, err) }, jobs)
	require.Len(t, errors, 1)
	require.ErrorIs(t, errors[0], errStorage)
}

func TestFanIn(t *testing.T) {
	first := make(chan DeleteTask, 1)
	second := make(chan DeleteTask, 1)
	first <- DeleteTask{URLs: []string{"a"}}
	close(first)
	output := fanIn(first, second)
	select {
	case task := <-output:
		require.Equal(t, []string{"a"}, task.URLs)
	case <-time.After(time.Second):
		t.Fatal("first input was not forwarded")
	}
	second <- DeleteTask{URLs: []string{"b", "c"}}
	close(second)
	var urls []string
	done := make(chan struct{})
	go func() {
		for task := range output {
			urls = append(urls, task.URLs...)
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("fanIn did not close output")
	}
	require.Equal(t, []string{"b", "c"}, urls)
	select {
	case _, ok := <-fanIn():
		require.False(t, ok)
	case <-time.After(time.Second):
		t.Fatal("fanIn with no inputs did not close")
	}
}

func TestProcessDeleteTasksRejectsInvalidParameters(t *testing.T) {
	for _, tt := range []struct {
		size     int
		interval time.Duration
	}{
		{0, time.Second}, {-1, time.Second}, {1, 0}, {1, -time.Second},
	} {
		var errors []error
		Service{}.ProcessDeleteTasks(func(err error) { errors = append(errors, err) }, tt.size, tt.interval, make(chan DeleteTask))
		require.Len(t, errors, 1)
	}
}
