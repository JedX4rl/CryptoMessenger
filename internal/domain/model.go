package domain

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
	SenderName string `json:"sender_name"`
	RoomID     string `json:"room_id"`
	Prime      string `json:"prime"`
	G          string `json:"g"`
	PublicKey  string `json:"public_key"`
	MessageID  string `json:"message_id"`
}

type InvitationReaction struct {
	SenderId     string `json:"sender_id"`
	SenderName   string `json:"sender_name"`
	ReceiverName string `json:"receiver_name"`
	RoomID       string `json:"room_id"`
	PublicKey    string `json:"public_key"`
	Accepted     bool   `json:"accepted"`
	MessageID    string `json:"message_id"`
}
