package sessions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"reflect"
	"sync"
)

type sesDataLazyLoader struct {
	sesId   SesId
	o       *sync.Once
	sesData map[string][]byte
	rdb     *redis.Client
}

var KeyNotFoundErr = errors.New("this key doesn't present in session")

// HADD map field "val"
func newLazyLoader(sesId SesId, rdb *redis.Client) *sesDataLazyLoader {
	return &sesDataLazyLoader{
		sesId:   sesId,
		o:       &sync.Once{},
		sesData: make(map[string][]byte),
		rdb:     rdb,
	}
}

// user:123 -> [profile, avatar]
// user:123:profile -> "123"
// user:123:avatar -> [blob]
func (ll *sesDataLazyLoader) load(ctx context.Context, key string, dst interface{}) error {
	if reflect.ValueOf(dst).Kind() != reflect.Ptr {
		return fmt.Errorf("dst must be a pointer")
	}

	if err := ll.ensureSesDataLoaded(ctx); err != nil {
		return fmt.Errorf("cannot load ses data: %w", err)
	}

	keyData, ok := ll.sesData[key]
	if !ok {
		return KeyNotFoundErr
	}

	if err := json.Unmarshal(keyData, dst); err != nil {
		return fmt.Errorf("cannot extract your data: %w", err)
	}

	return nil
}

func (ll *sesDataLazyLoader) onKeyInserted(key string, val []byte) {
	ll.sesData[key] = val
}

func (ll *sesDataLazyLoader) ensureSesDataLoaded(ctx context.Context) error {
	var returnErr error
	ll.o.Do(func() {
		keys, err := ll.rdb.LRange(ctx, ll.sesId.redisKeysList(), 0, -1).Result()
		if err != nil {
			returnErr = fmt.Errorf("cannot get keys from redis: %w", err)
			return
		}

		pipe := ll.rdb.Pipeline()
		for _, k := range keys {
			pipe.Get(ctx, ll.sesId.redisKey(k))
		}

		results, err := pipe.Exec(ctx)
		if err != nil {
			returnErr = fmt.Errorf("cannot exec redis pipe: %w", err)
			return
		}

		resultMap := make(map[string][]byte)
		for i, r := range results {
			val, err := r.(*redis.StringCmd).Bytes()
			if err != nil {
				returnErr = fmt.Errorf("cannot get redis result from pipe cmd: %w", err)
				return
			}

			resultMap[keys[i]] = val
		}

		ll.sesData = resultMap
	})

	return returnErr
}
