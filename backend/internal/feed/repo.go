package feed

import (
	"backend/internal/video"
	"context"
	"time"

	"gorm.io/gorm"
)

type FeedRepository struct {
	db *gorm.DB
}

func NewFeedRepository(db *gorm.DB) *FeedRepository {
	return &FeedRepository{db: db}
}

//`latestBefore time.Time`：**游标时间**，
//上一页最后一条视频的`create_time`，
//代表：只查**比这个时间更早**的数据

func (r *FeedRepository) ListLatest(ctx context.Context, limit int, latestBefore time.Time) ([]*video.Video, error) {
	var videos []*video.Video
	//`create_time DESC`：**最新的排在最前面**
	//最新发布的视频时间大，`DESC`从大到小排，第一页拿到最新视频。
	query := r.db.WithContext(ctx).Model(&video.Video{}).Order("create_time DESC")
	//- `latestBefore.IsZero()`：判断时间是不是零值（`time.Time{}`，没有赋值）
	//- **第一页请求的时候**：前端没传游标 → `latestBefore`是零值 → 这个 if 不执行，
	//**不加 where 条件**，直接查最新 N 条
	//- **下拉加载更多（第二页、第三页）**：
	//前端传上一页最后一条视频的 create_time，
	//`latestBefore`不为零，追加条件：
	if !latestBefore.IsZero() {
		query = query.Where("create_time < ?", latestBefore)
	}
	if err := query.Limit(limit).Find(&videos).Error; err != nil {
		return nil, err
	}
	return videos, nil
}

func (r *FeedRepository) ListLikedIDs(ctx context.Context, accountID uint, videoIDs []uint) (map[uint]bool, error) {
	liked := make(map[uint]bool)
	if accountID == 0 || len(videoIDs) == 0 {
		return liked, nil
	}
	// likes 表在 T401 才建。现在查会报错，已登录的信息流会 500。
	var ids []uint
	err := r.db.WithContext(ctx).Table("likes").
		Where("account_id = ? AND video_id IN ?", accountID, videoIDs).
		//`Pluck` = **单独提取某一列的值，放到切片里**
		Pluck("video_id", &ids).Error
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		liked[id] = true
	}
	return liked, nil
}

func (r *FeedRepository) ListByFollowing(ctx context.Context, limit int, followerID uint, latestBefore time.Time) ([]*video.Video, error) {
	if followerID == 0 {
		return []*video.Video{}, nil
	}
	sub := r.db.WithContext(ctx).Table("socials").
		Select("vlogger_id").
		Where("follower_id = ?", followerID)
	query := r.db.WithContext(ctx).Model(&video.Video{}).
		Where("author_id IN (?)", sub).
		Order("create_time DESC")
	if !latestBefore.IsZero() {
		query = query.Where("create_time < ?", latestBefore)
	}
	var videos []*video.Video
	err := query.Limit(limit).Find(&videos).Error
	return videos, err
}
