package dao

import (
	"errors"
	"strings"
	"time"

	"wxcloudrun-golang/db"
	"wxcloudrun-golang/db/model"

	"gorm.io/gorm"
)

// UserQuery 用户列表查询条件
type UserQuery struct {
	Keyword string
	Role    string
	Status  int // -1 表示不限
	Page    int
	Size    int
}

// FindUserByUsername 按用户名查询账号
func FindUserByUsername(username string) (*model.AdminUser, error) {
	var user model.AdminUser
	err := db.Get().Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindUserById 按主键查询账号
func FindUserById(id int64) (*model.AdminUser, error) {
	var user model.AdminUser
	err := db.Get().Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// ListUsers 分页查询账号
func ListUsers(q UserQuery) ([]model.AdminUser, int64, error) {
	cli := db.Get().Model(&model.AdminUser{})
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		like := "%" + kw + "%"
		cli = cli.Where("username LIKE ? OR nickname LIKE ? OR email LIKE ? OR phone LIKE ?", like, like, like, like)
	}
	if q.Role != "" {
		cli = cli.Where("roleCode = ?", q.Role)
	}
	if q.Status >= 0 {
		cli = cli.Where("status = ?", q.Status)
	}

	var total int64
	if err := cli.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	list := make([]model.AdminUser, 0)
	err := cli.Order("id asc").Offset((q.Page - 1) * q.Size).Limit(q.Size).Find(&list).Error
	return list, total, err
}

// CreateUser 新建账号
func CreateUser(user *model.AdminUser) error {
	return db.Get().Create(user).Error
}

// UpdateUserFields 更新指定字段
func UpdateUserFields(id int64, fields map[string]interface{}) error {
	fields["updatedAt"] = time.Now()
	return db.Get().Model(&model.AdminUser{}).Where("id = ?", id).Updates(fields).Error
}

// DeleteUser 删除账号
func DeleteUser(id int64) error {
	return db.Get().Where("id = ?", id).Delete(&model.AdminUser{}).Error
}

// TouchUserLogin 记录登录信息
func TouchUserLogin(id int64, ip string, at time.Time) error {
	return db.Get().Model(&model.AdminUser{}).Where("id = ?", id).Updates(map[string]interface{}{
		"lastLoginAt": at,
		"lastLoginIp": ip,
	}).Error
}

// CountUsers 统计账号数量，status < 0 表示不限
func CountUsers(status int) (int64, error) {
	cli := db.Get().Model(&model.AdminUser{})
	if status >= 0 {
		cli = cli.Where("status = ?", status)
	}
	var n int64
	err := cli.Count(&n).Error
	return n, err
}

// CountUsersCreatedSince 统计某时间点之后创建的账号数
func CountUsersCreatedSince(since time.Time) (int64, error) {
	var n int64
	err := db.Get().Model(&model.AdminUser{}).Where("createdAt >= ?", since).Count(&n).Error
	return n, err
}

// CountUsersByRole 统计各角色账号数
func CountUsersByRole() (map[string]int64, error) {
	type row struct {
		RoleCode string `gorm:"column:roleCode"`
		Total    int64  `gorm:"column:total"`
	}
	var rows []row
	err := db.Get().Model(&model.AdminUser{}).
		Select("roleCode, count(*) as total").
		Group("roleCode").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, r := range rows {
		out[r.RoleCode] = r.Total
	}
	return out, nil
}

// IsNotFound 判断是否为记录不存在
func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
