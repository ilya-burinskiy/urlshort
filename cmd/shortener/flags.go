package main

import "flag"

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
}
