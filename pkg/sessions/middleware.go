package sessions

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"net/http"
	"time"
)

type SesId string

func (s SesId) redisKeysList() string {
	return "ses_strg_keys:" + string(s)
}

func (s SesId) redisKey(key string) string {
	return fmt.Sprintf("ses_strg:%s:%s", s, key)
}

type SesExtractor func(r *http.Request) SesId

type sesMngCtxKey struct{}
type sesMng struct {
	ll *sesDataLazyLoader
	s  *sesDataSaver
}

func Middleware(sesExtractor SesExtractor, redisClient *redis.Client, expireTime time.Duration) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sesId := sesExtractor(r)
			r = r.WithContext(context.WithValue(r.Context(), sesMngCtxKey{}, sesMng{
				ll: newLazyLoader(sesId, redisClient),
				s:  newSaver(sesId, redisClient, expireTime),
			}))

			next.ServeHTTP(w, r)
		})
	}
}

func Get[T any](ctx context.Context, key string) (val T, err error) {
	mng, ok := ctx.Value(sesMngCtxKey{}).(sesMng)
	if !ok {
		return val, fmt.Errorf("cannot get ses manager from context, please add middleware to you http pipeline")
	}

	err = mng.ll.load(ctx, key, &val)
	return val, err
}

func Set[T any](ctx context.Context, key string, val T) error {
	mng, ok := ctx.Value(sesMngCtxKey{}).(sesMng)
	if !ok {
		return fmt.Errorf("cannot get ses manager from context, please add middleware to you http pipeline")
	}

	b, err := mng.s.save(ctx, key, val)
	if err != nil {
		return fmt.Errorf("cannot save val: %w", err)
	}

	mng.ll.onKeyInserted(key, b)
	return nil
}
