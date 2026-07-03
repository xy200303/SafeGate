package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	client *redis.Client
}

func New(addr, password string) *Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})
	return &Client{client: rdb}
}

func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) IncrementAttempt(ctx context.Context, ruleID uint64, identity string, windowSeconds int) (int64, error) {
	key := fmt.Sprintf("attempt:%d:%s", ruleID, identity)
	pipe := c.client.Pipeline()
	incr := pipe.Incr(ctx, key)
	if windowSeconds > 0 {
		pipe.Expire(ctx, key, time.Duration(windowSeconds)*time.Second)
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}
	return incr.Val(), nil
}

func (c *Client) GetAttemptCount(ctx context.Context, ruleID uint64, identity string) (int64, error) {
	key := fmt.Sprintf("attempt:%d:%s", ruleID, identity)
	v, err := c.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return v, err
}

func (c *Client) BlacklistJWT(ctx context.Context, jti string, ttl time.Duration) error {
	return c.client.Set(ctx, fmt.Sprintf("jwt:blacklist:%s", jti), "1", ttl).Err()
}

func (c *Client) IsJWTBlacklisted(ctx context.Context, jti string) (bool, error) {
	exists, err := c.client.Exists(ctx, fmt.Sprintf("jwt:blacklist:%s", jti)).Result()
	return exists > 0, err
}
