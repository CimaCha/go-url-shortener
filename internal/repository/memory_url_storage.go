package repository

import (
	"context"
	"errors"
	"maps"
	"sync"

	"github.com/CimaCha/go-url-shortener/internal/model"
)

var (
	ErrURLHasGone     = errors.New("URL was been deleted")
	ErrURLNotFound    = errors.New("URL not found")
	ErrUserNotFound   = errors.New("user not found")
	ErrShortURLExists = errors.New("short URL already exists")
	ErrFullURLExists  = errors.New("full URL already exists")
)

type MemoryURLStorage struct {
	mu           sync.RWMutex
	urls         map[string]URLData
	backwardUrls map[string]string
	userMap      map[string][]*model.UserRecord
}

type URLData struct {
	UserID      string
	OriginalURL string
	DeletedFlag bool
}

func NewMemoryURLStorage(urls map[string]URLData) *MemoryURLStorage {
	if urls == nil {
		urls = make(map[string]URLData)
	}
	backwardUrls := arrangeMap(urls)
	userMap := setUserMap(urls)
	return &MemoryURLStorage{urls: urls, backwardUrls: backwardUrls, userMap: userMap}
}

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

func (s *MemoryURLStorage) FindFullURL(_ context.Context, shortURL string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fullURLData, ok := s.urls[shortURL]
	if !ok {
		return "", ErrURLNotFound
	}
	if fullURLData.DeletedFlag {
		return "", ErrURLHasGone
	}

	return fullURLData.OriginalURL, nil
}

func (s *MemoryURLStorage) Snapshot() map[string]URLData {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return maps.Clone(s.urls)
}

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

func (s *MemoryURLStorage) GetUserURLs(_ context.Context, userID string) ([]*model.UserRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pairs, ok := s.userMap[userID]
	if !ok {
		return nil, ErrUserNotFound
	}
	return pairs, nil
}

func (s *MemoryURLStorage) GetShortURLData(_ context.Context, shortURL string) (*model.StorageRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

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
