package model

type ShortenBatchRequest []*OriginalURLRecord

type OriginalURLRecord struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type ShortenBatchResponse []*ShortURLRecord

type ShortURLRecord struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

type URLRecord struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
	ShortURL      string `json:"short_url"`
}
