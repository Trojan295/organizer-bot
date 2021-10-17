package todo

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"strings"
	"time"

	"github.com/Trojan295/organizer-bot/internal/redisutils"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

var redisExpirationTime = 30 * 24 * time.Hour

type RedisTodoStore struct {
	redisClient *redis.Client
}

func NewRedisTodoStore(client *redis.Client) *RedisTodoStore {
	return &RedisTodoStore{
		redisClient: client,
	}
}

func (store *RedisTodoStore) GetEntry(ctx context.Context, channelID, entryID string) (*Entry, error) {
	key := fmt.Sprintf("todo:%s:%s", channelID, entryID)

	data, err := store.redisClient.Get(ctx, key).Bytes()
	if err != nil {
		return nil, errors.Wrapf(err, "while getting key %s", key)
	}

	entry, err := store.unmarshalEntry(data)
	if err != nil {
		return nil, errors.Wrap(err, "while unmarshaling entry")
	}

	return entry, nil
}

func (store *RedisTodoStore) ListEntries(ctx context.Context, channelID string) ([]string, error) {
	keyPattern := fmt.Sprintf("todo:%s:*", channelID)

	allKeys, err := redisutils.ScanKeys(ctx, store.redisClient, keyPattern)
	if err != nil {
		return nil, errors.Wrap(err, "while scanning all keys")
	}

	var allIDs []string

	for _, key := range allKeys {
		parts := strings.Split(key, ":")
		ID := parts[len(parts)-1]
		allIDs = append(allIDs, ID)
	}

	return allIDs, nil
}

func (store *RedisTodoStore) GetEntries(ctx context.Context, channelID string) ([]*Entry, error) {
	IDs, err := store.ListEntries(ctx, channelID)
	if err != nil {
		return nil, errors.Wrap(err, "while listing entry IDs")
	}

	var entries []*Entry
	for _, entryID := range IDs {
		entry, err := store.GetEntry(ctx, channelID, entryID)
		if err != nil {
			return nil, errors.Wrapf(err, "while getting entry %s", entryID)
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func (store *RedisTodoStore) AddEntry(ctx context.Context, channelID string, entry *Entry) (string, error) {
	UUID := uuid.New()
	entry.ID = UUID.String()

	key := fmt.Sprintf("todo:%s:%s", channelID, entry.ID)
	data, err := store.marshalEntry(entry)
	if err != nil {
		return "", errors.Wrap(err, "while marshaling entry")
	}

	if err := store.redisClient.Set(ctx, key, data, redisExpirationTime).Err(); err != nil {
		return "", errors.Wrapf(err, "while SET on key %s", key)
	}

	return entry.ID, nil
}

func (store *RedisTodoStore) RemoveEntry(ctx context.Context, channelID, entryID string) error {
	key := fmt.Sprintf("todo:%s:%s", channelID, entryID)

	if err := store.redisClient.Del(ctx, key).Err(); err != nil {
		return errors.Wrapf(err, "while DEL key %s", key)
	}

	return nil
}

func (store *RedisTodoStore) unmarshalEntry(data []byte) (*Entry, error) {
	var (
		entry = &Entry{}
		buf   = bytes.NewBuffer(data)
	)

	if err := gob.NewDecoder(buf).Decode(entry); err != nil {
		return nil, err
	}

	return entry, nil
}

func (store *RedisTodoStore) marshalEntry(entry *Entry) ([]byte, error) {
	buf := &bytes.Buffer{}
	if err := gob.NewEncoder(buf).Encode(entry); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
