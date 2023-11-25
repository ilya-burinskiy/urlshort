package configs

import (
	"flag"
	"os"
)

type Config struct {
	ServerAddress        string
	ShortenedURLBaseAddr string
	FileStoragePath      string
}

func Parse() Config {
	config := Config{}
	flag.StringVar(&config.ServerAddress, "a", "localhost:8080", "server's address")
	flag.StringVar(
		&config.ShortenedURLBaseAddr,
		"b", "http://localhost:8080",
		"base address of the resulting shortened URL")
	flag.StringVar(
		&config.FileStoragePath,
		"f", "../../storage",
		"file storage path",
	)
	flag.Parse()

	if envServerAddress := os.Getenv("SERVER_ADDRESS"); envServerAddress != "" {
		config.ServerAddress = envServerAddress
	}
	if envShortenedURLBaseAddr := os.Getenv("BASE_URL"); envShortenedURLBaseAddr != "" {
		config.ShortenedURLBaseAddr = envShortenedURLBaseAddr
	}
	if envFileStoragePath := os.Getenv("FILE_STORAGE_PATH"); envFileStoragePath != "" {
		config.FileStoragePath = envFileStoragePath
	}

	return config
}
