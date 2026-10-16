package http

import (
	"backend/internal/account"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	jwtmw "backend/internal/middleware/jwt"
)

func SetRouter(gdb *gorm.DB) *gin.Engine {
	if gdb == nil {
		log.Fatal("gdb is nil")
	}
	r := gin.Default()
	//告诉 Gin 不要相信任何`X‑Forwarded‑For`这类代理请求头，客户端 IP 直接从 TCP 连接获取；适合本地开发，避免 IP 伪造安全风险
	if err := r.SetTrustedProxies(nil); err != nil {
		log.Printf("SetTrustedProxies failed: %v", err)
	}
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	repo := account.NewAccountRepository(gdb)
	svc := account.NewAccountService(repo)
	h := account.NewAccountHandler(svc)
	g := r.Group("/account")
	{
		g.POST("/register", h.CreateAccount)
		g.POST("/login", h.Login)
		g.POST("/findByID", h.FindByID)
		g.POST("/findByUsername", h.FindByUsername)
		g.POST("/refresh", h.Refresh)
	}
	protected := g.Group("")
	{
		protected.Use(jwtmw.JWTAuth(repo))
		protected.POST("/logout", h.Logout)
	}
	return r
}
