package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisRepository struct {
	Client *redis.Client
}

// NewRedisRepository creates a new instance of the Redis repository
func NewRedisRepository(client *redis.Client) *RedisRepository {
	return &RedisRepository{
		Client: client,
	}
}

// IsBlacklisted checks if a user or IP is in the blacklist set
func (r *RedisRepository) IsBlacklisted(ctx context.Context, userID, ip string) (bool, error) {
	// Check User Blacklist
	isUserBanned, err := r.Client.SIsMember(ctx, "blacklist:users", userID).Result()
	if err != nil {
		return false, err
	}
	if isUserBanned {
		return true, nil
	}

	// Check IP Blacklist
	isIPBanned, err := r.Client.SIsMember(ctx, "blacklist:ips", ip).Result()
	if err != nil {
		return false, err
	}
	return isIPBanned, nil
}

// IncrementVelocity increments the transaction count for a user
func (r *RedisRepository) IncrementVelocity(ctx context.Context, userID string, window time.Duration) (int64, error) {
	key := "velocity:" + userID

	// Increment counter
	count, err := r.Client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	// Set expiration on first increment
	if count == 1 {
		r.Client.Expire(ctx, key, window)
	}

	return count, nil
}

// AddToBlacklist adds a value to the specific blacklist (users or ips)
func (r *RedisRepository) AddToBlacklist(ctx context.Context, listType string, value string) error {
	key := "blacklist:" + listType // e.g., "blacklist:users"
	return r.Client.SAdd(ctx, key, value).Err()
}
