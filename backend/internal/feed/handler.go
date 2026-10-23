package feed

import (
	"backend/internal/apierror"
	jwtmw "backend/internal/middleware/jwt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type FeedHandler struct {
	svc *FeedService
}

func NewFeedHandler(svc *FeedService) *FeedHandler {
	return &FeedHandler{svc: svc}
}

func (h *FeedHandler) ListLatest(c *gin.Context) {
	var req ListLatestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = 10
	}
	var latestBefore time.Time
	if req.LatestTime > 0 {
		latestBefore = time.UnixMilli(req.LatestTime)
	}
	viewerID, err := jwtmw.GetAccountID(c)
	if err != nil {
		viewerID = 0
	}
	resp, err := h.svc.ListLatest(c.Request.Context(), req.Limit, latestBefore, viewerID)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if resp.VideoList == nil {
		resp.VideoList = []FeedVideoItem{}
	}
	c.JSON(http.StatusOK, resp)
}

func (h *FeedHandler) ListByFollowing(c *gin.Context) {
	var req ListByFollowingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = 10
	}
	viewerID, err := jwtmw.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	var latestBefore time.Time
	if req.LatestTime > 0 {
		// listLatest 是 time.UnixMilli。这里按原项目用秒。
		latestBefore = time.Unix(req.LatestTime, 0)
	}
	resp, err := h.svc.ListByFollowing(c.Request.Context(), req.Limit, latestBefore, viewerID)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if resp.VideoList == nil {
		resp.VideoList = []FeedVideoItem{}
	}
	c.JSON(http.StatusOK, resp)
}

func (h *FeedHandler) ListByPopularity(c *gin.Context) {
	var req ListByPopularityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = 10
	}
	if req.Offset < 0 {
		req.Offset = 0
	}
	if req.LatestPopularity < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "latest_popularity must be >= 0"})
		return
	}
	anyCursor := !req.LatestBefore.IsZero() || req.LatestIDBefore != nil
	if anyCursor && (req.LatestBefore.IsZero() || req.LatestIDBefore == nil || *req.LatestIDBefore == 0) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "latest_before and latest_id_before must be provided together"})
		return
	}
	viewerID, err := jwtmw.GetAccountID(c)
	if err != nil {
		viewerID = 0
	}
	resp, err := h.svc.ListByPopularity(c.Request.Context(), req, viewerID)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if resp.VideoList == nil {
		resp.VideoList = []FeedVideoItem{}
	}
	c.JSON(http.StatusOK, resp)
}

func (h *FeedHandler) ListLikesCount(c *gin.Context) {
	var req ListLikesCountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = 10
	}
	var cursor *LikesCountCursor
	if req.LikesCountBefore != nil || req.IDBefore != nil {
		if req.LikesCountBefore == nil || req.IDBefore == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "likes_count_before and id_before must be provided together"})
			return
		}
		if *req.LikesCountBefore < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cursor: likes_count_before must be >= 0"})
			return
		}
		if *req.IDBefore == 0 {
			if *req.LikesCountBefore != 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cursor: id_before must be > 0"})
				return
			}
		} else {
			cursor = &LikesCountCursor{LikesCount: *req.LikesCountBefore, ID: *req.IDBefore}
		}
	}
	viewerID, err := jwtmw.GetAccountID(c)
	if err != nil {
		viewerID = 0
	}
	resp, err := h.svc.ListLikesCount(c.Request.Context(), req.Limit, cursor, viewerID)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if resp.VideoList == nil {
		resp.VideoList = []FeedVideoItem{}
	}
	c.JSON(http.StatusOK, resp)
}

func (h *FeedHandler) ListByTag(c *gin.Context) {
	var req ListByTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.TagName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tag_name is required"})
		return
	}
	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = 10
	}
	viewerID, err := jwtmw.GetAccountID(c)
	if err != nil {
		viewerID = 0
	}
	items, err := h.svc.ListByTag(c.Request.Context(), req.TagName, req.Limit, viewerID)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if items == nil {
		items = []FeedVideoItem{}
	}
	c.JSON(http.StatusOK, gin.H{"video_list": items})
}
