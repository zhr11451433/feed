package account

import (
	"context"

	"gorm.io/gorm"
)

type AccountRepository struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

//负责**插入一行记录（新增数据）**

func (r *AccountRepository) CreateAccount(ctx context.Context, account *Account) error {
	return r.db.WithContext(ctx).Create(account).Error
}

func (r *AccountRepository) FindByUsername(ctx context.Context, username string) (*Account, error) {
	var account Account
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *AccountRepository) FindByID(ctx context.Context, id uint) (*Account, error) {
	var account Account
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *AccountRepository) Login(ctx context.Context, id uint, token, refreshToken string) error {
	return r.db.WithContext(ctx).Model(&Account{}).Where("id = ?", id).
		Updates(map[string]any{"token": token, "refresh_token": refreshToken}).Error
}

func (r *AccountRepository) Logout(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&Account{}).Where("id = ?", id).
		Updates(map[string]any{"token": "", "refresh_token": ""}).Error
}

func (r *AccountRepository) UpdateToken(ctx context.Context, id uint, token string) error {
	return r.db.WithContext(ctx).Model(&Account{}).Where("id = ?", id).Update("token", token).Error
}

func (r *AccountRepository) FindByRefreshToken(ctx context.Context, refreshToken string) (*Account, error) {
	var account Account
	if err := r.db.WithContext(ctx).Where("refresh_token = ?", refreshToken).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}
