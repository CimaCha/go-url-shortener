package file

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"maps"
	"slices"
	"strconv"
	"sync"

	"github.com/CimaCha/go-url-shortener/internal/model"
)

var (
	ErrURLNotFound    = errors.New("URL not found")
	ErrShortURLExists = errors.New("short URL already exists")
)

type Storage struct {
	logger   *zap.Logger
	writer   *Writer
	mu       sync.RWMutex
	urls     map[string]*model.FileRecord
	nextUUID int
}

func NewFileStorage(logger *zap.Logger, filePath string) (*Storage, error) {
	reader, err := NewReader(logger.With(zap.String("file worker", "reader")), filePath)
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

	urls := make(map[string]*model.FileRecord, len(records))
	nextUUID := 1
	for _, record := range records {
		urls[record.ShortURL] = record
		if uuid, parseErr := strconv.Atoi(record.UUID); parseErr == nil && uuid >= nextUUID {
			nextUUID = uuid + 1
		}
	}

	return &Storage{
		logger:   logger,
		writer:   NewWriter(logger.With(zap.String("file worker", "writer")), filePath),
		urls:     urls,
		nextUUID: nextUUID,
	}, nil
}

func (f *Storage) SetShortURL(shortURL string, fullURL string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.urls[shortURL]; exists {
		return ErrShortURLExists
	}

	record := model.FileRecord{
		UUID:        strconv.Itoa(f.nextUUID),
		ShortURL:    shortURL,
		OriginalURL: fullURL,
	}

	f.urls[shortURL] = &record
	if err := f.writer.WriteRecords(slices.Collect(maps.Values(f.urls))); err != nil {
		return fmt.Errorf("persist short URL: %w", err)
	}
	f.nextUUID++
	return nil
}

func (f *Storage) GetFullURL(shortURL string) (string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	record, exists := f.urls[shortURL]
	if !exists {
		return "", ErrURLNotFound
	}
	return record.OriginalURL, nil
}
