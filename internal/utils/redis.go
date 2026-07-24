package utils

import (
	"context"
	"time"

	"github.com/eefenaxce/axce_blog/internal/db"
)

type RedisClient struct {
	client *db.RedisClient
}

func NewRedisClient(r *db.RedisClient) *RedisClient {
	return &RedisClient{client: r}
}

func (r *RedisClient) blacklistKey(token string) string {
	return "blacklist:" + token
}

func (r *RedisClient) failedAttemptsKey(username string) string {
	return "failed_attempts:" + username
}

func (r *RedisClient) blockedKey(username string) string {
	return "blocked:" + username
}

func (r *RedisClient) BlacklistToken(ctx context.Context, token string, expiration time.Duration) error {
	return r.client.Client.Set(ctx, r.blacklistKey(token), "1", expiration).Err()
}

func (r *RedisClient) IsBlacklisted(ctx context.Context, token string) (bool, error) {
	result, err := r.client.Client.Get(ctx, r.blacklistKey(token)).Result()
	if err != nil {
		return false, nil
	}
	return result == "1", nil
}

func (r *RedisClient) IncrementFailedAttempts(ctx context.Context, username string) error {
	key := r.failedAttemptsKey(username)
	count, err := r.client.Client.Incr(ctx, key).Result()
	if err != nil {
		return err
	}
	if count == 5 {
		r.client.Client.Set(ctx, r.blockedKey(username), "1", time.Minute*15)
	}
	r.client.Client.Expire(ctx, key, time.Minute*15)
	return nil
}

func (r *RedisClient) ClearFailedAttempts(ctx context.Context, username string) error {
	return r.client.Client.Del(ctx, r.failedAttemptsKey(username)).Err()
}

func (r *RedisClient) IsBlocked(ctx context.Context, username string) (bool, error) {
	result, err := r.client.Client.Get(ctx, r.blockedKey(username)).Result()
	if err != nil {
		return false, nil
	}
	return result == "1", nil
}

func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Client.Set(ctx, key, value, expiration).Err()
}

func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return r.client.Client.Get(ctx, key).Result()
}

func (r *RedisClient) Delete(ctx context.Context, key string) error {
	return r.client.Client.Del(ctx, key).Err()
}

func (r *RedisClient) FlushDB(ctx context.Context) error {
	return r.client.Client.FlushDB(ctx).Err()
}

func (r *RedisClient) DeleteByPattern(ctx context.Context, pattern string) error {
	var cursor uint64
	for {
		keys, nextCursor, err := r.client.Client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			r.client.Client.Del(ctx, keys...)
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}

func (r *RedisClient) Increment(ctx context.Context, key string, expirationSeconds int) error {
	pipe := r.client.Client.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, time.Second*time.Duration(expirationSeconds))
	_, err := pipe.Exec(ctx)
	return err
}
