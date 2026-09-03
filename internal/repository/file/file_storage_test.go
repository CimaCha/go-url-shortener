package file

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/CimaCha/go-url-shortener/internal/model"
	"github.com/CimaCha/go-url-shortener/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			ctx := context.Background()
			filename := filepath.Join(t.TempDir(), "storage.json")
			if tt.filename != nil {
				filename = tt.filename(t)
			}
			if tt.content != nil {
				writeRecords(t, filename, tt.content)
			} else if tt.rawContent != "" {
				require.NoError(t, os.WriteFile(filename, []byte(tt.rawContent), 0o600))
			}

			storage, err := NewFileStorage(filename)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, storage)
				return
			}
			require.NoError(t, err)
			for shortURL, wantFullURL := range tt.want {
				got, getErr := storage.FindFullURL(ctx, shortURL)
				require.NoError(t, getErr)
				assert.Equal(t, wantFullURL, got)
			}
			_, getErr := storage.FindFullURL(ctx, "missing")
			assert.ErrorIs(t, getErr, repository.ErrURLNotFound)
		})
	}
}

func TestStorageSetShortURL(t *testing.T) {
	tests := []struct {
		name       string
		writes     [][2]string
		wantStored string
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
		{
			name: "returns existing short URL for duplicate full URL",
			writes: [][2]string{
				{"first", "https://example.com"},
				{"second", "https://example.com"},
			},
			wantStored: "first",
			wantSetErr: repository.ErrFullURLExists,
			wantURLs: map[string]string{
				"first": "https://example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			filename := filepath.Join(t.TempDir(), "storage.json")
			require.NoError(t, os.WriteFile(filename, nil, 0o600))
			storage, err := NewFileStorage(filename)
			require.NoError(t, err)

			var storedShortURL string
			var setErr error
			for _, write := range tt.writes {
				storedShortURL, setErr = storage.SaveShortURL(ctx, write[0], write[1], "")
			}

			assert.ErrorIs(t, setErr, tt.wantSetErr)
			assert.Equal(t, tt.wantStored, storedShortURL)
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
			ctx := context.Background()
			root := t.TempDir()
			filename := filepath.Join(root, "storage.json")
			require.NoError(t, os.WriteFile(filename, nil, 0o600))
			storage, err := NewFileStorage(filename)
			require.NoError(t, err)
			storage.writer = NewWriter(filepath.Join(root, "missing", "storage.json"))

			_, err = storage.SaveShortURL(ctx, "short", "https://example.com", "")

			assert.ErrorIs(t, err, ErrOpenFileForWrite)
			got, getErr := storage.FindFullURL(ctx, "short")
			require.NoError(t, getErr)
			assert.Equal(t, "https://example.com", got)
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
			ctx := context.Background()
			filename := filepath.Join(t.TempDir(), "storage.json")
			writeRecords(t, filename, []*model.FileRecord{
				{UUID: "1", ShortURL: "first", OriginalURL: "https://first.example"},
			})
			storage, err := NewFileStorage(filename)
			require.NoError(t, err)
			writeRecords(t, filename, []*model.FileRecord{
				{UUID: "99", ShortURL: "external", OriginalURL: "https://external.example"},
			})

			got, err := storage.FindFullURL(ctx, "first")
			require.NoError(t, err)
			assert.Equal(t, "https://first.example", got)
			_, err = storage.FindFullURL(ctx, "external")
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
