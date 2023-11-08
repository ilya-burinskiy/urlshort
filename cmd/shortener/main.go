package main

import (
	"github.com/ilya-burinskiy/urlshort/configs"
	"github.com/ilya-burinskiy/urlshort/internal/app/handlers"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage"
	"github.com/ilya-burinskiy/urlshort/internal/app/services"
	"net/http"
)

func main() {
	config := configs.Parse()
	rndGen := services.StdRandHexStringGenerator{}
	storage := storage.Storage{}
	err := http.ListenAndServe(
		config.ServerAddress,
		handlers.ShortenURLRouter(config, rndGen, storage),
	)

	if err != nil {
		panic(err)
	}
}
