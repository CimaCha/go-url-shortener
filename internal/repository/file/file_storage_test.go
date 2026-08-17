package file

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/CimaCha/go-url-shortener/internal/model"
	"github.com/CimaCha/go-url-shortener/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewFileStorage(t *testing.T) {
	tests := []struct {
		name       string
		filename   func(*testing.T) string
		content    []*model.FileRecord
		rawContent string
		want       map[string]string
		wantErr    error
	}{
		{
			name:    "returns open error for missing file",
			wantErr: ErrOpenFile,
		},
		{
			name: "loads records",
			content: []*model.FileRecord{
				{UUID: "1", ShortURL: "first", OriginalURL: "https://first.example"},
				{UUID: "4", ShortURL: "second", OriginalURL: "https://second.example"},
			},
			want: map[string]string{
				"first":  "https://first.example",
				"second": "https://second.example",
			},
		},
		{
			name:       "returns read error for malformed JSON",
			rawContent: "not-json\n",
			wantErr:    ErrDecodeRecords,
		},
		{
			name: "returns open error for missing parent directory",
			filename: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "missing", "storage.json")
			},
			wantErr: ErrOpenFile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := filepath.Join(t.TempDir(), "storage.json")
			if tt.filename != nil {
				filename = tt.filename(t)
			}
			if tt.content != nil {
				writeRecords(t, filename, tt.content)
			} else if tt.rawContent != "" {
				require.NoError(t, os.WriteFile(filename, []byte(tt.rawContent), 0o600))
			}

			storage, err := NewFileStorage(zap.NewNop(), filename)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, storage)
				return
			}
			require.NoError(t, err)
			for shortURL, wantFullURL := range tt.want {
				got, getErr := storage.GetFullURL(shortURL)
				require.NoError(t, getErr)
				assert.Equal(t, wantFullURL, got)
			}
			_, getErr := storage.GetFullURL("missing")
			assert.ErrorIs(t, getErr, repository.ErrURLNotFound)
		})
	}
}

func TestStorageSetShortURL(t *testing.T) {
	tests := []struct {
		name       string
		writes     [][2]string
		wantSetErr error
		wantURLs   map[string]string
	}{
		{
			name: "persists successive writes",
			writes: [][2]string{
				{"first", "https://first.example"},
				{"second", "https://second.example"},
			},
			wantURLs: map[string]string{
				"first":  "https://first.example",
				"second": "https://second.example",
			},
		},
		{
			name: "rejects duplicate without changing persisted records",
			writes: [][2]string{
				{"short", "https://first.example"},
				{"short", "https://second.example"},
			},
			wantSetErr: repository.ErrShortURLExists,
			wantURLs: map[string]string{
				"short": "https://first.example",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := filepath.Join(t.TempDir(), "storage.json")
			require.NoError(t, os.WriteFile(filename, nil, 0o600))
			storage, err := NewFileStorage(zap.NewNop(), filename)
			require.NoError(t, err)

			var setErr error
			for _, write := range tt.writes {
				setErr = storage.SetShortURL(write[0], write[1])
			}

			assert.ErrorIs(t, setErr, tt.wantSetErr)
			snapshots := readSnapshots(t, filename)
			require.Len(t, snapshots, 1)
			gotURLs := make(map[string]string, len(snapshots[0]))
			for _, record := range snapshots[0] {
				gotURLs[record.ShortURL] = record.OriginalURL
			}
			assert.Equal(t, tt.wantURLs, gotURLs)
			assertJSONLines(t, filename, 1)
		})
	}
}

func TestStorageSetShortURLWriteFailure(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "write error keeps added record in memory"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			filename := filepath.Join(root, "storage.json")
			require.NoError(t, os.WriteFile(filename, nil, 0o600))
			storage, err := NewFileStorage(zap.NewNop(), filename)
			require.NoError(t, err)
			storage.writer = NewWriter(zap.NewNop(), filepath.Join(root, "missing", "storage.json"))

			err = storage.SetShortURL("short", "https://example.com")

			assert.ErrorIs(t, err, ErrOpenFileForWrite)
			got, getErr := storage.GetFullURL("short")
			require.NoError(t, getErr)
			assert.Equal(t, "https://example.com", got)
		})
	}
}

func TestStorageConcurrentSetSnapshot(t *testing.T) {
	tests := []struct {
		name    string
		workers int
	}{
		{name: "last snapshot contains all concurrent writes", workers: 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := filepath.Join(t.TempDir(), "storage.json")
			require.NoError(t, os.WriteFile(filename, nil, 0o600))
			storage, err := NewFileStorage(zap.NewNop(), filename)
			require.NoError(t, err)

			start := make(chan struct{})
			errors := make(chan error, tt.workers)
			var wg sync.WaitGroup
			for worker := range tt.workers {
				wg.Add(1)
				go func() {
					defer wg.Done()
					<-start
					shortURL := fmt.Sprintf("short-%d", worker)
					errors <- storage.SetShortURL(shortURL, "https://example.com/"+shortURL)
				}()
			}
			close(start)
			wg.Wait()
			close(errors)
			for setErr := range errors {
				require.NoError(t, setErr)
			}

			snapshots := readSnapshots(t, filename)
			require.Len(t, snapshots, 1)
			lastSnapshot := snapshots[len(snapshots)-1]
			byShortURL := make(map[string]string, len(lastSnapshot))
			for _, record := range lastSnapshot {
				byShortURL[record.ShortURL] = record.OriginalURL
			}
			for worker := range tt.workers {
				shortURL := fmt.Sprintf("short-%d", worker)
				assert.Equal(t, "https://example.com/"+shortURL, byShortURL[shortURL])
			}
			assert.Len(t, lastSnapshot, tt.workers)
		})
	}
}

func TestStorageUsesMemoryAsSourceOfTruth(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "does not reread externally replaced file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := filepath.Join(t.TempDir(), "storage.json")
			writeRecords(t, filename, []*model.FileRecord{
				{UUID: "1", ShortURL: "first", OriginalURL: "https://first.example"},
			})
			storage, err := NewFileStorage(zap.NewNop(), filename)
			require.NoError(t, err)
			writeRecords(t, filename, []*model.FileRecord{
				{UUID: "99", ShortURL: "external", OriginalURL: "https://external.example"},
			})

			got, err := storage.GetFullURL("first")
			require.NoError(t, err)
			assert.Equal(t, "https://first.example", got)
			_, err = storage.GetFullURL("external")
			assert.ErrorIs(t, err, repository.ErrURLNotFound)
		})
	}
}

func writeRecords(t *testing.T, filename string, records []*model.FileRecord) {
	t.Helper()
	var content bytes.Buffer
	require.NoError(t, json.NewEncoder(&content).Encode(records))
	require.NoError(t, os.WriteFile(filename, content.Bytes(), 0o600))
}

func readRecords(t *testing.T, filename string) []*model.FileRecord {
	t.Helper()
	reader, err := NewReader(zap.NewNop(), filename)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, reader.Close()) })
	records, err := reader.ReadRecords()
	require.NoError(t, err)
	return records
}

func readSnapshots(t *testing.T, filename string) [][]*model.FileRecord {
	t.Helper()
	file, err := os.Open(filename)
	require.NoError(t, err)
	defer file.Close()

	decoder := json.NewDecoder(file)
	snapshots := make([][]*model.FileRecord, 0)
	for {
		var records []*model.FileRecord
		err = decoder.Decode(&records)
		if err == io.EOF {
			return snapshots
		}
		require.NoError(t, err)
		snapshots = append(snapshots, records)
	}
}

func assertJSONLines(t *testing.T, filename string, wantLines int) {
	t.Helper()
	file, err := os.Open(filename)
	require.NoError(t, err)
	defer file.Close()

	lineCount := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lineCount++
		assert.True(t, json.Valid(scanner.Bytes()), "line %d must be valid JSON", lineCount)
	}
	require.NoError(t, scanner.Err())
	assert.Equal(t, wantLines, lineCount)
}
