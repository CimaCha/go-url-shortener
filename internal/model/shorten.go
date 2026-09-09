package model

type ShortenURLRequest struct {
	URL string `json:"url,omitempty"`
}

type ShortenURLResponse struct {
	Result string `json:"result,omitempty"`
}

type StorageRecord struct {
	UserID      string `json:"user_id,omitempty"`
	OriginalURL string `json:"original_url,omitempty"`
	DeletedFlag bool   `json:"deleted_flag,omitempty"`
}
