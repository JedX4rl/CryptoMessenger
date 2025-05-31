package service

import (
	"CryptoMessenger/internal/domain"
	natsjs "CryptoMessenger/internal/infrastructure/nats"
	"CryptoMessenger/internal/repository"
	"context"
	"fmt"
	"github.com/google/uuid"
	"log/slog"
)

type ChatService struct {
	rooms    repository.RoomRepo
	keys     repository.KeyRepo
	users    repository.UserRepo
	jsClient *natsjs.JSClient
}

func NewChatService(repo repository.RoomRepo, keys repository.KeyRepo, users repository.UserRepo, jsClient *natsjs.JSClient) *ChatService {
	return &ChatService{rooms: repo, keys: keys, users: users, jsClient: jsClient}
}

func (s *ChatService) CreateRoom(ctx context.Context, cfg domain.RoomConfig) (string, error) {
	cfg.RoomID = uuid.New().String()

	if err := s.rooms.Create(ctx, cfg); err != nil {
		return "", fmt.Errorf("cannot create room: %w", err)
	}
	return cfg.RoomID, nil
}

func (s *ChatService) InviteUser(ctx context.Context, invitation domain.ChatInvitation) (string, error) {

	var err error
	slog.Info(invitation.RoomID)

	sender, err := s.users.GetByID(ctx, invitation.SenderID)
	if err != nil {
		return "", fmt.Errorf("cannot get sender: %w", err)
	}

	receiver, err := s.users.GetByUsername(ctx, invitation.ReceiverName)
	if err != nil {
		return "", fmt.Errorf("user doesnt't exist")
	}

	if sender.Username == receiver.Username {
		return "", fmt.Errorf("sender and receiver cannot be the same user")
	}

	messageID := uuid.New().String()
	invitation.SenderName = sender.Username
	invitation.ReceiverID = receiver.ID
	invitation.MessageID = messageID

	if err = s.jsClient.PublishInvitation(ctx, invitation); err != nil {
		return "", fmt.Errorf("failed to publish invitation: %w", err)
	}

	if err = s.jsClient.EnsureMessagesConsumer(invitation.SenderID, messageID); err != nil {
		return "", fmt.Errorf("failed to ensure messages: %w", err)
	}

	return messageID, nil
}

func (s *ChatService) AckEvent(messageID string) error {
	return s.jsClient.AckEvent(messageID)
}

func (s *ChatService) ReceiveInvitation(ctx context.Context, userID string) (domain.ChatInvitation, error) {
	return s.jsClient.FetchOneInvitation(ctx, userID)
}

func (s *ChatService) ReceiveInvitationReaction(ctx context.Context, userID string) (domain.InvitationReaction, error) {
	return s.jsClient.FetchOneInvitationReaction(ctx, userID)
}

func (s *ChatService) ReactToInvitation(ctx context.Context, reaction domain.InvitationReaction) error {
	sender, err := s.users.GetByID(ctx, reaction.SenderID)
	if err != nil {
		return fmt.Errorf("cannot get sender: %w", err)
	}
	receiver, err := s.users.GetByUsername(ctx, reaction.ReceiverName)
	if err != nil {
		return fmt.Errorf("user doesnt't exist: %w", err)
	}
	messageID := uuid.New().String()

	slog.Info("roomID", reaction.RoomID)

	reaction.MessageID = messageID
	reaction.SenderName = sender.Username
	reaction.ReceiverName = receiver.Username
	reaction.ReceiverID = receiver.ID

	if err = s.jsClient.PublishInvitationReaction(ctx, reaction); err != nil {
		return fmt.Errorf("failed to publish invitation: %w", err)
	}

	if reaction.Accepted {
		if err = s.jsClient.EnsureMessagesConsumer(reaction.SenderID, reaction.RoomID); err != nil {
			return fmt.Errorf("failed to ensure messages: %w", err)
		}
	}

	return nil
}

func (s *ChatService) SendMessage(ctx context.Context, message *domain.ChatMessage) error {
	sender, err := s.users.GetByID(ctx, message.SenderID)
	if err != nil {
		return fmt.Errorf("cannot get sender: %w", err)
	}

	receiver, err := s.users.GetByUsername(ctx, message.ReceiverName)
	if err != nil {
		return fmt.Errorf("user doesnt't exist: %w", err)
	}

	message.ReceiverID = receiver.ID
	message.SenderName = sender.Username

	if err = s.jsClient.PublishChatMessage(ctx, message); err != nil {
		return fmt.Errorf("failed to publish invitation: %w", err)
	}
	return nil
}

func (s *ChatService) ReceiveMessage(ctx context.Context, userID, chatID string) (domain.ChatMessage, error) {
	msg, err := s.jsClient.FetchOneChatMessage(ctx, userID, chatID)
	if err != nil {
		return domain.ChatMessage{}, fmt.Errorf("failed to fetch chat message: %w", err)
	}
	return msg, nil
}

func (s *ChatService) CloseRoom(ctx context.Context, roomID string) error {
	if err := s.rooms.Delete(ctx, roomID); err != nil {
		return fmt.Errorf("cannot close room: %w", err)
	}
	return nil
}

func (s *ChatService) ClearChatHistory(ctx context.Context, action domain.ChatActions) error {
	user, err := s.users.GetByUsername(ctx, action.UserName)
	if err != nil {
		return fmt.Errorf("user doesnt't exist: %w", err)
	}
	action.UserID = user.ID
	return s.jsClient.PublishClearChatHistoryRequest(ctx, action)
}

func (s *ChatService) ReceiveClearChatHistoryRequest(ctx context.Context, userID string) (domain.ChatActions, error) {
	return s.jsClient.FetchClearChatHistoryRequest(ctx, userID)
}

func (s *ChatService) UpdateOrDeleteCipherKey(ctx context.Context, action domain.ChatActions) error {
	return nil
}

func (s *ChatService) JoinRoom(ctx context.Context, roomID, clientID string) error {
	return nil
}

func (s *ChatService) LeaveRoom(ctx context.Context, roomID, clientID string) error {
	return nil
}

func (s *ChatService) SendInvitation(ctx context.Context, invite domain.ChatInvitation) error {

	return nil
}

func (s *ChatService) SendPublicKey(ctx context.Context, roomID, clientID, pubHex string) error {
	return nil
}

func (s *ChatService) GetPublicKeys(ctx context.Context, roomID string) ([]domain.PublicKey, error) {
	return nil, nil
}

func (s *ChatService) GetRoomConfig(ctx context.Context, roomID string) (domain.RoomConfig, error) {
	return domain.RoomConfig{}, nil
}
