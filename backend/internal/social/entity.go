package social

import "backend/internal/account"

// 同一个 (follower_id, vlogger_id) 只能有一行。
// 最左列是 follower_id，「我关注了谁」能用上这条唯一索引。
// vlogger_id 上的普通索引给「谁关注了我」用。

type Social struct {
	ID         uint `gorm:"primaryKey"`
	FollowerID uint `gorm:"not null;uniqueIndex:idx_social_follower_vlogger,priority:1"`
	VloggerID  uint `gorm:"not null;uniqueIndex:idx_social_follower_vlogger,priority:2;index"`
}

type FollowRequest struct {
	VloggerID uint `json:"vlogger_id"`
}

type GetAllFollowersRequest struct {
	VloggerID uint `json:"vlogger_id"`
}

type GetAllVloggersRequest struct {
	FollowerID uint `json:"follower_id"`
}

type GetAllFollowersResponse struct {
	Followers     []account.Account `json:"followers"`
	FollowerCount int64             `json:"follower_count"`
}

type GetAllVloggersResponse struct {
	Vloggers     []account.Account `json:"vloggers"`
	VloggerCount int64             `json:"vlogger_count"`
}

type SocialCounts struct {
	FollowerCount int64 `json:"follower_count"`
	VloggerCount  int64 `json:"vlogger_count"`
}
