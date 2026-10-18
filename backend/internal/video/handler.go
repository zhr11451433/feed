package video

import (
	"backend/internal/apierror"
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	jwtmw "backend/internal/middleware/jwt"

	"github.com/gin-gonic/gin"
)

var (
	errMissingFile = errors.New("missing file")
	errInvalidSize = errors.New("invalid file size")
	errBadContent  = errors.New("invalid file content")
)

type VideoHandler struct {
	store *Store
	svc   *VideoService
}

func NewVideoHandler(store *Store, svc *VideoService) *VideoHandler {
	return &VideoHandler{store: store, svc: svc}
}

func (h *VideoHandler) UploadVideo(c *gin.Context) {
	h.upload(c, "videos", 200<<20, map[string]string{".mp4": "video/mp4"}, "only .mp4 is allowed", "play_url")
}

func (h *VideoHandler) UploadCover(c *gin.Context) {
	h.upload(c, "covers", 10<<20, map[string]string{
		".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png", ".webp": "image/webp",
	}, "only .jpg/.jpeg/.png/.webp is allowed", "cover_url")
}

func (h *VideoHandler) PublishVideo(c *gin.Context) {
	var req PublishVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	authorID, err := jwtmw.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	username, err := jwtmw.GetUsername(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	v := &Video{
		AuthorID:    authorID,
		Username:    username,
		Title:       req.Title,
		Description: req.Description,
		PlayURL:     req.PlayURL,
		CoverURL:    req.CoverURL,
	}
	if err := h.svc.Publish(c.Request.Context(), v); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *VideoHandler) GetDetail(c *gin.Context) {
	var req GetDetailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	v, err := h.svc.GetDetail(c.Request.Context(), req.ID)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *VideoHandler) ListByAuthorID(c *gin.Context) {
	var req ListByAuthorIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	videos, err := h.svc.ListByAuthorID(c.Request.Context(), req.AuthorID)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if videos == nil {
		videos = []Video{}
	}
	c.JSON(http.StatusOK, videos)
}

func (h *VideoHandler) upload(c *gin.Context, kind string, maxSize int64, allow map[string]string, extErr, urlField string) {
	authorID, err := jwtmw.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	publicURL, err := readAndSave(c.Request, h.store, authorID, kind, maxSize, allow, extErr)
	if err != nil {
		status := http.StatusBadRequest
		if !errors.Is(err, errMissingFile) && !errors.Is(err, errInvalidSize) && !errors.Is(err, errBadContent) && err.Error() != extErr {
			status = http.StatusInternalServerError
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": publicURL, urlField: publicURL})
}

// 解析 multipart 表单，提取文件，完成扩展名、真实类型、大小三重校验，最终调用 Store 落盘
func readAndSave(r *http.Request, store *Store, authorID uint, kind string, maxSize int64, allow map[string]string, extErr string) (string, error) {
	mr, err := r.MultipartReader()
	if err != nil {
		return "", errMissingFile
	}
	var part *multipart.Part
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if p.FormName() == "file" {
			part = p
			break
		}
		p.Close() // 跳过的 part 要读完丢掉，NextPart 才能继续
	}
	if part == nil {
		return "", errMissingFile
	}
	defer part.Close()

	ext := strings.ToLower(filepath.Ext(part.FileName()))
	want, ok := allow[ext]
	if !ok {
		return "", errors.New(extErr)
	}

	head := make([]byte, 512)
	n, err := io.ReadFull(part, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && err != io.EOF {
		return "", err
	}
	head = head[:n]
	if n == 0 {
		return "", errInvalidSize
	}
	// 看文件头，不看客户端自己填的 Content-Type
	if http.DetectContentType(head) != want {
		return "", errBadContent
	}
	if int64(n) > maxSize {
		return "", errInvalidSize
	}
	src := io.MultiReader(bytes.NewReader(head), &limitedReader{r: part, remain: maxSize - int64(n)})
	return store.Save(r, authorID, kind, ext, src)
}

// remain 用完后再读到字节，就说明文件超过上限。
// Save 看到这个错误会删掉半成品。
// 包装输入流，强制限制最大读取总量，超过上限立刻返回错误，实现流式拦截，不用等整个文件上传完才报错。
type limitedReader struct {
	r      io.Reader
	remain int64
}

func (l *limitedReader) Read(p []byte) (int, error) {
	// 1. 剩余额度已经用完
	if l.remain <= 0 {
		// 再试探读1个字节
		var b [1]byte
		n, err := l.r.Read(b[:])
		if n > 0 {
			// 还有数据，说明文件超限，返回大小错误
			return 0, errInvalidSize
		}
		if err == io.EOF || err == nil {
			return 0, io.EOF
		}
		return 0, err
	}
	if int64(len(p)) > l.remain {
		p = p[:l.remain]
	}
	n, err := l.r.Read(p)
	l.remain -= int64(n)
	return n, err
}
