package service

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"wxcloudrun-golang/db/dao"
	"wxcloudrun-golang/db/model"
)

// permItem 单项权限
type permItem struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Desc string `json:"desc"`
}

// permGroup 权限分组
type permGroup struct {
	Group string     `json:"group"`
	Items []permItem `json:"items"`
}

// permissionCatalog 权限字典，前端角色编辑页按此渲染
var permissionCatalog = []permGroup{
	{Group: "工作台", Items: []permItem{
		{Code: "dashboard", Name: "数据看板", Desc: "查看看板与统计图表"},
	}},
	{Group: "账号与权限", Items: []permItem{
		{Code: "users.view", Name: "账号查看", Desc: "查看账号列表"},
		{Code: "users", Name: "账号管理", Desc: "新建、编辑、禁用、删除账号"},
		{Code: "roles.view", Name: "角色查看", Desc: "查看角色与权限配置"},
		{Code: "roles", Name: "角色管理", Desc: "新建、编辑、删除角色"},
	}},
	{Group: "课程内容", Items: []permItem{
		{Code: "majors.view", Name: "专业查看", Desc: "查看专业分类"},
		{Code: "majors", Name: "专业管理", Desc: "新建、编辑、删除专业与调整排序"},
		{Code: "courses.view", Name: "课程查看", Desc: "查看课程与章节内容"},
		{Code: "courses", Name: "课程管理", Desc: "新建、编辑、发布、删除课程与章节"},
	}},
	{Group: "业务数据", Items: []permItem{
		{Code: "counters.view", Name: "计数查看", Desc: "查看计数记录"},
		{Code: "counters", Name: "计数管理", Desc: "新增、修改、删除计数记录"},
		{Code: "mp.view", Name: "小程序查看", Desc: "查看小程序接入状态"},
		{Code: "mp", Name: "小程序管理", Desc: "维护小程序端内容与配置"},
	}},
	{Group: "系统", Items: []permItem{
		{Code: "logs.view", Name: "日志查看", Desc: "查看登录日志与操作日志"},
		{Code: "logs", Name: "日志管理", Desc: "清空日志"},
		{Code: "configs.view", Name: "设置查看", Desc: "查看系统设置"},
		{Code: "configs", Name: "设置管理", Desc: "修改系统设置"},
		{Code: "system.view", Name: "系统信息", Desc: "查看运行时与数据库状态"},
	}},
}

// roleItem 带账号数量的角色视图
type roleItem struct {
	model.AdminRole
	UserCount int64 `json:"userCount"`
}

// roleList 查询全部角色
func roleList(c *adminCtx, r *http.Request) (interface{}, error) {
	roles, err := dao.ListRoles()
	if err != nil {
		return nil, serverError("查询角色失败: " + err.Error())
	}
	counts, err := dao.CountUsersByRole()
	if err != nil {
		return nil, serverError(err.Error())
	}
	items := make([]roleItem, 0, len(roles))
	for _, role := range roles {
		items = append(items, roleItem{AdminRole: role, UserCount: counts[role.Code]})
	}
	return map[string]interface{}{"list": items, "total": len(items)}, nil
}

// rolePermissionCatalogHandler 返回权限字典
func rolePermissionCatalogHandler(c *adminCtx, r *http.Request) (interface{}, error) {
	return map[string]interface{}{"groups": permissionCatalog}, nil
}

// roleSave 新建或更新角色
func roleSave(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id          int64    `json:"id"`
		Code        string   `json:"code"`
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Permissions []string `json:"permissions"`
		Sort        int      `json:"sort"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	req.Code = strings.TrimSpace(req.Code)
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return nil, badRequest("角色名称不能为空")
	}

	permStr := normalizePermissions(req.Permissions, req.Code)
	now := time.Now()

	if req.Id > 0 {
		role, err := dao.FindRoleById(req.Id)
		if err != nil {
			return nil, notFound("角色不存在")
		}
		if role.Builtin && role.Code != "super" && req.Name == "" {
			return nil, badRequest("内置角色名称不能为空")
		}
		if role.Code == "super" {
			permStr = "*"
		}
		role.Name = req.Name
		role.Description = req.Description
		role.Permissions = permStr
		role.Sort = req.Sort
		role.UpdatedAt = now
		if err := dao.UpdateRole(role); err != nil {
			return nil, serverError("保存失败: " + err.Error())
		}
		return role, nil
	}

	if len(req.Code) < 2 || len(req.Code) > 32 {
		return nil, badRequest("角色编码长度需在 2-32 个字符之间")
	}
	if _, err := dao.FindRoleByCode(req.Code); err == nil {
		return nil, badRequest("角色编码 " + req.Code + " 已存在")
	}
	role := &model.AdminRole{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		Permissions: permStr,
		Sort:        req.Sort,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := dao.CreateRole(role); err != nil {
		return nil, serverError("创建失败: " + err.Error())
	}
	return role, nil
}

// roleDelete 删除角色
func roleDelete(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id int64 `json:"id"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	role, err := dao.FindRoleById(req.Id)
	if err != nil {
		return nil, notFound("角色不存在")
	}
	if role.Builtin {
		return nil, badRequest("内置角色不允许删除")
	}
	count, err := dao.CountUsersInRole(role.Code)
	if err != nil {
		return nil, serverError(err.Error())
	}
	if count > 0 {
		return nil, badRequest("该角色下还有 " + strconv.FormatInt(count, 10) + " 个账号，请先调整账号角色")
	}
	if err := dao.DeleteRole(req.Id); err != nil {
		return nil, serverError("删除失败: " + err.Error())
	}
	return map[string]interface{}{"id": req.Id}, nil
}

// normalizePermissions 去重并过滤非法权限编码
func normalizePermissions(list []string, roleCode string) string {
	if roleCode == "super" {
		return "*"
	}
	valid := map[string]bool{}
	for _, group := range permissionCatalog {
		for _, item := range group.Items {
			valid[item.Code] = true
		}
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(list))
	for _, p := range list {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] || !valid[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return strings.Join(out, ",")
}
