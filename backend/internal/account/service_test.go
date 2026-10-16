package account

import (
	"strings"
	"testing"
)

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("123456")
	if err != nil {
		t.Fatal(err)
	}
	//$2 开头 = bcrypt 生成成功
	if hash == "123456" || !strings.HasPrefix(hash, "$2") {
		t.Fatalf("not a bcrypt hash: %s", hash)
	}
	again, err := HashPassword("123456")
	if err != nil {
		t.Fatal(err)
	}
	// 同一个密码，两次哈希结果必须不同
	if hash == again {
		t.Fatal("two hashes of the same password must differ")
	}
	if err := CheckPassword(hash, "123456"); err != nil {
		t.Fatal(err)
	}
	if err := CheckPassword(hash, "wrong"); err == nil {
		t.Fatal("wrong password was accepted")
	}
}
