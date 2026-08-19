package file

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/CimaCha/go-url-shortener/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriterWriteRecords(t *testing.T) {
	first := []*model.FileRecord{
		{UUID: "1", ShortURL: "first", OriginalURL: "https://first.example"},
	}
	second := []*model.FileRecord{
		{UUID: "2", ShortURL: "second", OriginalURL: "https://second.example"},
		{UUID: "3", ShortURL: "third", OriginalURL: "https://third.example"},
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

func TestWriterAtomicallyReplacesFile(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "storage.json")
	oldRecords := []*model.FileRecord{{UUID: "1", ShortURL: "old", OriginalURL: "https://old.example"}}
	oldContent, err := json.Marshal(oldRecords)
	if err != nil {
		t.Fatal(err)
	}
	oldContent = append(oldContent, '\n')
	if err = os.WriteFile(filename, oldContent, 0o600); err != nil {
		t.Fatal(err)
	}

	oldFile, err := os.Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer oldFile.Close()

	newRecords := []*model.FileRecord{{UUID: "2", ShortURL: "new", OriginalURL: "https://new.example"}}
	if err = NewWriter(filename).WriteRecords(newRecords); err != nil {
		t.Fatal(err)
	}

	gotOldContent, err := io.ReadAll(oldFile)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotOldContent, oldContent) {
		t.Fatalf("open file changed: got %q, want %q", gotOldContent, oldContent)
	}

	newContent, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	var gotNewRecords []*model.FileRecord
	if err = json.Unmarshal(newContent, &gotNewRecords); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotNewRecords, newRecords) {
		t.Fatalf("replacement contains %#v, want %#v", gotNewRecords, newRecords)
	}
}
