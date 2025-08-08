package grpc

import (
	"context"
	"errors"
	"strconv"

	"github.com/ilya-burinskiy/urlshort/internal/app/configs"
	"github.com/ilya-burinskiy/urlshort/internal/app/logger"
	"github.com/ilya-burinskiy/urlshort/internal/app/models"
	"github.com/ilya-burinskiy/urlshort/internal/app/services"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// URLsServer
type URLsServer struct {
	UnimplementedURLServiceServer
	config            configs.Config
	store             storage.Storage
	userAuthenticator services.UserAuthenticator
	shortener  services.URLShortener
	urlDeleter        services.BatchDeleter
}

// NewURLsServer
func NewURLsServer(
	config configs.Config,
	store storage.Storage,
	userAuthenticator services.UserAuthenticator,
	shortener services.URLShortener,
	urlDeleter services.BatchDeleter) URLsServer {

	return URLsServer{
		config:            config,
		store:             store,
		userAuthenticator: userAuthenticator,
		shortener:  shortener,
		urlDeleter:        urlDeleter,
	}
}

// CreateURL
func (s URLsServer) CreateURL(ctx context.Context, in *CreateURLRequest) (*CreateURLResponse, error) {
	jwtStr := getJWT(ctx)
	user, jwtStr, err := s.userAuthenticator.AuthOrRegister(ctx, jwtStr)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if err := grpc.SendHeader(ctx, metadata.New(map[string]string{"jwt": jwtStr})); err != nil {
		return nil, status.Error(codes.Internal, "failed to set JWT")
	}

	record, err := s.shortener.Shortify(in.OriginalUrl, user)
	if err != nil {
		var notUniqErr *storage.ErrNotUnique
		if errors.As(err, &notUniqErr) {
			return nil, status.Errorf(codes.AlreadyExists, "")
		}
		return nil, status.Error(codes.Internal, "failed to create url")
	}

	return &CreateURLResponse{ShortUrl: s.config.BaseURL + "/" + record.ShortenedPath}, nil
}

// GetOriginalURL
func (s URLsServer) GetOriginalURL(ctx context.Context, in *GetOriginalURLRequest) (*GetOriginalURLResponse, error) {
	record, err := s.store.FindByShortenedPath(ctx, in.ShortUrl)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, status.Errorf(codes.NotFound, "original URL for \"%s\" not found", in.ShortUrl)
	}
	if record.IsDeleted {
		return nil, status.Error(codes.NotFound, "deleted")
	}

	return &GetOriginalURLResponse{OriginalUrl: record.OriginalURL}, nil
}

// BatchCreateURL
func (s URLsServer) BatchCreateURL(ctx context.Context, in *BatchCreateURLRequest) (*BatchCreateURLResponse, error) {
	jwtStr := getJWT(ctx)
	user, jwtStr, err := s.userAuthenticator.AuthOrRegister(ctx, jwtStr)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if err := grpc.SendHeader(ctx, metadata.New(map[string]string{"jwt": jwtStr})); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	records := make([]models.Record, len(in.Items))
	for i, item := range in.Items {
		records[i] = models.Record{OriginalURL: item.OriginalUrl, CorrelationID: item.CorrelationId}
	}
	savedRecords, err := s.shortener.BatchShortify(records, user)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	responseItems := make([]*BatchCreateURLResponse_Item, len(savedRecords))
	for i, record := range savedRecords {
		responseItems[i] = &BatchCreateURLResponse_Item{
			CorrelationId: record.CorrelationID,
			ShortUrl:      s.config.BaseURL + "/" + record.ShortenedPath,
		}
	}

	return &BatchCreateURLResponse{Items: responseItems}, nil
}

// GetUserURLs. User must be authenticated
func (s URLsServer) GetUserURLs(ctx context.Context, in *GetUserURLsRequest) (*GetUserURLsResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	userID, _ := strconv.Atoi(md.Get("user_id")[0])
	records, err := s.store.FindByUser(ctx, models.User{ID: userID})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	responseItems := make([]*GetUserURLsResponse_Item, len(records))
	for i, record := range records {
		responseItems[i] = &GetUserURLsResponse_Item{
			OriginalUrl: record.OriginalURL,
			ShortUrl:    s.config.BaseURL + "/" + record.ShortenedPath,
		}
	}

	return &GetUserURLsResponse{Items: responseItems}, nil
}

// DeleteUserURLs. User must be authenticated
func (s URLsServer) DeleteUserURLs(ctx context.Context, in *DeleteUserURLsRequest) (*DeleteUserURLsResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	userID, _ := strconv.Atoi(md.Get("user_id")[0])
	for _, shortPath := range in.ShortUrls {
		s.urlDeleter.Delete(models.Record{
			ShortenedPath: shortPath,
			UserID:        userID,
		})
	}

	return &DeleteUserURLsResponse{}, nil
}

// GetStats
func (s URLsServer) GetStats(ctx context.Context, in *GetStatsRequest) (*GetStatsResponse, error) {
	usersCount, err := s.store.UsersCount(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	urlsCount, err := s.store.URLsCount(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &GetStatsResponse{Urls: uint64(urlsCount), Users: uint64(usersCount)}, nil
}

// PingDB
func (s URLsServer) PingDB(ctx context.Context, in *PingDBRequest) (*PingDBResponse, error) {
	conn, err := pgx.Connect(ctx, s.config.DatabaseDSN)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	defer func() {
		if err = conn.Close(ctx); err != nil {
			logger.Log.Info("failed to close db connection", zap.Error(err))
		}
	}()

	return &PingDBResponse{}, nil
}

func getJWT(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	values := md.Get("jwt")
	if len(values) == 0 {
		return ""
	}

	return values[0]
}
