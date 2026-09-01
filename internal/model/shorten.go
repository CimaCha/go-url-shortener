package model

type ShortenURLRequest struct {
	URL string `json:"url,omitempty"`
}

type ShortenURLResponse struct {
	Result string `json:"result,omitempty"`
}
