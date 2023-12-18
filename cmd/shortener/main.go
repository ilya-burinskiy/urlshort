package main

import (
	"context"
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
	if err := logger.Initialize("info"); err != nil {
		panic(err)
	}

	var s storage.Storage
	if config.UseDBStorage() {
		var err error
		s, err = storage.NewDBStorage(config.DatabaseDSN)
		if err != nil {
			panic(err)
		}
	} else if config.UseFileStorage() {
		fs := storage.NewFileStorage(config.FileStoragePath)
		s = storage.NewMapStorage(fs)
		err := fs.Restore(s.(storage.MapStorage))
		if err != nil {
			panic(err)
		}

		go services.StorageDumper(s.(storage.MapStorage), 5*time.Second)
	} else {
		s = storage.NewMapStorage(nil)
	}

	server := http.Server{
		Handler: handlers.ShortenURLRouter(config, rndGen, s),
		Addr:    config.ServerAddress,
	}
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)
	go onExit(exit, &server, s)

	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		panic(err)
	}

}

func onExit(exit <-chan os.Signal, server *http.Server, s storage.Storage) {
	<-exit
	switch s := s.(type) {
	case storage.MapStorage:
		err := s.Dump()
		if err != nil {
			logger.Log.Info("on exit error", zap.String("err", err.Error()))
		}
	case *storage.DBStorage:
		s.Close()
	}

	server.Shutdown(context.TODO())
}
