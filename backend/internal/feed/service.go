package feed

import (
	"context"
	"time"
)

type FeedService struct {
	repo *FeedRepository
}

func NewFeedService(repo *FeedRepository) *FeedService {
	return &FeedService{repo: repo}
}

//游客的 viewerID 是 0，map 是空的，
//每一条都是 false。登录用户以后只要 map 里有这个视频 id，
//就是 true

func (s *FeedService) ListLatest(ctx context.Context, limit int, latestBefore time.Time, viewerID uint) (ListLatestResponse, error) {
	videos, err := s.repo.ListLatest(ctx, limit, latestBefore)
	if err != nil {
		return ListLatestResponse{}, err
	}
	ids := make([]uint, 0, len(videos))
	for _, v := range videos {
		if v != nil {
			ids = append(ids, v.ID)
		}
	}
	liked, err := s.repo.ListLikedIDs(ctx, viewerID, ids)
	if err != nil {
		return ListLatestResponse{}, err
	}
	items := make([]FeedVideoItem, 0, len(videos))
	for _, video := range videos {
		if video == nil {
			continue
		}
		items = append(items, NewFeedVideoItem(*video, liked[video.ID]))
	}
	var nextTime int64
	if n := len(videos); n > 0 && videos[n-1] != nil {
		nextTime = videos[n-1].CreateTime.UnixMilli()
	}
	return ListLatestResponse{
		VideoList: items,
		NextTime:  nextTime,
		HasMore:   len(videos) == limit,
	}, nil
}
