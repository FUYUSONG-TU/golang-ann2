package dao

import (
	"wxcloudrun-golang/db"
	"wxcloudrun-golang/db/model"
)

// ListRoles 查询全部角色，按排序值升序
func ListRoles() ([]model.AdminRole, error) {
	list := make([]model.AdminRole, 0)
	err := db.Get().Order("sort asc, id asc").Find(&list).Error
	return list, err
}

// FindRoleByCode 按编码查询角色
func FindRoleByCode(code string) (*model.AdminRole, error) {
	var role model.AdminRole
	err := db.Get().Where("code = ?", code).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// FindRoleById 按主键查询角色
func FindRoleById(id int64) (*model.AdminRole, error) {
	var role model.AdminRole
	err := db.Get().Where("id = ?", id).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// CreateRole 新建角色
func CreateRole(role *model.AdminRole) error {
	return db.Get().Create(role).Error
}

// UpdateRole 更新角色
func UpdateRole(role *model.AdminRole) error {
	return db.Get().Model(&model.AdminRole{}).Where("id = ?", role.Id).Updates(map[string]interface{}{
		"name":        role.Name,
		"description": role.Description,
		"permissions": role.Permissions,
		"sort":        role.Sort,
		"updatedAt":   role.UpdatedAt,
	}).Error
}

// DeleteRole 删除角色
func DeleteRole(id int64) error {
	return db.Get().Where("id = ?", id).Delete(&model.AdminRole{}).Error
}

// CountUsersInRole 统计使用该角色的账号数
func CountUsersInRole(code string) (int64, error) {
	var n int64
	err := db.Get().Model(&model.AdminUser{}).Where("roleCode = ?", code).Count(&n).Error
	return n, err
}
