package todo

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
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

func (store *RedisTodoStore) Get(ctx context.Context, id string) (*List, error) {
	key := store.getListKey(id)

	data, err := store.redisClient.Get(ctx, key).Bytes()

	if err == redis.Nil {
		return &List{}, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "while getting key %s", key)
	}

	dec := gob.NewDecoder(bytes.NewBuffer(data))

	list := &List{}
	if err := dec.Decode(list); err != nil {
		return nil, errors.Wrap(err, "while decoding list value")
	}

	return list, nil
}

func (store *RedisTodoStore) Save(ctx context.Context, id string, l *List) error {
	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)

	if err := enc.Encode(l); err != nil {
		return errors.Wrap(err, "while encoding list value")
	}

	key := store.getListKey(id)

	if err := store.redisClient.Set(ctx, key, buf.Bytes(), redisExpirationTime).Err(); err != nil {
		return errors.Wrapf(err, "while setting key %s", key)
	}

	return nil
}

func (store *RedisTodoStore) getListKey(id string) string {
	return fmt.Sprintf("todo:%s", id)
}
