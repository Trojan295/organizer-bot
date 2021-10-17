package reminder

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// ZSET serving as a delayed queue for reminders
// key: "reminder:queue"
// member: "<channelID>:<reminderID>"
// score: timestamp in epoch
//
// STRING for storing reminders
// key: "reminder:reminders:<channelID>:<reminderID>"
type RedisReminderStore struct {
	redisClient *redis.Client
}

func NewRedisReminderStore(client *redis.Client) *RedisReminderStore {
	return &RedisReminderStore{
		redisClient: client,
	}
}

func (store *RedisReminderStore) Add(ctx context.Context, channelID string, reminder *Reminder) (string, error) {
	UUID := uuid.New()

	reminderID := UUID.String()

	stringKey := fmt.Sprintf("reminder:reminders:%s:%s", channelID, reminderID)
	zsetKey := "reminder:queue"

	zsetMember := fmt.Sprintf("%s:%s", channelID, reminderID)
	timestamp := reminder.Date.Unix()

	reminderBytes, err := store.serializeReminder(reminder)
	if err != nil {
		return "", errors.Wrap(err, "while serializing reminder")
	}

	_, err = store.redisClient.TxPipelined(ctx, func(p redis.Pipeliner) error {
		if err := p.Set(ctx, stringKey, reminderBytes, 0).Err(); err != nil {
			return errors.Wrapf(err, "while SET to %s", stringKey)
		}

		if err := p.ZAdd(ctx, zsetKey, &redis.Z{
			Member: zsetMember,
			Score:  float64(timestamp),
		}).Err(); err != nil {
			return errors.Wrapf(err, "while adding ZSET member to %s", zsetKey)
		}

		return nil
	})
	if err != nil {
		return "", errors.Wrap(err, "while executing TX pipeline")
	}

	return UUID.String(), nil
}

func (store *RedisReminderStore) Remove(ctx context.Context, channelID, reminderID string) error {
	stringKey := fmt.Sprintf("reminder:reminders:%s:%s", channelID, reminderID)
	zsetKey := "reminder:queue"

	zsetMember := fmt.Sprintf("%s:%s", channelID, reminderID)

	_, err := store.redisClient.TxPipelined(ctx, func(p redis.Pipeliner) error {
		if err := p.Del(ctx, stringKey).Err(); err != nil {
			return errors.Wrapf(err, "while DEL key %s", stringKey)
		}

		if err := p.ZRem(ctx, zsetKey, zsetMember).Err(); err != nil {
			return errors.Wrapf(err, "while ZREM key %s", zsetKey)
		}

		return nil
	})
	if err != nil {
		return errors.Wrap(err, "while executing TX pipeline")
	}

	return nil
}

func (store *RedisReminderStore) List(ctx context.Context, channelID string) ([]string, error) {
	var (
		allIDs        []string
		currentCursor uint64
	)

	keyPattern := fmt.Sprintf("reminder:reminders:%s:*", channelID)

	for {
		keys, cursor, err := store.redisClient.Scan(ctx, currentCursor, keyPattern, 0).Result()
		if err != nil {
			return nil, errors.Wrapf(err, "while SCANing %s", keyPattern)
		}

		for _, key := range keys {
			parts := strings.Split(key, ":")
			allIDs = append(allIDs, parts[len(parts)-1])
		}

		if cursor == 0 {
			break
		}
	}

	return allIDs, nil
}

func (store *RedisReminderStore) Get(ctx context.Context, channelID, reminderID string) (*Reminder, error) {
	stringKey := fmt.Sprintf("reminder:reminders:%s:%s", channelID, reminderID)

	data, err := store.redisClient.Get(ctx, stringKey).Bytes()
	if err != nil {
		return nil, errors.Wrapf(err, "while GET key %s", stringKey)
	}

	r, err := store.deserializeReminder(data)
	if err != nil {
		return nil, errors.Wrap(err, "while deserializing reminder")
	}

	return r, nil
}

func (store *RedisReminderStore) serializeReminder(r *Reminder) ([]byte, error) {
	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)

	if err := enc.Encode(*r); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (store *RedisReminderStore) deserializeReminder(data []byte) (*Reminder, error) {
	buf := bytes.NewBuffer(data)

	reminder := &Reminder{}
	if err := gob.NewDecoder(buf).Decode(reminder); err != nil {
		return nil, err
	}

	return reminder, nil
}
