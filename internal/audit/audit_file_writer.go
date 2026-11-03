package audit

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

var (
	ErrOpenFileForWrite = errors.New("open file for write error")
)

type FileWriteSubscriber struct {
	id       string
	fileName string
}

func NewFileWriteSubscriber(id string, fileName string) *FileWriteSubscriber {
	return &FileWriteSubscriber{
		id:       id,
		fileName: fileName,
	}
}

func (s *FileWriteSubscriber) update(exposure Exposure) error {
	return s.writeAudit(exposure)
}

func (s *FileWriteSubscriber) getID() string {
	return s.id
}

func (s *FileWriteSubscriber) writeAudit(exposure Exposure) error {
	file, err := os.OpenFile(s.fileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrOpenFileForWrite, err)
	}
	defer file.Close()

	if err = json.NewEncoder(file).Encode(exposure); err != nil {
		return fmt.Errorf("encode exposure: %w", err)
	}
	if err = file.Sync(); err != nil {
		return fmt.Errorf("sync file: %w", err)
	}
	if err = file.Close(); err != nil {
		return fmt.Errorf("close file: %w", err)
	}
	return nil
}
