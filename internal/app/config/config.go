package config

import (
	"flag"
	"github.com/caarlos0/env/v6"
)

type Config struct {
	ServerAddress   string `env:"SERVER_ADDRESS" envDefault:"localhost:8080"`
	BaseURL         string `env:"BASE_URL" envDefault:"http://localhost:8080"`
	LoggerLevel     string `env:"LOG_LEVEL" envDefault:"info"`
	FileStoragePath string `env:"FILE_STORAGE_PATH" envDefault:"storage.json"`
}

func LoadConfig() *Config {
	config := &Config{}
	err := env.Parse(config)

	if err != nil {
		panic(err)
	}

	ParseFlags(config)

	return config
}

func ParseFlags(config *Config) {
	flag.StringVar(&config.ServerAddress, "a", config.ServerAddress, "address and port to run server")
	flag.StringVar(&config.BaseURL, "b", config.BaseURL, "address and port to link")
	flag.StringVar(&config.LoggerLevel, "l", config.LoggerLevel, "log level")
	flag.StringVar(&config.FileStoragePath, "f", config.FileStoragePath, "file storage path")
	flag.Parse()
}
