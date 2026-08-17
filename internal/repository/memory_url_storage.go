package repository

import (
	"errors"
	"maps"
	"sync"
)

var (
	ErrURLNotFound    = errors.New("URL not found")
	ErrShortURLExists = errors.New("short URL already exists")
)

type MemoryURLStorage struct {
	mu   sync.RWMutex
	urls map[string]string
}

func NewMemoryURLStorage(urls map[string]string) *MemoryURLStorage {
	return &MemoryURLStorage{urls: urls}
}

func (s *MemoryURLStorage) SetShortURL(shortURL string, fullURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.urls[shortURL]
	if ok {
		return ErrShortURLExists
	}
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

func (s *MemoryURLStorage) Snapshot() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return maps.Clone(s.urls)
}
