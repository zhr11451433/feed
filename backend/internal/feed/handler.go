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
