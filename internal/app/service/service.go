package service

import (
	"context"

	"github.com/issafronov/shortener/internal/app/models"
)

// Service определяет бизнес-логику для работы с сокращёнными URL
type Service interface {
	// CreateURL создаёт сокращённый URL для одного оригинального URL
	CreateURL(ctx context.Context, originalURL, userID string) (shortKey string, err error)

	// CreateURLBatch создаёт сокращённые URL по батч-запросу
	CreateURLBatch(ctx context.Context, batch []models.BatchURLData, userID string) ([]models.BatchURLDataResponse, error)

	// GetOriginalURL возвращает оригинальный URL по его короткому ключу
	GetOriginalURL(ctx context.Context, shortKey string) (string, error)

	// GetUserURLs возвращает все сокращённые URL, созданные пользователем
	GetUserURLs(ctx context.Context, userID, host string) ([]models.ShortURLResponse, error)

	// DeleteUserURLs удаляет список сокращённых ссылок пользователя
	DeleteUserURLs(ctx context.Context, userID string, ids []string) error

	// GetStats возвращает статистику: количество URL и количество пользователей
	GetStats(ctx context.Context) (urls int64, users int64, err error)

	// Ping пингует сервис
	Ping(ctx context.Context) error
}
