package configs

import (
	"flag"
	"os"
)

type Config struct {
	ServerAddress        string
	ShortenedURLBaseAddr string
}

func Parse() Config {
	config := Config{}
	flag.StringVar(&config.ServerAddress, "a", "localhost:8080", "server's address")
	flag.StringVar(
		&config.ShortenedURLBaseAddr,
		"b", "http://localhost:8080",
		"base address of the resulting shortened URL")
	flag.Parse()

	if envServerAddress := os.Getenv("SERVER_ADDRESS"); envServerAddress != "" {
		config.ServerAddress = envServerAddress
	}
	if envShortenedURLBaseAddr := os.Getenv("BASE_URL"); envShortenedURLBaseAddr != "" {
		config.ShortenedURLBaseAddr = envShortenedURLBaseAddr
	}

	return config
}
