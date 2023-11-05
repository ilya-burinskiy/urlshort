package main

import (
	"flag"
	"os"
)

var config struct {
	serverAddress        string
	shortenedURLBaseAddr string
}

func parseFlags() {
	flag.StringVar(&config.serverAddress, "a", "localhost:8080", "server's address")
	flag.StringVar(
		&config.shortenedURLBaseAddr,
		"b", "http://localhost:8080",
		"base address of the resulting shortened URL")
	flag.Parse()

	if envServerAddress := os.Getenv("SERVER_ADDRESS"); envServerAddress != "" {
		config.serverAddress = envServerAddress
	}
	if envShortenedURLBaseAddr := os.Getenv("BASE_URL"); envShortenedURLBaseAddr != "" {
		config.shortenedURLBaseAddr = envShortenedURLBaseAddr
	}
}
