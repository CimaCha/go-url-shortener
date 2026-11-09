package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFileWriteSubscriberPreservesEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.json")
	subscriber := NewFileWriteSubscriber("file", path)
	want := []Exposure{
		{Timestamp: 1, Action: Shorten, UserID: "user", URL: "https://example.com/first"},
		{Timestamp: 2, Action: Follow, UserID: "user", URL: "https://example.com/second"},
	}
	for _, event := range want {
		require.NoError(t, subscriber.update(event))
	}

	file, err := os.Open(path)
	require.NoError(t, err)
	defer file.Close()
	decoder := json.NewDecoder(file)
	for _, event := range want {
		var got Exposure
		require.NoError(t, decoder.Decode(&got))
		require.Equal(t, event, got)
	}
}
