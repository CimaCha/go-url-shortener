package file

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/CimaCha/go-url-shortener/internal/model"
	"go.uber.org/zap"
)

var (
	ErrWriteNullRecords = errors.New("write null records")
	ErrOpenFileForWrite = errors.New("open file for write error")
)

type Writer struct {
	logger   *zap.Logger
	filename string
}

func NewWriter(log *zap.Logger, filename string) *Writer {
	return &Writer{
		logger:   log,
		filename: filename,
	}
}

func (w *Writer) WriteRecords(records []*model.FileRecord) error {
	if records == nil {
		w.logger.Error("write null records")
		return ErrWriteNullRecords
	}

	info, err := os.Stat(w.filename)
	if err != nil {
		w.logger.Error("open file for write", zap.Error(err))
		return ErrOpenFileForWrite
	}
	file, err := os.CreateTemp(filepath.Dir(w.filename), ".storage-*.tmp")
	if err != nil {
		w.logger.Error("open file for write", zap.Error(err))
		return ErrOpenFileForWrite
	}
	temporaryName := file.Name()
	defer func() {
		_ = file.Close()
		_ = os.Remove(temporaryName)
	}()
	if err = file.Chmod(info.Mode().Perm()); err != nil {
		return fmt.Errorf("set temporary file mode: %w", err)
	}

	if err = json.NewEncoder(file).Encode(records); err != nil {
		w.logger.Error("encode record", zap.Error(err))
		return fmt.Errorf("encode record: %w", err)
	}
	if err = file.Sync(); err != nil {
		w.logger.Error("sync file", zap.Error(err))
		return fmt.Errorf("sync file: %w", err)
	}
	if err = file.Close(); err != nil {
		w.logger.Error("close file", zap.Error(err))
		return fmt.Errorf("close file: %w", err)
	}
	if err = os.Rename(temporaryName, w.filename); err != nil {
		w.logger.Error("rename file", zap.Error(err))
		return fmt.Errorf("rename file: %w", err)
	}
	return nil
}
