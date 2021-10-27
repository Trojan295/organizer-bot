package redisutils

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
)

func ScanKeys(ctx context.Context, cli *redis.Client, keyPattern string) ([]string, error) {
	var (
		currentCursor uint64
		allKeys       []string
	)

	for {
		keys, cursor, err := cli.Scan(ctx, currentCursor, keyPattern, 0).Result()
		if err != nil {
			return nil, errors.Wrapf(err, "while SCANing %s", keyPattern)
		}

		allKeys = append(allKeys, keys...)

		if cursor == 0 {
			break
		}
	}

	return allKeys, nil
}
