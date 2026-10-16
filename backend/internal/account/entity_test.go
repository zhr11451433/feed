package account

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAccountSecretsOmitted(t *testing.T) {
	a := Account{
		Username: "alice", Password: "secret",
		Token: "access", RefreshToken: "refresh",
	}
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	if strings.Contains(got, "secret") || strings.Contains(got, "access") || strings.Contains(got, "refresh") {
		t.Fatalf("secret leaked: %s", got)
	}
}
