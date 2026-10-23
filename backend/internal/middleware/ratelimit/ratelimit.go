package ratelimit

import (
	"backend/internal/middleware/jwt"
	rediscache "backend/internal/middleware/redis"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

//string：限流主体的标识，例如客户端 IP、账号 ID。
//bool：是否成功获取标识。true 表示可以限流，false 表示跳过限流。

type KeyFunc func(ctx *gin.Context) (string, bool)

//keyPrefix	限流场景的前缀	"login"
//maxRequests	窗口内允许的最大请求数	10
//window	窗口持续时间	time.Minute
//keyFunc	提取限流主体的函数	KeyByIP

func Limit(cache *rediscache.Client, keyPrefix string, maxRequests int64, window time.Duration, keyFunc KeyFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		//只要有一个条件成立，就跳过限流。
		//- cache == nil：没有 Redis 客户端。
		//- keyFunc == nil：没有办法确定应该对谁限流。
		//- maxRequests <= 0：最大请求数不合理。
		//- window <= 0：窗口时长不合理。
		if cache == nil || keyFunc == nil || maxRequests <= 0 || window <= 0 {
			c.Next()
			return
		}
		//如果传的是 KeyByIP，就提取 IP；
		//如果传的是 KeyByAccount，就提取账号 ID
		subject, ok := keyFunc(c)
		if !ok {
			c.Next()
			return
		}
		count, err := cache.IncrementWithExpire(c.Request.Context(), buildKey(keyPrefix, subject), window)
		//err != nil：Redis 计数失败，放行。
		//count <= maxRequests：还没超过限制，放行。
		//只有计数成功且 count > maxRequests，才进入下面的拒绝逻辑。
		if err != nil || count <= maxRequests {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
	}
}

func buildKey(keyPrefix, subject string) string {
	keyPrefix = strings.TrimSpace(keyPrefix)
	if keyPrefix == "" {
		keyPrefix = "default"
	}
	return fmt.Sprintf("feedsystem:ratelimit:%s:%s", keyPrefix, strings.TrimSpace(subject))
}

func KeyByIP(c *gin.Context) (string, bool) {
	ip := strings.TrimSpace(c.ClientIP())
	if ip == "" {
		return "", false
	}
	return ip, true
}

func KeyByAccount(c *gin.Context) (string, bool) {
	accountID, err := jwt.GetAccountID(c)
	if err != nil || accountID == 0 {
		return "", false
	}
	return strconv.FormatUint(uint64(accountID), 10), true
}
