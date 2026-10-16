package auth

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// `TestMain` 是特殊测试函数，**整个包的测试入口**
func TestMain(m *testing.M) {
	_ = os.Setenv("JWT_SECRET", "test-secret")
	//`m` 是 `*testing.M` 对象，代表整个包的测试集合。
	//`m.Run()`：**执行本 package 下面所有 TestXxx 测试函数**，返回一个整数退出码
	os.Exit(m.Run())
}

// 测试**生成 JWT 和解析 JWT 整套流程是否正常**
func TestGenerateAndParse(t *testing.T) {
	raw, err := GenerateToken(7, "alice")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		t.Fatalf("want 3 parts, got %s", raw)
	}
	claims, err := ParseToken(raw)
	if err != nil {
		t.Fatal(err)
	}
	if claims.AccountID != 7 || claims.Username != "alice" {
		t.Fatalf("%+v", claims)
	}
	if time.Until(claims.ExpiresAt.Time) < 14*time.Minute {
		t.Fatal("exp is shorter than 15 minutes")
	}
}

// 过期的 JWT token 必须解析失败，不能放行
func TestParseTokenRejectsExpired(t *testing.T) {
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		AccountID: 7,
		Username:  "alice",
		RegisteredClaims: jwt.RegisteredClaims{
			//-time 表示已过期
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
		},
	}).SignedString(jwtSecret())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseToken(raw); err == nil {
		t.Fatal("expired token was accepted")
	}
}
