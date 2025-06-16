package models

// URLData представляет входную структуру для сокращения URL
type URLData struct {
	URL string `json:"url"`
}

// ShortURLData представляет выходную структуру после создания короткой ссылки
type ShortURLData struct {
	Result string `json:"result"`
}

// ShortURLResponse содержит пару оригинального и сокращённого URL
// Используется при получении списка ссылок, привязанных к пользователю
type ShortURLResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// BatchURLData используется для пакетной отправки ссылок на сокращение
// Каждая запись содержит оригинальный URL и связанный с ним correlation_id
type BatchURLData struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

// BatchURLDataResponse используется для ответа на пакетную обработку ссылок
// Каждая запись содержит correlation_id и соответствующий короткий URL
type BatchURLDataResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}
