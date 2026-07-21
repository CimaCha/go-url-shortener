package service

import (
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/CimaCha/go-url-shortener/internal/repository"
)

var (
	ErrEmptyURL = errors.New("empty URL")
)

type Service struct {
	storage repository.URLStorage
}

func NewService(storage repository.URLStorage) Service {
	return Service{storage: storage}
}

func ShortenURL(fullURL string) string {
	digest := sha1.Sum([]byte(fullURL))
	shortURL := base64.RawURLEncoding.EncodeToString(digest[:])
	return shortURL
}

func (s Service) SetShortURL(fullURL string) (string, error) {
	if fullURL == "" {
		return "", ErrEmptyURL
	}
	shortURL := ShortenURL(fullURL)
	s.storage.SetShortURL(shortURL, fullURL)
	return shortURL, nil
}

func (s Service) GetFullURL(shortUrl string) (string, error) {
	if shortUrl == "" {
		return "", ErrEmptyURL
	}
	fullURL, err := s.storage.GetFullURL(shortUrl)
	if err != nil {
		return "", err
	}
	return fmt.Sprint(fullURL), nil
}
