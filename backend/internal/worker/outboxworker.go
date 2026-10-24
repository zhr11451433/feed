package worker

import (
	"backend/internal/middleware/rabbitmq"
	"backend/internal/video"
	"context"
	"log"
	"time"

	"gorm.io/gorm"
)

// 【未学】发件箱轮询：扫 pending，投递成功才删除

//这个函数确实是在轮询发件箱，发现待发送消息后就投递到 RabbitMQ。
//使用 goroutine 不仅是为了不卡住后续操作，也是为了让这个后台任务独立运行，不阻塞调用它的主流程。

func StartOutboxPoller(db *gorm.DB, tmq *rabbitmq.TimelineMQ) {
	if db == nil || tmq == nil {
		log.Printf("Outbox poller disabled: timeline mq is not initialized")
		return
	}
	go func() {
		log.Printf("Outbox poller started")
		for {
			var messages []video.OutboxMsg
			err := db.Where("status = ?", "pending").Order("create_time ASC").Limit(100).Find(&messages).Error
			if err != nil || len(messages) == 0 {
				time.Sleep(time.Second)
				continue
			}
			for _, msg := range messages {
				err := tmq.PublishVideo(context.Background(), msg.VideoID, msg.CreateTime)
				if err != nil {
					log.Printf("投递MQ失败: VideoID: %d, err: %v", msg.VideoID, err)
					continue
				}
				if err := db.Delete(&msg).Error; err != nil {
					log.Printf("删除 outbox 消息失败: id=%d, err=%v", msg.ID, err)
				}
			}
		}
	}()
}
