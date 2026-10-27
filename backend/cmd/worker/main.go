package main

import (
	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/middleware/rabbitmq"
	rediscache "backend/internal/middleware/redis"
	"backend/internal/observability"
	"backend/internal/video"
	"backend/internal/worker"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println(".env not found; continuing")
	}
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/config.yaml"
	}
	cfg, _, err := config.LoadLocalDev(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	sqlDB, err := db.NewDB(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}
	defer func() {
		_ = db.CloseDB(sqlDB)
	}()
	//redis
	cache, err := rediscache.NewFromEnv(&cfg.Redis)
	if err != nil {
		log.Printf("Redis config error (cache disabled): %v", err)
		cache = nil
	} else {
		pingCtx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()
		if err := cache.Ping(pingCtx); err != nil {
			log.Printf("Redis not available (cache disabled): %v", err)
			_ = cache.Close()
			cache = nil
		} else {
			defer func() {
				_ = cache.Close()
			}()
			log.Printf("Redis connected (cache enabled)")
		}
	}
	//rabbitmq
	rmq, err := rabbitmq.NewRabbitMQ(&cfg.RabbitMQ)
	if err != nil {
		log.Fatalf("RabbitMQ not available: %v", err)
	}
	defer func() {
		_ = rmq.Close()
	}()
	if err := rmq.DeclareTopology(); err != nil {
		log.Fatalf("RabbitMQ declare failed: %v", err)
	}
	log.Printf("RabbitMQ connected")
	// 【未学】没有 HTTP 服务。信号到来前，进程停在这一行。
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var popMQ *rabbitmq.PopularityMQ
	if mq, err := rabbitmq.NewPopularityMQ(rmq); err != nil {
		log.Printf("PopularityMQ init failed (redis direct): %v", err)
	} else {
		popMQ = mq
	}
	pprofServer, err := observability.NewPprofServer(
		"Worker",
		cfg.ObservabilityConfig.Pprof.Enabled,
		cfg.ObservabilityConfig.Pprof.WorkerAddr,
	)
	if err != nil {
		log.Printf("Failed to start worker pprof server: %v", err)
	}
	if pprofServer != nil {
		defer func() {
			_ = pprofServer.Close()
		}()
	}
	go runWorkerWithRetry(ctx, "LikeWorker", rmq.Conn, func(ch *amqp.Channel) error {
		return worker.NewLikeWorker(
			ch,
			video.NewLikeRepository(sqlDB),
			video.NewVideoRepository(sqlDB),
			cache,
			popMQ,
			"like.events",
		).Run(ctx)
	})
	if cache != nil {
		go runWorkerWithRetry(ctx, "PopularityWorker", rmq.Conn, func(ch *amqp.Channel) error {
			return worker.NewPopularityWorker(ch, cache, "video.popularity.events").Run(ctx)
		})
	}
	log.Printf("worker ready")
	<-ctx.Done()
	log.Printf("Worker shutting down...")
	time.Sleep(2 * time.Second)
	log.Printf("Worker stopped")
}

func runWorkerWithRetry(ctx context.Context, name string, conn *amqp.Connection, fn func(*amqp.Channel) error) {
	for {
		if ctx.Err() != nil {
			return
		}
		ch, err := conn.Channel()
		if err != nil {
			log.Printf("%s: 创建 Channel 失败: %v, 5秒后重试", name, err)
			time.Sleep(5 * time.Second)
			continue
		}
		if err := ch.Qos(50, 0, false); err != nil {
			log.Printf("%s: QoS 设置失败: %v", name, err)
		}
		log.Printf("%s started, consuming", name)
		err = fn(ch)
		_ = ch.Close()
		if ctx.Err() != nil {
			return
		}
		log.Printf("%s: %v, 5秒后重连...", name, err)
		time.Sleep(5 * time.Second)
	}
}
