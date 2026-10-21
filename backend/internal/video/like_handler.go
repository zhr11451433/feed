package video

import (
	"backend/internal/apierror"
	jwtmw "backend/internal/middleware/jwt"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type LikeHandler struct {
	svc *LikeService
}

func NewLikeHandler(svc *LikeService) *LikeHandler {
	return &LikeHandler{svc: svc}
}

func (h *LikeHandler) Like(c *gin.Context) {
	h.write(c, true)
}

func (h *LikeHandler) Unlike(c *gin.Context) {
	h.write(c, false)
}

func (h *LikeHandler) IsLiked(c *gin.Context) {
	var req LikeRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.VideoID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "video_id is required"})
		return
	}
	accountID, err := jwtmw.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	ok, err := h.svc.IsLiked(c.Request.Context(), accountID, req.VideoID)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"is_liked": ok})
}

func (h *LikeHandler) write(c *gin.Context, like bool) {
	var req LikeRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.VideoID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "video_id is required"})
		return
	}
	accountID, err := jwtmw.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	if like {
		err = h.svc.Like(c.Request.Context(), accountID, req.VideoID)
	} else {
		err = h.svc.Unlike(c.Request.Context(), accountID, req.VideoID)
	}
	switch {
	case err == nil:
		msg := "unlike success"
		if like {
			msg = "like success"
		}
		c.JSON(http.StatusOK, gin.H{"message": msg})
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "video not found"})
	case errors.Is(err, ErrAlreadyLiked):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, ErrNotLiked), errors.Is(err, apierror.ErrValidation):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
	}
}

func (h *LikeHandler) ListMyLikedVideos(c *gin.Context) {
	accountID, err := jwtmw.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	videos, err := h.svc.ListLikedVideos(c.Request.Context(), accountID)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if videos == nil {
		videos = []Video{}
	}
	c.JSON(http.StatusOK, videos)
}
