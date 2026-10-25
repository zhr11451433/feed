package worker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	jwtmw "backend/internal/middleware/jwt"

	"github.com/gin-gonic/gin"
)

type SSEHub struct {
	//读写互斥锁
	//可以有多个 goroutine 同时持有读锁。
	//写锁只能由一个 goroutine 持有。
	//写锁持有期间，其他 goroutine 不能获取读锁或写锁

	mu      sync.RWMutex
	clients map[uint][]chan *Notification
}

func NewSSEHub() *SSEHub {
	return &SSEHub{clients: make(map[uint][]chan *Notification)}
}

func (h *SSEHub) Push(userID uint, n *Notification) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, ch := range h.clients[userID] {
		select {
		case ch <- n:
		default:
		}
	}
}

func (h *SSEHub) Subscribe(userID uint) chan *Notification {
	ch := make(chan *Notification, 20)
	h.mu.Lock()
	h.clients[userID] = append(h.clients[userID], ch)
	h.mu.Unlock()
	return ch
}

func (h *SSEHub) Unsubscribe(userID uint, ch chan *Notification) {
	h.mu.Lock()
	defer h.mu.Unlock()
	list := h.clients[userID]
	for i, c := range list {
		if c != ch {
			continue
		}
		h.clients[userID] = append(list[:i], list[i+1:]...)
		if len(h.clients[userID]) == 0 {
			delete(h.clients, userID)
		}
		close(c)
		return
	}
}

// 【未学】SSE：这条响应不结束，靠 data 行和 keepalive 往外推
//是把这些通知转换成 SSE 格式并写入 HTTP 响应的地方

func (h *SSEHub) Stream(c *gin.Context) {
	userID, err := jwtmw.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.WriteHeader(http.StatusOK)

	ch := h.Subscribe(userID)
	defer h.Unsubscribe(userID, ch)
	flusher, _ := c.Writer.(http.Flusher)
	if flusher != nil {
		flusher.Flush()
	}

	ctx := c.Request.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case n, ok := <-ch:
			if !ok {
				return
			}
			b, _ := json.Marshal(n)
			_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", b)
			if flusher != nil {
				flusher.Flush()
			}
		case <-time.After(30 * time.Second):
			_, _ = fmt.Fprintf(c.Writer, ": keepalive\n\n")
			if flusher != nil {
				flusher.Flush()
			}
		}
	}
}
