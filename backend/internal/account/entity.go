package account

//`entity.go`存放**这个业务域所有核心数据模型**
//带 json:"-"，以后 json.Marshal 不会把它们写进响应

type Account struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	Username  string `gorm:"uniqueIndex;size:64" json:"username"`
	Password  string `gorm:"size:255" json:"-"`
	AvatarURL string `gorm:"size:512" json:"avatar_url,omitempty"` //存储用户头像的图片 URL 地址
	Bio       string `gorm:"size:255" json:"bio,omitempty"`        //用户个人签名 / 简介，自我介绍
	//token 存入数据库，就是为了实现**强制下线**功能，比如改密码之后所有旧登录全部失效
	Token        string `gorm:"size:512" json:"-"`
	RefreshToken string `gorm:"size:512" json:"-"`
}

type CreateAccountRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
	//omitempty 序列化转 JSON 的时候：如果这个字段是【零值】，就直接忽略这个字段，不输出到 JSON 里
	RefreshToken string `json:"refresh_token,omitempty"`
	AccountID    uint   `json:"account_id"`
	Username     string `json:"username"`
}

type FindByIDRequest struct {
	ID uint `json:"id"`
}

type FindByUsernameRequest struct {
	Username string `json:"username"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type GetProfileRequest struct {
	AccountID uint `json:"account_id"`
}

type GetProfileResponse struct {
	Account       Account `json:"account"`
	VideoCount    int64   `json:"video_count"`
	TotalLikes    int64   `json:"total_likes"`
	FollowerCount int64   `json:"follower_count"`
	VloggerCount  int64   `json:"vlogger_count"`
}

type UpdateProfileRequest struct {
	AvatarURL string `json:"avatar_url"`
	Bio       string `json:"bio"`
}

type RenameRequest struct {
	NewUsername string `json:"new_username"`
}

type ChangePasswordRequest struct {
	Username    string `json:"username"`
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}
