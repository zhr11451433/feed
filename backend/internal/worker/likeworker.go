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

//LikeWorker：从 RabbitMQ 取出事件，
//真正执行点赞或取消点赞的业务逻辑

type LikeWorker struct {
	ch     *amqp.Channel
	likes  *video.LikeRepository
	videos *video.VideoRepository
	cache  *rediscache.Client
	queue  string
	pop    *rabbitmq.PopularityMQ
}

func NewLikeWorker(ch *amqp.Channel, likes *video.LikeRepository, videos *video.VideoRepository, cache *rediscache.Client, pop *rabbitmq.PopularityMQ, queue string) *LikeWorker {
	return &LikeWorker{ch: ch, likes: likes, videos: videos, cache: cache, queue: queue, pop: pop}
}

func (w *LikeWorker) Run(ctx context.Context) error {
	if w == nil || w.ch == nil || w.likes == nil || w.videos == nil || w.queue == "" {
		return errors.New("like worker is not initialized")
	}
	//Consume() 的意思是：注册一个消费者，
	//让 RabbitMQ 把队列里的消息持续交给当前程序。
	deliveries, err := w.ch.Consume(w.queue, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	for {
		//deliveries：RabbitMQ 送过来的消息通道。
		//d：当前收到的一条消息。
		//ctx.Done()：外部通知 Worker 停止工作的信号。
		//ok == false：消息通道已关闭，Worker 结束运行。
		//handleDelivery()：对这条消息进行处理、重试和确认。
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

////失败重试，成功后确认消息

func (w *LikeWorker) handleDelivery(ctx context.Context, d amqp.Delivery) {
	const maxRetries = 3
	for i := 0; i <= maxRetries; i++ {
		if ctx.Err() != nil {
			_ = d.Nack(false, true)
			return
		}
		if err := w.process(ctx, d.Body); err != nil {
			if i >= maxRetries {
				log.Printf("like worker: 重试 %d 次后进入死信: %v", maxRetries, err)
				_ = d.Nack(false, false)
				return
			}
			wait := time.Duration(1<<uint(i)) * time.Second
			log.Printf("like worker: 处理失败, %v 后重试 (%d/%d): %v", wait, i+1, maxRetries, err)
			time.Sleep(wait)
			continue
		}
		_ = d.Ack(false)
		return
	}
}

// 解析事件并决定执行哪种业务
func (w *LikeWorker) process(ctx context.Context, body []byte) error {
	var evt rabbitmq.LikeEvent
	if err := json.Unmarshal(body, &evt); err != nil || evt.UserID == 0 || evt.VideoID == 0 {
		return nil
	}
	var err error
	switch evt.Action {
	case "like":
		err = w.applyLike(ctx, evt.UserID, evt.VideoID)
	case "unlike":
		err = w.applyUnlike(ctx, evt.UserID, evt.VideoID)
	default:
		return nil
	}
	if err != nil {
		return err
	}
	log.Printf("like worker: %s video=%d user=%d", evt.Action, evt.VideoID, evt.UserID)
	if w.cache != nil {
		_ = w.cache.Del(ctx, w.cache.Key("video:detail:id=%d", evt.VideoID))
	}
	return nil
}

func (w *LikeWorker) applyLike(ctx context.Context, userID, videoID uint) error {
	ok, err := w.videos.IsExist(ctx, videoID)
	if err != nil || !ok {
		return err
	}
	created, err := w.likes.LikeIgnoreDuplicate(ctx, &video.Like{
		VideoID: videoID, AccountID: userID, CreatedAt: time.Now(),
	})
	if err != nil || !created {
		return err
	}
	if err := w.videos.ChangeLikesCount(ctx, videoID, 1); err != nil {
		return err
	}
	if err := w.videos.ChangePopularity(ctx, videoID, 1); err != nil {
		return err
	}
	w.bump(ctx, videoID, 1)
	return nil
}

func (w *LikeWorker) applyUnlike(ctx context.Context, userID, videoID uint) error {
	ok, err := w.videos.IsExist(ctx, videoID)
	if err != nil || !ok {
		return err
	}
	deleted, err := w.likes.DeleteByVideoAndAccount(ctx, videoID, userID)
	if err != nil || !deleted {
		return err
	}
	if err := w.videos.ChangeLikesCount(ctx, videoID, -1); err != nil {
		return err
	}
	if err := w.videos.ChangePopularity(ctx, videoID, -1); err != nil {
		return err
	}
	w.bump(ctx, videoID, -1)
	return nil
}

func (w *LikeWorker) bump(ctx context.Context, videoID uint, change int64) {
	var err error
	if w.pop != nil {
		if err = w.pop.Update(ctx, videoID, change); err == nil {
			return
		}
		log.Printf("popularity publish failed, direct redis: %v", err)
	}
	video.UpdatePopularityCache(ctx, w.cache, videoID, change)
}
