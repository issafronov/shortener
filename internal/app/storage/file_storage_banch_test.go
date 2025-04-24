package storage

import (
	"context"
	"fmt"
	"testing"
)

func BenchmarkDeleteURLs(b *testing.B) {
	ctx := context.Background()
	userID := "bench-user"
	fileStorage := &FileStorage{}

	// Подготовка: заполняем Urls и UsersUrls
	keys := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("short-%d", i)
		original := fmt.Sprintf("https://example.com/%d", i)
		Urls[key] = ShortenerURL{OriginalURL: original}
		UsersUrls[userID] = []string{key, original}
		keys[i] = key
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fileStorage.DeleteURLs(ctx, userID, keys)
	}
}
