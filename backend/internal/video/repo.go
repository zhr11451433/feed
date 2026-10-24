package video

import (
	"context"

	"gorm.io/gorm"
)

type VideoRepository struct {
	db *gorm.DB
}

func NewVideoRepository(db *gorm.DB) *VideoRepository {
	return &VideoRepository{db: db}
}

func (r *VideoRepository) GetByID(ctx context.Context, id uint) (*Video, error) {
	var video Video
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&video).Error; err != nil {
		return nil, err
	}
	return &video, nil
}

func (r *VideoRepository) ListByAuthorID(ctx context.Context, authorID uint) ([]Video, error) {
	var videos []Video
	err := r.db.WithContext(ctx).
		Where("author_id = ?", authorID).
		Order("create_time desc").
		Limit(200).
		Find(&videos).Error
	if err != nil {
		return nil, err
	}
	return videos, nil
}

func (r *VideoRepository) IsExist(ctx context.Context, id uint) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&Video{}).Where("id = ?", id).Count(&n).Error
	return n > 0, err
}

func (r *VideoRepository) ChangeLikesCount(ctx context.Context, id uint, change int64) error {
	return r.db.WithContext(ctx).Model(&Video{}).Where("id = ?", id).
		UpdateColumn("likes_count", gorm.Expr("GREATEST(likes_count + ?, 0)", change)).Error
}

func (r *VideoRepository) ChangePopularity(ctx context.Context, id uint, change int64) error {
	return r.db.WithContext(ctx).Model(&Video{}).Where("id = ?", id).
		UpdateColumn("popularity", gorm.Expr("GREATEST(popularity + ?, 0)", change)).Error
}
