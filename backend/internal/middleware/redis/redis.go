package redis

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"backend/internal/config"

	goredis "github.com/redis/go-redis/v9"
)

const defaultKeyPrefix = "v1:"

type Client struct {
	rdb       *goredis.Client
	keyPrefix string
}

func NewFromEnv(cfg *config.RedisConfig) (*Client, error) {
	if cfg == nil || cfg.Host == "" || cfg.Port == 0 {
		return nil, errors.New("redis config is incomplete")
	}
	rdb := goredis.NewClient(&goredis.Options{
		Addr:     cfg.Host + ":" + strconv.Itoa(cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	return &Client{rdb: rdb, keyPrefix: defaultKeyPrefix}, nil
}

func (c *Client) Close() error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.Close()
}

func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.rdb == nil {
		return errors.New("redis client not initialized")
	}
	return c.rdb.Ping(ctx).Err()
}

// Key 统一加前缀，避免和本机 Redis 里别的项目撞 key。T702 开始用。
func (c *Client) Key(format string, args ...any) string {
	prefix := ""
	if c != nil {
		prefix = c.keyPrefix
	}
	return prefix + fmt.Sprintf(format, args...)
}

func (c *Client) GetBytes(ctx context.Context, key string) ([]byte, error) {
	if c == nil || c.rdb == nil {
		return nil, errors.New("redis client not initialized")
	}
	return c.rdb.Get(ctx, key).Bytes()
}

func (c *Client) SetBytes(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if c == nil || c.rdb == nil {
		return errors.New("redis client not initialized")
	}
	return c.rdb.Set(ctx, key, value, ttl).Err()
}

func (c *Client) Del(ctx context.Context, key string) error {
	if c == nil || c.rdb == nil {
		return errors.New("redis client not initialized")
	}
	return c.rdb.Del(ctx, key).Err()
}
