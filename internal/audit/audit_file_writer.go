package audit

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
)

var (
	// ErrOpenFileForWrite indicates that the audit file could not be opened.
	ErrOpenFileForWrite = errors.New("open file for write error")
)

// FileWriteSubscriber appends audit events to a file as JSON lines.
type FileWriteSubscriber struct {
	id   string
	mu   sync.Mutex
	file *os.File
}

// NewFileWriteSubscriber creates a file audit observer with the given ID.
func NewFileWriteSubscriber(id string, fileName string) (*FileWriteSubscriber, error) {
	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrOpenFileForWrite, err)
	}
	return &FileWriteSubscriber{id: id, file: file}, nil
}

func (s *FileWriteSubscriber) update(exposure Exposure) error {
	return s.writeAudit(exposure)
}

func (s *FileWriteSubscriber) getID() string {
	return s.id
}

func (s *FileWriteSubscriber) writeAudit(exposure Exposure) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file == nil {
		return os.ErrClosed
	}
	if err := json.NewEncoder(s.file).Encode(exposure); err != nil {
		return fmt.Errorf("encode exposure: %w", err)
	}
	return nil
}

// Close syncs and closes the audit file after all queued events are delivered.
func (s *FileWriteSubscriber) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file == nil {
		return nil
	}
	err := errors.Join(s.file.Sync(), s.file.Close())
	s.file = nil
	return err
}
