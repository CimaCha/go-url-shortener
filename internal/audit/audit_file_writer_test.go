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
	subscriber, err := NewFileWriteSubscriber("file", path)
	require.NoError(t, err)
	want := []Exposure{
		{Timestamp: 1, Action: Shorten, UserID: "user", URL: "https://example.com/first"},
		{Timestamp: 2, Action: Follow, UserID: "user", URL: "https://example.com/second"},
	}
	for _, event := range want {
		require.NoError(t, subscriber.update(event))
	}
	require.NoError(t, subscriber.Close())

	contents, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "{\"ts\":1,\"action\":\"shorten\",\"user_id\":\"user\",\"url\":\"https://example.com/first\"}\n"+
		"{\"ts\":2,\"action\":\"follow\",\"user_id\":\"user\",\"url\":\"https://example.com/second\"}\n", string(contents))

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

func TestFileWriteSubscriberKeepsFileOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.json")
	subscriber, err := NewFileWriteSubscriber("file", path)
	require.NoError(t, err)
	require.NoError(t, subscriber.update(Exposure{Timestamp: 1}))
	moved := path + ".old"
	require.NoError(t, os.Rename(path, moved))
	require.NoError(t, subscriber.update(Exposure{Timestamp: 2}))
	require.NoError(t, subscriber.Close())

	contents, err := os.ReadFile(moved)
	require.NoError(t, err)
	require.Contains(t, string(contents), "\"ts\":2")
	_, err = os.Stat(path)
	require.True(t, os.IsNotExist(err))
}
