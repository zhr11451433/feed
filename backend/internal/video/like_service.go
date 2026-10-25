package video

import (
	"backend/internal/apierror"
	"backend/internal/middleware/rabbitmq"
	rediscache "backend/internal/middleware/redis"
	"context"
	"errors"
	"log"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

var (
	ErrAlreadyLiked = errors.New("user has liked this video")
	ErrNotLiked     = errors.New("user has not liked this video")
)

type LikeService struct {
	db     *gorm.DB
	cache  *rediscache.Client
	likeMQ *rabbitmq.LikeMQ
}

func NewLikeService(db *gorm.DB, cache *rediscache.Client, likeMQ *rabbitmq.LikeMQ) *LikeService {
	return &LikeService{db: db, cache: cache, likeMQ: likeMQ}
}

func (s *LikeService) Like(ctx context.Context, accountID, videoID uint) error {
	if accountID == 0 || videoID == 0 {
		return apierror.ErrValidation
	}
	var n int64
	if err := s.db.WithContext(ctx).Model(&Video{}).Where("id = ?", videoID).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return gorm.ErrRecordNotFound
	}
	var err error
	if s.likeMQ != nil {
		if err = s.likeMQ.Like(ctx, accountID, videoID); err == nil {
			return nil
		}
		log.Printf("like publish failed, direct write: %v", err)
	}
	liked, err := s.IsLiked(ctx, accountID, videoID)
	if err != nil {
		return err
	}
	if liked {
		return ErrAlreadyLiked
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
		if err := tx.Model(&Video{}).Where("id = ?", videoID).
			UpdateColumn("likes_count", gorm.Expr("likes_count + 1")).Error; err != nil {
			return err
		}
		return tx.Model(&Video{}).Where("id = ?", videoID).
			UpdateColumn("popularity", gorm.Expr("popularity + 1")).Error
	})
	if err != nil {
		return err
	}
	s.forgetDetail(ctx, videoID)
	UpdatePopularityCache(ctx, s.cache, videoID, 1)
	return nil
}

func (s *LikeService) Unlike(ctx context.Context, accountID, videoID uint) error {
	if accountID == 0 || videoID == 0 {
		return apierror.ErrValidation
	}
	var n int64
	if err := s.db.WithContext(ctx).Model(&Video{}).Where("id = ?", videoID).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return gorm.ErrRecordNotFound
	}
	var err error
	if s.likeMQ != nil {
		if err = s.likeMQ.Unlike(ctx, accountID, videoID); err == nil {
			return nil
		}
		log.Printf("unlike publish failed, direct write: %v", err)
	}
	liked, err := s.IsLiked(ctx, accountID, videoID)
	if err != nil {
		return err
	}
	if !liked {
		return ErrNotLiked
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
		if err := tx.Model(&Video{}).Where("id = ?", videoID).
			UpdateColumn("likes_count", gorm.Expr("GREATEST(likes_count - 1, 0)")).Error; err != nil {
			return err
		}
		return tx.Model(&Video{}).Where("id = ?", videoID).
			UpdateColumn("popularity", gorm.Expr("GREATEST(popularity - 1, 0)")).Error
	})
	if err != nil {
		return err
	}
	s.forgetDetail(ctx, videoID)
	UpdatePopularityCache(ctx, s.cache, videoID, -1)
	return nil
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

func (s *LikeService) ListLikedVideos(ctx context.Context, accountID uint) ([]Video, error) {
	videos := make([]Video, 0)
	if accountID == 0 {
		return videos, nil
	}
	err := s.db.WithContext(ctx).
		Model(&Video{}).
		Joins("JOIN likes ON likes.video_id = videos.id").
		Where("likes.account_id = ?", accountID).
		Order("likes.created_at DESC").
		Limit(200).
		Find(&videos).Error
	return videos, err
}

func (s *LikeService) forgetDetail(ctx context.Context, id uint) {
	if s.cache == nil {
		return
	}
	opCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	_ = s.cache.Del(opCtx, s.cache.Key("video:detail:id=%d", id))
}
