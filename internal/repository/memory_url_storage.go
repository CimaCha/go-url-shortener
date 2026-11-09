package repository

import (
	"context"
	"errors"
	"maps"
	"sync"

	"github.com/CimaCha/go-url-shortener/internal/model"
)

var (
	// ErrURLHasGone indicates that the URL was deleted.
	ErrURLHasGone = errors.New("URL was been deleted")
	// ErrURLNotFound indicates that the short ID is absent.
	ErrURLNotFound = errors.New("URL not found")
	// ErrUserNotFound indicates that the user has no stored URLs.
	ErrUserNotFound = errors.New("user not found")
	// ErrShortURLExists indicates a duplicate short ID.
	ErrShortURLExists = errors.New("short URL already exists")
	// ErrFullURLExists indicates a duplicate original URL.
	ErrFullURLExists = errors.New("full URL already exists")
)

// MemoryURLStorage stores URL mappings and ownership in memory.
type MemoryURLStorage struct {
	mu           sync.Mutex
	urls         map[string]URLData
	backwardUrls map[string]string
	userMap      map[string][]*model.UserRecord
}

// URLData contains the owner, original URL, and deletion state of a short ID.
type URLData struct {
	UserID      string
	OriginalURL string
	DeletedFlag bool
}

// NewMemoryURLStorage initializes storage from a map of short IDs to URL data.
func NewMemoryURLStorage(urls map[string]URLData) *MemoryURLStorage {
	if urls == nil {
		urls = make(map[string]URLData)
	}
	backwardUrls := arrangeMap(urls)
	userMap := setUserMap(urls)
	return &MemoryURLStorage{urls: urls, backwardUrls: backwardUrls, userMap: userMap}
}

// SaveShortURL stores a mapping or reports a duplicate short or original URL.
func (s *MemoryURLStorage) SaveShortURL(_ context.Context, shortURL, fullURL, userID string) (string, error) {
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

	s.urls[shortURL] = URLData{UserID: userID, OriginalURL: fullURL}
	s.backwardUrls[fullURL] = shortURL
	userURLList, ok := s.userMap[userID]
	if !ok {
		s.userMap[userID] = []*model.UserRecord{{ShortURL: shortURL, OriginalURL: fullURL}}
	} else {
		s.userMap[userID] = append(userURLList, &model.UserRecord{ShortURL: shortURL, OriginalURL: fullURL})
	}
	return "", nil
}

// FindFullURL returns the original URL unless the short ID is absent or deleted.
func (s *MemoryURLStorage) FindFullURL(_ context.Context, shortURL string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fullURLData, ok := s.urls[shortURL]
	if !ok {
		return "", ErrURLNotFound
	}
	if fullURLData.DeletedFlag {
		return "", ErrURLHasGone
	}

	return fullURLData.OriginalURL, nil
}

// Snapshot returns a copy of the current short ID mappings.
func (s *MemoryURLStorage) Snapshot() map[string]URLData {
	s.mu.Lock()
	defer s.mu.Unlock()

	return maps.Clone(s.urls)
}

// SaveShortURLBatch stores all mappings atomically or returns a duplicate URL error.
func (s *MemoryURLStorage) SaveShortURLBatch(_ context.Context, URLRecords []*model.URLRecord, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	urls := maps.Clone(s.urls)
	backwardURLs := maps.Clone(s.backwardUrls)
	for _, record := range URLRecords {
		_, ok := urls[record.ShortURL]
		if ok {
			return ErrShortURLExists
		}
		urls[record.ShortURL] = URLData{UserID: userID, OriginalURL: record.OriginalURL}

		_, ok = backwardURLs[record.OriginalURL]
		if ok {
			return ErrFullURLExists
		}
		backwardURLs[record.OriginalURL] = record.ShortURL

	}
	s.urls = urls
	s.backwardUrls = backwardURLs
	s.userMap = setUserMap(urls)
	return nil
}

// GetUserURLs returns the mappings belonging to a user.
func (s *MemoryURLStorage) GetUserURLs(_ context.Context, userID string) ([]*model.UserRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pairs, ok := s.userMap[userID]
	if !ok {
		return nil, ErrUserNotFound
	}
	return pairs, nil
}

// GetShortURLData returns ownership and deletion metadata for a short ID.
func (s *MemoryURLStorage) GetShortURLData(_ context.Context, shortURL string) (*model.StorageRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, ok := s.urls[shortURL]
	if !ok {
		return nil, ErrURLNotFound
	}
	return &model.StorageRecord{
		UserID:      data.UserID,
		OriginalURL: data.OriginalURL,
		DeletedFlag: data.DeletedFlag,
	}, nil
}

// DeleteURLsBatch marks only the user's matching short IDs as deleted.
func (s *MemoryURLStorage) DeleteURLsBatch(_ context.Context, shortURLs []string, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, shortURL := range shortURLs {
		data, ok := s.urls[shortURL]
		if !ok || data.UserID != userID {
			continue
		}
		data.DeletedFlag = true
		s.urls[shortURL] = data
	}
	return nil
}

func arrangeMap(oldMap map[string]URLData) map[string]string {
	newMap := make(map[string]string)
	for k, v := range oldMap {
		newMap[v.OriginalURL] = k
	}
	return newMap
}

func setUserMap(oldMap map[string]URLData) map[string][]*model.UserRecord {
	userMap := make(map[string][]*model.UserRecord)
	for k, v := range oldMap {
		userID, ok := userMap[v.UserID]
		if !ok {
			userMap[v.UserID] = []*model.UserRecord{{ShortURL: k, OriginalURL: v.OriginalURL}}
		} else {
			userMap[v.UserID] = append(userID, &model.UserRecord{ShortURL: k, OriginalURL: v.OriginalURL})
		}
	}
	return userMap
}
