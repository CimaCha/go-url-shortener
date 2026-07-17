package service

import (
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
)

var (
	ErrEmptyURL    = errors.New("empty URL")
	ErrURLNotFound = errors.New("URL not found")
)

type Service struct {
	storage *sync.Map
}

func NewService(storage *sync.Map) Service {
	return Service{storage: storage}
}

func ShortenUrl(fullURL string) string {
	digest := sha1.Sum([]byte(fullURL))
	shortURL := base64.RawURLEncoding.EncodeToString(digest[:])
	return shortURL
}

func (s Service) SetShortUrl(fullURL string) (string, error) {
	if fullURL == "" {
		return "", ErrEmptyURL
	}
	shortUrl := ShortenUrl(fullURL)
	s.storage.Store(shortUrl, fullURL)
	return shortUrl, nil
}

func (s Service) GetFullUrl(shortUrl string) (string, error) {
	if shortUrl == "" {
		return "", ErrEmptyURL
	}
	fullURL, exists := s.storage.Load(shortUrl)
	if !exists {
		return "", ErrURLNotFound
	}
	return fmt.Sprint(fullURL), nil
}
