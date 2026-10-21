package jwt

import (
	"backend/internal/account"
	"backend/internal/auth"
	"backend/internal/config"
	"backend/internal/db"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	_ = os.Setenv("JWT_SECRET", "dev-secret")
	os.Exit(m.Run())
}

func TestJWTAuthMissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", JWTAuth(nil, nil), func(c *gin.Context) { c.Status(http.StatusOK) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), "missing authorization header") {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestJWTAuthRejectsBadHeaderAndFakeToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", JWTAuth(nil, nil), func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Token abc")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), "invalid authorization header") {
		t.Fatal(w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer not-a-jwt")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), "invalid or expired token") {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestJWTAuthChecksStoredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gdb, err := db.NewDB(config.DefaultLocalConfig().Database)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.CloseDB(gdb) })

	const name = "jwt-t106"
	gdb.Where("username = ?", name).Delete(&account.Account{})
	repo := account.NewAccountRepository(gdb)
	a := &account.Account{Username: name, Password: "not-used"}
	if err := repo.CreateAccount(context.Background(), a); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { gdb.Where("username = ?", name).Delete(&account.Account{}) })

	saved, err := auth.GenerateToken(a.ID, a.Username)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)
	other, err := auth.GenerateToken(a.ID, a.Username)
	if err != nil {
		t.Fatal(err)
	}
	if saved == other {
		t.Fatal("two tokens are identical")
	}
	if err := repo.Login(context.Background(), a.ID, saved, "refresh"); err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.GET("/x", JWTAuth(repo, nil), func(c *gin.Context) {
		id, err := GetAccountID(c)
		if err != nil || id != a.ID {
			t.Errorf("id=%d err=%v", id, err)
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+saved)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatal(w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+other)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), "token has been revoked") {
		t.Fatal(w.Code, w.Body.String())
	}
}
