package repository

import (
	"errors"
	"sync"
)

var (
	ErrURLNotFound = errors.New("URL not found")
)

type MemoryURLStorage struct {
	mu   sync.RWMutex
	urls map[string]string
}

func NewMemoryURLStorage() *MemoryURLStorage {
	urls := map[string]string{}
	return &MemoryURLStorage{urls: urls}
}

func (s *MemoryURLStorage) SetShortURL(shortURL string, fullURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.urls[shortURL] = fullURL
	return nil
}

func (s *MemoryURLStorage) GetFullURL(shortURL string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fullURL, ok := s.urls[shortURL]
	if !ok {
		return "", ErrURLNotFound
	}

	return fullURL, nil
}
