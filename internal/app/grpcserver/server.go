package grpcserver

import (
	"context"
	"errors"
	"fmt"

	"github.com/issafronov/shortener/internal/app/config"
	"github.com/issafronov/shortener/internal/app/contextkeys"
	"github.com/issafronov/shortener/internal/app/models"
	"github.com/issafronov/shortener/internal/app/service"
	pb "github.com/issafronov/shortener/proto"
	"google.golang.org/grpc/metadata"
)

type GRPCHandler struct {
	pb.UnimplementedShortenerServer
	svc    service.Service
	config *config.Config
}

func NewGRPCHandler(svc service.Service, config *config.Config) *GRPCHandler {
	return &GRPCHandler{svc: svc, config: config}
}

func (h *GRPCHandler) CreateShortURL(ctx context.Context, req *pb.CreateShortURLRequest) (*pb.ShortURLResponse, error) {
	if req.Url == "" {
		return nil, errors.New("url is empty")
	}

	userID, _ := getKeyFromCtx(ctx, string(contextkeys.UserIDKey))
	shortKey, err := h.svc.CreateURL(ctx, req.Url, userID)
	if err != nil {
		return nil, err
	}

	fullURL := fmt.Sprintf("%s/%s", h.config.BaseURL, shortKey)
	return &pb.ShortURLResponse{Result: fullURL}, nil
}

func (h *GRPCHandler) CreateShortURLJSON(ctx context.Context, req *pb.CreateShortURLRequest) (*pb.ShortURLResponse, error) {
	return h.CreateShortURL(ctx, req)
}

func (h *GRPCHandler) CreateShortURLBatch(ctx context.Context, req *pb.CreateShortURLBatchRequest) (*pb.CreateShortURLBatchResponse, error) {
	if req.UserId == "" {
		return nil, errors.New("user_id is empty")
	}

	var batchReqs []models.BatchURLData
	for _, u := range req.Urls {
		batchReqs = append(batchReqs, models.BatchURLData{
			CorrelationID: u.CorrelationId,
			OriginalURL:   u.OriginalUrl,
		})
	}

	batchResponses, err := h.svc.CreateURLBatch(ctx, batchReqs, req.UserId)
	if err != nil {
		return nil, err
	}

	var pbBatchResponses []*pb.BatchURLDataResponse
	for _, r := range batchResponses {
		pbBatchResponses = append(pbBatchResponses, &pb.BatchURLDataResponse{
			CorrelationId: r.CorrelationID,
			ShortUrl:      r.ShortURL,
		})
	}

	return &pb.CreateShortURLBatchResponse{Urls: pbBatchResponses}, nil
}

func (h *GRPCHandler) GetOriginalURL(ctx context.Context, req *pb.GetOriginalURLRequest) (*pb.OriginalURLResponse, error) {
	if req.ShortUrl == "" {
		return nil, errors.New("short_url is empty")
	}

	originalURL, err := h.svc.GetOriginalURL(ctx, req.ShortUrl)
	if err != nil {
		return nil, err
	}

	return &pb.OriginalURLResponse{OriginalUrl: originalURL}, nil
}

func (h *GRPCHandler) GetUserURLs(ctx context.Context, req *pb.UserIDRequest) (*pb.UserURLsResponse, error) {
	if req.UserId == "" {
		return nil, errors.New("user_id is empty")
	}

	host, _ := getKeyFromCtx(ctx, string(contextkeys.HostKey))
	userURLs, err := h.svc.GetUserURLs(ctx, req.UserId, host)
	if err != nil {
		return nil, err
	}

	var pbUserURLs []*pb.UserURL
	for _, u := range userURLs {
		pbUserURLs = append(pbUserURLs, &pb.UserURL{
			ShortUrl:    u.ShortURL,
			OriginalUrl: u.OriginalURL,
		})
	}

	return &pb.UserURLsResponse{Urls: pbUserURLs}, nil
}

func (h *GRPCHandler) DeleteUserURLs(ctx context.Context, req *pb.DeleteUserURLsRequest) (*pb.DeleteUserURLsResponse, error) {
	if req.UserId == "" || len(req.ShortUrlIds) == 0 {
		return &pb.DeleteUserURLsResponse{Success: false}, errors.New("user_id or short_url_ids missing")
	}

	err := h.svc.DeleteUserURLs(ctx, req.UserId, req.ShortUrlIds)
	if err != nil {
		return &pb.DeleteUserURLsResponse{Success: false}, err
	}

	return &pb.DeleteUserURLsResponse{Success: true}, nil
}

func (h *GRPCHandler) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	err := h.svc.Ping(ctx)
	if err != nil {
		return &pb.PingResponse{Status: "FAIL"}, err
	}
	return &pb.PingResponse{Status: "OK"}, nil
}

func (h *GRPCHandler) GetStats(ctx context.Context, req *pb.GetStatsRequest) (*pb.GetStatsResponse, error) {
	urlsCount, usersCount, err := h.svc.GetStats(ctx)
	if err != nil {
		return nil, err
	}
	return &pb.GetStatsResponse{
		Urls:  urlsCount,
		Users: usersCount,
	}, nil
}

// getKeyFromCtx функция для получения данных из metadata
func getKeyFromCtx(ctx context.Context, key string) (string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}
	ids := md.Get(key)
	if len(ids) == 0 {
		return "", false
	}
	return ids[0], true
}
