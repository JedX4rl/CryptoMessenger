package natsjs

import (
	"CryptoMessenger/internal/domain"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	StreamName                   = "CHAT"
	InvitesSubjectPrefix         = "chat.invite.%s"
	InvitesReactionSubjectPrefix = "chat.invite.reaction.%s"
	MessagesSubjectPrefix        = "chat.messages.%s.%s"
	InvitesConsumerName          = "invite_consumer_%s"
	InviteReactionsConsumerName  = "invite_reactions_consumer_%s"
	MessagesConsumerName         = "message_consumer_%s_%s"
)

type JSClient struct {
	Conn          *nats.Conn
	JS            nats.JetStreamContext
	pendingEvents sync.Map
}

func NewJSClient(url string) *JSClient {
	nc, err := nats.Connect(url,
		nats.MaxReconnects(10),
		nats.ReconnectWait(2*time.Second),
		nats.ErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, err error) {
			log.Printf("NATS error: %v", err)
		}),
	)
	if err != nil {
		log.Fatalf("nats connect: %v", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("nats jetstream: %v", err)
	}

	_, err = js.AddStream(&nats.StreamConfig{
		Name:      StreamName,
		Subjects:  []string{"chat.>"},
		Retention: nats.WorkQueuePolicy,
		MaxAge:    24 * time.Hour,
	})
	if err != nil && !errors.Is(err, nats.ErrStreamNameAlreadyInUse) {
		log.Fatalf("stream creation failed: %v", err)
	}

	return &JSClient{Conn: nc, JS: js}
}

func (c *JSClient) EnsureInvitesConsumer(userID string) error {
	consumerName := fmt.Sprintf(InvitesConsumerName, userID)
	subject := fmt.Sprintf(InvitesSubjectPrefix, userID)

	_, err := c.JS.AddConsumer(StreamName, &nats.ConsumerConfig{
		Durable:       consumerName,
		FilterSubject: subject,
		AckPolicy:     nats.AckExplicitPolicy,
		AckWait:       5 * time.Second,
		MaxDeliver:    3,
		DeliverPolicy: nats.DeliverAllPolicy,
		ReplayPolicy:  nats.ReplayInstantPolicy,
	})

	if err != nil && !isConsumerExists(err) {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	return nil
}

func (c *JSClient) EnsureInviteReactionsConsumer(userID string) error {
	consumerName := fmt.Sprintf(InviteReactionsConsumerName, userID)
	subject := fmt.Sprintf(InvitesReactionSubjectPrefix, userID)

	_, err := c.JS.AddConsumer(StreamName, &nats.ConsumerConfig{
		Durable:       consumerName,
		FilterSubject: subject,
		AckPolicy:     nats.AckExplicitPolicy,
		AckWait:       5 * time.Second,
		MaxDeliver:    3,
		DeliverPolicy: nats.DeliverAllPolicy,
		ReplayPolicy:  nats.ReplayInstantPolicy,
	})

	if err != nil && !isConsumerExists(err) {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	return nil
}

func (c *JSClient) EnsureMessagesConsumer(userID, chatID string) error {
	consumerName := fmt.Sprintf(MessagesConsumerName, chatID, userID)
	subject := fmt.Sprintf(MessagesSubjectPrefix, chatID, userID)

	if userID == "904d9fb3-4e71-41d2-bd66-a36f7cb237bb" {
		a := 4
		_ = a
	}

	_, err := c.JS.AddConsumer(StreamName, &nats.ConsumerConfig{
		Durable:       consumerName,
		FilterSubject: subject,
		AckPolicy:     nats.AckExplicitPolicy,
		AckWait:       5 * time.Second,
		MaxDeliver:    3,
		DeliverPolicy: nats.DeliverAllPolicy,
		ReplayPolicy:  nats.ReplayInstantPolicy,
	})

	if err != nil && !isConsumerExists(err) {
		return fmt.Errorf("failed to create consumer: %w", err)
	}
	return nil
}

func (c *JSClient) PublishInvitation(ctx context.Context, message domain.ChatInvitation) error {
	var err error
	subject := fmt.Sprintf(InvitesSubjectPrefix, message.ReceiverID)

	msg := nats.NewMsg(subject)
	msg.Header.Set("Message-ID", message.MessageID)
	msg.Data, err = json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	_, err = c.JS.PublishMsg(msg, nats.MsgId(message.MessageID), nats.Context(ctx))
	if err != nil {
		return fmt.Errorf("publish failed: %v", err)
	}
	return nil
}

func (c *JSClient) PublishInvitationReaction(ctx context.Context, message domain.InvitationReaction) error {
	var err error
	subject := fmt.Sprintf(InvitesReactionSubjectPrefix, message.ReceiverID)

	msg := nats.NewMsg(subject)
	msg.Header.Set("Message-ID", message.MessageID)
	msg.Data, err = json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	_, err = c.JS.PublishMsg(msg, nats.MsgId(message.MessageID), nats.Context(ctx))
	if err != nil {
		return fmt.Errorf("publish failed: %v", err)
	}
	return nil
}

func (c *JSClient) PublishChatMessage(ctx context.Context, msg *domain.ChatMessage) error {
	subject := fmt.Sprintf(MessagesSubjectPrefix, msg.ChatID, msg.ReceiverID)

	slog.Info("trying to publish chat message", msg)

	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("failed to marshal message", err.Error(), msg)
		return fmt.Errorf("marshal: %w", err)
	}

	natsMsg := nats.NewMsg(subject)
	natsMsg.Header.Set("Message-ID", msg.MessageID)
	natsMsg.Data = data

	_, err = c.JS.PublishMsg(natsMsg, nats.MsgId(msg.MessageID), nats.Context(ctx))
	if err != nil {
		slog.Error("failed to publish message", err.Error(), msg)
		return fmt.Errorf("publish: %w", err)
	}
	slog.Info("success")
	return nil
}

func (c *JSClient) FetchOneChatMessage(ctx context.Context, userID, chatID string) (domain.ChatMessage, error) {
	subject := fmt.Sprintf(MessagesSubjectPrefix, chatID, userID)
	consumerName := fmt.Sprintf(MessagesConsumerName, chatID, userID)

	sub, err := c.JS.PullSubscribe(subject, consumerName)
	if err != nil {
		return domain.ChatMessage{}, fmt.Errorf("pull subscribe: %w", err)
	}

	msgs, err := sub.Fetch(1, nats.MaxWait(1*time.Second))

	if err != nil && !errors.Is(err, ctx.Err()) && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, nats.ErrTimeout) {
		slog.Error("failed to fetch message: %w", err)
		return domain.ChatMessage{}, fmt.Errorf("fetch: %w", err)
	}

	if len(msgs) == 0 {
		return domain.ChatMessage{}, nats.ErrMsgNotFound
	}

	msg := msgs[0]
	var chatMsg domain.ChatMessage
	if err = json.Unmarshal(msg.Data, &chatMsg); err != nil {
		msg.Nak()
		return domain.ChatMessage{}, fmt.Errorf("unmarshal: %w", err)
	}

	c.pendingEvents.Store(chatMsg.MessageID, msg)
	return chatMsg, nil
}

func (c *JSClient) AckEvent(messageID string) error {
	val, ok := c.pendingEvents.LoadAndDelete(messageID)
	if !ok {
		return nil
	}
	msg, ok := val.(*nats.Msg)
	if !ok {
		return fmt.Errorf("invalid message type")
	}
	return msg.Ack()
}

func (c *JSClient) FetchOneInvitation(ctx context.Context, userID string) (domain.ChatInvitation, error) {
	subject := fmt.Sprintf(InvitesSubjectPrefix, userID)
	consumerName := fmt.Sprintf(InvitesConsumerName, userID)

	sub, err := c.JS.PullSubscribe(subject, consumerName)
	if err != nil {
		return domain.ChatInvitation{}, fmt.Errorf("pull subscribe: %w", err)
	}

	msgs, err := sub.Fetch(1, nats.MaxWait(1*time.Second))

	if err != nil && !errors.Is(err, ctx.Err()) && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, nats.ErrTimeout) {
		slog.Error("failed to fetch invitation: %w", err)
		return domain.ChatInvitation{}, fmt.Errorf("fetch: %w", err)
	}

	if len(msgs) == 0 {
		return domain.ChatInvitation{}, nats.ErrMsgNotFound
	}

	msg := msgs[0]
	var invite domain.ChatInvitation
	if err = json.Unmarshal(msg.Data, &invite); err != nil {
		msg.Nak()
		return domain.ChatInvitation{}, fmt.Errorf("unmarshal: %w", err)
	}

	c.pendingEvents.Store(invite.MessageID, msg)
	return invite, nil
}

func (c *JSClient) FetchOneInvitationReaction(ctx context.Context, userID string) (domain.InvitationReaction, error) {
	subject := fmt.Sprintf(InvitesReactionSubjectPrefix, userID)
	consumerName := fmt.Sprintf(InviteReactionsConsumerName, userID)

	sub, err := c.JS.PullSubscribe(subject, consumerName)
	if err != nil {
		return domain.InvitationReaction{}, fmt.Errorf("pull subscribe: %w", err)
	}

	msgs, err := sub.Fetch(1, nats.MaxWait(1*time.Second))
	if err != nil && !errors.Is(err, ctx.Err()) && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, nats.ErrTimeout) {
		slog.Error("failed to fetch invitation reactions: %w", err)
		return domain.InvitationReaction{}, fmt.Errorf("fetch: %w", err)
	}

	if len(msgs) == 0 {
		return domain.InvitationReaction{}, nats.ErrMsgNotFound
	}

	msg := msgs[0]
	var invite domain.InvitationReaction
	if err = json.Unmarshal(msg.Data, &invite); err != nil {
		msg.Nak()
		return domain.InvitationReaction{}, fmt.Errorf("unmarshal: %w", err)
	}

	c.pendingEvents.Store(invite.MessageID, msg)
	return invite, nil
}

func isConsumerExists(err error) bool {
	return strings.Contains(err.Error(), "already exists")
}
