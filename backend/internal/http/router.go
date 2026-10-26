package http

import (
	"backend/internal/account"
	"backend/internal/feed"
	"backend/internal/message"
	"backend/internal/middleware/rabbitmq"
	"backend/internal/middleware/ratelimit"
	rediscache "backend/internal/middleware/redis"
	"backend/internal/social"
	"backend/internal/video"
	"backend/internal/worker"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	jwtmw "backend/internal/middleware/jwt"
)

func SetRouter(gdb *gorm.DB, cache *rediscache.Client, rmq *rabbitmq.RabbitMQ) *gin.Engine {
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
	loginLimiter := ratelimit.Limit(cache, "account_login", 10, time.Minute, ratelimit.KeyByIP)
	registerLimiter := ratelimit.Limit(cache, "account_register", 5, time.Hour, ratelimit.KeyByIP)
	likeLimiter := ratelimit.Limit(cache, "like_write", 30, time.Minute, ratelimit.KeyByAccount)
	commentLimiter := ratelimit.Limit(cache, "comment_write", 10, time.Minute, ratelimit.KeyByAccount)
	socialLimiter := ratelimit.Limit(cache, "social_write", 20, time.Minute, ratelimit.KeyByAccount)
	repo := account.NewAccountRepository(gdb)
	svc := account.NewAccountService(repo, cache)
	h := account.NewAccountHandler(svc)
	g := r.Group("/account")
	{
		g.POST("/register", registerLimiter, h.CreateAccount)
		g.POST("/login", loginLimiter, h.Login)
		g.POST("/changePassword", h.ChangePassword)
		g.POST("/findByID", h.FindByID)
		g.POST("/findByUsername", h.FindByUsername)
		g.POST("/refresh", h.Refresh)
		//原因是这一次要同时读三张表，而这三张表分属三个包：
		//accounts 在 account
		//videos 在 video
		//socials 在 social
		//account 的 repo 或 service
		//如果去查视频和关注，account 就要依赖
		//video 和 social。现在的依赖是路由把这三个包装在一起，
		//账号包自己并不知道视频和关注。
		g.POST("/getProfile", func(c *gin.Context) {
			var req account.GetProfileRequest
			if err := c.ShouldBindJSON(&req); err != nil || req.AccountID == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "account_id is required"})
				return
			}
			acc, err := svc.FindByID(c.Request.Context(), req.AccountID)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
				return
			}
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			var stats struct {
				VideoCount    int64 `gorm:"column:video_count"`
				TotalLikes    int64 `gorm:"column:total_likes"`
				FollowerCount int64 `gorm:"column:follower_count"`
				VloggerCount  int64 `gorm:"column:vlogger_count"`
			}
			err = gdb.WithContext(c.Request.Context()).Raw(`
		SELECT
			(SELECT COUNT(*) FROM videos WHERE author_id = ?) AS video_count,
			(SELECT COALESCE(SUM(likes_count), 0) FROM videos WHERE author_id = ?) AS total_likes,
			(SELECT COUNT(*) FROM socials WHERE vlogger_id = ?) AS follower_count,
			(SELECT COUNT(*) FROM socials WHERE follower_id = ?) AS vlogger_count
	`, req.AccountID, req.AccountID, req.AccountID, req.AccountID).Scan(&stats).Error
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, account.GetProfileResponse{
				Account:       *acc,
				VideoCount:    stats.VideoCount,
				TotalLikes:    stats.TotalLikes,
				FollowerCount: stats.FollowerCount,
				VloggerCount:  stats.VloggerCount,
			})
		})
	}
	protected := g.Group("")
	{
		protected.Use(jwtmw.JWTAuth(repo, cache))
		protected.POST("/logout", h.Logout)
		protected.POST("/uploadAvatar", h.UploadAvatar)
		protected.POST("/rename", h.Rename)
		protected.POST("/updateProfile", h.UpdateProfile)
	}
	vh := video.NewVideoHandler(video.NewStore(), video.NewVideoService(gdb, cache))
	vg := r.Group("/video")
	vg.POST("/getDetail", vh.GetDetail)
	vg.POST("/listByAuthorID", vh.ListByAuthorID)
	protectedVideo := vg.Group("")
	protectedVideo.Use(jwtmw.JWTAuth(repo, cache))
	chunkHandler := video.NewChunkHandler(video.NewChunkSessionStore(cache))
	{
		protectedVideo.POST("/uploadVideo", vh.UploadVideo)
		protectedVideo.POST("/uploadCover", vh.UploadCover)
		protectedVideo.POST("/chunk/init", chunkHandler.Init)
		protectedVideo.POST("/chunk/status", chunkHandler.Status)
		protectedVideo.POST("/publish", vh.PublishVideo)
		protectedVideo.POST("/chunk/complete", chunkHandler.Complete)
		protectedVideo.POST("/chunk/upload", chunkHandler.Upload)
	}
	fh := feed.NewFeedHandler(feed.NewFeedService(feed.NewFeedRepository(gdb), cache))
	feedGroup := r.Group("/feed")
	following := r.Group("/feed")
	following.Use(jwtmw.JWTAuth(repo, cache))
	following.POST("/listByFollowing", fh.ListByFollowing)
	{
		feedGroup.Use(jwtmw.SoftJWTAuth(repo, cache))
		feedGroup.POST("/listLatest", fh.ListLatest)
		feedGroup.POST("/listLikesCount", fh.ListLikesCount)
		feedGroup.POST("/listByTag", fh.ListByTag)
		feedGroup.POST("/listByPopularity", fh.ListByPopularity)
	}
	var likeMQ *rabbitmq.LikeMQ
	if rmq != nil {
		var err error
		likeMQ, err = rabbitmq.NewLikeMQ(rmq)
		if err != nil {
			log.Printf("LikeMQ init failed (mq disabled): %v", err)
			likeMQ = nil
		}
	}
	lh := video.NewLikeHandler(video.NewLikeService(gdb, cache, likeMQ))
	likeGroup := r.Group("/like")
	likeGroup.Use(jwtmw.JWTAuth(repo, cache))
	{
		likeGroup.POST("/like", likeLimiter, lh.Like)
		likeGroup.POST("/unlike", likeLimiter, lh.Unlike)
		likeGroup.POST("/isLiked", lh.IsLiked)
		likeGroup.POST("/listMyLikedVideos", lh.ListMyLikedVideos)
	}
	var popMQ *rabbitmq.PopularityMQ
	if rmq != nil {
		var err error
		popMQ, err = rabbitmq.NewPopularityMQ(rmq)
		if err != nil {
			log.Printf("PopularityMQ init failed (mq disabled): %v", err)
			popMQ = nil
		}
	}
	var commentMQ *rabbitmq.CommentMQ
	if rmq != nil {
		var err error
		commentMQ, err = rabbitmq.NewCommentMQ(rmq)
		if err != nil {
			log.Printf("PopularityMQ init failed (mq disabled): %v", err)
			commentMQ = nil
		}
	}
	ch := video.NewCommentHandler(video.NewCommentService(gdb, cache, popMQ, commentMQ))
	commentGroup := r.Group("/comment")
	commentGroup.POST("/listAll", ch.ListAll)
	protectedComment := commentGroup.Group("")
	protectedComment.Use(jwtmw.JWTAuth(repo, cache))
	{
		protectedComment.POST("/publish", commentLimiter, ch.Publish)
		protectedComment.POST("/delete", commentLimiter, ch.Delete)
	}
	var socialMQ *rabbitmq.SocialMQ
	if rmq != nil {
		var err error
		socialMQ, err = rabbitmq.NewSocialMQ(rmq)
		if err != nil {
			log.Printf("PopularityMQ init failed (mq disabled): %v", err)
			socialMQ = nil
		}
	}

	sh := social.NewSocialHandler(social.NewSocialService(gdb, repo, socialMQ))
	socialGroup := r.Group("/social")
	socialGroup.Use(jwtmw.JWTAuth(repo, cache))
	{
		socialGroup.POST("/follow", socialLimiter, sh.Follow)
		socialGroup.POST("/unfollow", socialLimiter, sh.Unfollow)
		socialGroup.POST("/getAllFollowers", sh.GetAllFollowers)
		socialGroup.POST("/getAllVloggers", sh.GetAllVloggers)
		socialGroup.POST("/getCounts", sh.GetCounts)
	}
	var timelineMQ *rabbitmq.TimelineMQ
	if rmq != nil {
		var err error
		timelineMQ, err = rabbitmq.NewTimelineMQ(rmq)
		if err != nil {
			log.Printf("TimelineMQ init failed (mq disabled): %v", err)
			timelineMQ = nil
		}
	}
	nh := worker.NewNotificationHandler(gdb)
	notificationGroup := r.Group("/notification")
	notificationGroup.Use(jwtmw.JWTAuth(repo, cache))
	{
		notificationGroup.POST("/list", nh.List)
		notificationGroup.POST("/markRead", nh.MarkRead)
		notificationGroup.POST("/delete", nh.Delete)
		notificationGroup.POST("/unreadCount", nh.UnreadCount)
	}
	worker.StartOutboxPoller(gdb, timelineMQ)
	hub := worker.NewSSEHub()
	worker.StartNotificationConsumers(rmq, gdb, hub)
	stream := r.Group("/notification")
	stream.Use(jwtmw.SSEAuth(repo, cache))
	stream.GET("/stream", hub.Stream)
	mh := message.NewHandler(message.NewRepository(gdb))
	messageGroup := r.Group("/message")
	messageGroup.Use(jwtmw.JWTAuth(repo, cache))
	{
		messageGroup.POST("/send", mh.Send)
		messageGroup.POST("/list", mh.List)
	}
	return r
}
