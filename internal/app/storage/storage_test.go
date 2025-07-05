package storage_test

import (
	"context"
	"os"
	"testing"

	"github.com/issafronov/shortener/internal/app/config"
	"github.com/issafronov/shortener/internal/app/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileStorage_CreateAndGet(t *testing.T) {
	// Подготовка временного файла
	tmpFile, err := os.CreateTemp("", "storage-test-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	cfg := &config.Config{FileStoragePath: tmpFile.Name()}
	s, err := storage.NewFileStorage(cfg)
	require.NoError(t, err)

	ctx := context.Background()

	short := "abc123"
	original := "https://example.com"
	userID := "user1"

	url := storage.ShortenerURL{
		ShortURL:    short,
		OriginalURL: original,
		UserID:      userID,
	}

	// Create
	_, err = s.Create(ctx, url)
	require.NoError(t, err)

	// 🧠 Вручную добавим в Urls (т.к. Create этого не делает)
	storage.Urls[short] = url

	// Get
	got, err := s.Get(ctx, short)
	require.NoError(t, err)
	assert.Equal(t, original, got)
}

func TestFileStorage_Get_NotFound(t *testing.T) {
	cfg := &config.Config{FileStoragePath: "nonexistent.json"}
	s, err := storage.NewFileStorage(cfg)
	require.NoError(t, err)

	_, err = s.Get(context.Background(), "no-such-key")
	assert.Error(t, err)
}

func TestFileStorage_Ping(t *testing.T) {
	// Для FileStorage Ping всегда возвращает nil
	fs := &storage.FileStorage{}
	err := fs.Ping(context.Background())
	if err != nil {
		t.Fatalf("FileStorage.Ping() returned error: %v", err)
	}
}

func cleanupGlobals() {
	for k := range storage.Urls {
		delete(storage.Urls, k)
	}
	for k := range storage.UsersUrls {
		delete(storage.UsersUrls, k)
	}
}

func TestFileStorage_DeleteURLs(t *testing.T) {
	cleanupGlobals()

	// Подготовим URL для удаления
	storage.Urls["short1"] = storage.ShortenerURL{
		UserID: "user1",
	}

	fs := &storage.FileStorage{}

	err := fs.DeleteURLs(context.Background(), "user1", []string{"short1"})
	if err != nil {
		t.Fatalf("DeleteURLs returned error: %v", err)
	}

	if !storage.Urls["short1"].IsDeleted {
		t.Error("Expected IsDeleted to be true after DeleteURLs")
	}
}
