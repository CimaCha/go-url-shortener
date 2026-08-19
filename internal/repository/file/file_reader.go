package file

import (
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/CimaCha/go-url-shortener/internal/model"
)

var (
	ErrOpenFile      = errors.New("error open file")
	ErrDecodeRecords = errors.New("error decode json records")
)

type Reader struct {
	file    *os.File
	decoder *json.Decoder
}

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

func (r *Reader) ReadRecords() ([]*model.FileRecord, error) {
	records := make([]*model.FileRecord, 0)

	err := r.decoder.Decode(&records)
	if err == io.EOF {
		return records, nil
	}
	if err != nil {
		return nil, ErrDecodeRecords
	}
	return records, nil
}

func (r *Reader) Close() error {
	return r.file.Close()
}
