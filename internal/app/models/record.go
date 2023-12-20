package models

type Record struct {
	OriginalURL   string `json:"original_url"`
	ShortenedPath string `json:"shortened_path"`
	CorrelationID string `json:"correlation_id"`
	UserID        int    `json:"user_id"`
}
