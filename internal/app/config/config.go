package config

import (
	"encoding/json"
	"flag"
	"os"

	"github.com/caarlos0/env/v6"
)

// Config содержит все конфигурационные параметры приложения
type Config struct {
	ServerAddress   string `json:"server_address" env:"SERVER_ADDRESS" envDefault:"localhost:8080"`
	BaseURL         string `json:"base_url" env:"BASE_URL" envDefault:"http://localhost:8080"`
	LoggerLevel     string `json:"log_level" env:"LOG_LEVEL" envDefault:"info"`
	FileStoragePath string `json:"file_storage_path" env:"FILE_STORAGE_PATH" envDefault:"internal/app/storage/storage.json"`
	DatabaseDSN     string `json:"database_dsn" env:"DATABASE_DSN"`
	EnableHTTPS     bool   `json:"enable_https" env:"ENABLE_HTTPS"`
	ConfigFile      string `json:"-" env:"CONFIG"`
}

// LoadConfig загружает конфигурацию из переменных окружения и флагов командной строки или JSON конфиг файла
func LoadConfig() *Config {
	config := &Config{}
	err := env.Parse(config)

	if err != nil {
		panic(err)
	}

	ParseFlags(config)

	if config.ConfigFile != "" {
		fileConfig, err := loadConfigFromFile(config.ConfigFile)
		if err != nil {
			panic(err)
		}
		mergeConfigs(config, fileConfig)
	}

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
	flag.BoolVar(&config.EnableHTTPS, "s", config.EnableHTTPS, "enable HTTPS")
	flag.Parse()
}

func loadConfigFromFile(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg Config
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) isDefault(field string) bool {
	switch field {
	case "ServerAddress":
		return c.ServerAddress == "localhost:8080"
	case "BaseURL":
		return c.BaseURL == "http://localhost:8080"
	case "LoggerLevel":
		return c.LoggerLevel == "info"
	case "FileStoragePath":
		return c.FileStoragePath == "internal/app/storage/storage.json"
	case "DatabaseDSN":
		return c.DatabaseDSN == ""
	case "EnableHTTPS":
		return !c.EnableHTTPS
	default:
		return false
	}
}

func mergeConfigs(dst, src *Config) {
	if src.ServerAddress != "" && dst.isDefault("ServerAddress") {
		dst.ServerAddress = src.ServerAddress
	}
	if src.BaseURL != "" && dst.isDefault("BaseURL") {
		dst.BaseURL = src.BaseURL
	}
	if src.LoggerLevel != "" && dst.isDefault("LoggerLevel") {
		dst.LoggerLevel = src.LoggerLevel
	}
	if src.FileStoragePath != "" && dst.isDefault("FileStoragePath") {
		dst.FileStoragePath = src.FileStoragePath
	}
	if src.DatabaseDSN != "" && dst.isDefault("DatabaseDSN") {
		dst.DatabaseDSN = src.DatabaseDSN
	}
	if src.EnableHTTPS && dst.isDefault("EnableHTTPS") {
		dst.EnableHTTPS = src.EnableHTTPS
	}
}
