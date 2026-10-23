package feed

import (
	rediscache "backend/internal/middleware/redis"
	"backend/internal/video"
	"context"
	"strconv"
	"time"
)

type FeedService struct {
	repo  *FeedRepository
	cache *rediscache.Client
}

func NewFeedService(repo *FeedRepository, cache *rediscache.Client) *FeedService {
	return &FeedService{repo: repo, cache: cache}
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

func (s *FeedService) ListByFollowing(ctx context.Context, limit int, latestBefore time.Time, viewerID uint) (ListByFollowingResponse, error) {
	videos, err := s.repo.ListByFollowing(ctx, limit, viewerID, latestBefore)
	if err != nil {
		return ListByFollowingResponse{}, err
	}
	ids := make([]uint, 0, len(videos))
	for _, v := range videos {
		if v != nil {
			ids = append(ids, v.ID)
		}
	}
	liked, err := s.repo.ListLikedIDs(ctx, viewerID, ids)
	if err != nil {
		return ListByFollowingResponse{}, err
	}
	items := make([]FeedVideoItem, 0, len(videos))
	for _, v := range videos {
		if v == nil {
			continue
		}
		items = append(items, NewFeedVideoItem(*v, liked[v.ID]))
	}
	var nextTime int64
	if n := len(videos); n > 0 && videos[n-1] != nil {
		// 秒。listLatest 这里写的是 UnixMilli()。
		nextTime = videos[n-1].CreateTime.Unix()
	}
	return ListByFollowingResponse{
		VideoList: items,
		NextTime:  nextTime,
		HasMore:   len(videos) == limit,
	}, nil
}

func (s *FeedService) ListByPopularity(ctx context.Context, req ListByPopularityRequest, viewerID uint) (ListByPopularityResponse, error) {
	// 1. 如果 s.cache == nil (Redis 没配置或挂了)，直接跳过这里，去最后一行查 DB
	if s.cache != nil {
		//resp：查询结果。
		//ok：这次是否成功使用了 Redis 快照。
		//err：是否发生需要向上返回的业务错误
		resp, ok, err := s.listHotSnapshot(ctx, req, viewerID)
		// 【第一个 if】处理异常：如果 Redis 报错了（网络问题、超时等），直接把错误往上抛给上层。
		// 注意：这里通常不会回退到 DB，因为如果是系统级故障，回退 DB 可能会把 DB 打垮。
		if err != nil {
			return ListByPopularityResponse{}, err
		}
		if ok {
			return resp, nil
		}
	}
	return s.listHotFromDB(ctx, req, viewerID)
}

func (s *FeedService) listHotSnapshot(ctx context.Context, req ListByPopularityRequest, viewerID uint) (ListByPopularityResponse, bool, error) {
	//第一次请求没有 AsOf，使用当前 UTC 时间。
	//后续请求携带 AsOf，就恢复之前的时间。
	//Truncate(time.Minute) 把时间截断到整分钟。
	asOf := time.Now().UTC().Truncate(time.Minute)
	if req.AsOf > 0 {
		asOf = time.UnixMilli(req.AsOf).UTC().Truncate(time.Minute)
	}
	//找到最近 60 分钟的热度数据
	keys := make([]string, 0, 60)
	for i := 0; i < 60; i++ {
		keys = append(keys, s.cache.Key("hot:video:1m:%s", asOf.Add(-time.Duration(i)*time.Minute).Format("200601021504")))
	}
	dest := s.cache.Key("hot:video:merge:1m:%s", asOf.Format("200601021504"))
	opCtx, cancel := context.WithTimeout(ctx, 80*time.Millisecond)
	defer cancel()
	//如果榜单不存在，就合并生成
	exists, err := s.cache.Exists(opCtx, dest)
	if err != nil {
		return ListByPopularityResponse{}, false, nil
	}
	if !exists {
		if err := s.cache.ZUnionStore(opCtx, dest, keys); err != nil {
			return ListByPopularityResponse{}, false, nil
		}
		_ = s.cache.Expire(opCtx, dest, 2*time.Minute)
	}
	//按照 offset 从热榜中取视频
	start := int64(req.Offset)
	members, err := s.cache.ZRevRange(opCtx, dest, start, start+int64(req.Limit)-1)
	if err != nil || (len(members) == 0 && req.Offset == 0) {
		return ListByPopularityResponse{}, false, nil
	}
	if len(members) == 0 {
		return ListByPopularityResponse{
			VideoList:  []FeedVideoItem{},
			AsOf:       asOf.UnixMilli(),
			NextOffset: req.Offset,
			HasMore:    false,
		}, true, nil
	}
	//把 Redis 返回的视频 ID 转成完整视频
	//Redis 热榜里存的是视频 ID 和热度分数，不是完整的视频详情
	ids := make([]uint, 0, len(members))
	for _, m := range members {
		n, err := strconv.ParseUint(m, 10, 64)
		if err == nil && n > 0 {
			ids = append(ids, uint(n))
		}
	}
	videos, err := s.repo.GetByIDs(ctx, ids)
	if err != nil {
		return ListByPopularityResponse{}, false, err
	}
	byID := make(map[uint]*video.Video, len(videos))
	for _, v := range videos {
		if v != nil {
			byID[v.ID] = v
		}
	}
	ordered := make([]*video.Video, 0, len(ids))
	for _, id := range ids {
		if v := byID[id]; v != nil {
			ordered = append(ordered, v)
		}
	}
	//给视频补充当前用户的点赞状态
	items, err := s.feedItems(ctx, ordered, viewerID)
	if err != nil {
		return ListByPopularityResponse{}, false, err
	}
	return ListByPopularityResponse{
		VideoList:  items,
		AsOf:       asOf.UnixMilli(),
		NextOffset: req.Offset + len(items),
		HasMore:    len(items) == req.Limit,
	}, true, nil
}

func (s *FeedService) listHotFromDB(ctx context.Context, req ListByPopularityRequest, viewerID uint) (ListByPopularityResponse, error) {
	var idBefore uint
	if req.LatestIDBefore != nil {
		idBefore = *req.LatestIDBefore
	}
	videos, err := s.repo.ListByPopularity(ctx, req.Limit, req.LatestPopularity, req.LatestBefore, idBefore)
	if err != nil {
		return ListByPopularityResponse{}, err
	}
	items, err := s.feedItems(ctx, videos, viewerID)
	if err != nil {
		return ListByPopularityResponse{}, err
	}
	resp := ListByPopularityResponse{VideoList: items, HasMore: len(items) == req.Limit}
	if n := len(videos); n > 0 && videos[n-1] != nil {
		last := videos[n-1]
		pop := last.Popularity
		tm := last.CreateTime
		id := last.ID
		resp.NextLatestPopularity = &pop
		resp.NextLatestBefore = &tm
		resp.NextLatestIDBefore = &id
	}
	return resp, nil
}

func (s *FeedService) feedItems(ctx context.Context, videos []*video.Video, viewerID uint) ([]FeedVideoItem, error) {
	ids := make([]uint, 0, len(videos))
	for _, v := range videos {
		if v != nil {
			ids = append(ids, v.ID)
		}
	}
	liked, err := s.repo.ListLikedIDs(ctx, viewerID, ids)
	if err != nil {
		return nil, err
	}
	items := make([]FeedVideoItem, 0, len(videos))
	for _, v := range videos {
		if v != nil {
			items = append(items, NewFeedVideoItem(*v, liked[v.ID]))
		}
	}
	return items, nil
}

func (s *FeedService) ListLikesCount(ctx context.Context, limit int, cursor *LikesCountCursor, viewerID uint) (ListLikesCountResponse, error) {
	videos, err := s.repo.ListLikesCount(ctx, limit, cursor)
	if err != nil {
		return ListLikesCountResponse{}, err
	}
	items, err := s.feedItems(ctx, videos, viewerID)
	if err != nil {
		return ListLikesCountResponse{}, err
	}
	resp := ListLikesCountResponse{VideoList: items, HasMore: len(videos) == limit}
	if n := len(videos); n > 0 && videos[n-1] != nil {
		last := videos[n-1]
		likes := last.LikesCount
		id := last.ID
		resp.NextLikesCountBefore = &likes
		resp.NextIDBefore = &id
	}
	return resp, nil
}

func (s *FeedService) ListByTag(ctx context.Context, tagName string, limit int, viewerID uint) ([]FeedVideoItem, error) {
	videos, err := s.repo.ListByTag(ctx, tagName, limit)
	if err != nil {
		return nil, err
	}
	return s.feedItems(ctx, videos, viewerID)
}
