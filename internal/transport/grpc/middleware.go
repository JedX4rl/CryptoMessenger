package grpc

import (
	"CryptoMessenger/internal/auth"
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"strings"
)

func AuthInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {

		if info.FullMethod == "/chat.ChatService/Register" || info.FullMethod == "/chat.ChatService/Login" {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		authHeader := md["authorization"]
		if len(authHeader) == 0 {
			return nil, status.Error(codes.Unauthenticated, "authorization header is not provided")
		}

		token := strings.TrimPrefix(authHeader[0], "Bearer ")
		claims, err := auth.ParseToken(token)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
		}

		ctx = context.WithValue(ctx, "client_id", claims.ClientID)
		return handler(ctx, req)
	}
}

func GetClientID(ctx context.Context) (string, error) {
	clientID, ok := ctx.Value("client_id").(string)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing client id")
	}
	return clientID, nil
}
