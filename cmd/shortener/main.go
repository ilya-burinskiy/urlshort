package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"

	"github.com/ilya-burinskiy/urlshort/internal/app/configs"
	"github.com/ilya-burinskiy/urlshort/internal/app/handlers"
	"github.com/ilya-burinskiy/urlshort/internal/app/logger"
	"github.com/ilya-burinskiy/urlshort/internal/app/middlewares"
	"github.com/ilya-burinskiy/urlshort/internal/app/services"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage"
)

var (
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
)

func main() {
	config := configs.Parse()
	if err := logger.Initialize("info"); err != nil {
		panic(err)
	}
	showBuildInfo()

	store := configureStorage(config)
	urlCreateService := services.NewCreateURLService(
		8,
		services.StdRandHexStringGenerator{},
		store,
	)
	urlDeleter := services.NewBatchDeleter(store)
	go urlDeleter.Run()

	server := http.Server{
		Handler: configureRouter(config, urlCreateService, urlDeleter, store),
		Addr:    config.ServerAddress,
	}
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)
	go onExit(exit, &server, store)

	var serveErr error
	if config.UseHTTPS() {
		manager := &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist("urlshort.ru"),
		}
		server.TLSConfig = manager.TLSConfig()
		serveErr = server.ListenAndServeTLS("", "")
	} else {
		serveErr = server.ListenAndServe()
	}

	if serveErr != nil && serveErr != http.ErrServerClosed {
		panic(serveErr)
	}
}

func onExit(exit <-chan os.Signal, server *http.Server, s storage.Storage) {
	<-exit
	switch s := s.(type) {
	case *storage.MapStorage:
		err := s.Dump()
		if err != nil {
			logger.Log.Info("on exit error", zap.String("err", err.Error()))
		}
	case *storage.DBStorage:
		s.Close()
	}

	if err := server.Shutdown(context.TODO()); err != nil {
		logger.Log.Info("failed to shutdown", zap.Error(err))
	}
}

func configureRouter(
	config configs.Config,
	urlCreateService services.CreateURLService,
	urlDeleter services.BatchDeleter,
	s storage.Storage) chi.Router {

	router := chi.NewRouter()
	handlers := handlers.NewHandlers(config, s)
	router.Use(
		middlewares.ResponseLogger,
		middlewares.RequestLogger,
		middlewares.GzipCompress,
		middleware.AllowContentEncoding("gzip"),
	)
	router.Group(func(router chi.Router) {
		router.Use(middleware.AllowContentType("text/plain", "application/x-gzip"))
		router.Post("/", handlers.CreateURL(urlCreateService))
		router.Get("/{id}", handlers.GetOriginalURL)
		router.Get("/ping", handlers.PingDB)
	})
	router.Group(func(router chi.Router) {
		router.Use(middleware.AllowContentType("application/json", "application/x-gzip"))
		router.Post("/api/shorten", handlers.CreateURLFromJSON(urlCreateService))
		router.Post("/api/shorten/batch", handlers.BatchCreateURL(urlCreateService))
		router.Group(func(router chi.Router) {
			router.Use(middlewares.Authenticate)
			router.Get("/api/user/urls", handlers.GetUserURLs)
			router.Delete("/api/user/urls", handlers.DeleteUserURLs(urlDeleter))
		})
	})

	return router
}

func configureStorage(config configs.Config) storage.Storage {
	var store storage.Storage
	if config.UseDBStorage() {
		var err error
		store, err = storage.NewDBStorage(config.DatabaseDSN)
		if err != nil {
			panic(err)
		}
	} else if config.UseFileStorage() {
		fs := storage.NewFileStorage(config.FileStoragePath)
		store = storage.NewMapStorage(fs)
		records, err := fs.Snapshot()
		if err != nil {
			panic(err)
		}
		store.(*storage.MapStorage).Restore(records)
		dumper := services.NewStorageDumper(store.(*storage.MapStorage), 5*time.Second)
		dumper.Start()
	} else {
		store = storage.NewMapStorage(nil)
	}

	return store
}

func showBuildInfo() {
	logger.Log.Info("build info", zap.String("build version", buildVersion))
	logger.Log.Info("build info", zap.String("build date", buildDate))
	logger.Log.Info("build info", zap.String("build commit", buildCommit))
}
