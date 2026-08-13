package file

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CimaCha/go-url-shortener/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriterWriteRecords(t *testing.T) {
	first := []*model.FileRecord{
		{UUID: "1", ShortUrl: "first", OriginalUrl: "https://first.example"},
	}
	second := []*model.FileRecord{
		{UUID: "2", ShortUrl: "second", OriginalUrl: "https://second.example"},
		{UUID: "3", ShortUrl: "third", OriginalUrl: "https://third.example"},
	}
	tests := []struct {
		name     string
		filename func(*testing.T) string
		create   bool
		initial  []*model.FileRecord
		records  []*model.FileRecord
		want     [][]*model.FileRecord
		wantErr  error
	}{
		{
			name: "writes array to empty existing file",
			filename: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "storage.json")
			},
			create:  true,
			records: first,
			want:    [][]*model.FileRecord{first},
		},
		{
			name: "replaces existing array",
			filename: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "storage.json")
			},
			initial: first,
			records: second,
			want:    [][]*model.FileRecord{second},
		},
		{
			name: "returns open error for missing file",
			filename: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "storage.json")
			},
			records: first,
			wantErr: ErrOpenFileForWrite,
		},
		{
			name: "returns open error for missing parent",
			filename: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "missing", "storage.json")
			},
			records: first,
			wantErr: ErrOpenFileForWrite,
		},
		{
			name: "rejects nil records",
			filename: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "storage.json")
			},
			wantErr: ErrWriteNullRecords,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := tt.filename(t)
			if tt.initial != nil {
				writeRecords(t, filename, tt.initial)
			} else if tt.create {
				require.NoError(t, os.WriteFile(filename, nil, 0o600))
			}
			writer := NewWriter(filename)

			err := writer.WriteRecords(tt.records)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, readSnapshots(t, filename))
			assertJSONLines(t, filename, len(tt.want))
		})
	}
}
