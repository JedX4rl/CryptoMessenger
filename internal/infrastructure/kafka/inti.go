package kafka

import (
	"github.com/segmentio/kafka-go"
	"time"
)

type Client struct {
	Producer *kafka.Writer
	Consumer *kafka.Reader
}

func NewKafkaClient(brokerAddress, topic, groupID string) *Client {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokerAddress),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{brokerAddress},
		GroupID:     groupID,
		Topic:       topic,
		MinBytes:    1,    // 1B
		MaxBytes:    10e6, // 10MB
		StartOffset: kafka.LastOffset,
		MaxWait:     1 * time.Second,
	})

	return &Client{
		Producer: writer,
		Consumer: reader,
	}
}
