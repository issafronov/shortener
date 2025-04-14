package models

type URLData struct {
	URL string `json:"url"`
}

type ShortURLData struct {
	Result string `json:"result"`
}

type BatchURLData struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type BatchURLDataResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}
