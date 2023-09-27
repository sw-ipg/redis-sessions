package sessions

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

type sesDataSaver struct {
	sesId SesId
	rdb   *redis.Client
	ttl   time.Duration
}

func newSaver(sesId SesId, rdb *redis.Client, ttl time.Duration) *sesDataSaver {
	return &sesDataSaver{
		sesId: sesId,
		rdb:   rdb,
		ttl:   ttl,
	}
}

func (s *sesDataSaver) save(ctx context.Context, key string, data interface{}) ([]byte, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal val: %w", err)
	}

	currentKeys, err := s.rdb.LRange(ctx, s.sesId.redisKeysList(), 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("cannot get current keys: %w", err)
	}

	shouldLPush := true
	for _, ck := range currentKeys {
		if ck == key {
			shouldLPush = false
		}
	}

	if shouldLPush {
		if err := s.rdb.LPush(ctx, s.sesId.redisKeysList(), key).Err(); err != nil {
			return nil, fmt.Errorf("cannot lpush: %w", err)
		}
	}

	if err := s.rdb.ExpireNX(ctx, s.sesId.redisKeysList(), s.ttl).Err(); err != nil {
		return nil, fmt.Errorf("cannot expire nx: %w", err)
	}

	if err := s.rdb.Set(ctx, s.sesId.redisKey(key), b, s.ttl+5*time.Second).Err(); err != nil {
		return nil, fmt.Errorf("cannot set redis key: %w", err)
	}

	return b, nil
}
