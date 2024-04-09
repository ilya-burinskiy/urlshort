package grpc

import (
	"context"
	"errors"
	"net"
	"strconv"

	"github.com/ilya-burinskiy/urlshort/internal/app/configs"
	"github.com/ilya-burinskiy/urlshort/internal/app/services"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var ingnoreAuthMethods = []string{
	URLService_CreateURL_FullMethodName,
	URLService_GetOriginalURL_FullMethodName,
	URLService_BatchCreateURL_FullMethodName,
	URLService_PingDB_FullMethodName,
}

var ingoreIPCheckMethods = []string{
	URLService_CreateURL_FullMethodName,
	URLService_GetOriginalURL_FullMethodName,
	URLService_BatchCreateURL_FullMethodName,
	URLService_GetUserURLs_FullMethodName,
	URLService_DeleteUserURLs_FullMethodName,
	URLService_PingDB_FullMethodName,
}

// AuthenticateInterceptor
func AuthenticateInterceptor(userAuthenticator services.UserAuthService) func(
	context.Context,
	interface{},
	*grpc.UnaryServerInfo,
	grpc.UnaryHandler) (interface{}, error) {

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		method, _ := grpc.Method(ctx)
		for _, imethod := range ingnoreAuthMethods {
			if method == imethod {
				return handler(ctx, req)
			}
		}

		meta, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing token")
		}
		values := meta.Get("jwt")
		if len(values) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing token")
		}

		user, err := userAuthenticator.Auth(values[0])
		if errors.Is(err, services.ErrInvalidJWT) {
			return nil, status.Error(codes.Unauthenticated, "invalid jwt")
		}
		meta.Append("user_id", strconv.Itoa(user.ID))
		ctx = metadata.NewIncomingContext(ctx, meta)

		return handler(ctx, req)
	}
}

// TrustedIPInterceptor
func TrustedIPInterceptor(config configs.Config) func(context.Context, interface{}, *grpc.UnaryServerInfo, grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		method, _ := grpc.Method(ctx)
		for _, imethod := range ingoreIPCheckMethods {
			if method == imethod {
				return handler(ctx, req)
			}
		}
		_, ipv4Net, err := net.ParseCIDR(config.TrustedSubnet)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to parse trusted subnet CIDR")
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.PermissionDenied, "missing \"x-real-ip\"")
		}
		values := md.Get("x-real-ip")
		if len(values) == 0 {
			return nil, status.Error(codes.PermissionDenied, "missing \"x-real-ip\"")
		}

		realIP := net.ParseIP(values[0])
		if !ipv4Net.Contains(realIP) {
			return nil, status.Error(codes.PermissionDenied, "forbidden")
		}

		return handler(ctx, req)
	}
}
