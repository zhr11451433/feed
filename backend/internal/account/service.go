package account

import (
	"backend/internal/apierror"
	"backend/internal/auth"
	rediscache "backend/internal/middleware/redis"
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func HashPassword(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

//CheckPassword 返回 nil 表示密码正确。不要用 hash == plain，也不要自己比两个哈希字符串：盐不同，同一个密码的两次哈希也不相等。

func CheckPassword(hash string, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}

var (
	ErrNewUsernameRequired = errors.New("new_username is required")
	ErrUsernameTaken       = errors.New("用户名已存在")
)

type AccountService struct {
	repo  *AccountRepository
	cache *rediscache.Client
}

func NewAccountService(repo *AccountRepository, cache *rediscache.Client) *AccountService {
	return &AccountService{repo: repo, cache: cache}
}

func (s *AccountService) CreateAccount(ctx context.Context, account *Account) error {
	if account.Username == "" || account.Password == "" {
		return apierror.ErrValidation
	}
	hash, err := HashPassword(account.Password)
	if err != nil {
		return err
	}
	account.Password = hash
	err = s.repo.CreateAccount(ctx, account)
	if mysqlErr, ok := errors.AsType[*mysql.MySQLError](err); ok && mysqlErr.Number == 1062 {
		return ErrUsernameTaken
	}
	return err
}

func (s *AccountService) Login(ctx context.Context, username, password string) (*Account, string, string, error) {
	account, err := s.repo.FindByUsername(ctx, username)
	if err != nil {
		return nil, "", "", err
	}
	if err := CheckPassword(account.Password, password); err != nil {
		return nil, "", "", apierror.ErrUnauthorized
	}
	access, err := auth.GenerateToken(account.ID, account.Username)
	if err != nil {
		return nil, "", "", err
	}
	refresh, err := auth.GenerateRefreshToken(account.ID)
	if err != nil {
		return nil, "", "", err
	}
	if err := s.repo.Login(ctx, account.ID, access, refresh); err != nil {
		return nil, "", "", err
	}
	s.storeAccessToken(ctx, account.ID, access)
	return account, access, refresh, nil
}

func (s *AccountService) FindByUsername(ctx context.Context, username string) (*Account, error) {
	if username == "" {
		return nil, apierror.ErrValidation
	}
	return s.repo.FindByUsername(ctx, username)
}

func (s *AccountService) FindByID(ctx context.Context, id uint) (*Account, error) {
	if id == 0 {
		return nil, apierror.ErrValidation
	}
	return s.repo.FindByID(ctx, id)
}

func (s *AccountService) Logout(ctx context.Context, id uint) error {
	if err := s.repo.Logout(ctx, id); err != nil {
		return err
	}
	s.storeAccessToken(ctx, id, "")
	return nil
}

func (s *AccountService) RefreshAccessToken(ctx context.Context, refreshToken string) (string, uint, string, error) {
	if refreshToken == "" {
		return "", 0, "", errors.New("invalid refresh token")
	}
	a, err := s.repo.FindByRefreshToken(ctx, refreshToken)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", 0, "", errors.New("invalid refresh token")
	}
	if err != nil {
		return "", 0, "", err
	}
	access, err := auth.GenerateToken(a.ID, a.Username)
	if err != nil {
		return "", 0, "", err
	}
	if err := s.repo.UpdateToken(ctx, a.ID, access); err != nil {
		return "", 0, "", err
	}
	s.storeAccessToken(ctx, a.ID, access)
	return access, a.ID, a.Username, nil
}

func (s *AccountService) UpdateAvatar(ctx context.Context, id uint, avatarURL string) error {
	if avatarURL == "" {
		return apierror.ErrValidation
	}
	return s.repo.UpdateAvatar(ctx, id, avatarURL)
}

func (s *AccountService) UpdateProfile(ctx context.Context, id uint, req *UpdateProfileRequest) error {
	updates := map[string]any{}
	if req.Bio != "" {
		updates["bio"] = strings.TrimSpace(req.Bio)
	}
	if req.AvatarURL != "" {
		updates["avatar_url"] = strings.TrimSpace(req.AvatarURL)
	}
	if len(updates) == 0 {
		return apierror.ErrValidation
	}
	return s.repo.UpdateFields(ctx, id, updates)
}

func (s *AccountService) Rename(ctx context.Context, id uint, newUsername string) (string, error) {
	if newUsername == "" {
		return "", ErrNewUsernameRequired
	}
	token, err := auth.GenerateToken(id, newUsername)
	if err != nil {
		return "", err
	}
	err = s.repo.RenameWithToken(ctx, id, newUsername, token)
	if mysqlErr, ok := errors.AsType[*mysql.MySQLError](err); ok && mysqlErr.Number == 1062 {
		return "", ErrUsernameTaken
	}
	if err != nil {
		return "", err
	}
	s.storeAccessToken(ctx, id, token)
	return token, nil
}

func (s *AccountService) ChangePassword(ctx context.Context, username, oldPassword, newPassword string) error {
	if username == "" || oldPassword == "" || newPassword == "" {
		return apierror.ErrValidation
	}
	a, err := s.repo.FindByUsername(ctx, username)
	if err != nil {
		return err
	}
	if err := CheckPassword(a.Password, oldPassword); err != nil {
		return apierror.ErrUnauthorized
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.repo.ChangePassword(ctx, a.ID, hash); err != nil {
		return err
	}
	s.storeAccessToken(ctx, a.ID, "")
	return nil
}

func (s *AccountService) storeAccessToken(ctx context.Context, id uint, token string) {
	if s.cache == nil {
		return
	}
	cacheCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	key := s.cache.Key("account:%d", id)
	var err error
	if token == "" {
		err = s.cache.Del(cacheCtx, key)
	} else {
		err = s.cache.SetBytes(cacheCtx, key, []byte(token), 24*time.Hour)
	}
	if err != nil {
		log.Printf("token cache: %v", err)
	}
}
