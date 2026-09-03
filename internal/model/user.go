package model

type UserRecord struct {
	ShortURL    string `json:"short_url,omitempty"`
	OriginalURL string `json:"original_url,omitempty"`
}

type UserURLsResponse []*UserRecord
