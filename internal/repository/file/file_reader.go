package file

import (
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/CimaCha/go-url-shortener/internal/model"
)

var (
	// ErrOpenFile indicates that the snapshot file could not be opened.
	ErrOpenFile = errors.New("error open file")
	// ErrDecodeRecords indicates invalid snapshot JSON.
	ErrDecodeRecords = errors.New("error decode json records")
)

// Reader decodes the URL snapshot from a file.
type Reader struct {
	file    *os.File
	decoder *json.Decoder
}

// NewReader opens or creates a URL snapshot file.
func NewReader(filename string) (*Reader, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, ErrOpenFile
	}
	return &Reader{
		file:    file,
		decoder: json.NewDecoder(file),
	}, nil
}

// ReadRecords decodes a single JSON array of file records.
func (r *Reader) ReadRecords() ([]*model.FileRecord, error) {
	records := make([]*model.FileRecord, 0)

	err := r.decoder.Decode(&records)
	if err == io.EOF {
		return records, nil
	}
	if err != nil {
		return nil, ErrDecodeRecords
	}
	if records == nil {
		return nil, ErrDecodeRecords
	}
	for _, record := range records {
		if record == nil {
			return nil, ErrDecodeRecords
		}
	}
	if err := r.decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, ErrDecodeRecords
	}
	return records, nil
}

// Close closes the snapshot file.
func (r *Reader) Close() error {
	return r.file.Close()
}
