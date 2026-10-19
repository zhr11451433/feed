package video

import (
	"backend/internal/apierror"
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
)

var ErrForbidden = errors.New("forbidden")

type CommentService struct {
	db *gorm.DB
}

func NewCommentService(db *gorm.DB) *CommentService {
	return &CommentService{db: db}
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
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Select("id").First(&Video{}, comment.VideoID).Error; err != nil {
			return err
		}
		return tx.Create(comment).Error
	})
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
