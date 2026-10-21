package social

import (
	"backend/internal/account"
	"backend/internal/apierror"
	"context"
	"errors"
	"fmt"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

var (
	ErrFollowSelf      = errors.New("can not follow self")
	ErrAlreadyFollowed = errors.New("already followed")
	ErrNotFollowed     = errors.New("not followed")
)

type SocialService struct {
	db       *gorm.DB
	accounts *account.AccountRepository
}

func NewSocialService(db *gorm.DB, accounts *account.AccountRepository) *SocialService {
	return &SocialService{db: db, accounts: accounts}
}

func (s *SocialService) Follow(ctx context.Context, followerID, vloggerID uint) error {
	if followerID == 0 || vloggerID == 0 {
		return apierror.ErrValidation
	}
	if followerID == vloggerID {
		return ErrFollowSelf
	}
	if _, err := s.accounts.FindByID(ctx, vloggerID); err != nil {
		return err
	}
	err := s.db.WithContext(ctx).Create(&Social{FollowerID: followerID, VloggerID: vloggerID}).Error
	if mysqlErr, ok := errors.AsType[*mysql.MySQLError](err); ok && mysqlErr.Number == 1062 {
		return ErrAlreadyFollowed
	}
	return err
}

func (s *SocialService) Unfollow(ctx context.Context, followerID, vloggerID uint) error {
	if followerID == 0 || vloggerID == 0 {
		return apierror.ErrValidation
	}
	res := s.db.WithContext(ctx).
		Where("follower_id = ? AND vlogger_id = ?", followerID, vloggerID).
		Delete(&Social{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFollowed
	}
	return nil
}

type accountRow struct {
	ID        uint   `gorm:"column:id"`
	Username  string `gorm:"column:username"`
	AvatarURL string `gorm:"column:avatar_url"`
	Bio       string `gorm:"column:bio"`
	Total     int64  `gorm:"column:total"`
}

//查询某个用户的粉丝列表，或者他关注的人，并返回列表总数

//followers
//true：查询这个用户的粉丝。
//false：查询这个用户关注的人

func (s *SocialService) ListBy(ctx context.Context, accountID uint, followers bool) ([]account.Account, int64, error) {
	if _, err := s.accounts.FindByID(ctx, accountID); err != nil {
		return nil, 0, err
	}
	//whereCol 决定：根据哪一列筛选当前用户的关系。
	//joinCol 决定：通过哪一列找到关系另一端的用户资料
	//follower 追随者
	//vlogger 视频博主
	whereCol, joinCol := "follower_id", "vlogger_id"
	if followers {
		whereCol, joinCol = "vlogger_id", "follower_id"
	}
	query := fmt.Sprintf(`
		SELECT a.id, a.username, a.avatar_url, a.bio, COUNT(*) OVER() AS total
		FROM socials s
		JOIN accounts a ON a.id = s.%s
		WHERE s.%s = ?
		LIMIT 200`, joinCol, whereCol)
	var rows []accountRow
	//Raw(query, accountID)：执行原生 SQL，并将 accountID 绑定到 ?
	//Scan(&rows)：把查询结果映射到 rows 中每个 accountRow
	if err := s.db.WithContext(ctx).Raw(query, accountID).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	if len(rows) == 0 {
		return []account.Account{}, 0, nil
	}
	out := make([]account.Account, len(rows))
	for i, row := range rows {
		out[i] = account.Account{ID: row.ID, Username: row.Username, AvatarURL: row.AvatarURL, Bio: row.Bio}
	}
	return out, rows[0].Total, nil
}

//查询某个用户有多少粉丝、关注了多少人

func (s *SocialService) Counts(ctx context.Context, accountID uint) (SocialCounts, error) {
	var counts SocialCounts
	err := s.db.WithContext(ctx).Raw(`
		SELECT
			(SELECT COUNT(*) FROM socials WHERE vlogger_id = ?) AS follower_count,
			(SELECT COUNT(*) FROM socials WHERE follower_id = ?) AS vlogger_count
	`, accountID, accountID).Scan(&counts).Error
	return counts, err
}
