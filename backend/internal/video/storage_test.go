package video

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestSaveUsesRequestHost(t *testing.T) {
	s := &Store{Root: t.TempDir()}
	req := httptest.NewRequest(http.MethodPost, "/video/uploadVideo", nil)
	req.Host = "127.0.0.1:8080"

	got, err := s.Save(req, 7, "videos", ".mp4", strings.NewReader("fake-bytes"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "http://127.0.0.1:8080/static/videos/7/") || !strings.HasSuffix(got, ".mp4") {
		t.Fatalf("url = %s", got)
	}

	req.Host = "feed.example.com"
	req.Header.Set("X-Forwarded-Proto", "https")
	got, err = s.Save(req, 7, "videos", ".mp4", strings.NewReader("x"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "https://feed.example.com/static/videos/7/") {
		t.Fatalf("url = %s", got)
	}

	entries, err := os.ReadDir(s.Root)
	if err != nil || len(entries) == 0 {
		t.Fatal("file was not written")
	}
}
