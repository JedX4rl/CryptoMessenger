package service

import (
	"CryptoMessenger/internal/domain"
	natsjs "CryptoMessenger/internal/infrastructure/nats"
	"CryptoMessenger/internal/repository"
	"context"
)

type Auth interface {
	Register(ctx context.Context, username, password string) (string, error)
	Login(ctx context.Context, username, password string) (string, error)
}

type Chat interface {
	CreateRoom(ctx context.Context, cfg domain.RoomConfig) (string, error)
	CloseRoom(ctx context.Context, roomID string) error
	JoinRoom(ctx context.Context, roomID, clientID string) error
	LeaveRoom(ctx context.Context, roomID, clientID string) error
	SendPublicKey(ctx context.Context, roomID, clientID, pubHex string) error
	GetPublicKeys(ctx context.Context, roomID string) ([]domain.PublicKey, error)
	SendMessage(ctx context.Context, msg domain.EncryptedMessage) error
	ReceiveMessages(ctx context.Context, roomID, clientID string) (<-chan domain.EncryptedMessage, error)
	GetRoomConfig(ctx context.Context, roomID string) (domain.RoomConfig, error)
	SendInvitation(ctx context.Context, invite domain.ChatInvitation) error
	InviteUser(ctx context.Context, invitation domain.ChatInvitation) (string, error)
	ReceiveInvitation(ctx context.Context, userID string) (domain.ChatInvitation, error)
	ReceiveInvitationReaction(ctx context.Context, userID string) (domain.InvitationReaction, error)
	ReactToInvitation(ctx context.Context, reaction domain.InvitationReaction) error
	AckInvite(messageID string) error
}

type Service struct {
	Auth
	Chat
}

func NewService(repositories *repository.Repository, jsClient *natsjs.JSClient) *Service {
	return &Service{
		Auth: NewAuthService(repositories.UserRepo, jsClient),
		Chat: NewChatService(repositories.RoomRepo, repositories.KeyRepo, repositories.UserRepo, jsClient),
	}
}
