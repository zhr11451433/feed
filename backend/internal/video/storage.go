package video

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Store struct {
	Root string // 磁盘根目录，默认 .run/uploads
}

func NewStore() *Store {
	return &Store{Root: filepath.Join(".run", "uploads")}
}

//这是一个**本地文件上传存储函数**，
//负责把前端上传的文件流（视频、封面、头像）
//按照「分类 / 用户 ID / 日期 / 随机文件名」的规则存到服务器磁盘
//最后返回可以通过 HTTP 访问的静态资源 URL。

func (s *Store) Save(r *http.Request, authorID uint, kind, ext string, src io.Reader) (string, error) {
	if authorID == 0 {
		return "", errors.New("authorID is required")
	}
	switch kind {
	case "videos", "covers", "avatars":
	default:
		return "", errors.New("invalid kind")
	}
	if len(ext) <= 2 || ext[0] != '.' || ext != strings.ToLower(ext) || strings.ContainsAny(ext, `/\`) {
		return "", errors.New("invalid ext")
	}
	date := time.Now().Format("20060102")
	// 生成16位随机十六进制文件名
	name, err := randHex(16)
	if err != nil {
		return "", err
	}
	filename := name + ext
	author := strconv.FormatUint(uint64(authorID), 10)
	absDir := filepath.Join(s.Root, kind, author, date)
	if err := os.MkdirAll(absDir, 0o777); err != nil {
		return "", err
	}
	absPath := filepath.Join(absDir, filename)
	f, err := os.Create(absPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	// 把上传的文件流拷贝到磁盘文件
	if _, err := io.Copy(f, src); err != nil {
		// 写入失败，删除半成品文件，不留下脏文件
		_ = os.Remove(absPath)
		return "", err
	}
	// URL必须用正斜杠，所以用path.Join，不用filepath.Join（Windows会生成反斜杠）
	urlPath := path.Join("/static", kind, author, date, filename)
	return PublicURL(r, urlPath), nil
}

func PublicURL(r *http.Request, urlPath string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if xf := r.Header.Get("X-Forwarded-Proto"); xf != "" {
		scheme = xf
	}
	return fmt.Sprintf("%s://%s%s", scheme, r.Host, urlPath)
}

func randHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand.Read: %w", err)
	}
	return hex.EncodeToString(b), nil
}
