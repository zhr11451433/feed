package video

import (
	jwtmw "backend/internal/middleware/jwt"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
)

type ChunkHandler struct {
	store *ChunkSessionStore
}

func NewChunkHandler(store *ChunkSessionStore) *ChunkHandler {
	return &ChunkHandler{store: store}
}

func (h *ChunkHandler) Init(c *gin.Context) {
	if h.store == nil || h.store.Cache == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": ErrChunkCacheUnavailable.Error()})
		return
	}
	var req struct {
		Filename    string `json:"filename"`
		FileSize    int64  `json:"file_size"`
		ChunkSize   int64  `json:"chunk_size"`
		TotalChunks int    `json:"total_chunks"`
		FileHash    string `json:"file_hash"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	accountID, err := jwtmw.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	if req.Filename == "" || req.FileHash == "" || req.FileSize < 1 || req.TotalChunks < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "filename, file_hash, file_size and total_chunks are required"})
		return
	}
	if req.FileSize > 200<<20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file size exceeds 200MB limit"})
		return
	}
	session, _, err := h.store.Begin(c.Request.Context(), accountID, req.Filename, req.FileSize, req.ChunkSize, req.TotalChunks, req.FileHash)
	if err != nil {
		if err.Error() == "invalid chunk session" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": ErrChunkCacheUnavailable.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"upload_id":       session.UploadID,
		"uploaded_chunks": session.UploadedChunks(),
	})
}

func (h *ChunkHandler) Status(c *gin.Context) {
	if h.store == nil || h.store.Cache == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": ErrChunkCacheUnavailable.Error()})
		return
	}
	var req struct {
		UploadID string `json:"upload_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.UploadID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "upload_id is required"})
		return
	}
	accountID, err := jwtmw.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	session, err := h.store.Get(c.Request.Context(), req.UploadID)
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			c.JSON(http.StatusNotFound, gin.H{"error": "upload session not found"})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": ErrChunkCacheUnavailable.Error()})
		return
	}
	if session.AccountID != accountID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"upload_id":       session.UploadID,
		"uploaded_chunks": session.UploadedChunks(),
		"total_chunks":    session.TotalChunks,
	})
}

//这个用户上传的这片数据，确实属于这个上传任务，并且校验通过后才算完成

func (h *ChunkHandler) Upload(c *gin.Context) {
	if h.store == nil || h.store.Cache == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": ErrChunkCacheUnavailable.Error()})
		return
	}
	uploadID := c.PostForm("upload_id")
	hashFromClient := c.PostForm("chunk_hash")
	indexText := c.PostForm("chunk_index")
	if uploadID == "" || hashFromClient == "" || indexText == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "upload_id, chunk_index and chunk_hash are required"})
		return
	}
	//把分片编号从字符串转成整数
	chunkIndex, err := strconv.Atoi(indexText)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chunk_index"})
		return
	}
	accountID, err := jwtmw.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	session, err := h.store.Get(c.Request.Context(), uploadID)
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			c.JSON(http.StatusNotFound, gin.H{"error": "upload session not found"})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": ErrChunkCacheUnavailable.Error()})
		return
	}
	//上传会话归属校验
	if session.AccountID != accountID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	if chunkIndex < 0 || chunkIndex >= session.TotalChunks {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chunk_index"})
		return
	}
	//重复请求不重复上传
	if session.UploadedBits[chunkIndex] {
		c.JSON(http.StatusOK, gin.H{"chunk_index": chunkIndex})
		return
	}
	//获取真正的分片文件
	f, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing file"})
		return
	}
	chunkFile, err := f.Open() //才是真正打开文件数据的操作
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read chunk"})
		return
	}
	defer func() {
		_ = chunkFile.Close()
	}()

	sum := md5.New()
	if _, err := io.Copy(sum, chunkFile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash chunk"})
		return
	}
	actual := fmt.Sprintf("%x", sum.Sum(nil))

	if actual != hashFromClient {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":    "chunk hash mismatch",
			"expected": hashFromClient,
			"actual":   actual,
		})
		return
	}
	//第一次读取：计算哈希。
	//Seek：回到开头。
	//第二次读取：把数据写入正式的分片临时文件
	if _, err := chunkFile.Seek(0, io.SeekStart); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read chunk"})
		return
	}

	dir := filepath.Join(".run", "uploads", "tmp", uploadID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create temp dir"})
		return
	}
	dst, err := os.Create(filepath.Join(dir, strconv.Itoa(chunkIndex)))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save chunk"})
		return
	}
	defer func() {
		_ = dst.Close()
	}()
	if _, err := io.Copy(dst, chunkFile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save chunk"})
		return
	}

	session.UploadedBits[chunkIndex] = true
	if err := h.store.Save(c.Request.Context(), session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"chunk_index": chunkIndex})
}

func (h *ChunkHandler) Complete(c *gin.Context) {
	if h.store == nil || h.store.Cache == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": ErrChunkCacheUnavailable.Error()})
		return
	}
	var req struct {
		UploadID string `json:"upload_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.UploadID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "upload_id is required"})
		return
	}
	accountID, err := jwtmw.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	session, err := h.store.Get(c.Request.Context(), req.UploadID)
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			c.JSON(http.StatusNotFound, gin.H{"error": "upload session not found"})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": ErrChunkCacheUnavailable.Error()})
		return
	}
	if session.AccountID != accountID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	for _, uploaded := range session.UploadedBits {
		if !uploaded {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":           "not all chunks uploaded",
				"uploaded_chunks": session.UploadedChunks(),
				"total_chunks":    session.TotalChunks,
			})
			return
		}
	}

	date := time.Now().Format("20060102")
	name, err := randHex(16)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate filename"})
		return
	}
	relDir := filepath.Join("videos", strconv.FormatUint(uint64(accountID), 10), date)
	absDir := filepath.Join(".run", "uploads", relDir)
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create output dir"})
		return
	}
	finalPath := filepath.Join(absDir, name+".mp4")
	finalFile, err := os.Create(finalPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create final file"})
		return
	}
	defer finalFile.Close()

	tmpDir := filepath.Join(".run", "uploads", "tmp", req.UploadID)
	for i := 0; i < session.TotalChunks; i++ {
		cf, err := os.Open(filepath.Join(tmpDir, strconv.Itoa(i)))
		if err != nil {
			finalFile.Close()
			_ = os.Remove(finalPath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("chunk %d missing", i)})
			return
		}
		_, copyErr := io.Copy(finalFile, cf)
		cf.Close()
		if copyErr != nil {
			finalFile.Close()
			_ = os.Remove(finalPath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to merge chunks"})
			return
		}
	}

	_ = os.RemoveAll(tmpDir)
	_ = h.store.Cache.Del(c.Request.Context(), h.store.sessionKey(req.UploadID))
	_ = h.store.Cache.Del(c.Request.Context(), h.store.hashKey(accountID, session.FileHash))

	urlPath := path.Join("/static", "videos", strconv.FormatUint(uint64(accountID), 10), date, name+".mp4")
	playURL := PublicURL(c.Request, urlPath)
	c.JSON(http.StatusOK, gin.H{"url": playURL, "play_url": playURL})
}
