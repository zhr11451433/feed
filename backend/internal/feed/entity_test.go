package feed

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"backend/internal/video"
)

func TestNewFeedVideoItem(t *testing.T) {
	created := time.Date(2026, 10, 18, 14, 26, 13, 157_000_000, time.FixedZone("CST", 8*3600))
	item := NewFeedVideoItem(video.Video{
		ID: 1, AuthorID: 10, Username: "zhr", Title: "标题",
		CreateTime: created, LikesCount: 3, Popularity: 9,
	}, false)

	if item.Author.ID != 10 || item.Author.Username != "zhr" {
		t.Fatalf("author = %+v", item.Author)
	}
	if item.CreateTime != created.UnixMilli() {
		t.Fatalf("create_time = %d", item.CreateTime)
	}
	if item.IsLiked {
		t.Fatal("is_liked should be false")
	}

	b, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	if strings.Contains(got, "author_id") || strings.Contains(got, "popularity") {
		t.Fatalf("table fields leaked: %s", got)
	}
	if !strings.Contains(got, `"author":{"id":10,"username":"zhr"}`) {
		t.Fatalf("author = %s", got)
	}
}
