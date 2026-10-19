package video

import (
	"backend/internal/apierror"
	jwtmw "backend/internal/middleware/jwt"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CommentHandler struct {
	svc *CommentService
}

func NewCommentHandler(svc *CommentService) *CommentHandler {
	return &CommentHandler{svc: svc}
}

func (h *CommentHandler) Publish(c *gin.Context) {
	var req PublishCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.VideoID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "video_id is required"})
		return
	}
	if req.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content is required"})
		return
	}
	accountID, err := jwtmw.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	username, err := jwtmw.GetUsername(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	comment := &Comment{Username: username, VideoID: req.VideoID, AuthorID: accountID, Content: req.Content}
	if err := h.svc.Publish(c.Request.Context(), comment); err != nil {
		h.writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "comment published successfully"})
}

func (h *CommentHandler) Delete(c *gin.Context) {
	var req DeleteCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.CommentID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "comment_id is required"})
		return
	}
	accountID, err := jwtmw.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.Delete(c.Request.Context(), req.CommentID, accountID); err != nil {
		h.writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "comment deleted successfully"})
}

func (h *CommentHandler) ListAll(c *gin.Context) {
	var req ListCommentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.VideoID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "video_id is required"})
		return
	}
	comments, err := h.svc.ListAll(c.Request.Context(), req.VideoID)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	if comments == nil {
		comments = []Comment{}
	}
	c.JSON(http.StatusOK, comments)
}

func (h *CommentHandler) writeErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	case errors.Is(err, ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, apierror.ErrValidation):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		msg := err.Error()
		if msg == "content is required" {
			c.JSON(http.StatusBadRequest, gin.H{"error": msg})
			return
		}
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": msg})
	}
}
