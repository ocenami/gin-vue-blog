package model

import (
	"time"

	"gorm.io/gorm"
)

type UserInfo struct {
	Model
	Email    string `json:"email" gorm:"type:varchar(30)"`
	Nickname string `json:"nickname" gorm:"unique;type:varchar(30);not null"`
	Avatar   string `json:"avatar" gorm:"type:varchar(1024);not null"`
	Intro    string `json:"intro" gorm:"type:varchar(255)"`
	Website  string `json:"website" gorm:"type:varchar(255)"`
}

type UserInfoVO struct {
	UserInfo
	ArticleLikeSet []string `json:"article_like_set"`
	CommentLikeSet []string `json:"comment_like_set"`
}

// FIXME: 组合 UserInfo, UserAuth
type UserVO struct {
	ID            int       `json:"id"`
	UserInfoId    int       `json:"user_info_id"`
	Avatar        string    `json:"avatar"`
	Nickname      string    `json:"nickname"`
	LoginType     int       `json:"login_type"`
	IpAddress     string    `json:"ip_address"`
	IpSource      string    `json:"ip_source"`
	CreatedAt     time.Time `json:"created_at"`
	LastLoginTime time.Time `json:"last_login_time"`
	IsDisable     bool      `json:"is_disable"`

	Roles []Role `json:"roles" gorm:"many2many:user_role;foreignKey:UserInfoId;joinForeignKey:UserId;"`
}

func GetUserInfoById(db *gorm.DB, id int) (*UserInfo, error) {
	var userInfo UserInfo
	result := db.Model(&userInfo).Where("id", id).First(&userInfo)
	if result.Error != nil {
		return nil, result.Error
	}
	return &userInfo, nil
}

func GetUserAuthInfoByName(db *gorm.DB, name string) (*UserAuth, error) {
	var userAuth UserAuth
	result := db.Where(&UserAuth{Username: name}).First(&userAuth)
	if result.Error != nil {
		return nil, result.Error
	}
	return &userAuth, nil
}

func GetUserList(db *gorm.DB, page, size int, loginType int8, nickname, username string) (list []UserAuth, total int64, err error) {
	if loginType != 0 {
		db = db.Where("login_type = ?", loginType)
	}

	if username != "" {
		db = db.Where("username LIKE ?", "%"+username+"%")
	}

	result := db.Model(&UserAuth{}).
		Joins("LEFT JOIN user_info ON user_info.id = user_auth.user_info_id").
		Where("user_info.nickname LIKE ?", "%"+nickname+"%").
		Preload("UserInfo").
		Preload("Roles").
		Count(&total).
		Scopes(Paginate(page, size)).
		Find(&list)

	if result.Error != nil {
		return nil, 0, result.Error
	}

	return list, total, nil
}

// 更新用户昵称及角色信息
func UpdateUserNicknameAndRole(db *gorm.DB, authId int, nickname string, roleIds []int) error {
	userAuth, err := GetUserAuthInfoById(db, authId)
	if err != nil {
		return err
	}

	userInfo := UserInfo{
		Model:    Model{ID: userAuth.UserInfoId},
		Nickname: nickname,
	}
	result := db.Model(&userInfo).Updates(userInfo)
	if result.Error != nil {
		return result.Error
	}

	// 至少有一个角色
	if len(roleIds) == 0 {
		return nil
	}

	// 更新用户角色, 清空原本的 user_role 关系, 添加新的关系
	result = db.Where(UserAuthRole{UserAuthId: userAuth.UserInfoId}).Delete(UserAuthRole{})
	if result.Error != nil {
		return result.Error
	}

	var userRoles []UserAuthRole
	for _, id := range roleIds {
		userRoles = append(userRoles, UserAuthRole{
			RoleId:     id,
			UserAuthId: userAuth.ID,
		})
	}
	result = db.Create(&userRoles)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func UpdateUserPassword(db *gorm.DB, id int, password string) error {
	userAuth := UserAuth{
		Model:    Model{ID: id},
		Password: password,
	}
	result := db.Model(&userAuth).Updates(userAuth)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func UpdateUserInfo(db *gorm.DB, id int, nickname, avatar, intro, website string) error {
	userInfo := UserInfo{
		Model:    Model{ID: id},
		Nickname: nickname,
		Avatar:   avatar,
		Intro:    intro,
		Website:  website,
	}

	result := db.
		Select("nickname", "avatar", "intro", "website").
		Updates(userInfo)
	return result.Error
}

func UpdateUserDisable(db *gorm.DB, id int, isDisable bool) error {
	userAuth := UserAuth{
		Model:     Model{ID: id},
		IsDisable: isDisable,
	}
	result := db.Model(&userAuth).Select("is_disable").Updates(&userAuth)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// 更新用户登录信息
func UpdateUserLoginInfo(db *gorm.DB, id int, ipAddress, ipSource string) error {
	now := time.Now()
	userAuth := UserAuth{
		IpAddress:     ipAddress,
		IpSource:      ipSource,
		LastLoginTime: &now,
	}

	result := db.Where("id", id).Updates(userAuth)
	if result.Error != nil {
		return result.Error
	}

	return nil
}
