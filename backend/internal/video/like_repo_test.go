package video_test

import (
	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/video"
	"context"
	"testing"
)

func TestLikeIgnoreDuplicateRejectsEmpty(t *testing.T) {
	repo := video.NewLikeRepository(nil)
	inserted, err := repo.LikeIgnoreDuplicate(context.Background(), nil)
	if inserted || err != nil {
		t.Fatalf("nil like: inserted=%v err=%v", inserted, err)
	}
	inserted, err = repo.LikeIgnoreDuplicate(context.Background(), &video.Like{})
	if inserted || err != nil {
		t.Fatalf("zero ids: inserted=%v err=%v", inserted, err)
	}
}

func TestLikeIgnoreDuplicate(t *testing.T) {
	gdb, err := db.NewDB(config.DefaultLocalConfig().Database)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.CloseDB(gdb) })
	if err := gdb.AutoMigrate(&video.Like{}); err != nil {
		t.Fatal(err)
	}
	const videoID, accountID uint = 900001, 900001
	t.Cleanup(func() {
		gdb.Where("video_id = ? AND account_id = ?", videoID, accountID).Delete(&video.Like{})
	})
	repo := video.NewLikeRepository(gdb)
	inserted, err := repo.LikeIgnoreDuplicate(context.Background(), &video.Like{VideoID: videoID, AccountID: accountID})
	if err != nil || !inserted {
		t.Fatalf("first insert: inserted=%v err=%v", inserted, err)
	}
	inserted, err = repo.LikeIgnoreDuplicate(context.Background(), &video.Like{VideoID: videoID, AccountID: accountID})
	if err != nil {
		t.Fatal(err)
	}
	if inserted {
		t.Fatal("second like inserted a duplicate row")
	}
}
