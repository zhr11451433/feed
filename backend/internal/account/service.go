package account

import (
	"backend/internal/apierror"
	"backend/internal/auth"
	"context"
	"errors"

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

var ErrUsernameTaken = errors.New("用户名已存在")

type AccountService struct {
	repo *AccountRepository
}

func NewAccountService(repo *AccountRepository) *AccountService {
	return &AccountService{repo: repo}
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
	return s.repo.Logout(ctx, id)
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
	return access, a.ID, a.Username, nil
}
