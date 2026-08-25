package file

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/CimaCha/go-url-shortener/internal/model"
	"github.com/CimaCha/go-url-shortener/internal/repository"
)

type Storage struct {
	memory *repository.MemoryURLStorage
	writer *Writer
	mu     sync.Mutex
}

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

	urls := make(map[string]string, len(records))
	for _, record := range records {
		urls[record.ShortURL] = record.OriginalURL
	}
	memory := repository.NewMemoryURLStorage(urls)

	return &Storage{
		memory: memory,
		writer: NewWriter(filePath),
	}, nil
}

func (f *Storage) SaveShortURL(ctx context.Context, shortURL string, fullURL string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := f.memory.SaveShortURL(ctx, shortURL, fullURL); err != nil {
		return err
	}

	urls := f.memory.Snapshot()
	records := make([]*model.FileRecord, 0, len(urls))
	uuid := 0
	for currentShortURL, currentFullURL := range urls {
		records = append(records, &model.FileRecord{
			UUID:        strconv.Itoa(uuid),
			ShortURL:    currentShortURL,
			OriginalURL: currentFullURL,
		})
		uuid++
	}

	if err := f.writer.WriteRecords(records); err != nil {
		return fmt.Errorf("persist short URL: %w", err)
	}
	return nil
}

func (f *Storage) FindFullURL(ctx context.Context, shortURL string) (string, error) {
	return f.memory.FindFullURL(ctx, shortURL)
}

func (f *Storage) SaveShortUrlBatch(ctx context.Context, URLRecords []*model.URLRecord) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := f.memory.SaveShortUrlBatch(ctx, URLRecords); err != nil {
		return err
	}

	urls := f.memory.Snapshot()
	records := make([]*model.FileRecord, 0, len(urls))
	uuid := 0
	for currentShortURL, currentFullURL := range urls {
		records = append(records, &model.FileRecord{
			UUID:        strconv.Itoa(uuid),
			ShortURL:    currentShortURL,
			OriginalURL: currentFullURL,
		})
		uuid++
	}

	if err := f.writer.WriteRecords(records); err != nil {
		return fmt.Errorf("persist short URL: %w", err)
	}
	return nil
}
