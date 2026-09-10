package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"sync"

	"github.com/CimaCha/go-url-shortener/internal/model"
	"github.com/CimaCha/go-url-shortener/internal/repository"
)

const (
	maxShortURLAttempts    = 5
	deleteBatchConcurrency = 10
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

type Service struct {
	storage URLStorage
}

func NewService(storage URLStorage) Service {
	return Service{storage: storage}
}

func (s Service) Shorten(ctx context.Context, fullURL string, userID string) (string, error) {
	if fullURL == "" {
		return "", ErrEmptyURL
	}
	for range maxShortURLAttempts {
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

	for range maxShortURLAttempts {
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

	type lookupResult struct {
		shortURL string
		data     *model.StorageRecord
		err      error
	}

	lookupCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan string)
	results := make(chan lookupResult)
	var wg sync.WaitGroup
	workerCount := min(deleteBatchConcurrency, len(shortURLList))
	wg.Add(workerCount)

	for range workerCount {
		go func() {
			defer wg.Done()
			for shortURL := range jobs {
				data, err := s.storage.GetShortURLData(lookupCtx, shortURL)
				select {
				case results <- lookupResult{shortURL: shortURL, data: data, err: err}:
				case <-lookupCtx.Done():
					return
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, shortURL := range shortURLList {
			select {
			case jobs <- shortURL:
			case <-lookupCtx.Done():
				return
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
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
