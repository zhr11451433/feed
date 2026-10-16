package main

import (
	"backend/internal/config"
	"backend/internal/db"
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	apphttp "backend/internal/http"
)

func main() {
	// 伪代码，按这个顺序写：
	// 1. godotenv.Load() 失败只打日志，不退出
	// 2. 读 CONFIG_PATH，空则用 "configs/config.yaml"
	// 3. cfg, usedDefault, err := config.LoadLocalDev(configPath)；err != nil 则 log.Fatal
	// 4. db.NewDB(cfg.Database) 失败则 log.Fatal
	// 5. db.AutoMigrate 失败则 log.Fatal
	// 6. defer db.CloseDB(gdb)
	// 7. r := apphttp.SetRouter(gdb)
	// 8. 用下面的 http.Server 监听，不要 r.Run
	if err := godotenv.Load(); err != nil {
		log.Println(".env not found; continuing")
	}
	// 加载配置
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/config.yaml"
	}
	log.Printf("Loading config from %s", configPath)
	cfg, usedDefault, err := config.LoadLocalDev(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	if usedDefault {
		log.Printf("Config File %s not found, using default local config", configPath)
	} else {
		log.Printf("Config loaded from file: %s", configPath)
	}
	// 连接数据库
	//log.Printf("Database config: %v", cfg.Database)
	sqlDB, err := db.NewDB(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}
	if err := db.AutoMigrate(sqlDB); err != nil {
		log.Fatalf("Failed to auto migrate database: %v", err)
	}
	defer db.CloseDB(sqlDB)
	// 设置路由
	r := apphttp.SetRouter(sqlDB)
	//优雅关闭
	srv := &http.Server{
		Addr:    ":" + strconv.Itoa(cfg.Server.Port),
		Handler: r,
	}
	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	//os.Interrupt：对应键盘 Ctrl+C
	//syscall.SIGTERM：docker/k8s/kill 命令发送的终止信号。
	//当收到 Ctrl+C /kill 信号，这个`ctx`就会被取消，`ctx.Done()`通道会可读
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case err := <-errCh:
		log.Fatalf("listen: %v", err)
	case <-ctx.Done():
		log.Println("shutting down")
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
	//log.Printf("Server is running on port %d", cfg.Server.Port)
	//if err := r.Run(":" + strconv.Itoa(cfg.Server.Port)); err != nil {
	//	log.Fatalf("Failed to run server: %v", err)
	//}
}
