package config

import (
	"flag"

	"github.com/caarlos0/env/v6"
)

// Config содержит все конфигурационные параметры приложения
type Config struct {
	ServerAddress   string `env:"SERVER_ADDRESS" envDefault:"localhost:8080"`
	BaseURL         string `env:"BASE_URL" envDefault:"http://localhost:8080"`
	LoggerLevel     string `env:"LOG_LEVEL" envDefault:"info"`
	FileStoragePath string `env:"FILE_STORAGE_PATH" envDefault:"internal/app/storage/storage.json"`
	DatabaseDSN     string `env:"DATABASE_DSN"`
}

// LoadConfig загружает конфигурацию из переменных окружения и флагов командной строки
func LoadConfig() *Config {
	config := &Config{}
	err := env.Parse(config)

	if err != nil {
		panic(err)
	}

	ParseFlags(config)

	return config
}

// ParseFlags добавляет флаги командной строки для параметров конфигурации
// и переопределяет значения, если они указаны в аргументах запуска.
func ParseFlags(config *Config) {
	flag.StringVar(&config.ServerAddress, "a", config.ServerAddress, "address and port to run server")
	flag.StringVar(&config.BaseURL, "b", config.BaseURL, "address and port to link")
	flag.StringVar(&config.LoggerLevel, "l", config.LoggerLevel, "log level")
	flag.StringVar(&config.FileStoragePath, "f", config.FileStoragePath, "file storage path")
	flag.StringVar(&config.DatabaseDSN, "d", config.DatabaseDSN, "database DSN")
	flag.Parse()
}
