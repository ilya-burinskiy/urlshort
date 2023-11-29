package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ilya-burinskiy/urlshort/internal/app/configs"
	"github.com/ilya-burinskiy/urlshort/internal/app/handlers"
	"github.com/ilya-burinskiy/urlshort/internal/app/logger"
	"github.com/ilya-burinskiy/urlshort/internal/app/services"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage"
	"go.uber.org/zap"
)

func main() {
	config := configs.Parse()
	rndGen := services.StdRandHexStringGenerator{}

	storage := storage.New(config.FileStoragePath)
	err := storage.Load()
	if err != nil {
		panic(err)
	}
	go func() {
		for {
			storage.Dump()
			time.Sleep(5 * time.Second)
		}
	}()

	if err := logger.Initialize("info"); err != nil {
		panic(err)
	}

	server := http.Server{
		Handler: handlers.ShortenURLRouter(config, rndGen, storage),
		Addr:    config.ServerAddress,
	}
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		err = server.ListenAndServe()
		if err != nil {
			panic(err)
		}
	}()

	<-exit
	err = storage.Dump()
	if err != nil {
		logger.Log.Info("dump error", zap.String("msg", err.Error()))
	}
}
