package storage

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/issafronov/shortener/internal/app/config"
	"github.com/issafronov/shortener/internal/app/contextkeys"
	"github.com/issafronov/shortener/internal/app/models"
	"github.com/issafronov/shortener/internal/middleware/logger"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx"
	"go.uber.org/zap"
	"os"
)

var Urls = make(map[string]ShortenerURL)
var UsersUrls = make(map[string][]string)
var ErrConflict = errors.New("conflict")

type ShortenerURL struct {
	UUID          int    `json:"uuid"`
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
	OriginalURL   string `json:"original_url"`
	UserID        string `json:"user_id"`
	IsDeleted     bool   `json:"is_deleted"`
}

type Storage interface {
	Create(ctx context.Context, url ShortenerURL) (string, error)
	Get(ctx context.Context, url string) (string, error)
	GetByUser(ctx context.Context, username string) ([]models.ShortURLResponse, error)
	Ping(ctx context.Context) error
	DeleteURLs(ctx context.Context, userID string, urls []string) error
}

type FileStorage struct {
	file   *os.File
	writer *bufio.Writer
	reader *bufio.Reader
}

func (f *FileStorage) Ping(ctx context.Context) error {
	return nil
}

func (f *FileStorage) Create(ctx context.Context, url ShortenerURL) (string, error) {
	data, err := json.Marshal(url)
	if err != nil {
		logger.Log.Info("Failed to marshal shortener URL", zap.Error(err))
		return "", err
	}
	if _, err := f.writer.Write(data); err != nil {
		logger.Log.Info("Failed to write URL", zap.String("url", url.ShortURL), zap.Error(err))
		return "", err
	}
	if err := f.writer.WriteByte('\n'); err != nil {
		logger.Log.Info("Error writing data new line", zap.Error(err))
		return "", err
	}
	return "", f.writer.Flush()
}

func (f *FileStorage) Get(ctx context.Context, url string) (string, error) {
	link, ok := Urls[url]
	if !ok {
		return "", errors.New("url not found")
	}
	return link.OriginalURL, nil
}

func (f *FileStorage) GetByUser(ctx context.Context, username string) ([]models.ShortURLResponse, error) {
	var result []models.ShortURLResponse
	for key, value := range UsersUrls {
		if key == username {
			result = append(result, models.ShortURLResponse{
				ShortURL:    ctx.Value(contextkeys.HostKey).(string) + "/" + value[0],
				OriginalURL: value[1],
			})
		}
	}
	return result, nil
}

func (f *FileStorage) DeleteURLs(ctx context.Context, userID string, ids []string) error {
	for _, id := range ids {
		shortenerURL, exists := Urls[id]
		if !exists {
			continue
		}
		if shortenerURL.UserID == userID {
			shortenerURL.IsDeleted = true
			Urls[id] = shortenerURL
		}
	}
	return nil
}

func NewFileStorage(config *config.Config) (*FileStorage, error) {
	file, err := os.OpenFile(config.FileStoragePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	return &FileStorage{
		file:   file,
		writer: bufio.NewWriter(file),
		reader: bufio.NewReader(file),
	}, nil
}

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(ctx context.Context, dsn string) (*PostgresStorage, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	createTableQuery := `
	CREATE TABLE IF NOT EXISTS urls (
	   id SERIAL PRIMARY KEY,
	   short_url TEXT NOT NULL UNIQUE,
	   original_url TEXT NOT NULL UNIQUE,
	   user_id TEXT NOT NULL,
	   is_deleted BOOLEAN DEFAULT FALSE
	);
	`

	if _, err = db.ExecContext(ctx, createTableQuery); err != nil {
		return nil, err
	}

	return &PostgresStorage{db: db}, nil
}

func (s *PostgresStorage) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *PostgresStorage) Create(ctx context.Context, url ShortenerURL) (string, error) {
	query := `
	INSERT INTO urls (
	    short_url,
	    original_url,
		user_id
	    )
	VALUES ($1, $2, $3)
	`
	_, err := s.db.ExecContext(ctx, query, url.ShortURL, url.OriginalURL, url.UserID)

	if err != nil {
		var pgErr pgx.PgError
		if errors.As(err, &pgErr) && pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
			var shortKey string
			err = s.db.QueryRowContext(
				ctx,
				"SELECT short_url FROM urls WHERE original_url = $1",
				url.OriginalURL,
			).Scan(&shortKey)
			if err != nil {
				return "", err
			}
			return shortKey, ErrConflict
		}
		return "", err
	}

	return "", nil
}

func (s *PostgresStorage) Get(ctx context.Context, url string) (string, error) {
	var originalURL string
	var isDeleted bool
	err := s.db.QueryRowContext(
		ctx,
		"SELECT original_url, is_deleted FROM urls WHERE short_url = $1",
		url,
	).Scan(&originalURL, &isDeleted)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errors.New("url not found")
		}
		return "", err
	}
	if isDeleted {
		return "", errors.New("url gone")
	}
	return originalURL, nil
}

func (s *PostgresStorage) GetByUser(ctx context.Context, username string) ([]models.ShortURLResponse, error) {
	var result []models.ShortURLResponse
	rows, err := s.db.Query("SELECT short_url, original_url FROM urls WHERE user_id = $1", username)
	if err != nil {
		logger.Log.Info("Failed to get shortener URLs", zap.String("username", username), zap.Error(err))
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var shortURL string
		var originalURL string
		if err := rows.Scan(&shortURL, &originalURL); err != nil {
			return nil, err
		}
		fmt.Println(ctx.Value(contextkeys.HostKey).(string), shortURL)
		result = append(result, models.ShortURLResponse{ShortURL: ctx.Value(contextkeys.HostKey).(string) + "/" + shortURL, OriginalURL: originalURL})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *PostgresStorage) DeleteURLs(ctx context.Context, userID string, urls []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `UPDATE urls SET is_deleted = TRUE WHERE short_url = $1 AND user_id = $2`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, url := range urls {
		if _, err := stmt.ExecContext(ctx, url, userID); err != nil {
			return err
		}
	}

	return tx.Commit()
}
