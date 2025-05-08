package main

import (
	"CryptoMessenger/internal/config/serverConfig"
	"CryptoMessenger/internal/config/storageConfig"
	natsjs "CryptoMessenger/internal/infrastructure/nats"
	"CryptoMessenger/internal/infrastructure/postgres"
	"CryptoMessenger/internal/repository"
	"CryptoMessenger/internal/service"
	"CryptoMessenger/internal/transport/grpc"
	"github.com/joho/godotenv"
	"log"
	"log/slog"
	"os"
)

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		slog.Error("Error loading .env file")
		os.Exit(1)
	}
}

func main() {

	storageCfg, err := storageConfig.MustLoadStorageConfig()
	if err != nil {
		log.Fatalf(err.Error())
	}

	config, err := serverConfig.MustLoadServerConfig()
	if err != nil {
		log.Fatalf(err.Error())
	}

	dataBase, err := postgres.NewStorage(storageCfg)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	repos := repository.NewRepository(dataBase)

	broker := natsjs.NewJSClient("localhost:4222")

	defer broker.Conn.Close()

	chatService := service.NewService(repos, broker)

	if err := grpc.RunGRPCServer(config.Server, chatService); err != nil {
		log.Fatalf("cannot start gRPC server: %v", err)
	}

}
