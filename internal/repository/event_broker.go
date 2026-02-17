package repository

import (
	"context"
	"github.com/redis/go-redis/v9"
)

type EventBroker struct {
	client *redis.Client
}

func NewEventBroker(client *redis.Client) *EventBroker {
	return &EventBroker{client: client}
}

// Publish replaces the Kafka Producer PublishAlert
func (b *EventBroker) PublishFraudAlert(ctx context.Context, channel string, message string) error {
	// We use Redis Pub/Sub to broadcast the alert
	return b.client.Publish(ctx, channel, message).Err()
}