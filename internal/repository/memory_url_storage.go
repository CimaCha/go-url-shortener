package repository

import (
	"context"
	"errors"
	"maps"
	"sync"

	"github.com/CimaCha/go-url-shortener/internal/model"
)

var (
	ErrURLNotFound    = errors.New("URL not found")
	ErrShortURLExists = errors.New("short URL already exists")
	ErrFullURLExists  = errors.New("full URL already exists")
)

type MemoryURLStorage struct {
	mu           sync.RWMutex
	urls         map[string]string
	backwardUrls map[string]string
}

func NewMemoryURLStorage(urls map[string]string) *MemoryURLStorage {
	backwardUrls := arrangeMap(urls)
	return &MemoryURLStorage{urls: urls, backwardUrls: backwardUrls}
}

func (s *MemoryURLStorage) SaveShortURL(_ context.Context, shortURL, fullURL string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.urls[shortURL]
	if ok {
		return "", ErrShortURLExists
	}

	storedShortURL, ok := s.backwardUrls[fullURL]
	if ok {
		return storedShortURL, ErrFullURLExists
	}

	s.urls[shortURL] = fullURL
	s.backwardUrls[fullURL] = shortURL
	return "", nil
}

func (s *MemoryURLStorage) FindFullURL(_ context.Context, shortURL string) (string, error) {
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

func (s *MemoryURLStorage) SaveShortUrlBatch(_ context.Context, URLRecords []*model.URLRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	urls := maps.Clone(s.urls)
	backwardURLs := maps.Clone(s.backwardUrls)
	for _, record := range URLRecords {
		_, ok := urls[record.ShortURL]
		if ok {
			return ErrShortURLExists
		}
		urls[record.ShortURL] = record.OriginalURL

		_, ok = backwardURLs[record.OriginalURL]
		if ok {
			return ErrFullURLExists
		}
		backwardURLs[record.OriginalURL] = record.ShortURL
	}
	s.urls = urls
	s.backwardUrls = backwardURLs
	return nil
}

func arrangeMap(oldMap map[string]string) map[string]string {
	newMap := make(map[string]string)
	for k, v := range oldMap {
		newMap[v] = k
	}
	return newMap
}
