package video

import (
	"backend/internal/apierror"
	rediscache "backend/internal/middleware/redis"
	"context"
	"encoding/json"
	"strings"
	"time"

	"gorm.io/gorm"
)

type VideoService struct {
	repo  *VideoRepository
	cache *rediscache.Client
}

func NewVideoService(db *gorm.DB, cache *rediscache.Client) *VideoService {
	return &VideoService{repo: NewVideoRepository(db), cache: cache}
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
		msg := OutboxMsg{
			VideoID:    v.ID,
			EventType:  "video_published",
			Status:     "pending",
			CreateTime: v.CreateTime,
		}
		if err := tx.Create(&msg).Error; err != nil {
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

func (s *VideoService) ListByAuthorID(ctx context.Context, authorID uint) ([]Video, error) {
	return s.repo.ListByAuthorID(ctx, authorID)
}

//为什么redis未查到要加锁而不是直接去数据库里查？

//### 核心原因：防止重复回填缓存
//想象场景：热点视频缓存过期了，**同时几十个请求打到这个接口**
//1. 所有请求第一步：读 Redis → 全部缓存未命中
//2. 它们都进入抢分布式锁的环节
//   - ✅ 只有 1 个请求抢到锁
//   - ❌ 剩下一堆请求抢锁失败，进入等待
//如果抢到锁的请求直接去查 MySQL、写缓存：
//> 没问题，但是有一个情况：**在你抢到锁之前，别的请求已经把缓存写好了！**
//举个时序：
//1. 请求 A：缓存未命中 → 抢锁 ✅ 成功
//2. 请求 B：缓存未命中 → 抢锁 ❌ 失败，等待
//3. 请求 A：查 MySQL，把数据写回 Redis，释放锁
//4. 请求 B：等待结束，抢到锁了！
//👉 这时如果 B 不先读 Redis，直接又去查一遍 MySQL、重复写缓存，就白白多查一次 DB。

func (s *VideoService) GetDetail(ctx context.Context, id uint) (*Video, error) {
	if id == 0 {
		return nil, apierror.ErrValidation
	}
	//这属于降级处理：如果没有配置缓存客户端，就直接查询数据库。
	if s.cache == nil {
		return s.repo.GetByID(ctx, id)
	}
	cacheKey := s.cache.Key("video:detail:id=%d", id)
	//如果缓存命中，直接返回，不需要查询 MySQL。
	//如果没命中，就继续尝试加锁。
	v, err := s.readDetail(ctx, cacheKey)
	if err == nil {
		return v, nil
	}
	if !rediscache.IsMiss(err) {
		return s.repo.GetByID(ctx, id)
	}
	//缓存 Key：video:detail:id=123
	//锁的 Key：lock:video:detail:id=123
	lockKey := "lock:" + cacheKey
	lockCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	//token：当前请求的锁标识。
	//locked：是否成功拿到锁。
	//lockErr：获取锁时是否发生错误。
	token, locked, lockErr := s.cache.Lock(lockCtx, lockKey, 2*time.Second)
	cancel()
	//加锁成功：再次查缓存
	if lockErr == nil && locked {
		//为什么拿到锁后还要再次查询 Redis？
		//因为从第一次读缓存到拿到锁之间，可能已经有其他请求完成了查库和缓存回填
		defer func() { _ = s.cache.Unlock(context.Background(), lockKey, token) }()
		if v, err := s.readDetail(ctx, cacheKey); err == nil {
			return v, nil
		}
		video, err := s.repo.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		s.writeDetail(ctx, cacheKey, video)
		return video, nil
	}
	//加锁失败时：等待其他请求回填缓存
	//当请求没能成功拿到锁时，可能是另一个请求正在查数据库并重建缓存
	for i := 0; i < 5; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(20 * time.Millisecond):
		}
		if v, ok := s.readDetail(ctx, cacheKey); ok == nil {
			return v, nil
		}
	}
	video, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.writeDetail(ctx, cacheKey, video)
	return video, nil
}

func (s *VideoService) readDetail(ctx context.Context, key string) (*Video, error) {
	opCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	b, err := s.cache.GetBytes(opCtx, key)
	if err != nil {
		return nil, err
	}
	var video Video
	if err := json.Unmarshal(b, &video); err != nil {
		return nil, err
	}
	return &video, nil
}

func (s *VideoService) writeDetail(ctx context.Context, key string, video *Video) {
	b, err := json.Marshal(video)
	if err != nil {
		return
	}
	opCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	_ = s.cache.SetBytes(opCtx, key, b, 5*time.Minute)
}

func (s *VideoService) ForgetDetail(ctx context.Context, id uint) {
	if s.cache == nil || id == 0 {
		return
	}
	opCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	_ = s.cache.Del(opCtx, s.cache.Key("video:detail:id=%d", id))
}
