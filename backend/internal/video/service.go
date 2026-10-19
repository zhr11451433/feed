package video

import (
	"backend/internal/apierror"
	"context"
	"strings"

	"gorm.io/gorm"
)

type VideoService struct {
	repo *VideoRepository
}

func NewVideoService(db *gorm.DB) *VideoService {
	return &VideoService{repo: NewVideoRepository(db)}
}

func (s *VideoService) Publish(ctx context.Context, v *Video) error {
	v.Title = strings.TrimSpace(v.Title)
	v.Description = strings.TrimSpace(v.Description)
	v.PlayURL = strings.TrimSpace(v.PlayURL)
	v.CoverURL = strings.TrimSpace(v.CoverURL)
	if v.Title == "" || v.PlayURL == "" || v.CoverURL == "" {
		return apierror.ErrValidation
	}
	return s.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(v).Error; err != nil {
			return err
		}
		for _, name := range ExtractTags(v.Title + " " + v.Description) {
			var tag Tag
			if err := tx.Where("name = ?", name).FirstOrCreate(&tag, Tag{Name: name}).Error; err != nil {
				return err
			}
			if err := tx.Create(&VideoTag{VideoID: v.ID, TagID: tag.ID}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *VideoService) GetDetail(ctx context.Context, id uint) (*Video, error) {
	if id == 0 {
		return nil, apierror.ErrValidation
	}
	return s.repo.GetByID(ctx, id)
}

func (s *VideoService) ListByAuthorID(ctx context.Context, authorID uint) ([]Video, error) {
	return s.repo.ListByAuthorID(ctx, authorID)
}
