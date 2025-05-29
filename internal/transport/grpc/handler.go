package grpc

import (
	"CryptoMessenger/internal/auth"
	"CryptoMessenger/internal/domain"
	myErrors "CryptoMessenger/internal/errors"
	"CryptoMessenger/internal/service"
	pb "CryptoMessenger/proto/chatpb"
	"context"
	"database/sql"
	"errors"
	"github.com/nats-io/nats.go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"log/slog"
)

type ChatHandler struct {
	services *service.Service
	pb.UnimplementedChatServiceServer
}

func NewChatHandler(services *service.Service) *ChatHandler {
	return &ChatHandler{
		services: services,
	}
}

func (h *ChatHandler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	slog.Info("Register request received")
	userID, err := h.services.Register(ctx, req.GetUsername(), req.GetPassword())
	if err != nil {
		if errors.Is(err, myErrors.ErrUserExists) {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	token, err := auth.GenerateToken(userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	slog.Info("Register response sent")

	return &pb.RegisterResponse{
		Token:  token,
		UserID: userID,
	}, nil
}

func (h *ChatHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	slog.Info("Login request received")
	userID, err := h.services.Login(ctx, req.GetUsername(), req.GetPassword())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		if errors.Is(err, myErrors.ErrInvalidPassword) {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	token, err := auth.GenerateToken(userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	slog.Info("Login response sent")
	return &pb.LoginResponse{
		Token:  token,
		UserID: userID,
	}, nil
}

func (h *ChatHandler) CreateRoom(ctx context.Context, req *pb.CreateRoomRequest) (*pb.CreateRoomResponse, error) {
	slog.Info("CreateRoom request received")
	roomID, err := h.services.CreateRoom(ctx, domain.RoomConfig{
		RoomName:    req.RoomName,
		Algorithm:   req.Algorithm,
		Mode:        req.Mode,
		Padding:     req.Padding,
		PrimeHex:    req.Prime,
		Iv:          req.Iv,
		RandomDelta: req.RandomDelta,
	})
	if err != nil {
		return &pb.CreateRoomResponse{}, status.Error(codes.Internal, err.Error())
	}
	slog.Info("CreateRoom response sent")
	return &pb.CreateRoomResponse{RoomId: roomID}, nil
}

func (h *ChatHandler) InviteUser(ctx context.Context, req *pb.Invitation) (*emptypb.Empty, error) {
	slog.Info("InviteUser request received")

	senderID, err := GetClientID(ctx)
	if err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	invitation := domain.ChatInvitation{
		SenderID:     senderID,
		ReceiverName: req.ReceiverName,
		RoomID:       req.RoomId,
		RoomName:     req.RoomName,
		Prime:        req.Prime,
		G:            req.G,
		PublicKey:    req.PublicKey,
		Algorithm:    req.Algorithm,
		Mode:         req.Mode,
		Padding:      req.Padding,
		Iv:           req.Iv,
	}

	_, err = h.services.Chat.InviteUser(ctx, invitation)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	slog.Info("InviteUser response sent")

	return &emptypb.Empty{}, nil
}

func (h *ChatHandler) AckEvent(ctx context.Context, req *pb.AckRequest) (*emptypb.Empty, error) {
	slog.Info("AckEvent request received")
	if err := h.services.Chat.AckEvent(req.MessageId); err != nil {
		return nil, status.Error(codes.NotFound, "message not found")
	}
	slog.Info("AckEvent response sent")
	return &emptypb.Empty{}, nil
}

func (h *ChatHandler) CloseRoom(ctx context.Context, req *pb.CloseRoomRequest) (*emptypb.Empty, error) {
	if err := h.services.CloseRoom(ctx, req.RoomId); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

func (h *ChatHandler) JoinRoom(ctx context.Context, req *pb.JoinRoomRequest) (*emptypb.Empty, error) {
	clientID, err := GetClientID(ctx)
	if err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}
	if err = h.services.JoinRoom(ctx, req.RoomId, clientID); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

func (h *ChatHandler) LeaveRoom(ctx context.Context, req *pb.LeaveRoomRequest) (*emptypb.Empty, error) {
	clientID, err := GetClientID(ctx)
	if err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}
	if err = h.services.LeaveRoom(ctx, req.RoomId, clientID); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

func (h *ChatHandler) ReceiveInvitation(ctx context.Context, _ *emptypb.Empty) (*pb.Invitation, error) {
	clientID, err := GetClientID(ctx)
	if err != nil {
		return &pb.Invitation{}, status.Error(codes.PermissionDenied, err.Error())
	}
	invitation, err := h.services.Chat.ReceiveInvitation(ctx, clientID)
	if err != nil {
		if errors.Is(err, nats.ErrMsgNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, err
	}
	return &pb.Invitation{
		SenderName:  invitation.SenderName,
		RoomId:      invitation.RoomID,
		Prime:       invitation.Prime,
		G:           invitation.G,
		PublicKey:   invitation.PublicKey,
		RoomName:    invitation.RoomName,
		Algorithm:   invitation.Algorithm,
		Mode:        invitation.Mode,
		Padding:     invitation.Padding,
		Iv:          invitation.Iv,
		RandomDelta: invitation.RandomDelta,
		MessageId:   invitation.MessageID,
	}, nil
}

func (h *ChatHandler) ReactToInvitation(ctx context.Context, reaction *pb.InvitationReaction) (*emptypb.Empty, error) {
	clientID, err := GetClientID(ctx)
	if err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	invitationReaction := domain.InvitationReaction{
		SenderID:     clientID,
		ReceiverName: reaction.ReceiverName,
		RoomID:       reaction.RoomId,
		PublicKey:    reaction.PublicKey,
		Accepted:     reaction.Accepted,
	}
	if err = h.services.Chat.ReactToInvitation(ctx, invitationReaction); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

func (h *ChatHandler) ReceiveInvitationReaction(ctx context.Context, _ *emptypb.Empty) (*pb.InvitationReaction, error) {
	clientID, err := GetClientID(ctx)
	if err != nil {
		return &pb.InvitationReaction{}, status.Error(codes.PermissionDenied, err.Error())
	}
	reaction, err := h.services.Chat.ReceiveInvitationReaction(ctx, clientID)
	if err != nil {
		if errors.Is(err, nats.ErrMsgNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, err
	}
	return &pb.InvitationReaction{
		SenderName:   reaction.SenderName,
		ReceiverName: reaction.ReceiverName,
		RoomId:       reaction.RoomID,
		PublicKey:    reaction.PublicKey,
		MessageId:    reaction.MessageID,
		Accepted:     reaction.Accepted,
	}, nil
}

func (h *ChatHandler) SendMessage(ctx context.Context, req *pb.ChatMessage) (*emptypb.Empty, error) {
	senderID, err := GetClientID(ctx)
	if err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	chatMessage := &domain.ChatMessage{
		MessageID:    req.MessageId,
		SenderID:     senderID,
		ReceiverName: req.ReceiverName,
		ChatID:       req.ChatId,
		Timestamp:    req.Timestamp.AsTime(),
	}

	switch payload := req.Payload.(type) {
	case *pb.ChatMessage_Text:
		chatMessage.Text = domain.TextPayload{
			Content: payload.Text.Content,
		}
	case *pb.ChatMessage_Chunk:
		chatMessage.FileChunk = &domain.FileChunk{
			FileID:      payload.Chunk.FileId,
			Filename:    payload.Chunk.Filename,
			ChunkIndex:  int(payload.Chunk.ChunkIndex),
			TotalChunks: int(payload.Chunk.TotalChunks),
			ChunkData:   payload.Chunk.ChunkData,
		}
	default:
		return nil, status.Error(codes.InvalidArgument, "unknown payload type")
	}

	if err := h.services.Chat.SendMessage(ctx, chatMessage); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

func (h *ChatHandler) ReceiveMessage(ctx context.Context, req *pb.ReceiveMessagesRequest) (*pb.ChatMessage, error) {
	msg, err := h.services.Chat.ReceiveMessage(ctx, req.UserId, req.ChatId)
	if err != nil {
		if errors.Is(err, nats.ErrMsgNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	chatMsg := &pb.ChatMessage{
		MessageId:  msg.MessageID,
		SenderId:   msg.SenderID,
		SenderName: msg.SenderName,
		ChatId:     msg.ChatID,
		Timestamp:  timestamppb.New(msg.Timestamp),
	}

	switch {
	case msg.Text != domain.TextPayload{}:
		chatMsg.Payload = &pb.ChatMessage_Text{
			Text: &pb.TextPayload{
				Content: msg.Text.Content,
			},
		}
	case msg.FileChunk != nil:
		chatMsg.Payload = &pb.ChatMessage_Chunk{
			Chunk: &pb.FileChunk{
				FileId:      msg.FileChunk.FileID,
				Filename:    msg.FileChunk.Filename,
				ChunkIndex:  int32(msg.FileChunk.ChunkIndex),
				TotalChunks: int32(msg.FileChunk.TotalChunks),
				ChunkData:   msg.FileChunk.ChunkData,
			},
		}
	default:
		return nil, status.Error(codes.InvalidArgument, "unknown message payload")
	}

	return chatMsg, nil
}
