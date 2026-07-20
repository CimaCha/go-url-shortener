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
	storage repository.UrlStorage
}

func NewService(storage repository.UrlStorage) Service {
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
	err := s.storage.SetShortUrl(shortUrl, fullURL)
	if err != nil {
		return "", err
	}
	return shortUrl, nil
}

func (s Service) GetFullUrl(shortUrl string) (string, error) {
	if shortUrl == "" {
		return "", ErrEmptyURL
	}
	fullURL, err := s.storage.GetFullUrl(shortUrl)
	if err != nil {
		return "", err
	}
	return fmt.Sprint(fullURL), nil
}
