package kafka

type InvitationMessage struct {
	Type      string `json:"type"`       // "invite" или "response"
	From      string `json:"from"`       // кто отправил (client_id)
	To        string `json:"to"`         // кому
	RoomID    string `json:"room_id"`    // id комнаты
	Prime     string `json:"prime"`      // простое число (hex)
	G         string `json:"g"`          // генератор (hex)
	PublicKey string `json:"public_key"` // g^a mod p (или g^b mod p)
	Accepted  bool   `json:"accepted"`   // используется только в response
}
