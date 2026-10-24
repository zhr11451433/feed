package video

import (
	"backend/internal/apierror"
	"backend/internal/middleware/rabbitmq"
	rediscache "backend/internal/middleware/redis"
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
)

var ErrForbidden = errors.New("forbidden")

type CommentService struct {
	db    *gorm.DB
	cache *rediscache.Client
	pop   *rabbitmq.PopularityMQ
}

func NewCommentService(db *gorm.DB, cache *rediscache.Client, pop *rabbitmq.PopularityMQ) *CommentService {
	return &CommentService{db: db, cache: cache, pop: pop}
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
