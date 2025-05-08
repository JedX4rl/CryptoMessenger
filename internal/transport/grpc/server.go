package grpc

import (
	"CryptoMessenger/internal/config/serverConfig"
	"CryptoMessenger/internal/service"
	"fmt"
	"google.golang.org/grpc"
	"net"

	pb "CryptoMessenger/proto/chatpb"
)

func RunGRPCServer(config serverConfig.ServerConfig, service *service.Service) error {
	lis, err := net.Listen(config.Type, config.Address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", config.Address, err)
	}

	srv := grpc.NewServer(
		grpc.UnaryInterceptor(AuthInterceptor()))
	pb.RegisterChatServiceServer(srv, NewChatHandler(service))

	fmt.Printf("gRPC server listening at %s\n", config.Address)

	return srv.Serve(lis)
}
