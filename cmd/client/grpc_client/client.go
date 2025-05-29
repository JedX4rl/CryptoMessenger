package grpc_client

import (
	dh "CryptoMessenger/algorithm/diffie_hellman"
	"CryptoMessenger/algorithm/rc5"
	"CryptoMessenger/algorithm/rc6"
	"CryptoMessenger/algorithm/symmetric"
	"CryptoMessenger/cmd/client/domain"
	"CryptoMessenger/cmd/client/pkg"
	pb "CryptoMessenger/proto/chatpb"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"io"
	"io/ioutil"
	"log"
	"log/slog"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type ChatClient struct {
	conn          *grpc.ClientConn
	client        pb.ChatServiceClient
	Messages      sync.Map
	username      string
	UserID        string
	authToken     string
	CipherContext sync.Map
}

func NewChatClient(serverAddr string) (*ChatClient, error) {
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &ChatClient{
		conn:          conn,
		client:        pb.NewChatServiceClient(conn),
		Messages:      sync.Map{},
		CipherContext: sync.Map{},
	}, nil
}

func (c *ChatClient) Close() error {
	return c.conn.Close()
}

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

	if info.Receiver == c.username {
		return errors.New("creating a chat with yourself is not allowed")
	}

	dhParams, err := c.generateDHParams(2048)
	if err != nil {
		return err
	}

	iv := make([]byte, 16)
	_, err = rand.Read(iv)
	if err != nil {
		return fmt.Errorf("could not generate IV: %w", err)
	}

	info.IV = hex.EncodeToString(iv)

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

	chatPath := filepath.Join(dir, "chat.jsonl")
	f, err := os.Create(chatPath)
	if err != nil {
		return fmt.Errorf("could not create file %s: %w", chatPath, err)
	}
	defer f.Close()

	roomInfoPath := filepath.Join(dir, "room_info.json")
	f, err = os.Create(roomInfoPath)
	if err != nil {
		return fmt.Errorf("could not create file %s: %w", roomInfoPath, err)
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

	_, err = c.client.AckEvent(ctx, &pb.AckRequest{MessageId: invitation.MessageId})
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

		privateKey, err := dh.GeneratePrivateKey(params.Prime)
		if err != nil {
			return fmt.Errorf("could not generate private key: %v", err)
		}

		cipherKey := dh.GenerateSharedKey(privateKey, params.OtherPublicKey, params.Prime)

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

	_, err = c.client.AckEvent(ctx, &pb.AckRequest{MessageId: reaction.MessageId})
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

func (c *ChatClient) SendMessage(cancelContext context.Context, roomID, text, filePath string, progressFunc func(done, total int)) error {
	ctx, cancel := context.WithTimeout(c.AuthenticatedContext(), 8*time.Second)
	defer cancel()

	if text == "" && filePath == "" {
		return fmt.Errorf("must provide either text or filePath")
	}

	info, err := c.loadRoomInfoFromDisk(roomID)
	if err != nil {
		return fmt.Errorf("could not load room info from disk: %w", err)
	}

	var cipherContext *symmetric.CipherContext

	cipherCtxRaw, ok := c.CipherContext.Load(roomID)
	if !ok {
		cipherContext, err = c.newRoomCipher(info)
		if err != nil {
			return fmt.Errorf("could not create cipher context: %w", err)
		}
		c.CipherContext.Store(roomID, cipherContext)
	} else {
		cipherContext, ok = cipherCtxRaw.(*symmetric.CipherContext)
		if !ok {
			return fmt.Errorf("invalid cipher context type for room %s", roomID)
		}
	}

	messageID := uuid.New().String()
	timestamp := time.Now()

	var storedMsg domain.StoredMessage

	if text != "" {
		byteText, err := cipherContext.Encrypt([]byte(text), 0, 1)
		if err != nil {
			return fmt.Errorf("could not encrypt message: %w", err)
		}
		_, err = c.client.SendMessage(ctx, &pb.ChatMessage{
			MessageId:    messageID,
			ChatId:       roomID,
			ReceiverName: info.Companion,
			Timestamp:    timestamppb.New(timestamp),
			Payload: &pb.ChatMessage_Text{
				Text: &pb.TextPayload{
					Content: base64.StdEncoding.EncodeToString(byteText),
				},
			},
		})
		if err != nil {
			return fmt.Errorf("sending text message: %w", err)
		}

		storedMsg = domain.StoredMessage{
			MessageID: messageID,
			Sender:    info.MyClient,
			Type:      "text",
			Content:   text,
			Timestamp: timestamp,
		}

		if err = c.appendToChatFile(roomID, storedMsg); err != nil {
			return fmt.Errorf("save to chat file: %w", err)
		}

		c.Messages.Store(info.ID, struct{}{})
	}

	if filePath != "" {
		encryptedPath := filepath.Join(filepath.Dir(filePath), "encrypted_"+filepath.Base(filePath))
		if err := cipherContext.EncryptFile(cancelContext, filePath, encryptedPath, progressFunc); err != nil {
			return fmt.Errorf("could not encrypt file: %w", err)
		}

		encryptedFile, err := os.Open(encryptedPath)
		if err != nil {
			return fmt.Errorf("open encrypted file: %w", err)
		}
		defer encryptedFile.Close()

		infoStat, err := encryptedFile.Stat()
		if err != nil {
			return fmt.Errorf("stat encrypted file: %w", err)
		}
		fileSize := infoStat.Size()
		const chunkSize = 1024 * 256 // 256KB
		totalChunks := int((fileSize + chunkSize - 1) / chunkSize)

		filename := filepath.Base(filePath)
		fileID := uuid.New().String()
		messageID = uuid.New().String()

		for i := 0; ; i++ {
			buf := make([]byte, chunkSize)
			n, err := encryptedFile.Read(buf)
			if err != nil && err != io.EOF {
				return fmt.Errorf("read encrypted chunk: %w", err)
			}
			if n == 0 {
				break
			}
			slog.Error("sent chunk:", i, totalChunks)

			if _, err = c.client.SendMessage(ctx, &pb.ChatMessage{
				MessageId:    uuid.New().String(),
				ChatId:       roomID,
				ReceiverName: info.Companion,
				Timestamp:    timestamppb.New(timestamp),
				Payload: &pb.ChatMessage_Chunk{
					Chunk: &pb.FileChunk{
						FileId:      fileID,
						Filename:    filename,
						ChunkIndex:  int32(i),
						TotalChunks: int32(totalChunks),
						ChunkData:   buf[:n],
					},
				},
			}); err != nil {
				return fmt.Errorf("sending file chunk %d: %w", i, err)
			}
		}

		storedMsg = domain.StoredMessage{
			MessageID:   messageID,
			Sender:      info.MyClient,
			Type:        "file",
			Filename:    filename,
			Filepath:    filePath,
			TotalChunks: totalChunks,
			Timestamp:   timestamp,
		}

		if err = c.appendToChatFile(roomID, storedMsg); err != nil {
			return fmt.Errorf("save to chat file: %w", err)
		}

		c.Messages.Store(info.ID, struct{}{})

	}

	return nil
}

func (c *ChatClient) ReceiveMessage(roomID string, progressFunc func(done, total int)) error {
	ctx, cancel := context.WithTimeout(c.AuthenticatedContext(), 10*time.Second)
	defer cancel()

	resp, err := c.client.ReceiveMessage(ctx, &pb.ReceiveMessagesRequest{
		ChatId: roomID,
		UserId: c.UserID,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.NotFound {
			return nil
		}
		if errors.Is(err, ctx.Err()) {
			return nil
		}
		return fmt.Errorf("receive message: %w", err)
	}

	info, err := c.loadRoomInfoFromDisk(roomID)
	if err != nil {
		return fmt.Errorf("could not load room info from disk: %w", err)
	}

	var cipherContext *symmetric.CipherContext

	cipherCtxRaw, ok := c.CipherContext.Load(roomID)
	if !ok {
		cipherContext, err = c.newRoomCipher(info)
		if err != nil {
			return fmt.Errorf("could not create cipher context: %w", err)
		}
		c.CipherContext.Store(roomID, cipherContext)
	} else {
		cipherContext, ok = cipherCtxRaw.(*symmetric.CipherContext)
		if !ok {
			return fmt.Errorf("invalid cipher context type for room %s", roomID)
		}
	}

	timestamp := resp.Timestamp.AsTime()
	messageID := resp.MessageId

	switch payload := resp.Payload.(type) {

	case *pb.ChatMessage_Text:
		cipherBytes, err := base64.StdEncoding.DecodeString(payload.Text.Content)
		if err != nil {
			return fmt.Errorf("invalid base64 ciphertext: %w", err)
		}

		byteText, err := cipherContext.Decrypt(cipherBytes, 0, 1)
		if err != nil {
			return fmt.Errorf("could not decrypt message: %w", err)
		}

		storedMsg := domain.StoredMessage{
			MessageID: messageID,
			Sender:    resp.SenderName,
			Type:      "text",
			Content:   string(byteText),
			Timestamp: timestamp,
		}
		if err := c.appendToChatFile(roomID, storedMsg); err != nil {
			return fmt.Errorf("write to chat file: %w", err)
		}

	case *pb.ChatMessage_Chunk:
		slog.Error("Got chunk", payload.Chunk.ChunkIndex, payload.Chunk.TotalChunks)
		dirPath := filepath.Join("cmd/client", "users", c.UserID, "chats", roomID, "files")
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("mkdir for files: %w", err)
		}

		tempFileName := fmt.Sprintf("%s.part", payload.Chunk.FileId)
		tempFilePath := filepath.Join(dirPath, tempFileName)

		f, err := os.OpenFile(tempFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("open file for chunk: %w", err)
		}
		defer f.Close()

		if _, err := f.Write(payload.Chunk.ChunkData); err != nil {
			return fmt.Errorf("write chunk: %w", err)
		}

		if int(payload.Chunk.ChunkIndex) == int(payload.Chunk.TotalChunks)-1 {
			slog.Error("Here now")
			finalFilePath := filepath.Join(dirPath, payload.Chunk.Filename)
			decryptedFile, err := os.Create(finalFilePath)
			if err != nil {
				return fmt.Errorf("create encrypted file: %w", err)
			}
			defer decryptedFile.Close()

			if err = cipherContext.DecryptFile(tempFilePath, finalFilePath, progressFunc); err != nil {
				return fmt.Errorf("could not encrypt file: %w", err)
			}

			storedMsg := domain.StoredMessage{
				MessageID:   messageID,
				Sender:      resp.SenderName,
				Type:        "file",
				Filename:    payload.Chunk.Filename,
				Filepath:    finalFilePath,
				TotalChunks: int(payload.Chunk.TotalChunks),
				Timestamp:   timestamp,
			}

			if err = c.appendToChatFile(roomID, storedMsg); err != nil {
				return fmt.Errorf("write to chat file: %w", err)
			}
		}
	default:
		return fmt.Errorf("unknown message payload")
	}

	if _, err = c.client.AckEvent(ctx, &pb.AckRequest{MessageId: messageID}); err != nil {
		return fmt.Errorf("ack event: %w", err)
	}

	c.Messages.Store(resp.ChatId, struct{}{})

	return nil
}

func (c *ChatClient) appendToChatFile(chatID string, msg domain.StoredMessage) error {
	path := fmt.Sprintf("cmd/client/users/%s/chats/%s/chat.jsonl",
		c.UserID, chatID)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open chat file: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	if _, err = f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write message: %w", err)
	}
	return nil
}

func (c *ChatClient) loadDHParamsFromDisk(roomID string, myClient bool) (*domain.DiffieHellmanParams, error) {
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

func (c *ChatClient) newRoomCipher(info domain.RoomInfo) (*symmetric.CipherContext, error) {
	tmp, err := hex.DecodeString(info.CipherKey)
	if err != nil {
		return nil, fmt.Errorf("invalid hex cipher key: %w", err)
	}

	key := make([]byte, 64)
	copy(key, tmp)

	iv, err := hex.DecodeString(info.IV)
	if err != nil {
		return nil, fmt.Errorf("invalid IV hex: %w", err)
	}

	var cipher symmetric.CipherScheme
	var blockSize int
	switch strings.ToUpper(info.Algorithm) {
	case "RC5":
		cipher, err = rc5.NewRC5(64, 12, uint(len(key)), key)
		if err != nil {
			slog.Error("could not create RC5 block cipher: %w", err)
			return nil, err
		}
		blockSize = 16
	case "RC6":
		cipher, err = rc6.NewRC6(key)
		if err != nil {
			slog.Error("could not create RC6 block cipher: %w", err)
			return nil, err
		}
		blockSize = 16
	default:
		return nil, fmt.Errorf("unsupported algorithm: %s", info.Algorithm)
	}

	mode, err := pkg.ParseCipherMode(info.CipherMode)
	if err != nil {
		return nil, err
	}

	padding, err := pkg.ParsePaddingMode(info.Padding)
	if err != nil {
		return nil, err
	}

	ctx, err := symmetric.NewCipherContext(
		key,
		cipher,
		mode,
		padding,
		iv,
		blockSize,
	)
	if err != nil {
		return nil, err
	}
	return ctx, nil
}
