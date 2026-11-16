package grpcapi

import (
	"context"
	"crypto/subtle"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// ServiceTokenInterceptor 校验服务间调用的共享密钥，防止未授权微服务访问。
func ServiceTokenInterceptor(expected string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "service token is missing")
		}
		values := md.Get("authorization")
		if len(values) == 0 {
			return nil, status.Error(codes.Unauthenticated, "service token is missing")
		}
		provided := strings.TrimPrefix(values[0], "Bearer ")
		// 使用常量时间比较，避免通过响应时间探测密钥。
		if subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
			return nil, status.Error(codes.Unauthenticated, "invalid service token")
		}
		return handler(ctx, req)
	}
}
