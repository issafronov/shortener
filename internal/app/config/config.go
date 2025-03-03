package config

import (
	"flag"
	"github.com/caarlos0/env/v6"
)

type Config struct {
	ServerAddress string `env:"SERVER_ADDRESS" envDefault:"localhost:8080"`
	BaseURL       string `env:"BASE_URL" envDefault:"http://localhost:8080"`
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
	flag.Parse()
}
