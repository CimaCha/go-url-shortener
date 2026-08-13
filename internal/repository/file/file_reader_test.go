package file

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CimaCha/go-url-shortener/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReaderReadRecords(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []*model.FileRecord
		wantErr error
	}{
		{
			name: "reads JSON array",
			content: `[{"uuid":"1","short_url":"first","original_url":"https://first.example"},` +
				`{"uuid":"2","short_url":"second","original_url":"https://second.example"}]`,
			want: []*model.FileRecord{
				{UUID: "1", ShortUrl: "first", OriginalUrl: "https://first.example"},
				{UUID: "2", ShortUrl: "second", OriginalUrl: "https://second.example"},
			},
		},
		{
			name:    "reads empty file as empty slice",
			content: "",
			want:    []*model.FileRecord{},
		},
		{
			name:    "returns decode error for malformed JSON",
			content: "not-json",
			wantErr: ErrDecodeRecords,
		},
		{
			name:    "returns decode error for JSON object",
			content: `{"uuid":"1","short_url":"first"}`,
			wantErr: ErrDecodeRecords,
		},
		{
			name: "reads only first array from stream",
			content: `[{"uuid":"1","short_url":"first"}]` + "\n" +
				`[{"uuid":"2","short_url":"second"}]`,
			want: []*model.FileRecord{
				{UUID: "1", ShortUrl: "first"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := filepath.Join(t.TempDir(), "storage.json")
			require.NoError(t, os.WriteFile(filename, []byte(tt.content), 0o600))
			reader, err := NewReader(filename)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, reader.Close()) })

			got, err := reader.ReadRecords()

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.NotNil(t, got)
		})
	}
}

func TestNewReader(t *testing.T) {
	tests := []struct {
		name     string
		filename func(*testing.T) string
		wantErr  error
	}{
		{
			name: "creates missing file",
			filename: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "storage.json")
			},
		},
		{
			name: "returns open error for missing parent",
			filename: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "missing", "storage.json")
			},
			wantErr: ErrOpenFile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, err := NewReader(tt.filename(t))
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, reader)
				return
			}
			require.NoError(t, err)
			require.NoError(t, reader.Close())
		})
	}
}

func TestReaderClose(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "reports repeated close"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, err := NewReader(filepath.Join(t.TempDir(), "storage.json"))
			require.NoError(t, err)
			require.NoError(t, reader.Close())
			assert.Error(t, reader.Close())
		})
	}
}
