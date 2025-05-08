package kafka

import (
	"context"
	"encoding/json"
	"github.com/segmentio/kafka-go"
	"log"
)

type MessageHandlerFunc func(InvitationMessage)

type Consumer struct {
	reader  *kafka.Reader
	handler MessageHandlerFunc
}

func NewConsumer(brokerAddr, topic, groupID string, handler MessageHandlerFunc) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{brokerAddr},
		Topic:   topic,
		GroupID: groupID,
	})
	return &Consumer{
		reader:  reader,
		handler: handler,
	}
}

func (c *Consumer) Start(ctx context.Context) {
	go func() {
		defer c.reader.Close()

		for {
			select {
			case <-ctx.Done():
				log.Println("Kafka consumer stopped")
				return
			default:
				m, err := c.reader.ReadMessage(ctx)
				if err != nil {
					log.Println("Kafka read error:", err)
					continue
				}

				var msg InvitationMessage
				if err := json.Unmarshal(m.Value, &msg); err != nil {
					log.Println("Kafka JSON decode error:", err)
					continue
				}

				c.handler(msg)
			}
		}
	}()
}
