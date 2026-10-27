package feed

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestListLikesCountCursorBoundaries(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/feed/listLikesCount", NewFeedHandler(nil).ListLikesCount)
	r.POST("/feed/listByPopularity", NewFeedHandler(nil).ListByPopularity)

	cases := []struct {
		path string
		body string
		want string
	}{
		{"/feed/listLikesCount", `{"likes_count_before":1}`, "must be provided together"},
		{"/feed/listLikesCount", `{"id_before":1}`, "must be provided together"},
		{"/feed/listLikesCount", `{"likes_count_before":-1,"id_before":1}`, "must be >= 0"},
		{"/feed/listLikesCount", `{"likes_count_before":3,"id_before":0}`, "must be > 0"},
		{"/feed/listByPopularity", `{"latest_before":"2026-01-01T00:00:00Z"}`, "must be provided together"},
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		var got struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if w.Code != http.StatusBadRequest || !strings.Contains(got.Error, tc.want) {
			t.Fatalf("%s %s => %d %s", tc.path, tc.body, w.Code, got.Error)
		}
	}
}
