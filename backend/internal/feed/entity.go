package feed

import "backend/internal/video"

type FeedAuthor struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

type FeedVideoItem struct {
	ID          uint       `json:"id"`
	Author      FeedAuthor `json:"author"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	PlayURL     string     `json:"play_url"`
	CoverURL    string     `json:"cover_url"`
	CreateTime  int64      `json:"create_time"`
	LikesCount  int64      `json:"likes_count"`
	IsLiked     bool       `json:"is_liked"`
}

func NewFeedVideoItem(v video.Video, isLiked bool) FeedVideoItem {
	return FeedVideoItem{
		ID:          v.ID,
		Author:      FeedAuthor{ID: v.AuthorID, Username: v.Username},
		Title:       v.Title,
		Description: v.Description,
		PlayURL:     v.PlayURL,
		CoverURL:    v.CoverURL,
		CreateTime:  v.CreateTime.UnixMilli(),
		LikesCount:  v.LikesCount,
		IsLiked:     isLiked,
	}
}

type ListLatestRequest struct {
	Limit      int   `json:"limit"`       // 每页加载条数，比如 10 条
	LatestTime int64 `json:"latest_time"` // 游标：上一页最后一条视频的创建时间戳（毫秒）
}

type ListLatestResponse struct {
	VideoList []FeedVideoItem `json:"video_list"` // 当前页的视频数据列表
	NextTime  int64           `json:"next_time"`  // 下一页的游标：本页最后一条的创建时间戳
	HasMore   bool            `json:"has_more"`   // 还有没有更多数据，前端控制「加载更多」提示
}
