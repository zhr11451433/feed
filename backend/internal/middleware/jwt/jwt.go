package jwt

import (
	"backend/internal/account"
	"backend/internal/auth"
	rediscache "backend/internal/middleware/redis"
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

//受保护的接口以后都挂这个中间件。解析失败要 Abort，否则后面的 handler 仍会执行

func JWTAuth(accountRepo *account.AccountRepository, cache *rediscache.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}
		//SplitN(待分割字符串,分隔符,最多分割成几份)
		parts := strings.SplitN(header, " ", 2)
		//`EqualFold`：**大小写不敏感比较字符串**
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			return
		}
		claims, err := auth.ParseToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		check(c, claims, parts[1], accountRepo, cache)
	}
}

//软鉴权：没有 token 当游客；有 token 就必须通过 JWTAuth。

func SoftJWTAuth(accountRepo *account.AccountRepository, cache *rediscache.Client) gin.HandlerFunc {
	strict := JWTAuth(accountRepo, cache)
	return func(c *gin.Context) {
		if c.GetHeader("Authorization") == "" {
			c.Next()
			return
		}
		strict(c)
	}
}

//handler 不自己解析 JWT。中间件放进去的类型是 uint，取的时候要断言

func GetAccountID(c *gin.Context) (uint, error) {
	v, ok := c.Get("accountID")
	if !ok {
		return 0, errors.New("accountID not found")
	}
	id, ok := v.(uint)
	if !ok {
		return 0, errors.New("accountID has invalid type")
	}
	return id, nil
}

func GetUsername(c *gin.Context) (string, error) {
	v, ok := c.Get("username")
	if !ok {
		return "", errors.New("username not found")
	}
	name, ok := v.(string)
	if !ok || name == "" {
		return "", errors.New("username has invalid type")
	}
	return name, nil
}

func check(c *gin.Context, claims *auth.Claims, tokenString string, accountRepo *account.AccountRepository, cache *rediscache.Client) {
	if cache != nil {
		key := cache.Key("account:%d", claims.AccountID)
		cacheCtx, cancel := context.WithTimeout(c.Request.Context(), 50*time.Millisecond)
		defer cancel()
		b, err := cache.GetBytes(cacheCtx, key)
		if err == nil {
			if string(b) != tokenString {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token has been revoked"})
				return
			}
			c.Set("accountID", claims.AccountID)
			c.Set("username", claims.Username)
			c.Next()
			return
		}
	}
	info, err := accountRepo.FindByID(c.Request.Context(), claims.AccountID)
	if err != nil || info.Token == "" || info.Token != tokenString {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token has been revoked"})
		return
	}
	if cache != nil {
		cacheCtx, cancel := context.WithTimeout(c.Request.Context(), 50*time.Millisecond)
		defer cancel()
		if err := cache.SetBytes(cacheCtx, cache.Key("account:%d", claims.AccountID), []byte(tokenString), 24*time.Hour); err != nil {
			log.Printf("failed to set token cache: %v", err)
		}
	}
	c.Set("accountID", claims.AccountID)
	c.Set("username", claims.Username)
	c.Next()
}
