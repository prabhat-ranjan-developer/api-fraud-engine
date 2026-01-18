package repository

import (
	"context"
	"log"

	"github.com/segmentio/kafka-go"
)

type EventConsumer struct {
	reader *kafka.Reader
}

func NewEventConsumer(brokerUrl string, topic string) *EventConsumer {
	return &EventConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: []string{brokerUrl},
			Topic:   topic,
			GroupID: "fraud-engine-dashboard-group",

			// === FIX 1: Read from the beginning of history ===
			StartOffset: kafka.FirstOffset,

			MinBytes: 10e3, // 10KB
			MaxBytes: 10e6, // 10MB
		}),
	}
}

func (c *EventConsumer) Subscribe(ctx context.Context, msgChan chan<- string) {
	log.Println("🎧 Kafka Consumer Started... Waiting for messages.")
	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("❌ Consumer Error: %v", err)
			break
		}

		// === FIX 2: Debug Log to prove we received it ===
		log.Printf("📥 Consumer Received: %s", string(m.Value))

		// Send to the API
		msgChan <- string(m.Value)
	}
}

func (c *EventConsumer) Close() {
	c.reader.Close()
}
