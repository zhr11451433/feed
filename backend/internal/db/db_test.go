package db

import (
	"backend/internal/config"
	"testing"
)

// 这个测试只证明「能连上、空迁移不报错、能关掉」
func TestNewDBPingAndEmptyMigrate(t *testing.T) {
	cfg := config.DefaultLocalConfig()
	gdb, err := NewDB(cfg.Database)
	if err != nil {
		t.Fatal(err)
	}
	//作用：注册一个清理函数，在当前这个测试用例结束的时候自动执行（不管测试是 PASS 成功，还是t.Fatal()失败崩溃，都会跑）。
	//不是立刻执行里面的代码！只是 “登记”，等整个TestNewDB_PingAndEmptyMigrate函数跑完才执行
	t.Cleanup(func() { _ = CloseDB(gdb) })

	if err := AutoMigrate(gdb); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Fatal(err)
	}
}
