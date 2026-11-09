package file

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"sync"

	"github.com/CimaCha/go-url-shortener/internal/model"
	"github.com/CimaCha/go-url-shortener/internal/repository"
)

// Storage persists URL mappings in a file-backed snapshot.
type Storage struct {
	memory *repository.MemoryURLStorage
	writer *Writer
	mu     sync.Mutex
}

// NewFileStorage loads the snapshot at filePath into memory.
func NewFileStorage(filePath string) (*Storage, error) {
	reader, err := NewReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open storage reader: %w", err)
	}
	records, err := reader.ReadRecords()
	closeErr := reader.Close()
	if err != nil {
		return nil, fmt.Errorf("read storage: %w", err)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close storage reader: %w", closeErr)
	}

	urls := make(map[string]repository.URLData, len(records))
	for _, record := range records {
		urls[record.ShortURL] = repository.URLData{UserID: record.UserID, OriginalURL: record.OriginalURL, DeletedFlag: record.DeletedFlag}
	}
	memory := repository.NewMemoryURLStorage(urls)

	return &Storage{
		memory: memory,
		writer: NewWriter(filePath),
	}, nil
}

// SaveShortURL stores one mapping and persists the new snapshot.
func (f *Storage) SaveShortURL(ctx context.Context, shortURL, fullURL, userID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	candidate := repository.NewMemoryURLStorage(f.memory.Snapshot())
	if storedShortURL, err := candidate.SaveShortURL(ctx, shortURL, fullURL, userID); err != nil {
		return storedShortURL, err
	}
	if err := f.persist(candidate); err != nil {
		return "", fmt.Errorf("persist short URL: %w", err)
	}
	f.memory = candidate
	return "", nil
}

// FindFullURL resolves a short ID from the loaded snapshot.
func (f *Storage) FindFullURL(ctx context.Context, shortURL string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.memory.FindFullURL(ctx, shortURL)
}

// SaveShortURLBatch stores all mappings and persists one new snapshot.
func (f *Storage) SaveShortURLBatch(ctx context.Context, URLRecords []*model.URLRecord, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	candidate := repository.NewMemoryURLStorage(f.memory.Snapshot())
	if err := candidate.SaveShortURLBatch(ctx, URLRecords, userID); err != nil {
		return err
	}
	if err := f.persist(candidate); err != nil {
		return fmt.Errorf("persist short URL: %w", err)
	}
	f.memory = candidate
	return nil
}

// GetUserURLs returns the mappings belonging to a user.
func (f *Storage) GetUserURLs(ctx context.Context, userID string) ([]*model.UserRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.memory.GetUserURLs(ctx, userID)
}

// GetShortURLData returns ownership and deletion metadata for a short ID.
func (f *Storage) GetShortURLData(ctx context.Context, shortURL string) (*model.StorageRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.memory.GetShortURLData(ctx, shortURL)
}

// DeleteURLsBatch marks the user's URLs as deleted and persists the snapshot.
func (f *Storage) DeleteURLsBatch(ctx context.Context, shortURLs []string, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	candidate := repository.NewMemoryURLStorage(f.memory.Snapshot())
	if err := candidate.DeleteURLsBatch(ctx, shortURLs, userID); err != nil {
		return err
	}
	if err := f.persist(candidate); err != nil {
		return fmt.Errorf("persist deleted URLs: %w", err)
	}
	f.memory = candidate
	return nil
}

func (f *Storage) persist(memory *repository.MemoryURLStorage) error {
	urls := memory.Snapshot()
	shortURLs := slices.Sorted(maps.Keys(urls))
	records := make([]*model.FileRecord, 0, len(urls))
	for uuid, shortURL := range shortURLs {
		data := urls[shortURL]
		records = append(records, &model.FileRecord{
			UUID:        strconv.Itoa(uuid),
			ShortURL:    shortURL,
			OriginalURL: data.OriginalURL,
			UserID:      data.UserID,
			DeletedFlag: data.DeletedFlag,
		})
	}
	return f.writer.WriteRecords(records)
}
