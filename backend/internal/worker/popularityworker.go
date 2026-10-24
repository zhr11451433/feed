package worker

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"backend/internal/middleware/rabbitmq"
	rediscache "backend/internal/middleware/redis"
	"backend/internal/video"

	amqp "github.com/rabbitmq/amqp091-go"
)

type PopularityWorker struct {
	ch    *amqp.Channel
	cache *rediscache.Client
	queue string
}

func NewPopularityWorker(ch *amqp.Channel, cache *rediscache.Client, queue string) *PopularityWorker {
	return &PopularityWorker{ch: ch, cache: cache, queue: queue}
}

func (w *PopularityWorker) Run(ctx context.Context) error {
	if w == nil || w.ch == nil || w.cache == nil || w.queue == "" {
		return errors.New("popularity worker is not initialized")
	}
	deliveries, err := w.ch.Consume(w.queue, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-deliveries:
			if !ok {
				return errors.New("deliveries channel closed")
			}
			w.handleDelivery(ctx, d)
		}
	}
}

func (w *PopularityWorker) handleDelivery(ctx context.Context, d amqp.Delivery) {
	const maxRetries = 3
	for i := 0; i <= maxRetries; i++ {
		if ctx.Err() != nil {
			_ = d.Nack(false, true)
			return
		}
		if err := w.process(ctx, d.Body); err != nil {
			if i >= maxRetries {
				log.Printf("popularity worker: 重试 %d 次后仍失败, 丢弃: %v", maxRetries, err)
				_ = d.Ack(false)
				return
			}
			wait := time.Duration(1<<uint(i)) * time.Second
			log.Printf("popularity worker: 处理失败, %v 后重试 (%d/%d): %v", wait, i+1, maxRetries, err)
			time.Sleep(wait)
			continue
		}
		_ = d.Ack(false)
		return
	}
}

func (w *PopularityWorker) process(ctx context.Context, body []byte) error {
	var evt rabbitmq.PopularityEvent
	if err := json.Unmarshal(body, &evt); err != nil || evt.VideoID == 0 || evt.Change == 0 {
		return nil
	}
	video.UpdatePopularityCache(ctx, w.cache, evt.VideoID, evt.Change)
	log.Printf("popularity worker: video=%d change=%d", evt.VideoID, evt.Change)
	return nil
}
