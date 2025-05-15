package grpc_client

import (
	dh "CryptoMessenger/algorithm/diffie_hellman"
	"CryptoMessenger/cmd/client/domain"
	pb "CryptoMessenger/proto/chatpb"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"io/ioutil"
	"log"
	"log/slog"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type ChatClient struct {
	conn         *grpc.ClientConn
	client       pb.ChatServiceClient
	username     string
	UserID       string
	authToken    string
	privateKeys  sync.Map
	Invitations  sync.Map
	diffieParams sync.Map
	CipherKeys   sync.Map
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
	c.username = username
	c.UserID = resp.UserID

	info := struct {
		Username string `json:"user_name"`
		UserID   string `json:"user_id"`
	}{Username: username, UserID: resp.UserID}

	dir := filepath.Join("cmd", "client", "users", resp.UserID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create user directory: %w", err)
	}

	path := filepath.Join(dir, "user_info.json")
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create user info file: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err = enc.Encode(&info); err != nil {
		return fmt.Errorf("failed to write user info: %w", err)
	}

	return nil
}

func (c *ChatClient) LoadUserName() error {
	path := filepath.Join("cmd", "client", "users", c.UserID, "user_info.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	tmp := struct {
		UserName string `json:"user_name"`
	}{}
	if err = json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	c.username = tmp.UserName
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
	c.UserID = resp.UserID
	c.username = username

	return nil
}

func (c *ChatClient) CreateChat(info domain.Chat) error {
	ctx, cancel := context.WithTimeout(c.AuthenticatedContext(), time.Second*3)
	defer cancel()

	dhParams, err := c.generateDHParams(2048)
	if err != nil {
		return err
	}

	slog.Info("%v", dhParams)

	roomID, err := c.createAndInvite(ctx, info, dhParams)
	if err != nil {
		return err
	}

	roomInfo := domain.RoomInfo{
		ID:          roomID,
		Name:        info.ChatName,
		MyClient:    c.username,
		Companion:   info.Receiver,
		CipherKey:   "",
		P:           dhParams.Prime.Text(16),
		G:           dhParams.G.Text(16),
		PrivateKey:  dhParams.PrivateKey.Text(16),
		MyPublicKey: dhParams.MyPublicKey.Text(16),
		Algorithm:   info.Algorithm,
		CipherMode:  info.Mode,
		Padding:     info.Padding,
		RandomDelta: info.RandomDelta,
		IV:          info.IV,
	}

	return c.saveRoomInfo(roomInfo)
}

func (c *ChatClient) createAndInvite(ctx context.Context, info domain.Chat, params *domain.DiffieHellmanParams) (string, error) {
	resp, err := c.client.CreateRoom(ctx, &pb.CreateRoomRequest{
		RoomName:    info.ChatName,
		Algorithm:   info.Algorithm,
		Mode:        info.Mode,
		Padding:     info.Padding,
		Prime:       params.Prime.Text(16),
		Iv:          info.IV,
		RandomDelta: info.RandomDelta,
	})
	if err != nil {
		return "", fmt.Errorf("could not create room: %w", err)
	}
	roomID := resp.RoomId

	//c.diffieParams.Store(roomID, *params)

	_, err = c.client.InviteUser(ctx, &pb.Invitation{
		ReceiverName: info.Receiver,
		RoomId:       roomID,
		RoomName:     info.ChatName,
		Algorithm:    info.Algorithm,
		Mode:         info.Mode,
		Padding:      info.Padding,
		Iv:           info.IV,
		RandomDelta:  info.RandomDelta,
		Prime:        params.Prime.Text(16),
		G:            params.G.Text(16),
		PublicKey:    params.MyPublicKey.Text(16),
	})
	if err != nil {
		return "", fmt.Errorf("could not invite user: %w", err)
	}

	return roomID, nil
}

func (c *ChatClient) saveRoomInfo(roomInfo domain.RoomInfo) error {

	dir := filepath.Join("cmd", "client", "users", c.UserID, "chats", roomInfo.ID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("could not create dir %s: %w", dir, err)
	}
	filePath := filepath.Join(dir, "room_info.json")
	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("could not create file %s: %w", filePath, err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err = enc.Encode(&roomInfo); err != nil {
		return fmt.Errorf("could not write JSON: %w", err)
	}
	return nil
}

func (c *ChatClient) ReceiveInvitation() (domain.Invitation, error) {
	ctx, cancel := context.WithTimeout(c.AuthenticatedContext(), time.Second*4)
	defer cancel()

	invitation, err := c.client.ReceiveInvitation(ctx, &emptypb.Empty{})

	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.NotFound {
			return domain.Invitation{}, nil
		}
		if errors.Is(err, ctx.Err()) {
			return domain.Invitation{}, nil
		}
		return domain.Invitation{}, err

	}

	slog.Info("Received invitation", invitation)

	roomInfo := domain.RoomInfo{
		ID:             invitation.RoomId,
		Name:           invitation.RoomName,
		MyClient:       c.username,
		Companion:      invitation.SenderName,
		CipherKey:      "",
		P:              invitation.Prime,
		G:              invitation.G,
		PrivateKey:     "",
		OtherPublicKey: invitation.PublicKey,
		Algorithm:      invitation.Algorithm,
		CipherMode:     invitation.Mode,
		Padding:        invitation.Padding,
		RandomDelta:    invitation.RandomDelta,
		IV:             invitation.Iv,
	}

	err = c.saveRoomInfo(roomInfo)

	_, err = c.client.AckInvite(ctx, &pb.AckRequest{MessageId: invitation.MessageId})
	if err != nil {
		log.Printf("could not ack invitation: %v", err)
		return domain.Invitation{}, err
	}

	return domain.Invitation{
		Sender: invitation.SenderName,
		RoomID: invitation.RoomId,
	}, nil
}

func (c *ChatClient) ReactToInvitation(invitation domain.Invitation, accepted bool) error {
	ctx, cancel := context.WithTimeout(c.AuthenticatedContext(), 3*time.Second)
	defer cancel()

	publicKey := new(big.Int)

	if accepted {

		params, err := c.loadDHParamsFromDisk(invitation.RoomID, false)
		if err != nil {
			return fmt.Errorf("could not load DH params: %w", err)
		}

		slog.Info("Got params", params)

		privateKey, err := dh.GeneratePrivateKey(params.Prime)
		if err != nil {
			return fmt.Errorf("could not generate private key: %v", err)
		}

		slog.Info("Got private key", privateKey)

		//c.diffieParams.Store(invitation.RoomID, domain.DiffieHellmanParams{Prime: receivedPrime, G: receivedG, OtherPublicKey: receivedPublicKey, PrivateKey: privateKey})

		cipherKey := dh.GenerateSharedKey(privateKey, params.OtherPublicKey, params.Prime)
		//c.CipherKeys.Store(invitation.RoomID, cipherKey)

		publicKey = dh.GeneratePublicKey(params.G, privateKey, params.Prime)

		if err = c.updateRoomInfoOnDisk(invitation.RoomID, cipherKey.Text(16), domain.CipherKey); err != nil {
			return fmt.Errorf("could not update room info on disk: %w", err)
		}
		if err = c.updateRoomInfoOnDisk(invitation.RoomID, publicKey.Text(16), domain.MyPublicKey); err != nil {
			return fmt.Errorf("could not update room info on disk: %w", err)
		}
		if err = c.updateRoomInfoOnDisk(invitation.RoomID, privateKey.Text(16), domain.PrivateKey); err != nil {
			return fmt.Errorf("could not update room info on disk: %w", err)
		}

	}

	_, err := c.client.ReactToInvitation(ctx, &pb.InvitationReaction{ReceiverName: invitation.Receiver, RoomId: invitation.RoomID, PublicKey: publicKey.Text(16), Accepted: accepted})
	if err != nil {
		return fmt.Errorf("could not react to invitation: %v", err)
	}

	return nil
}

func (c *ChatClient) ReceiveInvitationResponse() (domain.Invitation, error) {
	ctx, cancel := context.WithTimeout(c.AuthenticatedContext(), time.Second*4)
	defer cancel()

	reaction, err := c.client.ReceiveInvitationReaction(ctx, &emptypb.Empty{})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.NotFound {
			return domain.Invitation{}, nil
		}
		if errors.Is(err, ctx.Err()) {
			return domain.Invitation{}, nil
		}
		return domain.Invitation{}, err
	}

	slog.Info("Got reaction: %v", reaction)

	_, err = c.client.AckInvite(ctx, &pb.AckRequest{MessageId: reaction.MessageId})
	if err != nil {
		log.Printf("could not ack invitation: %v", err)
		return domain.Invitation{}, err
	}

	slog.Info("Acked message: %v", reaction.MessageId)

	dhParams, err := c.loadDHParamsFromDisk(reaction.RoomId, true)
	if err != nil {
		return domain.Invitation{}, err
	}

	otherPublicKey, ok := new(big.Int).SetString(reaction.PublicKey, 16)
	if !ok {
		return domain.Invitation{}, fmt.Errorf("invalid public key")
	}

	cipherKey := dh.GenerateSharedKey(dhParams.PrivateKey, otherPublicKey, dhParams.Prime)

	if err = c.updateRoomInfoOnDisk(reaction.RoomId, cipherKey.Text(16), domain.CipherKey); err != nil {
		slog.Error("updateRoomInfoOnDisk 1")
		return domain.Invitation{}, fmt.Errorf("could not update room info on disk: %w", err)
	}
	if err = c.updateRoomInfoOnDisk(reaction.RoomId, reaction.PublicKey, domain.OtherPublicKey); err != nil {
		slog.Error("updateRoomInfoOnDisk 2")
		return domain.Invitation{}, fmt.Errorf("could not update room info on disk: %w", err)
	}

	return domain.Invitation{}, nil
}

func (c *ChatClient) generateDHParams(bits int) (*domain.DiffieHellmanParams, error) {
	prime, err := dh.GeneratePrime(bits)
	if err != nil {
		return nil, fmt.Errorf("could not generate prime: %w", err)
	}
	g := big.NewInt(2)

	privateKey, err := dh.GeneratePrivateKey(prime)
	if err != nil {
		return nil, fmt.Errorf("could not generate private key: %w", err)
	}
	publicKey := dh.GeneratePublicKey(g, privateKey, prime)

	return &domain.DiffieHellmanParams{
		Prime:       prime,
		G:           g,
		PrivateKey:  privateKey,
		MyPublicKey: publicKey,
	}, nil
}

func (c *ChatClient) sendMessage(roomID, text string, filepath string) error {
	ctx, cancel := context.WithTimeout(c.AuthenticatedContext(), time.Second*4)
	defer cancel()
	info, err := c.loadRoomInfoFromDisk(roomID)
	if err != nil {
		return fmt.Errorf("could not load room info: %w", err)
	}

	//TODO зашифровать сообщение + отправлять кусками в натс, но как?

	_, err := c.client.SendMessage(ctx, &pb.SendMessageRequest{RoomId: roomID, ReceiverName: info.Companion, EncryptedMessage: []ecrypted, MessageType: "smth"})

	//сообщение пришло, ошибки нет, упаковываем его в папку с диалогом
}

func (c *ChatClient) receiveMessage(roomID string) error {
	ctx, cancel := context.WithTimeout(c.AuthenticatedContext(), time.Second*4)
	defer cancel()
	info, err := c.loadRoomInfoFromDisk(roomID)
	if err != nil {
		return fmt.Errorf("could not load room info: %w", err)
	}

	_, err := c.client.ReceiveMessage(ctx, &pb.ReceiveMessagesRequest{RoomId: roomID})

	если ошибки нет, расшифровываем

	//сообщение пришло, ошибки нет, упаковываем его в папку с диалогом
}

func (c *ChatClient) loadDHParamsFromDisk(roomID string, myClient bool) (*domain.DiffieHellmanParams, error) {
	//path := filepath.Join("cmd", "client", "users", c.UserID, "chats", roomID, "room_info.json")
	//data, err := ioutil.ReadFile(path)
	//if err != nil {
	//	return nil, fmt.Errorf("could not read room_info.json: %w", err)
	//}
	//
	//var info domain.RoomInfo
	//if err = json.Unmarshal(data, &info); err != nil {
	//	return nil, fmt.Errorf("invalid JSON in room_info.json: %w", err)
	//}

	info, err := c.loadRoomInfoFromDisk(roomID)
	if err != nil {
		return nil, fmt.Errorf("could not load room_info: %w", err)
	}

	var otherPublicKey, privateKey *big.Int

	p, ok := new(big.Int).SetString(info.P, 16)
	if !ok {
		return nil, fmt.Errorf("invalid prime hex: %s", info.P)
	}

	g, ok := new(big.Int).SetString(info.G, 16)
	if !ok {
		return nil, fmt.Errorf("invalid G hex: %s", info.G)
	}

	if !myClient {
		otherPublicKey, ok = new(big.Int).SetString(info.OtherPublicKey, 16)
		if !ok {
			return nil, fmt.Errorf("invalid OtherPublicKey hex: %s", info.OtherPublicKey)
		}
	}

	if myClient {
		privateKey, ok = new(big.Int).SetString(info.PrivateKey, 16)
		if !ok {
			return nil, fmt.Errorf("invalid PrivateKey hex: %s", info.PrivateKey)
		}
	}

	return &domain.DiffieHellmanParams{
		Prime:          p,
		G:              g,
		OtherPublicKey: otherPublicKey,
		PrivateKey:     privateKey,
	}, nil
}

func (c *ChatClient) loadRoomInfoFromDisk(roomID string) (domain.RoomInfo, error) {
	path := filepath.Join("cmd", "client", "users", c.UserID, "chats", roomID, "room_info.json")
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return domain.RoomInfo{}, fmt.Errorf("could not read room_info.json: %w", err)
	}

	var info domain.RoomInfo
	if err = json.Unmarshal(data, &info); err != nil {
		return domain.RoomInfo{}, fmt.Errorf("invalid JSON in room_info.json: %w", err)
	}
	return info, nil
}

func (c *ChatClient) updateRoomInfoOnDisk(roomID, newField string, field int) error {
	path := filepath.Join("cmd", "client", "users", c.UserID, "chats", roomID, "room_info.json")

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("could not read room_info.json: %w", err)
	}
	var info domain.RoomInfo
	if err = json.Unmarshal(data, &info); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	switch field {
	case domain.CipherKey:
		info.CipherKey = newField
	case domain.MyPublicKey:
		info.MyPublicKey = newField
	case domain.PrivateKey:
		info.PrivateKey = newField
	case domain.OtherPublicKey:
		info.OtherPublicKey = newField
	}

	out, err := json.MarshalIndent(&info, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal updated JSON: %w", err)
	}
	if err = ioutil.WriteFile(path, out, 0o600); err != nil {
		return fmt.Errorf("could not write back room_info.json: %w", err)
	}
	return nil
}

func (c *ChatClient) AuthenticatedContext() context.Context {
	md := metadata.New(map[string]string{
		"authorization": "Bearer " + c.authToken,
	})
	return metadata.NewOutgoingContext(context.Background(), md)
}
