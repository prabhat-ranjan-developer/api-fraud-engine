package repository

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

type EventProducer struct {
	writer *kafka.Writer
}

func NewEventProducer(brokerUrl string, topic string) *EventProducer {
	return &EventProducer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokerUrl),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
		},
	}
}

func (p *EventProducer) PublishFraudAlert(ctx context.Context, txID string, reason string, payload interface{}) error {
	// Convert the transaction data to JSON
	val, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Create a structured message
	msg := kafka.Message{
		Key:   []byte(txID),
		Value: val,
		Headers: []kafka.Header{
			{Key: "reason", Value: []byte(reason)},
			{Key: "timestamp", Value: []byte(time.Now().String())},
		},
	}

	// Send to Kafka
	err = p.writer.WriteMessages(ctx, msg)
	if err != nil {
		log.Printf("❌ Failed to publish to Kafka: %v", err)
		return err
	}

	log.Printf("📨 Sent to Kafka: %s (Reason: %s)", txID, reason)
	return nil
}

func (p *EventProducer) Close() {
	err := p.writer.Close()
	if err != nil {
		return
	}
}
