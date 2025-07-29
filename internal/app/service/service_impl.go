package service

import (
	"context"
	"errors"

	"github.com/issafronov/shortener/internal/app/contextkeys"
	"github.com/issafronov/shortener/internal/app/models"
	"github.com/issafronov/shortener/internal/app/storage"
	"github.com/issafronov/shortener/internal/app/utils"
)

const shortKeyLength = 8

var ErrDeleted = errors.New("url gone")
var ErrConflict = errors.New("url conflict")

type shortenerService struct {
	storage storage.Storage
}

// NewService создаёт новый экземпляр сервиса
func NewService(storage storage.Storage) Service {
	return &shortenerService{storage: storage}
}

// CreateURL создаёт сокращённый URL
func (s *shortenerService) CreateURL(ctx context.Context, originalURL, userID string) (string, error) {
	shortKey := utils.CreateShortKey(shortKeyLength)
	uuid := len(storage.Urls) + 1

	shortenerURL := storage.ShortenerURL{
		UUID:        uuid,
		ShortURL:    shortKey,
		OriginalURL: originalURL,
		UserID:      userID,
	}

	storage.Urls[shortKey] = shortenerURL
	storage.UsersUrls[userID] = []string{shortKey, originalURL}

	_, err := s.storage.Create(ctx, shortenerURL)
	if err != nil {
		if errors.Is(err, storage.ErrConflict) {
			return "", ErrConflict
		}
		return "", err
	}

	return shortKey, nil
}

// CreateURLBatch создаёт пакет ссылок
func (s *shortenerService) CreateURLBatch(ctx context.Context, batch []models.BatchURLData, userID string) ([]models.BatchURLDataResponse, error) {
	var responses []models.BatchURLDataResponse

	for _, item := range batch {
		if item.OriginalURL == "" || item.CorrelationID == "" {
			continue
		}

		shortKey := utils.CreateShortKey(shortKeyLength)
		shortenerURL := storage.ShortenerURL{
			ShortURL:      shortKey,
			OriginalURL:   item.OriginalURL,
			CorrelationID: item.CorrelationID,
			UserID:        userID,
		}

		_, err := s.storage.Create(ctx, shortenerURL)
		if err != nil {
			if errors.Is(err, storage.ErrConflict) {
				return nil, ErrConflict
			}
			return nil, err
		}

		responses = append(responses, models.BatchURLDataResponse{
			ShortURL:      shortKey,
			CorrelationID: item.CorrelationID,
		})
	}

	return responses, nil
}

// GetOriginalURL возвращает оригинальный URL по ключу
func (s *shortenerService) GetOriginalURL(ctx context.Context, shortKey string) (string, error) {
	return s.storage.Get(ctx, shortKey)
}

// GetUserURLs возвращает все URL пользователя
func (s *shortenerService) GetUserURLs(ctx context.Context, userID, host string) ([]models.ShortURLResponse, error) {
	ctx = context.WithValue(ctx, contextkeys.HostKey, host)
	return s.storage.GetByUser(ctx, userID)
}

// DeleteUserURLs удаляет список ссылок
func (s *shortenerService) DeleteUserURLs(ctx context.Context, userID string, ids []string) error {
	return s.storage.DeleteURLs(ctx, userID, ids)
}

// GetStats возвращает общее количество URL и пользователей
func (s *shortenerService) GetStats(ctx context.Context) (int64, int64, error) {
	urls, err := s.storage.CountURLs(ctx)
	if err != nil {
		return 0, 0, err
	}

	users, err := s.storage.CountUsers(ctx)
	if err != nil {
		return 0, 0, err
	}

	return urls, users, nil
}

func (s *shortenerService) Ping(ctx context.Context) error {
	return s.storage.Ping(ctx)
}
