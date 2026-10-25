package video

import (
	"backend/internal/apierror"
	"backend/internal/middleware/rabbitmq"
	rediscache "backend/internal/middleware/redis"
	"context"
	"errors"
	"log"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

var ErrForbidden = errors.New("forbidden")

type CommentService struct {
	db        *gorm.DB
	cache     *rediscache.Client
	pop       *rabbitmq.PopularityMQ
	commentMQ *rabbitmq.CommentMQ
}

func NewCommentService(db *gorm.DB, cache *rediscache.Client, pop *rabbitmq.PopularityMQ, commentMQ *rabbitmq.CommentMQ) *CommentService {
	return &CommentService{db: db, cache: cache, pop: pop, commentMQ: commentMQ}
}
func (s *CommentService) Publish(ctx context.Context, comment *Comment) error {
	//删掉字符串两头的空白字符
	comment.Username = strings.TrimSpace(comment.Username)
	comment.Content = strings.TrimSpace(comment.Content)
	if comment.VideoID == 0 || comment.AuthorID == 0 || comment.Username == "" {
		return apierror.ErrValidation
	}
	if comment.Content == "" {
		return errors.New("content is required")
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Select("id").First(&Video{}, comment.VideoID).Error; err != nil {
			return err
		}
		if err := tx.Create(comment).Error; err != nil {
			return err
		}
		if err := tx.Model(&Video{}).Where("id = ?", comment.VideoID).
			UpdateColumn("popularity", gorm.Expr("popularity + 1")).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.notifyMentions(ctx, comment)
	if s.commentMQ != nil {
		if err := s.commentMQ.Publish(ctx, comment.Username, comment.VideoID, comment.AuthorID, comment.Content); err != nil {
			log.Printf("comment event failed: %v", err)
		}
	}
	if s.pop != nil {
		if err := s.pop.Update(ctx, comment.VideoID, 1); err == nil {
			return nil
		}
	}
	UpdatePopularityCache(ctx, s.cache, comment.VideoID, 1)
	return nil
}

func (s *CommentService) Delete(ctx context.Context, commentID, accountID uint) error {
	if commentID == 0 || accountID == 0 {
		return apierror.ErrValidation
	}
	var comment Comment
	if err := s.db.WithContext(ctx).First(&comment, commentID).Error; err != nil {
		return err
	}
	if comment.AuthorID != accountID {
		return ErrForbidden
	}
	return s.db.WithContext(ctx).Delete(&comment).Error
}

func (s *CommentService) ListAll(ctx context.Context, videoID uint) ([]Comment, error) {
	if videoID == 0 {
		return nil, apierror.ErrValidation
	}
	if err := s.db.WithContext(ctx).Select("id").First(&Video{}, videoID).Error; err != nil {
		return nil, err
	}
	var comments []Comment
	err := s.db.WithContext(ctx).
		Where("video_id = ?", videoID).
		Order("created_at asc").
		Limit(200).
		Find(&comments).Error
	return comments, err
}

// 当用户发表评论时，自动识别评论中提到的其他用户（@用户名），然后给被提及的人创建一条通知记录
var mentionRegex = regexp.MustCompile(`@(\w+)`)

func (s *CommentService) notifyMentions(ctx context.Context, comment *Comment) {
	matches := mentionRegex.FindAllStringSubmatch(comment.Content, -1)
	//创建去重 Map
	seen := map[string]bool{}
	for _, m := range matches {
		username := m[1]
		//跳过重复用户和评论作者本人
		if seen[username] || username == comment.Username {
			continue
		}
		seen[username] = true
		var accID uint
		if err := s.db.WithContext(ctx).Table("accounts").Where("username = ?", username).Select("id").Scan(&accID).Error; err != nil || accID == 0 {
			continue
		}
		err := s.db.WithContext(ctx).Table("notifications").Create(map[string]any{
			"recipient_id": accID,
			"sender_id":    comment.AuthorID,
			"type":         "mention",
			"target_id":    comment.VideoID,
			"content":      comment.Username + " 在评论中提到了你",
		}).Error
		if err != nil {
			log.Printf("create mention notification failed: %v", err)
		}
	}
}
