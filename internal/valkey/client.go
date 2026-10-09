package valkey

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/config"
)

// Client wraps go-redis for Valkey (Redis-compatible). A nil *Client is safe to
// call — every method returns zero/nil (fail-open when Valkey is disabled).
type Client struct {
	rdb *redis.Client
}

// New connects to Valkey when cfg.ValkeyURL is set. Returns (nil, nil) when the
// URL is empty — callers treat a nil *Client as "Valkey disabled".
func New(cfg *config.Config) (*Client, error) {
	if cfg.ValkeyURL == "" {
		return nil, nil
	}
	opts, err := redis.ParseURL(cfg.ValkeyURL)
	if err != nil {
		return nil, fmt.Errorf("valkey: parse url: %w", err)
	}
	rdb := redis.NewClient(opts)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("valkey: ping: %w", err)
	}
	return &Client{rdb: rdb}, nil
}

// Raw returns the underlying *redis.Client for callers that need the native
// client (e.g. the cache package's tiered backend). A nil *Client yields a nil
// *redis.Client, which those callers treat as "Valkey disabled".
func (c *Client) Raw() *redis.Client {
	if c == nil {
		return nil
	}
	return c.rdb
}

func (c *Client) Ping(ctx context.Context) error {
	if c == nil {
		return nil
	}
	return c.rdb.Ping(ctx).Err()
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	if c == nil {
		return "", nil
	}
	v, err := c.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return v, err
}

func (c *Client) SetEX(ctx context.Context, key string, val any, ttl time.Duration) error {
	if c == nil {
		return nil
	}
	return c.rdb.Set(ctx, key, val, ttl).Err()
}

func (c *Client) SetNX(ctx context.Context, key string, val any, ttl time.Duration) (bool, error) {
	if c == nil {
		return false, nil
	}
	return c.rdb.SetNX(ctx, key, val, ttl).Result()
}

func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	if c == nil {
		return 0, nil
	}
	return c.rdb.Incr(ctx, key).Result()
}

func (c *Client) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if c == nil {
		return nil
	}
	return c.rdb.Expire(ctx, key, ttl).Err()
}

func (c *Client) Del(ctx context.Context, keys ...string) error {
	if c == nil {
		return nil
	}
	return c.rdb.Del(ctx, keys...).Err()
}

func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	return c.rdb.Close()
}
