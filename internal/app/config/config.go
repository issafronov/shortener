package config

import (
	"flag"
)

type Config struct {
	Address string
	BaseURL string
}

func LoadConfig() *Config {
	config := &Config{}
	ParseFlags(config)
	return config
}

func ParseFlags(config *Config) {
	flag.StringVar(&config.Address, "a", "localhost:8080", "address and port to run server")
	flag.StringVar(&config.BaseURL, "b", "http://127.0.0.1:8080", "address and port to run server")
	flag.Parse()
}
