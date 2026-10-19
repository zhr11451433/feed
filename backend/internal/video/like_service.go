package video

import (
	"backend/internal/apierror"
	"context"
	"errors"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

var (
	ErrAlreadyLiked = errors.New("user has liked this video")
	ErrNotLiked     = errors.New("user has not liked this video")
)

type LikeService struct {
	db *gorm.DB
}

func NewLikeService(db *gorm.DB) *LikeService {
	return &LikeService{db: db}
}

func (s *LikeService) Like(ctx context.Context, accountID, videoID uint) error {
	if accountID == 0 || videoID == 0 {
		return apierror.ErrValidation
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Select("id").First(&Video{}, videoID).Error; err != nil {
			return err
		}
		err := tx.Create(&Like{VideoID: videoID, AccountID: accountID}).Error
		if mysqlErr, ok := errors.AsType[*mysql.MySQLError](err); ok && mysqlErr.Number == 1062 {
			return ErrAlreadyLiked
		}
		if err != nil {
			return err
		}
		//**只更新指定字段，不会自动更新 `UpdatedAt`
		//时间戳**（对比`Update`会自动更新 UpdatedAt）。
		//适合单纯计数器增减场景
		return tx.Model(&Video{}).Where("id = ?", videoID).
			UpdateColumn("likes_count", gorm.Expr("likes_count + 1")).Error
	})
}

func (s *LikeService) Unlike(ctx context.Context, accountID, videoID uint) error {
	if accountID == 0 || videoID == 0 {
		return apierror.ErrValidation
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Select("id").First(&Video{}, videoID).Error; err != nil {
			return err
		}
		res := tx.Where("video_id = ? AND account_id = ?", videoID, accountID).Delete(&Like{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrNotLiked
		}
		//`GREATEST(a,b)` 是 MySQL 内置函数：**返回两个值里面更大的那一个**
		//`GREATEST(likes_count -1, 0)`：点赞数减 1，但**最小不会低于 0，防止负数**。
		return tx.Model(&Video{}).Where("id = ?", videoID).
			UpdateColumn("likes_count", gorm.Expr("GREATEST(likes_count - 1, 0)")).Error
	})
}

//查询：某个用户 (accountID)，有没有给这条视频 (videoID) 点过赞

func (s *LikeService) IsLiked(ctx context.Context, accountID, videoID uint) (bool, error) {
	if accountID == 0 || videoID == 0 {
		return false, apierror.ErrValidation
	}
	var n int64
	err := s.db.WithContext(ctx).Model(&Like{}).
		Where("video_id = ? AND account_id = ?", videoID, accountID).
		Count(&n).Error
	return n > 0, err
}
