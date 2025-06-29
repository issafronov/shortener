package config_test

import (
	"flag"
	"os"
	"testing"

	"github.com/caarlos0/env/v6"
	"github.com/issafronov/shortener/internal/app/config"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	t.Setenv("SERVER_ADDRESS", "testhost:9999")
	t.Setenv("BASE_URL", "http://testhost")
	t.Setenv("LOG_LEVEL", "warn")
	t.Setenv("FILE_STORAGE_PATH", "/tmp/test.json")
	t.Setenv("DATABASE_DSN", "dsn")

	// очистим аргументы флагов, чтобы не мешали тесту
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"test"}

	cfg := config.LoadConfig()

	assert.Equal(t, "testhost:9999", cfg.ServerAddress)
	assert.Equal(t, "http://testhost", cfg.BaseURL)
	assert.Equal(t, "warn", cfg.LoggerLevel)
	assert.Equal(t, "/tmp/test.json", cfg.FileStoragePath)
	assert.Equal(t, "dsn", cfg.DatabaseDSN)
}

func TestLoadConfig_FromEnv(t *testing.T) {
	t.Setenv("SERVER_ADDRESS", "127.0.0.1:9000")
	t.Setenv("BASE_URL", "http://example.com")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("FILE_STORAGE_PATH", "/tmp/data.json")
	t.Setenv("DATABASE_DSN", "postgres://user:pass@localhost/db")

	cfg := &config.Config{}
	err := env.Parse(cfg)
	assert.NoError(t, err)

	// Флаги игнорируются в этом тесте — только env
	assert.Equal(t, "127.0.0.1:9000", cfg.ServerAddress)
	assert.Equal(t, "http://example.com", cfg.BaseURL)
	assert.Equal(t, "debug", cfg.LoggerLevel)
	assert.Equal(t, "/tmp/data.json", cfg.FileStoragePath)
	assert.Equal(t, "postgres://user:pass@localhost/db", cfg.DatabaseDSN)
}

func TestParseFlags(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	cfg := &config.Config{
		ServerAddress:   "default:8000",
		BaseURL:         "http://default",
		LoggerLevel:     "info",
		FileStoragePath: "/default/path.json",
		DatabaseDSN:     "default-dsn",
	}

	fs.StringVar(&cfg.ServerAddress, "a", cfg.ServerAddress, "")
	fs.StringVar(&cfg.BaseURL, "b", cfg.BaseURL, "")
	fs.StringVar(&cfg.LoggerLevel, "l", cfg.LoggerLevel, "")
	fs.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "")
	fs.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "")

	args := []string{
		"-a=0.0.0.0:9999",
		"-b=http://cli.example.com",
		"-l=error",
		"-f=/tmp/cli.json",
		"-d=cli-dsn",
	}

	err := fs.Parse(args)
	assert.NoError(t, err)

	assert.Equal(t, "0.0.0.0:9999", cfg.ServerAddress)
	assert.Equal(t, "http://cli.example.com", cfg.BaseURL)
	assert.Equal(t, "error", cfg.LoggerLevel)
	assert.Equal(t, "/tmp/cli.json", cfg.FileStoragePath)
	assert.Equal(t, "cli-dsn", cfg.DatabaseDSN)
}
