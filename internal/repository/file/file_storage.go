package file

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/CimaCha/go-url-shortener/internal/model"
	"github.com/CimaCha/go-url-shortener/internal/repository"
	"go.uber.org/zap"
)

type Storage struct {
	logger *zap.Logger
	memory *repository.MemoryURLStorage
	writer *Writer
	mu     sync.Mutex
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

	urls := make(map[string]string, len(records))
	for _, record := range records {
		urls[record.ShortURL] = record.OriginalURL
	}
	memory := repository.NewMemoryURLStorage(urls)

	return &Storage{
		logger: logger,
		memory: memory,
		writer: NewWriter(logger.With(zap.String("file worker", "writer")), filePath),
	}, nil
}

func (f *Storage) SetShortURL(shortURL string, fullURL string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := f.memory.SetShortURL(shortURL, fullURL); err != nil {
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
		f.logger.Error("can't persist short URL", zap.Error(err))
		return fmt.Errorf("persist short URL: %w", err)
	}
	return nil
}

func (f *Storage) GetFullURL(shortURL string) (string, error) {
	return f.memory.GetFullURL(shortURL)
}
