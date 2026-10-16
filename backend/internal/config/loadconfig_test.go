package config

import "testing"

//T005 之前没有进程入口。测试不依赖当前工作目录，避免 go test 在包目录里找不到 configs/config.yaml

// 测试目标：当 yaml 配置文件不存在的时候，会不会正确加载内置默认配置
func TestLoadLocalDev_MissingFileUsesDefault(t *testing.T) {
	cfg, usedDefault, err := LoadLocalDev("does-not-exist.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !usedDefault || cfg.Server.Port != 8080 || cfg.Database.DBName != "feed" {
		t.Fatalf("got usedDefault=%v port=%d db=%s", usedDefault, cfg.Server.Port, cfg.Database.DBName)
	}
}

// 测试目标：验证 `ApplyEnvOverrides` 环境变量覆盖逻辑是否生效
func TestApplyEnvOverrides_PortAndPassword(t *testing.T) {
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("MYSQL_ROOT_PASSWORD", "root-pass")
	t.Setenv("MYSQL_PASSWORD", "app-pass")
	cfg, usedDefault, err := LoadLocalDev("does-not-exist.yaml")
	if err != nil || !usedDefault {
		t.Fatal(err, usedDefault)
	}
	if cfg.Server.Port != 9090 || cfg.Database.Password != "app-pass" {
		t.Fatalf("port=%d password=%s", cfg.Server.Port, cfg.Database.Password)
	}
}
