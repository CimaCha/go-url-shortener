package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/CimaCha/go-url-shortener/internal/model"
	"github.com/CimaCha/go-url-shortener/internal/repository"
)

var (
	ErrURLHasGone     = errors.New("URL was deleted")
	ErrEmptyURL       = errors.New("empty URL")
	ErrEmptyURLList   = errors.New("empty URL list")
	ErrURLNotFound    = errors.New("URL not found")
	ErrUserNotFound   = errors.New("user not found")
	ErrRepository     = errors.New("error in repository")
	ErrFullURLExists  = errors.New("full URL already exists")
	ErrUniqueShortURL = errors.New("can't create unique short URL")
)

type lookupResult struct {
	shortURL string
	data     *model.StorageRecord
	err      error
}

type lookupJob struct {
	ctx      context.Context
	shortURL string
	results  chan<- lookupResult
	done     *sync.WaitGroup
}

type Service struct {
	storage             URLStorage
	maxShortURLAttempts int
	deleteJobs          chan lookupJob
	deleteWriterSlots   chan struct{}
	deleteWorkersDone   chan struct{}
}

func NewService(storage URLStorage, maxShortURLAttempts int, deleteBatchConcurrency int) Service {
	workerCount := max(1, deleteBatchConcurrency)
	s := Service{
		storage:             storage,
		maxShortURLAttempts: maxShortURLAttempts,
		deleteJobs:          make(chan lookupJob),
		deleteWriterSlots:   make(chan struct{}, workerCount),
		deleteWorkersDone:   make(chan struct{}),
	}
	var workers sync.WaitGroup
	workers.Add(workerCount)
	for range workerCount {
		go func() {
			defer workers.Done()
			for job := range s.deleteJobs {
				if job.ctx.Err() == nil {
					data, err := s.storage.GetShortURLData(job.ctx, job.shortURL)
					select {
					case job.results <- lookupResult{shortURL: job.shortURL, data: data, err: err}:
					case <-job.ctx.Done():
					}
				}
				job.done.Done()
			}
		}()
	}
	go func() {
		workers.Wait()
		close(s.deleteWorkersDone)
	}()
	return s
}

func (s Service) Close() {
	close(s.deleteJobs)
	<-s.deleteWorkersDone
}

func (s Service) Shorten(ctx context.Context, fullURL string, userID string) (string, error) {
	if fullURL == "" {
		return "", ErrEmptyURL
	}
	for range s.maxShortURLAttempts {
		shortURL := rand.Text()
		storedShortURL, err := s.storage.SaveShortURL(ctx, shortURL, fullURL, userID)
		if err == nil {
			return shortURL, nil
		}
		if errors.Is(err, repository.ErrFullURLExists) {
			return storedShortURL, ErrFullURLExists
		}
		if !errors.Is(err, repository.ErrShortURLExists) {
			return "", fmt.Errorf("%w: save short URL: %w", ErrRepository, err)
		}
	}

	return "", ErrUniqueShortURL
}

func (s Service) Resolve(ctx context.Context, shortURL string) (string, error) {
	if shortURL == "" {
		return "", ErrEmptyURL
	}
	fullURL, err := s.storage.FindFullURL(ctx, shortURL)
	if err != nil {
		if errors.Is(err, repository.ErrURLNotFound) {
			return "", ErrURLNotFound
		}
		if errors.Is(err, repository.ErrURLHasGone) {
			return "", ErrURLHasGone
		}
		return "", ErrRepository
	}
	return fullURL, nil
}

func (s Service) ShortenBatch(ctx context.Context, fullURLBatch []*model.OriginalURLRecord, userID string) ([]*model.ShortURLRecord, error) {
	if len(fullURLBatch) == 0 {
		return nil, ErrEmptyURLList
	}
	for _, record := range fullURLBatch {
		if record == nil || record.OriginalURL == "" {
			return nil, ErrEmptyURL
		}
	}

	for range s.maxShortURLAttempts {
		URLRecords := make([]*model.URLRecord, 0, len(fullURLBatch))
		shortURLRecords := make([]*model.ShortURLRecord, 0, len(fullURLBatch))
		for _, fullURL := range fullURLBatch {
			shortURL := rand.Text()
			URLRecords = append(URLRecords, &model.URLRecord{
				CorrelationID: fullURL.CorrelationID,
				OriginalURL:   fullURL.OriginalURL,
				ShortURL:      shortURL,
			})
			shortURLRecords = append(shortURLRecords, &model.ShortURLRecord{
				CorrelationID: fullURL.CorrelationID,
				ShortURL:      shortURL,
			})
		}

		err := s.storage.SaveShortURLBatch(ctx, URLRecords, userID)
		if err == nil {
			return shortURLRecords, nil
		}
		if !errors.Is(err, repository.ErrShortURLExists) {
			return nil, fmt.Errorf("%w: save short URL: %w", ErrRepository, err)
		}
	}

	return nil, ErrUniqueShortURL
}

func (s Service) GetUserURLs(ctx context.Context, userID string) ([]*model.UserRecord, error) {
	userURLsList, err := s.storage.GetUserURLs(ctx, userID)
	if errors.Is(err, repository.ErrUserNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%w: get user URLs: %w", ErrRepository, err)
	}
	return userURLsList, nil
}

func (s Service) DeleteBatch(ctx context.Context, shortURLList []string, userID string) error {
	if len(shortURLList) == 0 {
		return ErrEmptyURLList
	}

	select {
	case s.deleteWriterSlots <- struct{}{}:
		defer func() { <-s.deleteWriterSlots }()
	case <-ctx.Done():
		return fmt.Errorf("%w: get short URL data: %w", ErrRepository, ctx.Err())
	}
	lookupCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make(chan lookupResult)
	go func() {
		var pending sync.WaitGroup
		defer close(results)
		for _, shortURL := range shortURLList {
			pending.Add(1)
			select {
			case s.deleteJobs <- lookupJob{ctx: lookupCtx, shortURL: shortURL, results: results, done: &pending}:
			case <-lookupCtx.Done():
				pending.Done()
				pending.Wait()
				return
			}
		}
		pending.Wait()
	}()

	ownedURLs := make([]string, 0, len(shortURLList))
	var lookupErr error
	for result := range results {
		if result.err != nil {
			if errors.Is(result.err, repository.ErrURLNotFound) {
				continue
			}
			if lookupErr == nil {
				lookupErr = result.err
				cancel()
			}
			continue
		}
		if result.data == nil {
			if lookupErr == nil {
				lookupErr = errors.New("storage returned nil URL data")
				cancel()
			}
			continue
		}
		if result.data.UserID == userID && !result.data.DeletedFlag {
			ownedURLs = append(ownedURLs, result.shortURL)
		}
	}

	if lookupErr != nil {
		return fmt.Errorf("%w: get short URL data: %w", ErrRepository, lookupErr)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%w: get short URL data: %w", ErrRepository, err)
	}
	if len(ownedURLs) == 0 {
		return nil
	}
	if err := s.storage.DeleteURLsBatch(ctx, ownedURLs, userID); err != nil {
		return fmt.Errorf("%w: delete URLs: %w", ErrRepository, err)
	}
	return nil
}

type DeleteTask struct {
	Context context.Context
	URLs    []string
	UserID  string
}

func (s Service) ProcessDeleteTasks(onError func(error), batchSize int, timeout time.Duration, inputs ...<-chan DeleteTask) {
	if batchSize <= 0 || timeout <= 0 {
		onError(fmt.Errorf("delete batch size and timeout must be positive"))
		return
	}
	s.processDeleteTasks(batchSize, timeout, onError, inputs...)
}

func (s Service) processDeleteTasks(batchSize int, interval time.Duration, onError func(error), inputs ...<-chan DeleteTask) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	tasks := fanIn(inputs...)
	// ponytail: in-memory buffer; use a durable queue if tasks must survive restarts.
	groups := make(map[string]DeleteTask)
	count := 0
	flush := func() {
		for _, group := range groups {
			if err := s.DeleteBatch(group.Context, group.URLs, group.UserID); err != nil {
				onError(err)
			}
		}
		clear(groups)
		count = 0
	}
	for {
		select {
		case task, ok := <-tasks:
			if !ok {
				flush()
				return
			}
			for _, url := range task.URLs {
				group, exists := groups[task.UserID]
				if !exists {
					group = DeleteTask{Context: context.WithoutCancel(task.Context), UserID: task.UserID}
				}
				group.URLs = append(group.URLs, url)
				groups[task.UserID] = group
				count++
				if count == batchSize {
					flush()
				}
			}
		case <-ticker.C:
			flush()
		}
	}
}

func fanIn(inputs ...<-chan DeleteTask) <-chan DeleteTask {
	output := make(chan DeleteTask)
	var wg sync.WaitGroup
	wg.Add(len(inputs))
	for _, input := range inputs {
		go func() {
			defer wg.Done()
			for task := range input {
				output <- task
			}
		}()
	}
	go func() {
		wg.Wait()
		close(output)
	}()
	return output
}
