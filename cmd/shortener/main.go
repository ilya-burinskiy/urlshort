package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ilya-burinskiy/urlshort/internal/app/configs"
	"github.com/ilya-burinskiy/urlshort/internal/app/handlers"
	"github.com/ilya-burinskiy/urlshort/internal/app/logger"
	"github.com/ilya-burinskiy/urlshort/internal/app/services"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage"
)

func main() {
	config := configs.Parse()
	rndGen := services.StdRandHexStringGenerator{}
	if err := logger.Initialize("info"); err != nil {
		panic(err)
	}

	persistentStorage := storage.ConfigurePersistentStorage(config)
	storage := storage.NewMapStorage(persistentStorage)
	err := storage.Restore()
	if err != nil {
		panic(err)
	}

	server := http.Server{
		Handler: handlers.ShortenURLRouter(config, rndGen, storage),
		Addr:    config.ServerAddress,
	}
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)
	go onExit(exit, &server, storage)

	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}

func onExit(exit <-chan os.Signal, server *http.Server, storage storage.MapStorage) {
	<-exit
	server.Shutdown(context.TODO())
}
