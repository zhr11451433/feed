package social

import (
	"backend/internal/apierror"
	jwtmw "backend/internal/middleware/jwt"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SocialHandler struct {
	svc *SocialService
}

func NewSocialHandler(svc *SocialService) *SocialHandler {
	return &SocialHandler{svc: svc}
}

func (h *SocialHandler) Follow(c *gin.Context) {
	h.write(c, true)
}

func (h *SocialHandler) Unfollow(c *gin.Context) {
	h.write(c, false)
}

func (h *SocialHandler) write(c *gin.Context, follow bool) {
	var req FollowRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.VloggerID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "vlogger_id is required"})
		return
	}
	followerID, err := jwtmw.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	if follow {
		err = h.svc.Follow(c.Request.Context(), followerID, req.VloggerID)
	} else {
		err = h.svc.Unfollow(c.Request.Context(), followerID, req.VloggerID)
	}
	switch {
	case err == nil:
		msg := "unfollowed"
		if follow {
			msg = "followed"
		}
		c.JSON(http.StatusOK, gin.H{"message": msg})
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
	case errors.Is(err, ErrAlreadyFollowed):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, ErrFollowSelf), errors.Is(err, ErrNotFollowed), errors.Is(err, apierror.ErrValidation):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
	}
}

//true：查询这个用户的粉丝。
//false：查询这个用户关注的人

func (h *SocialHandler) GetAllFollowers(c *gin.Context) {
	var req GetAllFollowersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	vloggerID, ok := h.currentIfZero(c, req.VloggerID)
	if !ok {
		return
	}
	list, total, err := h.svc.ListBy(c.Request.Context(), vloggerID, true)
	if err != nil {
		h.writeReadErr(c, err)
		return
	}
	c.JSON(http.StatusOK, GetAllFollowersResponse{Followers: list, FollowerCount: total})
}

func (h *SocialHandler) GetAllVloggers(c *gin.Context) {
	var req GetAllVloggersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	followerID, ok := h.currentIfZero(c, req.FollowerID)
	if !ok {
		return
	}
	list, total, err := h.svc.ListBy(c.Request.Context(), followerID, false)
	if err != nil {
		h.writeReadErr(c, err)
		return
	}
	c.JSON(http.StatusOK, GetAllVloggersResponse{Vloggers: list, VloggerCount: total})
}

func (h *SocialHandler) GetCounts(c *gin.Context) {
	accountID, err := jwtmw.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	counts, err := h.svc.Counts(c.Request.Context(), accountID)
	if err != nil {
		h.writeReadErr(c, err)
		return
	}
	c.JSON(http.StatusOK, counts)
}

func (h *SocialHandler) currentIfZero(c *gin.Context, id uint) (uint, bool) {
	if id != 0 {
		return id, true
	}
	accountID, err := jwtmw.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return 0, false
	}
	return accountID, true
}

func (h *SocialHandler) writeReadErr(c *gin.Context, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}
	c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
}
