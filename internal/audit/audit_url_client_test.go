package audit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gentleman.v2"
)

func TestURLClientSubscriberSendsExposure(t *testing.T) {
	want := Exposure{Timestamp: 123, Action: Shorten, UserID: "user-1", URL: "https://example.com"}
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/audit", r.URL.Path)
		require.Equal(t, "source=shortener", r.URL.RawQuery)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		var got Exposure
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		require.Equal(t, want, got)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	subscriber := NewURLClientSubscriber("url", server.URL+"/audit?source=shortener", gentleman.New())
	require.NoError(t, subscriber.update(want))
	require.Equal(t, int32(1), requests.Load())
}

func TestURLClientSubscriberReportsFailedResponse(t *testing.T) {
	for _, status := range []int{http.StatusMultipleChoices, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
			}))
			defer server.Close()

			subscriber := NewURLClientSubscriber("url", server.URL, gentleman.New())
			require.ErrorIs(t, subscriber.update(Exposure{}), ErrSendRequest)
		})
	}
}

func TestURLClientSubscriberReportsTransportError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := server.URL
	server.Close()

	subscriber := NewURLClientSubscriber("url", url, gentleman.New())
	err := subscriber.update(Exposure{})
	require.ErrorIs(t, err, ErrSendRequest)
}

func TestURLClientSubscriberDoesNotFollowRedirect(t *testing.T) {
	var redirected atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			redirected.Store(true)
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Redirect(w, r, "/login", http.StatusFound)
	}))
	defer server.Close()

	subscriber := NewURLClientSubscriber("url", server.URL+"/audit", gentleman.New())
	require.ErrorIs(t, subscriber.update(Exposure{}), ErrSendRequest)
	require.False(t, redirected.Load())
}
