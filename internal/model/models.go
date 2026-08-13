package model

type ShortenURLRequest struct {
	URL string `json:"url,omitempty"`
}

type ShortenURLResponse struct {
	Result string `json:"result,omitempty"`
}

type FileRecord struct {
	UUID        string `json:"uuid,omitempty"`
	ShortUrl    string `json:"short_url,omitempty"`
	OriginalUrl string `json:"original_url,omitempty"`
}
