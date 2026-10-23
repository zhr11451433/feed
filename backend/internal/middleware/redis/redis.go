package redis

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

//区分「没有 key」和「Redis 坏了」。只有没有 key 才加锁

func IsMiss(err error) bool {
	return errors.Is(err, goredis.Nil)
}

func (c *Client) Lock(ctx context.Context, key string, ttl time.Duration) (token string, ok bool, err error) {
	if c == nil || c.rdb == nil {
		return "", false, nil
	}
	b := make([]byte, 16)
	if _, err = rand.Read(b); err != nil {
		return "", false, err
	}
	token = hex.EncodeToString(b)
	//SetNX 可以理解为 SET if Not exists：
	//只有当这个 Key 不存在时，才设置它。
	ok, err = c.rdb.SetNX(ctx, key, token, ttl).Result()
	return token, ok, err
}

var unlockScript = goredis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
else
  return 0
end
`)

func (c *Client) Unlock(ctx context.Context, key, token string) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	_, err := unlockScript.Run(ctx, c.rdb, []string{key}, token).Result()
	return err
}

func (c *Client) ZIncrBy(ctx context.Context, key, member string, score float64) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.ZIncrBy(ctx, key, score, member).Err()
}

func (c *Client) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.Expire(ctx, key, ttl).Err()
}

func (c *Client) ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	if c == nil || c.rdb == nil {
		return nil, nil
	}
	//按分数从高到低（降序）获取指定排名区间内的成员
	return c.rdb.ZRangeArgs(ctx, goredis.ZRangeArgs{
		Key:   key,
		Start: start,
		Stop:  stop,
		Rev:   true,
	}).Result()
}

//它的核心作用是检查指定的 key 是否存在于 Redis 中

func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	if c == nil || c.rdb == nil {
		return false, nil
	}
	n, err := c.rdb.Exists(ctx, key).Result()
	return n > 0, err
}

func (c *Client) ZUnionStore(ctx context.Context, dest string, keys []string) error {
	if c == nil || c.rdb == nil || len(keys) == 0 {
		return nil
	}
	return c.rdb.ZUnionStore(ctx, dest, &goredis.ZStore{
		Keys:      keys,
		Aggregate: "SUM",
	}).Err()
}

var incrementWithExpireScript = goredis.NewScript(`
local count = redis.call("INCR", KEYS[1])
if count == 1 then
  redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
return count
`)

func (c *Client) IncrementWithExpire(ctx context.Context, key string, expire time.Duration) (int64, error) {
	if c == nil || c.rdb == nil {
		return 0, nil
	}
	return incrementWithExpireScript.Run(
		ctx,
		c.rdb,
		[]string{key},
		expire.Milliseconds(),
	).Int64()
}
