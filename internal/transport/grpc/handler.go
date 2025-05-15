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
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
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

func (h *ChatHandler) AckInvite(ctx context.Context, req *pb.AckRequest) (*emptypb.Empty, error) {
	slog.Info("AckEvent request received")
	if err := h.services.Chat.AckInvite(req.MessageId); err != nil {
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
	if clientID == "ebc58cc6-dd67-4b40-ae1d-df763613a48d" {
		var a = 5
		_ = a
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

func (h *ChatHandler) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*emptypb.Empty, error) {
	//err := h.chatUC.SendMessage(ctx, usecase.SendMessageParams{
	//	RoomID:           req.RoomId,
	//	ClientID:         req.ClientId,
	//	EncryptedMessage: req.EncryptedMessage,
	//	MessageType:      req.MessageType,
	//	FileName:         req.FileName,
	//})
	return &emptypb.Empty{}, nil
}

func (h *ChatHandler) ReceiveMessages(req *pb.ReceiveMessagesRequest, g grpc.ServerStreamingServer[pb.ReceiveMessagesResponse]) (*emptypb.Empty, error) {
	//msgsCh, err := h.chatUC.ReceiveMessages(stream.Context(), req.RoomId, req.ClientId)
	//if err != nil {
	//	return err
	//}
	//
	//for {
	//	select {
	//	case <-stream.Context().Done():
	//		return stream.Context().Err()
	//	case msg, ok := <-msgsCh:
	//		if !ok {
	//			return nil
	//		}
	//		if err := stream.Send(&pb.ReceiveMessagesResponse{
	//			SenderId:         msg.SenderID,
	//			EncryptedMessage: msg.Data,
	//			MessageType:      msg.Type,
	//			FileName:         msg.FileName,
	//			ChunkIndex:       int32(msg.ChunkIndex),
	//			TotalChunks:      int32(msg.TotalChunks),
	//		}); err != nil {
	//			return err
	//		}
	//	}
	//}
	return &emptypb.Empty{}, nil
}

//func (h *ChatHandler) GetRoom(ctx context.Context, req *pb.GetRoomRequest) (*pb.GetRoomResponse, error) {
//	cfg, err := h.services.GetRoomConfig(ctx, req.RoomId)
//	if err != nil {
//		return nil, err
//	}
//	return &pb.GetRoomResponse{
//		Algorithm: cfg.Algorithm,
//		Mode:      cfg.Mode,
//		Padding:   cfg.Padding,
//		Prime:     cfg.PrimeHex,
//	}, nil
//}

func (h *ChatHandler) MustEmbedUnimplementedChatservicesServer() {
	//TODO implement me
	panic("implement me")
}
