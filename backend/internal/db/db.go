package db

import (
	"backend/internal/account"
	"backend/internal/config"
	"backend/internal/video"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

//把「连哪台库」收成一个函数。parseTime=True 让驱动把 DATETIME 扫进 time.Time；不加的话，以后模型里的时间字段一查就报错。

func NewDB(dbcfg config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbcfg.User, dbcfg.Password, dbcfg.Host, dbcfg.Port, dbcfg.DBName)
	gdb, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, err
	}
	// 原项目没设连接池。下面三个数是本地学习用的起点，不是原作者的配置。
	sqlDB.SetMaxOpenConns(20)           // 同时打开的连接上限。0 表示不限制，容易把 MySQL 打满
	sqlDB.SetMaxIdleConns(10)           // 闲着留着的连接。默认只有 2，高并发时反复握手
	sqlDB.SetConnMaxLifetime(time.Hour) // 一条连接最多活多久，避免被 MySQL 的 wait_timeout 先掐掉
	return gdb, nil
}

//表名单放在切片里。后面每个阶段只往这里追加自己的模型，不要改 NewDB

var models = []any{
	&account.Account{},
	&video.Tag{},
	&video.Video{},
	&video.VideoTag{},
}

func AutoMigrate(gdb *gorm.DB) error {
	if len(models) == 0 {
		return nil
	}
	return gdb.AutoMigrate(models...)
}

func CloseDB(gdb *gorm.DB) error {
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
