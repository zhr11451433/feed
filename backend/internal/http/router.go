package http

import (
	"backend/internal/account"
	"backend/internal/feed"
	"backend/internal/video"
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
	r.Static("/static", "./.run/uploads")
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
	vh := video.NewVideoHandler(video.NewStore(), video.NewVideoService(gdb))
	vg := r.Group("/video")
	vg.POST("/getDetail", vh.GetDetail)
	vg.POST("/listByAuthorID", vh.ListByAuthorID)
	protectedVideo := vg.Group("")
	protectedVideo.Use(jwtmw.JWTAuth(repo))
	{
		protectedVideo.POST("/uploadVideo", vh.UploadVideo)
		protectedVideo.POST("/uploadCover", vh.UploadCover)
		protectedVideo.POST("/publish", vh.PublishVideo)
	}
	fh := feed.NewFeedHandler(feed.NewFeedService(feed.NewFeedRepository(gdb)))
	feedGroup := r.Group("/feed")
	{
		feedGroup.Use(jwtmw.SoftJWTAuth(repo))
		feedGroup.POST("/listLatest", fh.ListLatest)
	}
	lh := video.NewLikeHandler(video.NewLikeService(gdb))
	likeGroup := r.Group("/like")
	likeGroup.Use(jwtmw.JWTAuth(repo))
	{
		likeGroup.POST("/like", lh.Like)
		likeGroup.POST("/unlike", lh.Unlike)
		likeGroup.POST("/isLiked", lh.IsLiked)
	}
	ch := video.NewCommentHandler(video.NewCommentService(gdb))
	commentGroup := r.Group("/comment")
	commentGroup.POST("/listAll", ch.ListAll)
	protectedComment := commentGroup.Group("")
	protectedComment.Use(jwtmw.JWTAuth(repo))
	{
		protectedComment.POST("/publish", ch.Publish)
		protectedComment.POST("/delete", ch.Delete)
	}
	return r
}
