package reminder

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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

func (store *RedisReminderStore) AddReminder(ctx context.Context, channelID string, reminder *Reminder) (string, error) {
	UUID := uuid.New()
	reminder.ID = UUID.String()
	reminder.ChannelID = channelID

	stringKey := fmt.Sprintf("reminder:reminders:%s:%s", channelID, reminder.ID)
	zsetKey := "reminder:queue"

	zsetMember := fmt.Sprintf("%s:%s", channelID, reminder.ID)
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

	return reminder.ID, nil
}

func (store *RedisReminderStore) RemoveReminder(ctx context.Context, channelID, reminderID string) error {
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

func (store *RedisReminderStore) ListReminders(ctx context.Context, channelID string) ([]string, error) {
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

		currentCursor = cursor
	}

	return allIDs, nil
}

func (store *RedisReminderStore) GetReminder(ctx context.Context, channelID, reminderID string) (*Reminder, error) {
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

func (store *RedisReminderStore) GetReminders(ctx context.Context, channelID string) ([]*Reminder, error) {
	IDs, err := store.ListReminders(ctx, channelID)
	if err != nil {
		return nil, errors.Wrap(err, "while listing reminders")
	}

	var reminders []*Reminder
	for _, ID := range IDs {
		reminder, err := store.GetReminder(ctx, channelID, ID)
		if err != nil {
			return nil, errors.Wrapf(err, "while getting reminder %s", ID)
		}

		reminders = append(reminders, reminder)
	}

	return reminders, nil
}

func (store *RedisReminderStore) GetTriggeredReminders(ctx context.Context) ([]*Reminder, error) {
	zsetKey := "reminder:queue"
	timestampNow := time.Now().Unix()

	members, err := store.redisClient.ZRangeByScore(ctx, zsetKey, &redis.ZRangeBy{
		Max: fmt.Sprintf("%d", timestampNow),
	}).Result()
	if err != nil {
		return nil, errors.Wrapf(err, "while ZRANGE on key %s", zsetKey)
	}

	var reminders []*Reminder
	for _, member := range members {
		parts := strings.Split(member, ":")

		if len(parts) < 2 {
			logrus.WithField("member", member).Error("failed to parse channel and reminder ID")
			continue
		}

		channelID := parts[0]
		reminderID := parts[1]

		reminder, err := store.GetReminder(ctx, channelID, reminderID)
		if err != nil {
			return nil, errors.Wrapf(err, "while getting reminder %s", reminderID)
		}

		reminders = append(reminders, reminder)
	}

	return reminders, nil
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
