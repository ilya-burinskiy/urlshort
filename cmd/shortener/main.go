package main

import (
	"github.com/ilya-burinskiy/urlshort/configs"
	"github.com/ilya-burinskiy/urlshort/internal/app/handlers"
	"net/http"
)

func main() {
	config := configs.Parse()
	if err := http.ListenAndServe(config.ServerAddress, handlers.ShortenURLRouter(config)); err != nil {
		panic(err)
	}
}
