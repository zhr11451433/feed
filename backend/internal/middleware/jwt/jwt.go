package jwt

import (
	"backend/internal/account"
	"backend/internal/auth"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//受保护的接口以后都挂这个中间件。解析失败要 Abort，否则后面的 handler 仍会执行

func JWTAuth(accountRepo *account.AccountRepository) gin.HandlerFunc {
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
		info, err := accountRepo.FindByID(c.Request.Context(), claims.AccountID)
		if err != nil || info.Token == "" || info.Token != parts[1] {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token has been revoked"})
			return
		}
		c.Set("accountID", claims.AccountID)
		c.Set("username", claims.Username)
		c.Next()
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
