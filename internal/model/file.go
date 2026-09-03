package model

type FileRecord struct {
	UUID        string `json:"uuid,omitempty"`
	ShortURL    string `json:"short_url,omitempty"`
	OriginalURL string `json:"original_url,omitempty"`
	UserID      string `json:"user_id,omitempty"`
}
