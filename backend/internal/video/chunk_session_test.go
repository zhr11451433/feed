package video

import (
	"context"
	"testing"

	"backend/internal/config"
	rediscache "backend/internal/middleware/redis"
)

func TestChunkSessionReuse(t *testing.T) {
	cfg, err := config.Load("../../configs/config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	cache, err := rediscache.NewFromEnv(&cfg.Redis)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cache.Close() })
	if err := cache.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}

	store := NewChunkSessionStore(cache)
	ctx := context.Background()
	const (
		accountID = uint(1001)
		fileHash  = "t1001-reuse-hash"
	)
	t.Cleanup(func() {
		_ = cache.Del(ctx, store.hashKey(accountID, fileHash))
	})

	first, reused, err := store.Begin(ctx, accountID, "a.mp4", 10<<20, ChunkSize, 2, fileHash)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cache.Del(ctx, store.sessionKey(first.UploadID)) })
	if reused || first.UploadID == "" || len(first.UploadedChunks()) != 0 {
		t.Fatalf("first session: reused=%v id=%s chunks=%v", reused, first.UploadID, first.UploadedChunks())
	}

	first.UploadedBits[0] = true
	if err := store.Save(ctx, first); err != nil {
		t.Fatal(err)
	}

	second, reused, err := store.Begin(ctx, accountID, "a.mp4", 10<<20, ChunkSize, 2, fileHash)
	if err != nil {
		t.Fatal(err)
	}
	if !reused || second.UploadID != first.UploadID || len(second.UploadedChunks()) != 1 || second.UploadedChunks()[0] != 0 {
		t.Fatalf("reuse: reused=%v id=%s chunks=%v", reused, second.UploadID, second.UploadedChunks())
	}

	other, reused, err := store.Begin(ctx, accountID+1, "a.mp4", 10<<20, ChunkSize, 2, fileHash)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cache.Del(ctx, store.sessionKey(other.UploadID))
		_ = cache.Del(ctx, store.hashKey(accountID+1, fileHash))
	})
	if reused || other.UploadID == first.UploadID {
		t.Fatalf("other account reused upload id %s", other.UploadID)
	}
}
