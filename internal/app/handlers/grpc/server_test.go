package grpc_test

import (
	"context"
	"errors"
	"log"
	"net"

	"testing"

	"github.com/golang/mock/gomock"
	"github.com/ilya-burinskiy/urlshort/internal/app/auth"
	"github.com/ilya-burinskiy/urlshort/internal/app/models"
	"github.com/ilya-burinskiy/urlshort/internal/app/services"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/ilya-burinskiy/urlshort/internal/app/configs"
	pb "github.com/ilya-burinskiy/urlshort/internal/app/handlers/grpc"
)

var defaultConfig = configs.Config{
	BaseURL:         "http://localhost:8080",
	ServerAddress:   "http://localhost:8080",
	FileStoragePath: "storage",
}

type urlShortenerMock struct{ mock.Mock }

func (m *urlShortenerMock) Shortify(origURL string, user models.User) (models.Record, error) {
	args := m.Called(origURL, user)
	return args.Get(0).(models.Record), args.Error(1)
}

func (m *urlShortenerMock) BatchShortify(records []models.Record, user models.User) ([]models.Record, error) {
	args := m.Called(records, user)
	return args.Get(0).([]models.Record), args.Error(1)
}

type userAuthenticatorMock struct{ mock.Mock }

func (m *userAuthenticatorMock) AuthOrRegister(ctx context.Context, jwtStr string) (models.User, string, error) {
	args := m.Called(ctx, jwtStr)
	return args.Get(0).(models.User), args.String(1), args.Error(2)
}

func (m *userAuthenticatorMock) Auth(jwtStr string) (models.User, error) {
	args := m.Called(jwtStr)
	return args.Get(0).(models.User), args.Error(1)
}

func TestCreateURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mocks.NewMockStorage(ctrl)
	urlCreateService := new(urlShortenerMock)
	userAuthenticator := new(userAuthenticatorMock)
	user := models.User{ID: 1}
	userAuthenticator.On("AuthOrRegister", mock.Anything, mock.Anything).Return(
		user, "123", nil,
	)
	userAuthenticator.On("Auth", mock.Anything).Return(
		user, nil,
	)
	urlDeleter := services.NewDeferredDeleter(store)
	srvCloser := startServer(defaultConfig, store, userAuthenticator, urlCreateService, urlDeleter)
	defer srvCloser()

	client, closer := getClient()
	defer closer()

	type want struct {
		out *pb.CreateURLResponse
		err error
	}
	type createURLResult struct {
		record models.Record
		err    error
	}
	testCases := []struct {
		name      string
		in        *pb.CreateURLRequest
		createRes createURLResult
		want      want
	}{
		{
			name: "responds with ok",
			in:   &pb.CreateURLRequest{OriginalUrl: "http://example.com"},
			createRes: createURLResult{
				record: models.Record{ShortenedPath: "123"},
				err:    nil,
			},
			want: want{
				out: &pb.CreateURLResponse{ShortUrl: defaultConfig.BaseURL + "/" + "123"},
			},
		},
		{
			name: "responds with already exists status",
			in:   &pb.CreateURLRequest{OriginalUrl: "http://example.com"},
			createRes: createURLResult{
				record: models.Record{OriginalURL: "http://example.com", ShortenedPath: "123"},
				err: &storage.ErrNotUnique{
					Record: models.Record{OriginalURL: "http://example.com", ShortenedPath: "123"},
				},
			},
			want: want{
				out: &pb.CreateURLResponse{ShortUrl: defaultConfig.BaseURL + "/" + "123"},
				err: status.Error(codes.AlreadyExists, ""),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			mockCall := urlCreateService.On("Shortify", mock.Anything, mock.Anything).Return(
				tc.createRes.record, tc.createRes.err,
			)
			defer mockCall.Unset()

			out, err := client.CreateURL(ctx, tc.in)
			if err != nil {
				expectedErrStatus, ok := status.FromError(tc.want.err)
				require.True(t, ok)
				actualErrStatus, ok := status.FromError(err)
				require.True(t, ok)

				assert.Equal(t, expectedErrStatus, actualErrStatus)
				assert.Equal(t, tc.want.err.Error(), err.Error())
			}
			if out != nil {
				assert.Equal(t, tc.want.out.ShortUrl, out.ShortUrl)
			}
		})
	}
}

func TestGetOriginalURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mocks.NewMockStorage(ctrl)
	store.EXPECT().FindByShortenedPath(gomock.Any(), gomock.Any()).Return(models.Record{OriginalURL: "http://example.com"}, nil)
	store.EXPECT().FindByShortenedPath(gomock.Any(), gomock.Any()).Return(models.Record{}, storage.ErrNotFound)
	store.EXPECT().FindByShortenedPath(gomock.Any(), gomock.Any()).Return(models.Record{IsDeleted: true}, nil)
	urlCreateService := new(urlShortenerMock)
	userAuthenticator := new(userAuthenticatorMock)
	userAuthenticator.On("AuthOrRegister", mock.Anything, mock.Anything).Return(
		models.User{ID: 1}, "123", nil,
	)
	urlDeleter := services.NewDeferredDeleter(store)
	srvCloser := startServer(defaultConfig, store, userAuthenticator, urlCreateService, urlDeleter)
	defer srvCloser()

	client, closer := getClient()
	defer closer()

	type want struct {
		out *pb.GetOriginalURLResponse
		err error
	}
	testCases := []struct {
		name string
		in   *pb.GetOriginalURLRequest
		want want
	}{
		{
			name: "responds with ok status",
			in:   &pb.GetOriginalURLRequest{ShortUrl: "123"},
			want: want{
				out: &pb.GetOriginalURLResponse{OriginalUrl: "http://example.com"},
			},
		},
		{
			name: "responds with not found status",
			in:   &pb.GetOriginalURLRequest{ShortUrl: "123"},
			want: want{
				err: status.Error(codes.NotFound, "original URL for \"123\" not found"),
			},
		},
		{
			name: "responds with not found status if URL is deleted",
			in:   &pb.GetOriginalURLRequest{ShortUrl: "123"},
			want: want{
				err: status.Error(codes.NotFound, "deleted"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			out, err := client.GetOriginalURL(ctx, tc.in)
			if err != nil {
				expectedErrStatus, ok := status.FromError(tc.want.err)
				require.True(t, ok)
				actualErrStatus, ok := status.FromError(err)
				require.True(t, ok)

				assert.Equal(t, expectedErrStatus, actualErrStatus)
				assert.Equal(t, tc.want.err.Error(), err.Error())
			}
			if out != nil {
				assert.Equal(t, tc.want.out.OriginalUrl, out.OriginalUrl)
			}
		})
	}
}

func TestBatchCreateURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mocks.NewMockStorage(ctrl)
	store.EXPECT().CreateUser(gomock.Any()).AnyTimes().Return(models.User{ID: 1}, nil)
	urlCreateService := new(urlShortenerMock)
	userAuthenticator := new(userAuthenticatorMock)
	userAuthenticator.On("AuthOrRegister", mock.Anything, mock.Anything).Return(
		models.User{ID: 1}, "123", nil,
	)
	urlDeleter := services.NewDeferredDeleter(store)
	srvCloser := startServer(defaultConfig, store, userAuthenticator, urlCreateService, urlDeleter)
	defer srvCloser()

	client, closer := getClient()
	defer closer()

	type want struct {
		out *pb.BatchCreateURLResponse
		err error
	}
	type batchCreateURLResult struct {
		records []models.Record
		err     error
	}
	testCases := []struct {
		name      string
		in        *pb.BatchCreateURLRequest
		createRes batchCreateURLResult
		want      want
	}{
		{
			name: "responds with ok",
			in: &pb.BatchCreateURLRequest{
				Items: []*pb.BatchCreateURLRequest_Item{
					{OriginalUrl: "http://example0.com", CorrelationId: "1"},
					{OriginalUrl: "http://example1.com", CorrelationId: "2"},
				},
			},
			createRes: batchCreateURLResult{
				records: []models.Record{
					{OriginalURL: "http://example0.com", CorrelationID: "1", ShortenedPath: "1"},
					{OriginalURL: "http://example1.com", CorrelationID: "2", ShortenedPath: "2"},
				},
			},
			want: want{
				out: &pb.BatchCreateURLResponse{
					Items: []*pb.BatchCreateURLResponse_Item{
						{CorrelationId: "1", ShortUrl: defaultConfig.BaseURL + "/1"},
						{CorrelationId: "2", ShortUrl: defaultConfig.BaseURL + "/2"},
					},
				},
			},
		},
		{
			name: "responds with invalid argument status",
			in: &pb.BatchCreateURLRequest{
				Items: []*pb.BatchCreateURLRequest_Item{
					{OriginalUrl: "http://example0.com", CorrelationId: "1"},
					{OriginalUrl: "http://example1.com", CorrelationId: "2"},
				},
			},
			createRes: batchCreateURLResult{err: errors.New("error")},
			want: want{
				err: status.Error(codes.InvalidArgument, "error"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			mockCall := urlCreateService.On("BatchShortify", mock.Anything, mock.Anything).Return(
				tc.createRes.records, tc.createRes.err,
			)
			defer mockCall.Unset()

			out, err := client.BatchCreateURL(ctx, tc.in)
			if err != nil {
				expectedErrStatus, ok := status.FromError(tc.want.err)
				require.True(t, ok)
				actualErrStatus, ok := status.FromError(err)
				require.True(t, ok)

				assert.Equal(t, expectedErrStatus, actualErrStatus)
				assert.Equal(t, tc.want.err.Error(), err.Error())
			}
			if out != nil {
				assert.Equal(t, tc.want.out.Items, out.Items)
			}
		})
	}
}

func TestGetUserURLS(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mocks.NewMockStorage(ctrl)
	urlCreateService := new(urlShortenerMock)
	urlDeleter := services.NewDeferredDeleter(store)
	userAuthenticator := new(userAuthenticatorMock)
	user := models.User{ID: 1}
	userAuthenticator.On("AuthOrRegister", mock.Anything, mock.Anything).Return(
		user, "123", nil,
	)
	userAuthenticator.On("Auth", mock.Anything).Return(
		user, nil,
	)
	srvCloser := startServer(defaultConfig, store, userAuthenticator, urlCreateService, urlDeleter)
	defer srvCloser()

	userID := 1
	client, closer := getClient(authInterceptor(userID))
	defer closer()

	store.EXPECT().FindByUser(gomock.Any(), gomock.Any()).
		Return([]models.Record{
			{OriginalURL: "http://example0.com", ShortenedPath: "1"},
			{OriginalURL: "http://example1.com", ShortenedPath: "2"},
		}, nil)
	store.EXPECT().FindByUser(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	type want struct {
		out *pb.GetUserURLsResponse
		err error
	}
	testCases := []struct {
		name string
		in   *pb.GetUserURLsRequest
		want want
	}{
		{
			name: "responds with ok status",
			in:   &pb.GetUserURLsRequest{},
			want: want{
				out: &pb.GetUserURLsResponse{
					Items: []*pb.GetUserURLsResponse_Item{
						{OriginalUrl: "http://example0.com", ShortUrl: defaultConfig.BaseURL + "/1"},
						{OriginalUrl: "http://example1.com", ShortUrl: defaultConfig.BaseURL + "/2"},
					},
				},
			},
		},
		{
			name: "responds with internal error status",
			in:   &pb.GetUserURLsRequest{},
			want: want{
				err: status.Error(codes.Internal, "error"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			out, err := client.GetUserURLs(ctx, tc.in)
			if err != nil {
				expectedErrStatus, ok := status.FromError(tc.want.err)
				require.True(t, ok)
				actualErrStatus, ok := status.FromError(err)
				require.True(t, ok)

				assert.Equal(t, expectedErrStatus, actualErrStatus)
				assert.Equal(t, tc.want.err.Error(), err.Error())
			}
			if out != nil {
				assert.Equal(t, tc.want.out.Items, out.Items)
			}
		})
	}
}

func TestDeleteUserURLs(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mocks.NewMockStorage(ctrl)
	urlCreateService := new(urlShortenerMock)
	userAuthenticator := new(userAuthenticatorMock)
	user := models.User{ID: 1}
	userAuthenticator.On("AuthOrRegister", mock.Anything, mock.Anything).Return(
		user, "123", nil,
	)
	userAuthenticator.On("Auth", mock.Anything).Return(
		user, nil,
	)
	urlDeleter := services.NewDeferredDeleter(store)
	srvCloser := startServer(defaultConfig, store, userAuthenticator, urlCreateService, urlDeleter)
	defer srvCloser()

	userID := 1
	client, closer := getClient(authInterceptor(userID))
	defer closer()

	type want struct {
		out *pb.DeleteUserURLsResponse
		err error
	}
	testCases := []struct {
		name string
		in   *pb.DeleteUserURLsRequest
		want want
	}{
		{
			name: "responds with ok status",
			in: &pb.DeleteUserURLsRequest{
				ShortUrls: []string{"1", "2"},
			},
			want: want{
				out: &pb.DeleteUserURLsResponse{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			_, err := client.DeleteUserURLs(ctx, tc.in)
			if err != nil {
				expectedErrStatus, ok := status.FromError(tc.want.err)
				require.True(t, ok)
				actualErrStatus, ok := status.FromError(err)
				require.True(t, ok)

				assert.Equal(t, expectedErrStatus, actualErrStatus)
				assert.Equal(t, tc.want.err.Error(), err.Error())
			}
		})
	}
}

func startServer(
	config configs.Config,
	store storage.Storage,
	userAuthenticator services.UserAuthenticator,
	urlCreateService services.URLShortener,
	urlDeleter services.DeferredDeleter) func() {

	listen, err := net.Listen("tcp", ":3200")
	if err != nil {
		log.Fatal(err)
	}

	srv := grpc.NewServer(
		grpc.UnaryInterceptor(pb.AuthenticateInterceptor(userAuthenticator)),
	)
	pb.RegisterURLServiceServer(srv, pb.NewURLsServer(
		config,
		store,
		userAuthenticator,
		urlCreateService,
		urlDeleter,
	))
	go func() {
		if err := srv.Serve(listen); err != nil {
			log.Fatal(err)
		}
	}()

	return srv.Stop
}

func getClient(unaryInterceptors ...grpc.UnaryClientInterceptor) (pb.URLServiceClient, func()) {
	conn, err := grpc.Dial(
		":3200",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(unaryInterceptors...),
	)
	if err != nil {
		log.Fatal(err)
	}

	closer := func() {
		if err := conn.Close(); err != nil {
			log.Fatal(err)
		}
	}

	return pb.NewURLServiceClient(conn), closer
}

func authInterceptor(userID int) func(
	context.Context,
	string,
	interface{},
	interface{},
	*grpc.ClientConn,
	grpc.UnaryInvoker,
	...grpc.CallOption) error {

	jwtStr, err := auth.BuildJWTString(models.User{ID: userID})
	if err != nil {
		log.Fatal(err)
	}
	return func(
		ctx context.Context,
		method string,
		req interface{},
		reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption) error {

		md := metadata.New(map[string]string{"jwt": jwtStr})
		return invoker(
			metadata.NewOutgoingContext(ctx, md),
			method,
			req,
			reply,
			cc,
			opts...,
		)
	}
}
