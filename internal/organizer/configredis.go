package organizer

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisConfigStore struct {
	redisClient *redis.Client
}

func NewRedisConfigStore(client *redis.Client) *RedisConfigStore {
	return &RedisConfigStore{
		redisClient: client,
	}
}

// Returns the time.Location set on an channel. If it is not set, it will return nil.
func (store *RedisConfigStore) GetCurrentTimezone(ctx context.Context, id string) (*time.Location, error) {
	key := fmt.Sprintf("config:%s:timezone", id)

	tz, err := store.redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("while getting key %s: %w", key, err)
	}

	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("while loading location %s: %w", tz, err)
	}

	return loc, nil
}

func (store *RedisConfigStore) SetCurrentTimezone(ctx context.Context, id string, loc *time.Location) error {
	key := fmt.Sprintf("config:%s:timezone", id)

	if err := store.redisClient.Set(ctx, key, loc.String(), 0).Err(); err != nil {
		return fmt.Errorf("while setting key %s: %w", key, err)
	}

	return nil
}
