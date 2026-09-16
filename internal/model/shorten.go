package model

type ShortenURLRequest struct {
	URL string `json:"url,omitempty"`
}

type ShortenURLResponse struct {
	Result string `json:"result,omitempty"`
}

type StorageRecord struct {
	UserID      string
	OriginalURL string
	DeletedFlag bool
}
