package video

import (
	rediscache "backend/internal/middleware/redis"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const (
	ChunkSize  int64 = 5 << 20 // 5MB
	sessionTTL       = 24 * time.Hour
)

var ErrChunkCacheUnavailable = errors.New("chunk Cache unavailable")

type ChunkUploadSession struct {
	UploadID     string `json:"upload_id"`
	AccountID    uint   `json:"account_id"`
	Filename     string `json:"filename"`
	FileSize     int64  `json:"file_size"`
	ChunkSize    int64  `json:"chunk_size"`
	TotalChunks  int    `json:"total_chunks"`
	FileHash     string `json:"file_hash"`
	UploadedBits []bool `json:"uploaded_bits"`
}

func (s *ChunkUploadSession) UploadedChunks() []int {
	indices := make([]int, 0)
	for i, uploaded := range s.UploadedBits {
		if uploaded {
			indices = append(indices, i)
		}
	}
	return indices
}

type ChunkSessionStore struct {
	Cache *rediscache.Client
}

func NewChunkSessionStore(cache *rediscache.Client) *ChunkSessionStore {
	return &ChunkSessionStore{Cache: cache}
}

func (s *ChunkSessionStore) sessionKey(uploadID string) string {
	return s.Cache.Key("chunk_upload:%s", uploadID)
}

func (s *ChunkSessionStore) hashKey(accountID uint, fileHash string) string {
	return s.Cache.Key("chunk_upload_hash:%d:%s", accountID, fileHash)
}

func (s *ChunkSessionStore) Get(ctx context.Context, uploadID string) (*ChunkUploadSession, error) {
	if s == nil || s.Cache == nil {
		return nil, ErrChunkCacheUnavailable
	}
	b, err := s.Cache.GetBytes(ctx, s.sessionKey(uploadID))
	if err != nil {
		return nil, err
	}
	var session ChunkUploadSession
	if err := json.Unmarshal(b, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *ChunkSessionStore) Save(ctx context.Context, session *ChunkUploadSession) error {
	if s == nil || s.Cache == nil {
		return ErrChunkCacheUnavailable
	}
	b, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return s.Cache.SetBytes(ctx, s.sessionKey(session.UploadID), b, sessionTTL)
}

// Begin 按账号和 file_hash 找回没传完的会话。没有才新建。
func (s *ChunkSessionStore) Begin(ctx context.Context, accountID uint, filename string, fileSize, chunkSize int64, totalChunks int, fileHash string) (*ChunkUploadSession, bool, error) {
	if s == nil || s.Cache == nil {
		return nil, false, ErrChunkCacheUnavailable
	}
	if chunkSize != ChunkSize || accountID == 0 || fileHash == "" || totalChunks < 1 || fileSize < 1 {
		return nil, false, errors.New("invalid chunk session")
	}
	hashKey := s.hashKey(accountID, fileHash)
	existingID, err := s.Cache.GetBytes(ctx, hashKey)
	if err == nil && len(existingID) > 0 {
		session, sessErr := s.Get(ctx, string(existingID))
		if sessErr == nil && session.AccountID == accountID && session.FileHash == fileHash {
			if err := s.Cache.SetBytes(ctx, hashKey, existingID, sessionTTL); err != nil {
				return nil, false, err
			}
			if err := s.Save(ctx, session); err != nil {
				return nil, false, err
			}
			return session, true, nil
		}
	} else if err != nil && !errors.Is(err, goredis.Nil) {
		return nil, false, err
	}
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return nil, false, err
	}
	session := &ChunkUploadSession{
		UploadID:     hex.EncodeToString(raw) + strconv.FormatInt(time.Now().UnixNano(), 10),
		AccountID:    accountID,
		Filename:     filename,
		FileSize:     fileSize,
		ChunkSize:    chunkSize,
		TotalChunks:  totalChunks,
		FileHash:     fileHash,
		UploadedBits: make([]bool, totalChunks),
	}
	if err := s.Save(ctx, session); err != nil {
		return nil, false, err
	}
	if err := s.Cache.SetBytes(ctx, hashKey, []byte(session.UploadID), sessionTTL); err != nil {
		return nil, false, err
	}
	return session, false, nil
}
