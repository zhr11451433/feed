package video

import (
	"regexp"
	"time"
)

type Video struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	AuthorID    uint      `gorm:"index;not null" json:"author_id"`
	Username    string    `gorm:"size:255;not null" json:"username"`
	Title       string    `gorm:"size:255;not null" json:"title"`
	Description string    `gorm:"size:255" json:"description,omitempty"`
	PlayURL     string    `gorm:"size:255;not null" json:"play_url"`
	CoverURL    string    `gorm:"size:255;not null" json:"cover_url"`
	CreateTime  time.Time `gorm:"autoCreateTime;index:idx_videos_create_time,sort:desc" json:"create_time"`
	LikesCount  int64     `gorm:"not null;default:0" json:"likes_count"`
	Popularity  int64     `gorm:"not null;default:0" json:"popularity"`
}

type Tag struct {
	ID   uint   `gorm:"primaryKey" json:"id"`
	Name string `gorm:"uniqueIndex;size:100;not null" json:"name"`
}

// 同一个 (video_id, tag_id) 只能有一行。
// 联合索引的最左列是 video_id，所以「查某条视频的话题」能用上它。
// 单独再给 tag_id 一个索引，是为了「查某个话题下有哪些视频」。

type VideoTag struct {
	ID      uint `gorm:"primaryKey"`
	VideoID uint `gorm:"not null;uniqueIndex:uidx_video_tag,priority:1"`
	TagID   uint `gorm:"not null;uniqueIndex:uidx_video_tag,priority:2;index"`
}

type PublishVideoRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	PlayURL     string `json:"play_url"`
	CoverURL    string `json:"cover_url"`
}

// 用来**提取话题标签（# 标签）**，比如 `#go`、`#编程`、`#视频123`，从一段文本里匹配 `#xxx`，并且捕获 `#` 后面的标签文字。
var tagRegex = regexp.MustCompile(`#([\p{L}\p{N}_]+)`)

func ExtractTags(text string) []string {
	//`FindAllStringSubmatch`：**找出全部匹配项，同时返回捕获分组**
	//	- 参数 `-1`：匹配全部，不限制数量
	//- 返回二维切片：`m[0]` = 完整匹配（`#golang`）；`m[1]` = 捕获分组内容（`golang`）
	matches := tagRegex.FindAllStringSubmatch(text, -1)
	//map 用来**做去重**，key 是标签名；value 标记是否已经存在。
	seen := make(map[string]bool)
	var tags []string
	for _, m := range matches {
		name := m[1]
		if seen[name] {
			continue
		}
		seen[name] = true
		tags = append(tags, name)
	}
	return tags
}

type GetDetailRequest struct {
	ID uint `json:"id"`
}

type ListByAuthorIDRequest struct {
	AuthorID uint `json:"author_id"`
}
