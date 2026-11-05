package service

import (
	"net/http"
	"strings"
	"time"

	"wxcloudrun-golang/db/dao"
	"wxcloudrun-golang/db/model"
	"wxcloudrun-golang/pkg/security"
)

// userItem 带角色名的账号视图
type userItem struct {
	model.AdminUser
	RoleName string `json:"roleName"`
}

// userList 分页查询账号
func userList(c *adminCtx, r *http.Request) (interface{}, error) {
	page, size := pageParams(r)
	keyword := strings.TrimSpace(r.URL.Query().Get("keyword"))
	role := strings.TrimSpace(r.URL.Query().Get("role"))
	status := queryInt(r, "status", -1)

	list, total, err := dao.ListUsers(dao.UserQuery{
		Keyword: keyword, Role: role, Status: status, Page: page, Size: size,
	})
	if err != nil {
		return nil, serverError("查询账号失败: " + err.Error())
	}

	roleNames := map[string]string{}
	roleOptions := make([]map[string]string, 0)
	if roles, err := dao.ListRoles(); err == nil {
		for _, item := range roles {
			roleNames[item.Code] = item.Name
			roleOptions = append(roleOptions, map[string]string{"code": item.Code, "name": item.Name})
		}
	}

	items := make([]userItem, 0, len(list))
	for _, u := range list {
		name := roleNames[u.RoleCode]
		if name == "" {
			name = u.RoleCode
		}
		items = append(items, userItem{AdminUser: u, RoleName: name})
	}

	return map[string]interface{}{
		"list":  items,
		"total": total,
		"page":  page,
		"size":  size,
		"roles": roleOptions,
	}, nil
}

// userCreate 新建账号
func userCreate(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Nickname string `json:"nickname"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		RoleCode string `json:"roleCode"`
		Status   int    `json:"status"`
		Remark   string `json:"remark"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	req.Username = strings.TrimSpace(req.Username)
	if len(req.Username) < 3 || len(req.Username) > 32 {
		return nil, badRequest("账号长度需在 3-32 个字符之间")
	}
	if len(req.Password) < 6 {
		return nil, badRequest("密码长度不能少于 6 位")
	}
	if _, err := dao.FindUserByUsername(req.Username); err == nil {
		return nil, badRequest("账号 " + req.Username + " 已存在")
	}
	if req.RoleCode == "" {
		req.RoleCode = "operator"
	}
	if _, err := dao.FindRoleByCode(req.RoleCode); err != nil {
		return nil, badRequest("角色不存在: " + req.RoleCode)
	}

	hash, err := security.HashPassword(req.Password)
	if err != nil {
		return nil, badRequest(err.Error())
	}
	status := req.Status
	if status != 0 {
		status = 1
	}
	if req.Nickname == "" {
		req.Nickname = req.Username
	}

	now := time.Now()
	user := &model.AdminUser{
		Username:  req.Username,
		Password:  hash,
		Nickname:  req.Nickname,
		Email:     req.Email,
		Phone:     req.Phone,
		RoleCode:  req.RoleCode,
		Status:    status,
		Remark:    req.Remark,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := dao.CreateUser(user); err != nil {
		return nil, serverError("创建账号失败: " + err.Error())
	}
	return user, nil
}

// userUpdate 编辑账号资料
func userUpdate(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id       int64  `json:"id"`
		Nickname string `json:"nickname"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		RoleCode string `json:"roleCode"`
		Remark   string `json:"remark"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if req.Id <= 0 {
		return nil, badRequest("缺少账号 id")
	}
	user, err := dao.FindUserById(req.Id)
	if err != nil {
		return nil, notFound("账号不存在")
	}

	fields := map[string]interface{}{
		"nickname": req.Nickname,
		"email":    req.Email,
		"phone":    req.Phone,
		"remark":   req.Remark,
	}
	if req.RoleCode != "" && req.RoleCode != user.RoleCode {
		if _, err := dao.FindRoleByCode(req.RoleCode); err != nil {
			return nil, badRequest("角色不存在: " + req.RoleCode)
		}
		if user.Id == c.User.Id {
			return nil, badRequest("不能修改自己的角色")
		}
		fields["roleCode"] = req.RoleCode
	}
	if err := dao.UpdateUserFields(req.Id, fields); err != nil {
		return nil, serverError("保存失败: " + err.Error())
	}
	updated, _ := dao.FindUserById(req.Id)
	return updated, nil
}

// userDelete 删除账号
func userDelete(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id int64 `json:"id"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if req.Id <= 0 {
		return nil, badRequest("缺少账号 id")
	}
	if req.Id == c.User.Id {
		return nil, badRequest("不能删除当前登录账号")
	}
	user, err := dao.FindUserById(req.Id)
	if err != nil {
		return nil, notFound("账号不存在")
	}
	if user.RoleCode == "super" {
		total, err := dao.CountUsersInRole("super")
		if err != nil {
			return nil, serverError(err.Error())
		}
		if total <= 1 {
			return nil, badRequest("系统至少需要保留一个超级管理员")
		}
	}
	if err := dao.DeleteUser(req.Id); err != nil {
		return nil, serverError("删除失败: " + err.Error())
	}
	return map[string]interface{}{"id": req.Id}, nil
}

// userToggleStatus 启用或禁用账号
func userToggleStatus(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id     int64 `json:"id"`
		Status int   `json:"status"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if req.Id <= 0 {
		return nil, badRequest("缺少账号 id")
	}
	if req.Id == c.User.Id {
		return nil, badRequest("不能修改自己的账号状态")
	}
	if req.Status != 0 && req.Status != 1 {
		return nil, badRequest("状态值只能是 0 或 1")
	}
	if _, err := dao.FindUserById(req.Id); err != nil {
		return nil, notFound("账号不存在")
	}
	if err := dao.UpdateUserFields(req.Id, map[string]interface{}{"status": req.Status}); err != nil {
		return nil, serverError("保存失败: " + err.Error())
	}
	return map[string]interface{}{"id": req.Id, "status": req.Status}, nil
}

// userResetPassword 重置指定账号密码
func userResetPassword(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id       int64  `json:"id"`
		Password string `json:"password"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if req.Id <= 0 {
		return nil, badRequest("缺少账号 id")
	}
	if len(req.Password) < 6 {
		return nil, badRequest("新密码长度不能少于 6 位")
	}
	if _, err := dao.FindUserById(req.Id); err != nil {
		return nil, notFound("账号不存在")
	}
	hash, err := security.HashPassword(req.Password)
	if err != nil {
		return nil, badRequest(err.Error())
	}
	if err := dao.UpdateUserFields(req.Id, map[string]interface{}{
		"password":      hash,
		"mustChangePwd": true,
	}); err != nil {
		return nil, serverError("保存失败: " + err.Error())
	}
	return map[string]interface{}{"id": req.Id}, nil
}
