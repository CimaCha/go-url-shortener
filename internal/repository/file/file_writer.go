package file

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/CimaCha/go-url-shortener/internal/model"
)

var (
	// ErrWriteNullRecords indicates a nil snapshot.
	ErrWriteNullRecords = errors.New("write null records")
	// ErrOpenFileForWrite indicates that the snapshot cannot be written.
	ErrOpenFileForWrite = errors.New("open file for write error")
)

// Writer atomically replaces a URL snapshot file.
type Writer struct {
	filename string
}

// NewWriter creates a snapshot writer for filename.
func NewWriter(filename string) *Writer {
	return &Writer{
		filename: filename,
	}
}

// WriteRecords writes a JSON snapshot through a temporary file and rename.
func (w *Writer) WriteRecords(records []*model.FileRecord) error {
	if records == nil {
		return ErrWriteNullRecords
	}

	info, err := os.Stat(w.filename)
	if err != nil {
		return ErrOpenFileForWrite
	}
	file, err := os.CreateTemp(filepath.Dir(w.filename), ".storage-*.tmp")
	if err != nil {
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
		return fmt.Errorf("encode record: %w", err)
	}
	if err = file.Sync(); err != nil {
		return fmt.Errorf("sync file: %w", err)
	}
	if err = file.Close(); err != nil {
		return fmt.Errorf("close file: %w", err)
	}
	if err = os.Rename(temporaryName, w.filename); err != nil {
		return fmt.Errorf("rename file: %w", err)
	}
	return nil
}
