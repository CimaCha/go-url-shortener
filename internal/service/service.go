package service

import (
	"crypto/rand"
	"errors"
	"github.com/CimaCha/go-url-shortener/internal/repository"
	"go.uber.org/zap"
)

const maxShortURLAttempts = 5

var (
	ErrEmptyURL    = errors.New("empty URL")
	ErrURLNotFound = errors.New("URL not found")
	ErrRepository  = errors.New("error in repository")

	ErrUniqueShortURL = errors.New("can't create unique short URL")
)

type Service struct {
	logger  *zap.Logger
	storage URLStorage
}

func NewService(logger *zap.Logger, storage URLStorage) Service {
	return Service{
		logger:  logger,
		storage: storage,
	}
}

func (s Service) SetShortURL(fullURL string) (string, error) {
	if fullURL == "" {
		return "", ErrEmptyURL
	}
	for range maxShortURLAttempts {
		shortURL := rand.Text()
		err := s.storage.SetShortURL(shortURL, fullURL)
		if err == nil {
			return shortURL, nil
		}
		if !errors.Is(err, repository.ErrShortURLExists) {
			s.logger.Error("error set short url", zap.Error(err))
			return "", ErrRepository
		}
	}

	return "", ErrUniqueShortURL
}

func (s Service) GetFullURL(shortURL string) (string, error) {
	if shortURL == "" {
		return "", ErrEmptyURL
	}
	fullURL, err := s.storage.GetFullURL(shortURL)
	if err != nil {
		if errors.Is(err, repository.ErrURLNotFound) {
			s.logger.Error("URL not found by short URL", zap.String("short_url", shortURL))
			return "", ErrURLNotFound
		}
		s.logger.Error("error get full url", zap.Error(err))
		return "", ErrRepository
	}
	return fullURL, nil
}
