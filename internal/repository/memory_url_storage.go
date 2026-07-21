package repository

import (
	"errors"
	"sync"
)

var ErrURLNotFound = errors.New("URL not found")

type MemoryURLStorage struct {
	urls *sync.Map
}

func NewMemoryURLStorage() *MemoryURLStorage {
	urls := sync.Map{}
	return &MemoryURLStorage{urls: &urls}
}

func (s *MemoryURLStorage) SetShortURL(shortURL string, fullURL string) {
	s.urls.Store(shortURL, fullURL)
}

func (s *MemoryURLStorage) GetFullURL(shortURL string) (string, error) {
	fullURL, ok := s.urls.Load(shortURL)
	if !ok {
		return "", ErrURLNotFound
	}

	return fullURL.(string), nil
}
