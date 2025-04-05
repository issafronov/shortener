package storage

import (
	"context"
	"database/sql"
)

var Urls = make(map[string]string)

type ShortenerURL struct {
	UUID        int    `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type Storage interface {
	Ping(ctx context.Context) error
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
	   original_url TEXT NOT NULL UNIQUE
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
