package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"
	"google.golang.org/grpc"

	"github.com/ilya-burinskiy/urlshort/internal/app/configs"
	"github.com/ilya-burinskiy/urlshort/internal/app/handlers"
	pb "github.com/ilya-burinskiy/urlshort/internal/app/handlers/grpc"
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
	urlCreateService := services.NewURLShortener(
		8,
		services.RandHexStrGenerator{},
		store,
	)
	userAuthenticator := services.NewUserAuthenticator(store)
	urlDeleter := services.NewDeferredDeleter(store)
	ipChecker := services.NewIPChecker(config)
	go urlDeleter.Run()
	go startGRPCServer(config, store, userAuthenticator, ipChecker, urlCreateService, urlDeleter)
	startHTTPServer(config, store, userAuthenticator, ipChecker, urlCreateService, urlDeleter)
}

func startHTTPServer(
	config configs.Config,
	store storage.Storage,
	userAuthenticator services.UserAuthenticator,
	ipChecker services.IPChecker,
	shortener services.URLShortener,
	urlDeleter services.DeferredDeleter) {

	server := http.Server{
		Handler: configureRouter(store, config, userAuthenticator, ipChecker, shortener, urlDeleter),
		Addr:    config.ServerAddress,
	}
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
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

func startGRPCServer(
	config configs.Config,
	store storage.Storage,
	userAuthenticator services.UserAuthenticator,
	ipChecker services.IPChecker,
	shortener services.URLShortener,
	urlDeleter services.DeferredDeleter) {

	listen, err := net.Listen("tcp", config.GRPCServerAddress)
	if err != nil {
		panic(err)
	}
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			pb.AuthenticateInterceptor(userAuthenticator),
			pb.TrustedIPInterceptor(ipChecker),
		),
	)
	pb.RegisterURLServiceServer(
		srv,
		pb.NewURLsServer(config, store, userAuthenticator, shortener, urlDeleter),
	)
	if err := srv.Serve(listen); err != nil {
		panic(err)
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
	store storage.Storage,
	config configs.Config,
	userAuthenticator services.UserAuthenticator,
	ipChecker services.IPChecker,
	shortener services.URLShortener,
	urlDeleter services.DeferredDeleter) chi.Router {

	router := chi.NewRouter()
	handlers := handlers.NewHandlers(config, store)
	router.Use(
		middlewares.ResponseLogger,
		middlewares.RequestLogger,
		middlewares.GzipCompress,
		middleware.AllowContentEncoding("gzip"),
	)
	router.Group(func(router chi.Router) {
		router.Use(middleware.AllowContentType("text/plain", "application/x-gzip"))
		router.Post("/", handlers.CreateURL(shortener, userAuthenticator))
		router.Get("/{id}", handlers.GetOriginalURL)
		router.Get("/ping", handlers.PingDB)
	})
	router.Group(func(router chi.Router) {
		router.Use(middleware.AllowContentType("application/json", "application/x-gzip"))
		router.Post("/api/shorten", handlers.CreateURLFromJSON(shortener, userAuthenticator))
		router.Post("/api/shorten/batch", handlers.BatchCreateURL(shortener, userAuthenticator))
		router.Group(func(router chi.Router) {
			router.Use(middlewares.Authenticate(userAuthenticator))
			router.Get("/api/user/urls", handlers.GetUserURLs)
			router.Delete("/api/user/urls", handlers.DeleteUserURLs(urlDeleter))
		})
	})
	router.Group(func(router chi.Router) {
		router.Use(middlewares.OnlyTrustedIP(ipChecker), middleware.AllowContentType("application/json"))
		router.Get("/api/internal/stats", handlers.GetStats)
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
