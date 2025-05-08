package service

import (
	"CryptoMessenger/internal/domain"
	natsjs "CryptoMessenger/internal/infrastructure/nats"
	"CryptoMessenger/internal/repository"
	"context"
	"fmt"
	"github.com/google/uuid"
)

type ChatService struct {
	rooms    repository.RoomRepo
	keys     repository.KeyRepo
	users    repository.UserRepo
	jsClient *natsjs.JSClient
}

func NewChatService(repo repository.RoomRepo, keys repository.KeyRepo, users repository.UserRepo, broker *natsjs.JSClient) *ChatService {
	return &ChatService{rooms: repo, keys: keys, users: users, jsClient: broker}
}

func (s *ChatService) CreateRoom(ctx context.Context, cfg domain.RoomConfig) (string, error) {
	cfg.RoomID = uuid.New().String()
	if err := s.rooms.Create(ctx, cfg); err != nil {
		return "", fmt.Errorf("cannot create room: %w", err)
	}
	return cfg.RoomID, nil
}

func (s *ChatService) InviteUser(ctx context.Context, senderID, receiverName, roomID, prime, g, publicKey string) (string, error) {

	var err error

	sender, err := s.users.GetByID(ctx, senderID)
	if err != nil {
		return "", fmt.Errorf("cannot get sender: %w", err)
	}

	receiver, err := s.users.GetByUsername(ctx, receiverName)
	if err != nil {
		return "", fmt.Errorf("user doesnt't exist: %w", err)
	}

	if err = s.jsClient.EnsureInvitesConsumer(receiver.ID); err != nil {
		return "", fmt.Errorf("failed to init consumer: %w", err)
	}

	//if err = s.rooms.AddMember(ctx, roomID, to); err != nil {
	//	return "", fmt.Errorf("cannot add user to room: %w", err)
	//}

	messageID := uuid.New().String()
	message := &natsjs.InvitationMessage{
		MessageID:    messageID,
		SenderName:   sender.Username,
		ReceiverName: receiver.Username,
		ReceiverID:   receiver.ID,
		RoomID:       roomID,
		Prime:        prime,
		G:            g,
		PublicKey:    publicKey,
	}

	if err = s.jsClient.PublishInvitation(ctx, message); err != nil {
		return "", fmt.Errorf("failed to publish invitation: %w", err)
	}

	return messageID, nil
}

func (s *ChatService) AckInvite(messageID string) error {
	return s.jsClient.AckEvent(messageID)
}

func (s *ChatService) ReceiveInvitation(ctx context.Context, userID string) (domain.ChatInvitation, error) {
	return s.jsClient.FetchOneInvitation(ctx, userID)
}

func (s *ChatService) ReceiveInvitationReaction(ctx context.Context, userID string) (domain.InvitationReaction, error) {
	return s.jsClient.FetchOneInvitationReaction(ctx, userID)
}

func (s *ChatService) ReactToInvitation(ctx context.Context, reaction domain.InvitationReaction) error {
	sender, err := s.users.GetByID(ctx, reaction.SenderId)
	if err != nil {
		return fmt.Errorf("cannot get sender: %w", err)
	}
	receiver, err := s.users.GetByUsername(ctx, reaction.ReceiverName)
	if err != nil {
		return fmt.Errorf("user doesnt't exist: %w", err)
	}
	if err = s.jsClient.EnsureInvitesConsumer(receiver.ID); err != nil {
		return fmt.Errorf("failed to init consumer: %w", err)
	}
	messageID := uuid.New().String()
	message := &natsjs.InvitationMessage{
		MessageID:    messageID,
		SenderName:   sender.Username,
		ReceiverName: receiver.Username,
		ReceiverID:   receiver.ID,
		RoomID:       reaction.RoomID,
		PublicKey:    reaction.PublicKey,
		Accepted:     reaction.Accepted,
	}

	if err = s.jsClient.PublishInvitationReaction(ctx, message); err != nil {
		return fmt.Errorf("failed to publish invitation: %w", err)
	}

	return nil
}

func (s *ChatService) CloseRoom(ctx context.Context, roomID string) error {
	if err := s.rooms.Delete(ctx, roomID); err != nil {
		return fmt.Errorf("cannot close room: %w", err)
	}
	return nil
}

func (s *ChatService) JoinRoom(ctx context.Context, roomID, clientID string) error {
	// ничего не сохраняем по ТЗ
	return nil
}

func (s *ChatService) LeaveRoom(ctx context.Context, roomID, clientID string) error {
	// ничего не сохраняем по ТЗ
	return nil
}

func (s *ChatService) SendInvitation(ctx context.Context, invite domain.ChatInvitation) error {

	return nil
}

func (s *ChatService) SendPublicKey(ctx context.Context, roomID, clientID, pubHex string) error {
	//return s.keys.Store(ctx, roomID, clientID, pubHex)
	return nil
}

func (s *ChatService) GetPublicKeys(ctx context.Context, roomID string) ([]domain.PublicKey, error) {
	//return s.keys.List(ctx, roomID)
	return nil, nil
}

func (s *ChatService) SendMessage(ctx context.Context, msg domain.EncryptedMessage) error {
	// Сериализуем EncryptedMessage в protobuf-байты
	return nil
}

func (s *ChatService) ReceiveMessages(ctx context.Context, roomID, clientID string) (<-chan domain.EncryptedMessage, error) {

	out := make(chan domain.EncryptedMessage)

	return out, nil
}

func (s *ChatService) GetRoomConfig(ctx context.Context, roomID string) (domain.RoomConfig, error) {
	//return s.rooms.GetConfig(ctx, roomID)
	return domain.RoomConfig{}, nil
}
