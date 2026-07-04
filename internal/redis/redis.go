package redis

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	client *redis.Client
}

type AttemptEntry struct {
	Key        string `json:"key"`
	RuleID     uint64 `json:"rule_id"`
	Identity   string `json:"identity"`
	Count      int64  `json:"count"`
	TTLSeconds int64  `json:"ttl_seconds"`
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

func (c *Client) SetAttemptCount(ctx context.Context, ruleID uint64, identity string, count int64, ttlSeconds int) error {
	key := fmt.Sprintf("attempt:%d:%s", ruleID, identity)
	if ttlSeconds > 0 {
		return c.client.Set(ctx, key, count, time.Duration(ttlSeconds)*time.Second).Err()
	}
	return c.client.Set(ctx, key, count, 0).Err()
}

func (c *Client) ListAttempts(ctx context.Context) ([]AttemptEntry, error) {
	var cursor uint64
	var entries []AttemptEntry
	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, "attempt:*", 100).Result()
		if err != nil {
			return nil, err
		}
		cursor = nextCursor
		for _, key := range keys {
			count, err := c.client.Get(ctx, key).Int64()
			if err == redis.Nil {
				continue
			}
			if err != nil {
				return nil, err
			}
			ttl, err := c.client.TTL(ctx, key).Result()
			if err != nil {
				return nil, err
			}
			entry := parseAttemptKey(key)
			entry.Count = count
			entry.TTLSeconds = int64(ttl.Seconds())
			entries = append(entries, entry)
		}
		if cursor == 0 {
			return entries, nil
		}
	}
}

func (c *Client) DeleteAttempt(ctx context.Context, key string) (bool, error) {
	if !strings.HasPrefix(key, "attempt:") {
		return false, fmt.Errorf("invalid attempt key")
	}
	deleted, err := c.client.Del(ctx, key).Result()
	return deleted > 0, err
}

func (c *Client) ClearAttempts(ctx context.Context) (int64, error) {
	var cursor uint64
	var deleted int64
	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, "attempt:*", 100).Result()
		if err != nil {
			return deleted, err
		}
		cursor = nextCursor
		if len(keys) > 0 {
			n, err := c.client.Del(ctx, keys...).Result()
			if err != nil {
				return deleted, err
			}
			deleted += n
		}
		if cursor == 0 {
			return deleted, nil
		}
	}
}

func parseAttemptKey(key string) AttemptEntry {
	entry := AttemptEntry{Key: key}
	parts := strings.SplitN(key, ":", 3)
	if len(parts) < 3 {
		return entry
	}
	ruleID, _ := strconv.ParseUint(parts[1], 10, 64)
	entry.RuleID = ruleID
	entry.Identity = parts[2]
	return entry
}

func (c *Client) BlacklistJWT(ctx context.Context, jti string, ttl time.Duration) error {
	return c.client.Set(ctx, fmt.Sprintf("jwt:blacklist:%s", jti), "1", ttl).Err()
}

func (c *Client) IsJWTBlacklisted(ctx context.Context, jti string) (bool, error) {
	exists, err := c.client.Exists(ctx, fmt.Sprintf("jwt:blacklist:%s", jti)).Result()
	return exists > 0, err
}
