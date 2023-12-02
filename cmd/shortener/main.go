package main

import (
	"net/http"

	"github.com/ilya-burinskiy/urlshort/internal/app/configs"
	"github.com/ilya-burinskiy/urlshort/internal/app/handlers"
	"github.com/ilya-burinskiy/urlshort/internal/app/logger"
	"github.com/ilya-burinskiy/urlshort/internal/app/services"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage"
)

func main() {
	config := configs.Parse()
	rndGen := services.StdRandHexStringGenerator{}
	storage := storage.Storage{}
	if err := logger.Initialize("info"); err != nil {
		panic(err)
	}

	err := http.ListenAndServe(
		config.ServerAddress,
		handlers.ShortenURLRouter(config, rndGen, storage),
	)

	if err != nil {
		panic(err)
	}
}
