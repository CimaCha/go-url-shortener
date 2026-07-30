package service

import (
	"errors"
	"github.com/CimaCha/go-url-shortener/internal/repository"
	"strconv"
	"time"
)

var (
	ErrEmptyURL    = errors.New("empty URL")
	ErrURLNotFound = errors.New("URL not found")
	ErrRepository  = errors.New("error in repository")
)

type Service struct {
	storage URLStorage
}

func NewService(storage URLStorage) Service {
	return Service{storage: storage}
}

func generateShortUrl() string {
	timeStamp := time.Now().UnixNano()
	shortURL := strconv.FormatInt(timeStamp, 36)
	return shortURL
}

func (s Service) SetShortURL(fullURL string) (string, error) {
	if fullURL == "" {
		return "", ErrEmptyURL
	}
	shortURL := generateShortUrl()
	err := s.storage.SetShortURL(shortURL, fullURL)
	if err != nil {
		return "", ErrRepository
	}
	return shortURL, nil
}

func (s Service) GetFullURL(shortUrl string) (string, error) {
	if shortUrl == "" {
		return "", ErrEmptyURL
	}
	fullURL, err := s.storage.GetFullURL(shortUrl)
	if err != nil {
		if errors.Is(err, repository.ErrURLNotFound) {
			return "", ErrURLNotFound
		}
		return "", ErrRepository
	}
	return fullURL, nil
}
