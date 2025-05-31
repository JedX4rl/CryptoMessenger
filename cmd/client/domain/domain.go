package domain

import (
	"errors"
	"math/big"
	"time"
)

type Chat struct {
	ChatName    string
	Receiver    string
	Algorithm   string
	Mode        string
	Padding     string
	RandomDelta string
	IV          string
}

const (
	CipherKey = iota
	MyPublicKey
	PrivateKey
	OtherPublicKey
)

type RoomInfo struct {
	ID             string `json:"room_id"`
	Name           string `json:"room_name"`
	MyClient       string `json:"my_client"`
	Companion      string `json:"companion"`
	CipherKey      string `json:"cipher_key"`
	P              string `json:"p"`
	G              string `json:"g"`
	PrivateKey     string `json:"private_key"`
	MyPublicKey    string `json:"public_key"`
	OtherPublicKey string `json:"other_public_key"`
	Algorithm      string `json:"algorithm"`
	CipherMode     string `json:"cipher_mode"`
	Padding        string `json:"padding"`
	RandomDelta    string `json:"random_delta"`
	IV             string `json:"iv"`
}

type User struct {
	Name string `json:"user_name"`
}

type Invitation struct {
	Sender    string
	Receiver  string
	RoomID    string
	RoomName  string
	Prime     string
	G         string
	PublicKey string
	MessageID string
	SharedKey string
	Accepted  bool
}

type DiffieHellmanParams struct {
	Prime          *big.Int
	G              *big.Int
	MyPublicKey    *big.Int
	OtherPublicKey *big.Int
	PrivateKey     *big.Int
}

type StoredMessage struct {
	MessageID   string    `json:"message_id"`
	Sender      string    `json:"sender"`
	Type        string    `json:"type"`
	Content     string    `json:"content,omitempty"`
	Filename    string    `json:"filename,omitempty"`
	Filepath    string    `json:"filepath,omitempty"`
	TotalChunks int       `json:"total_chunks,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

var (
	EmptyFileError = errors.New("вы не можете отправить пустой файл")
	ErrNotFound    = errors.New("не найдено")
)
