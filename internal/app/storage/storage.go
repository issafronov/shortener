package storage

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/issafronov/shortener/internal/app/config"
	"github.com/issafronov/shortener/internal/middleware/logger"
	"go.uber.org/zap"
	"os"
)

var Urls = make(map[string]string)

type ShortenerURL struct {
	UUID          int    `json:"uuid"`
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
	OriginalURL   string `json:"original_url"`
}

type Storage interface {
	Create(ctx context.Context, url ShortenerURL) error
	Get(ctx context.Context, url string) (string, error)
	Ping(ctx context.Context) error
}

type FileStorage struct {
	file   *os.File
	writer *bufio.Writer
	reader *bufio.Reader
}

func (f *FileStorage) Ping(ctx context.Context) error {
	return nil
}

func (f *FileStorage) Create(ctx context.Context, url ShortenerURL) error {
	data, err := json.Marshal(url)
	if err != nil {
		logger.Log.Info("Failed to marshal shortener URL", zap.Error(err))
		return err
	}
	if _, err := f.writer.Write(data); err != nil {
		logger.Log.Info("Failed to write URL", zap.String("url", url.ShortURL), zap.Error(err))
		return err
	}
	if err := f.writer.WriteByte('\n'); err != nil {
		logger.Log.Info("Error writing data new line", zap.Error(err))
		return err
	}
	return f.writer.Flush()
}

func (f *FileStorage) Get(ctx context.Context, url string) (string, error) {
	link, ok := Urls[url]
	if !ok {
		return "", errors.New("url not found")
	}
	return link, nil
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
	   short_url TEXT NOT NULL,
	   original_url TEXT NOT NULL
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

func (s *PostgresStorage) Create(ctx context.Context, url ShortenerURL) error {
	query := `
	INSERT INTO urls (
	    short_url,
	    original_url
	    )
	VALUES ($1, $2)
	`
	_, err := s.db.ExecContext(ctx, query, url.ShortURL, url.OriginalURL)

	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresStorage) Get(ctx context.Context, url string) (string, error) {
	var originalURL string
	err := s.db.QueryRowContext(
		ctx,
		"SELECT original_url FROM urls WHERE short_url = $1",
		url,
	).Scan(&originalURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errors.New("url not found")
		}
		return "", err
	}
	return originalURL, nil
}
