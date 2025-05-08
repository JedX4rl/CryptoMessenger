package natsjs

type InvitationMessage struct {
	MessageID    string `json:"message_id"`
	SenderName   string `json:"sender_name"` // кто послал (client_id)
	ReceiverName string `json:"receiver_name"`
	ReceiverID   string // кому (client_id)
	RoomID       string `json:"room_id"`              // идентификатор комнаты
	Prime        string `json:"prime,omitempty"`      // p в hex (только в приглашении)
	G            string `json:"g,omitempty"`          // g в hex (только в приглашении)
	PublicKey    string `json:"public_key,omitempty"` // A или B в hex
	Accepted     bool   `json:"accepted,omitempty"`   // true = принято; false = отклонено (только в ответе)
}
