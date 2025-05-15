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
