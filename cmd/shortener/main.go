package main

import (
	"github.com/ilya-burinskiy/urlshort/configs"
	"net/http"
)

func main() {
	config := configs.Parse()
	if err := http.ListenAndServe(config.ServerAddress, ShortenURLRouter(config)); err != nil {
		panic(err)
	}
}
