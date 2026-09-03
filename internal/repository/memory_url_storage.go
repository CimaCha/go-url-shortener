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
	ErrUserNotFound   = errors.New("user not found")
	ErrShortURLExists = errors.New("short URL already exists")
	ErrFullURLExists  = errors.New("full URL already exists")
)

type MemoryURLStorage struct {
	mu           sync.RWMutex
	urls         map[string]UserPair
	backwardUrls map[string]string
	userMap      map[string][]*model.UserRecord
}

type UserPair struct {
	UserID      string
	OriginalURL string
}

func NewMemoryURLStorage(urls map[string]UserPair) *MemoryURLStorage {
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

	s.urls[shortURL] = UserPair{UserID: userID, OriginalURL: fullURL}
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

	fullURLPair, ok := s.urls[shortURL]
	if !ok {
		return "", ErrURLNotFound
	}

	return fullURLPair.OriginalURL, nil
}

func (s *MemoryURLStorage) Snapshot() map[string]UserPair {
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
		urls[record.ShortURL] = UserPair{UserID: userID, OriginalURL: record.OriginalURL}

		_, ok = backwardURLs[record.OriginalURL]
		if ok {
			return ErrFullURLExists
		}
		backwardURLs[record.OriginalURL] = record.ShortURL

		userURLList, ok := s.userMap[userID]
		if !ok {
			s.userMap[userID] = []*model.UserRecord{{ShortURL: record.ShortURL, OriginalURL: record.OriginalURL}}
		} else {
			s.userMap[userID] = append(userURLList, &model.UserRecord{ShortURL: record.ShortURL, OriginalURL: record.OriginalURL})
		}
	}
	s.urls = urls
	s.backwardUrls = backwardURLs
	return nil
}

func (s *MemoryURLStorage) GetUserURLs(_ context.Context, userID string) ([]*model.UserRecord, error) {
	pairs, ok := s.userMap[userID]
	if !ok {
		return nil, ErrUserNotFound
	}
	return pairs, nil
}

func arrangeMap(oldMap map[string]UserPair) map[string]string {
	newMap := make(map[string]string)
	for k, v := range oldMap {
		newMap[v.OriginalURL] = k
	}
	return newMap
}

func setUserMap(oldMap map[string]UserPair) map[string][]*model.UserRecord {
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
