package model

// ShortenURLRequest is the JSON body for POST /api/shorten.
type ShortenURLRequest struct {
	URL string `json:"url,omitempty"`
}

// ShortenURLResponse is the JSON result of POST /api/shorten.
type ShortenURLResponse struct {
	Result string `json:"result,omitempty"`
}

// StorageRecord contains ownership, destination, and deletion state for a short URL.
type StorageRecord struct {
	UserID      string
	OriginalURL string
	DeletedFlag bool
}
