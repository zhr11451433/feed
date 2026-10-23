package feed

import (
	"backend/internal/video"
	"time"
)

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

type ListByFollowingRequest struct {
	Limit int `json:"limit"`
	// 秒。listLatest 的 latest_time 是毫秒，这两个接口不要互相换算。
	LatestTime int64 `json:"latest_time"`
}

type ListByFollowingResponse struct {
	VideoList []FeedVideoItem `json:"video_list"`
	NextTime  int64           `json:"next_time"` // 秒
	HasMore   bool            `json:"has_more"`
}

type ListByPopularityRequest struct {
	Limit            int       `json:"limit"`
	AsOf             int64     `json:"as_of"`
	Offset           int       `json:"offset"`
	LatestPopularity int64     `json:"latest_popularity"`
	LatestBefore     time.Time `json:"latest_before"`
	LatestIDBefore   *uint     `json:"latest_id_before"`
}

type ListByPopularityResponse struct {
	VideoList            []FeedVideoItem `json:"video_list"`
	AsOf                 int64           `json:"as_of"`
	NextOffset           int             `json:"next_offset"`
	HasMore              bool            `json:"has_more"`
	NextLatestPopularity *int64          `json:"next_latest_popularity,omitempty"`
	NextLatestBefore     *time.Time      `json:"next_latest_before,omitempty"`
	NextLatestIDBefore   *uint           `json:"next_latest_id_before,omitempty"`
}

type ListLikesCountRequest struct {
	Limit            int    `json:"limit"`
	LikesCountBefore *int64 `json:"likes_count_before,omitempty"`
	IDBefore         *uint  `json:"id_before,omitempty"`
}

type LikesCountCursor struct {
	LikesCount int64
	ID         uint
}

type ListLikesCountResponse struct {
	VideoList            []FeedVideoItem `json:"video_list"`
	NextLikesCountBefore *int64          `json:"next_likes_count_before,omitempty"`
	NextIDBefore         *uint           `json:"next_id_before,omitempty"`
	HasMore              bool            `json:"has_more"`
}

type ListByTagRequest struct {
	TagName string `json:"tag_name"`
	Limit   int    `json:"limit"`
}
