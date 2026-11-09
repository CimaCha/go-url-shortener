package model

// ShortenBatchRequest is the JSON array accepted by the batch shortening endpoint.
type ShortenBatchRequest []*OriginalURLRecord

// OriginalURLRecord pairs a client correlation ID with an original URL.
type OriginalURLRecord struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

// ShortenBatchResponse is the JSON array returned by the batch shortening endpoint.
type ShortenBatchResponse []*ShortURLRecord

// ShortURLRecord pairs a client correlation ID with the resulting short URL.
type ShortURLRecord struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

// URLRecord carries an original URL and its generated short ID to storage.
type URLRecord struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
	ShortURL      string `json:"short_url"`
}
