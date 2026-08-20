package service

import (
	"crypto/rand"
	"errors"

	"github.com/CimaCha/go-url-shortener/internal/repository"
)

const maxShortURLAttempts = 5

var (
	ErrEmptyURL    = errors.New("empty URL")
	ErrURLNotFound = errors.New("URL not found")
	ErrRepository  = errors.New("error in repository")

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
			return "", ErrURLNotFound
		}
		return "", ErrRepository
	}
	return fullURL, nil
}
