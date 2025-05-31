package domain

import "time"

type User struct {
	ID           string
	Username     string
	PasswordHash string
}

type RoomConfig struct {
	RoomID      string
	RoomName    string
	Algorithm   string
	Mode        string
	Padding     string
	PrimeHex    string
	Iv          string
	RandomDelta string
}

type PublicKey struct {
	ClientID     string
	PublicKeyHex string
}

type EncryptedMessage struct {
	SenderID    string
	Data        []byte
	Type        string
	FileName    string
	ChunkIndex  int
	TotalChunks int
}

type ChatInvitation struct {
	MessageID string `json:"message_id"`

	SenderID   string `json:"-"`
	ReceiverID string `json:"-"`

	SenderName   string `json:"sender_name"`
	ReceiverName string `json:"receiver_name"`

	RoomID   string `json:"room_id"`
	RoomName string `json:"room_name"`

	Prime       string `json:"prime"`
	G           string `json:"g"`
	PublicKey   string `json:"public_key"`
	Algorithm   string `json:"algorithm"`
	Mode        string `json:"mode"`
	Padding     string `json:"padding"`
	Iv          string `json:"iv"`
	RandomDelta string `json:"random_delta"`
}

type InvitationReaction struct {
	MessageID string `json:"message_id"`

	SenderID   string `json:"sender_id"`
	ReceiverID string `json:"receiver_id"`

	SenderName   string `json:"sender_name"`
	ReceiverName string `json:"receiver_name"`

	RoomID    string `json:"room_id"`
	RoomName  string `json:"room_name"`
	PublicKey string `json:"public_key"`
	Accepted  bool   `json:"accepted"`
}

type ChatMessage struct {
	MessageID    string    `json:"message_id"`
	SenderID     string    `json:"sender_id"`
	SenderName   string    `json:"sender_name"`
	ReceiverID   string    `json:"receiver_id"`
	ReceiverName string    `json:"receiver_name"`
	ChatID       string    `json:"chat_id"`
	Timestamp    time.Time `json:"timestamp"`

	Text       TextPayload `json:"text"`
	FileHeader FileHeader  `json:"file_header"`
	FileChunk  *FileChunk  `json:"file_chunk"`
}

type TextPayload struct {
	Content string `json:"content"`
}

type FileHeader struct {
	FileID      string `json:"file_id"`
	Filename    string `json:"filename"`
	TotalChunks int    `json:"total_chunks"`
}

type FileChunk struct {
	FileID      string `json:"file_id"`
	Filename    string `json:"filename"`
	ChunkIndex  int    `json:"chunk_index"`
	TotalChunks int    `json:"total_chunks"`
	ChunkData   []byte `json:"chunk_data"`
}

type ChatActions struct {
	ID        string `json:"chat_id"`
	UserName  string `json:"user_name"`
	UserID    string `json:"user_id"`
	PublicKey string `json:"public_key"`
	MessageID string `json:"message_id"`
}
