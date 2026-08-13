package file

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/CimaCha/go-url-shortener/internal/logger"
	"github.com/CimaCha/go-url-shortener/internal/model"
	"go.uber.org/zap"
	"os"
)

var (
	ErrWriteNullRecords = errors.New("write null records")
	ErrOpenFileForWrite = errors.New("open file for write error")
)

type Writer struct {
	filename string
}

func NewWriter(filename string) *Writer {
	return &Writer{filename: filename}
}

func (w *Writer) WriteRecords(records []*model.FileRecord) (returnErr error) {
	if records == nil {
		logger.Log.Error("write null records")
		return ErrWriteNullRecords
	}

	file, err := os.OpenFile(w.filename, os.O_WRONLY|os.O_APPEND|os.O_TRUNC, 0o666)
	if err != nil {
		logger.Log.Error("open file for write", zap.Error(err))
		return ErrOpenFileForWrite
	}
	defer func() {
		if closeErr := file.Close(); returnErr == nil && closeErr != nil {
			returnErr = fmt.Errorf("close file: %w", closeErr)
		}
	}()

	if err = json.NewEncoder(file).Encode(records); err != nil {
		logger.Log.Error("encode record", zap.Error(err))
		return fmt.Errorf("encode record: %w", err)
	}
	if err = file.Sync(); err != nil {
		logger.Log.Error("sync file", zap.Error(err))
		return fmt.Errorf("sync file: %w", err)
	}
	return nil
}
