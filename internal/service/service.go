package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/CimaCha/go-url-shortener/internal/model"
	"github.com/CimaCha/go-url-shortener/internal/repository"
)

const maxShortURLAttempts = 5

var (
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
	return Service{
		storage: storage,
	}
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
		return "", ErrRepository
	}
	return fullURL, nil
}

func (s Service) ShortenBatch(ctx context.Context, fullURLBatch []*model.OriginalURLRecord, userID string) ([]*model.ShortURLRecord, error) {
	if len(fullURLBatch) == 0 {
		return nil, ErrEmptyURLList
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
