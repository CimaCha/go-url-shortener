package model

// UserRecord pairs a user's short URL with its original URL.
type UserRecord struct {
	ShortURL    string `json:"short_url,omitempty"`
	OriginalURL string `json:"original_url,omitempty"`
}

// UserURLsResponse is the JSON array returned by GET /api/user/urls.
type UserURLsResponse []*UserRecord
