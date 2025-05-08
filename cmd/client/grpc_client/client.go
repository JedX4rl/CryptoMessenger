package grpc_client

import (
	pb "CryptoMessenger/proto/chatpb"
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"sync"
	"time"
)

type ChatClient struct {
	conn        *grpc.ClientConn
	client      pb.ChatServiceClient
	authToken   string
	privateKeys sync.Map
}

func NewChatClient(serverAddr string) (*ChatClient, error) {
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &ChatClient{
		conn:   conn,
		client: pb.NewChatServiceClient(conn),
	}, nil
}

func (c *ChatClient) Close() error {
	return c.conn.Close()
}

// RegisterUser вызывает RPC RegisterUser
func (c *ChatClient) RegisterUser(username, password string) error {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	resp, err := c.client.Register(ctx, &pb.RegisterRequest{Username: username, Password: password})
	if err != nil {
		return err
	}

	c.authToken = resp.Token

	return nil
}

// LoginUser аналогично
func (c *ChatClient) LoginUser(username, password string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	resp, err := c.client.Login(ctx, &pb.LoginRequest{Username: username, Password: password})
	if err != nil {
		return err
	}

	c.authToken = resp.Token

	return nil
}

func (c *ChatClient) CreateChat(username, algo, mode, padding, chatName string) error {
	ctx, cancel := context.WithTimeout(c.AuthenticatedContext(), time.Second*5)
	defer cancel()

	roomId, err := c.client.CreateRoom(ctx, &pb.CreateRoomRequest{Algorithm: algo, Mode: mode, Padding: padding, RoomName: chatName})
	if err != nil {
		return err
	}
	p, g, a, A := "1", "2", "3", "4"

	c.privateKeys.Store(roomId, a+"123")

	_, err = c.client.InviteUser(ctx, &pb.Invitation{
		ReceiverName: username,
		RoomId:       roomId.RoomId,
		Prime:        p,
		G:            g,
		PublicKey:    A,
	})

	return err
}

func (c *ChatClient) AuthenticatedContext() context.Context {
	md := metadata.New(map[string]string{
		"authorization": "Bearer " + c.authToken,
	})
	return metadata.NewOutgoingContext(context.Background(), md)
}
