package main

import (
	"flag"
	"net/http"
)

var storage = make(Storage)
var config struct {
	serverAddress        string
	shortenedURLBaseAddr string
}

func main() {
	flag.StringVar(&config.serverAddress, "a", "localhost:8080", "server's address")
	flag.StringVar(
		&config.shortenedURLBaseAddr,
		"b", "http://localhost:8080",
		"base address of the resulting shortened URL")

	flag.Parse()

	if err := http.ListenAndServe(config.serverAddress, ShortenURLRouter()); err != nil {
		panic(err)
	}
}
