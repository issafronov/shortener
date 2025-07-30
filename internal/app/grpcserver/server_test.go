package grpcserver

import (
	"context"
	"errors"
	"testing"

	"github.com/issafronov/shortener/internal/app/config"
	"github.com/issafronov/shortener/internal/app/models"
	pb "github.com/issafronov/shortener/proto"
	"github.com/stretchr/testify/assert"
)

// stubService — простая реализация интерфейса Service для тестов
type stubService struct {
	CreateURLFn func(ctx context.Context, originalURL, userID string) (string, error)
	PingFn      func(ctx context.Context) error
}

func (s *stubService) CreateURL(ctx context.Context, originalURL, userID string) (string, error) {
	return s.CreateURLFn(ctx, originalURL, userID)
}

func (s *stubService) CreateURLBatch(ctx context.Context, batch []models.BatchURLData, userID string) ([]models.BatchURLDataResponse, error) {
	return nil, nil
}

func (s *stubService) GetOriginalURL(ctx context.Context, shortKey string) (string, error) {
	return "", nil
}

func (s *stubService) GetUserURLs(ctx context.Context, userID, host string) ([]models.ShortURLResponse, error) {
	return nil, nil
}

func (s *stubService) DeleteUserURLs(ctx context.Context, userID string, ids []string) error {
	return nil
}

func (s *stubService) GetStats(ctx context.Context) (int64, int64, error) {
	return 0, 0, nil
}

func (s *stubService) Ping(ctx context.Context) error {
	return s.PingFn(ctx)
}

func TestCreateShortURL(t *testing.T) {
	svc := &stubService{
		CreateURLFn: func(ctx context.Context, originalURL, userID string) (string, error) {
			return "abc123", nil
		},
	}
	cfg := &config.Config{BaseURL: "http://localhost"}
	handler := NewGRPCHandler(svc, cfg)

	resp, err := handler.CreateShortURL(context.Background(), &pb.CreateShortURLRequest{Url: "https://example.com"})
	assert.NoError(t, err)
	assert.Equal(t, "http://localhost/abc123", resp.Result)
}

func TestCreateShortURL_EmptyURL(t *testing.T) {
	handler := NewGRPCHandler(&stubService{}, &config.Config{})

	resp, err := handler.CreateShortURL(context.Background(), &pb.CreateShortURLRequest{Url: ""})
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestPing_OK(t *testing.T) {
	svc := &stubService{
		PingFn: func(ctx context.Context) error {
			return nil
		},
	}
	handler := NewGRPCHandler(svc, &config.Config{})

	resp, err := handler.Ping(context.Background(), &pb.PingRequest{})
	assert.NoError(t, err)
	assert.Equal(t, "OK", resp.Status)
}

func TestPing_Fail(t *testing.T) {
	svc := &stubService{
		PingFn: func(ctx context.Context) error {
			return errors.New("db down")
		},
	}
	handler := NewGRPCHandler(svc, &config.Config{})

	resp, err := handler.Ping(context.Background(), &pb.PingRequest{})
	assert.Error(t, err)
	assert.Equal(t, "FAIL", resp.Status)
}
