package service

import (
	"net/http"
	"strings"
)

// apiHandler 业务处理函数：返回数据即成功，返回 error 即失败
type apiHandler func(*adminCtx, *http.Request) (interface{}, error)

// apiRoute 路由定义
type apiRoute struct {
	method  string
	path    string
	perm    string // "" 表示登录即可；"users:view" 只读；"users" 需要管理权限
	module  string // 操作日志模块，为空则不计操作日志
	action  string
	handler apiHandler
}

// adminRoutes 后台全部接口
var adminRoutes = []apiRoute{
	// 登录态
	{http.MethodGet, "/auth/profile", "", "", "", authProfile},
	{http.MethodPost, "/auth/logout", "", "", "", authLogout},
	{http.MethodPost, "/auth/password", "", "auth", "修改密码", authChangePassword},

	// 数据看板
	{http.MethodGet, "/dashboard", "dashboard", "", "", dashboardOverview},
	{http.MethodGet, "/overview", "dashboard", "", "", dashboardOverview},

	// 账号管理
	{http.MethodGet, "/users", "users:view", "", "", userList},
	{http.MethodPost, "/users/create", "users", "users", "新建账号", userCreate},
	{http.MethodPost, "/users/update", "users", "users", "编辑账号", userUpdate},
	{http.MethodPost, "/users/delete", "users", "users", "删除账号", userDelete},
	{http.MethodPost, "/users/status", "users", "users", "启用/禁用账号", userToggleStatus},
	{http.MethodPost, "/users/reset-password", "users", "users", "重置账号密码", userResetPassword},

	// 角色与权限
	{http.MethodGet, "/roles", "roles:view", "", "", roleList},
	{http.MethodGet, "/roles/permissions", "roles:view", "", "", rolePermissionCatalogHandler},
	{http.MethodPost, "/roles/save", "roles", "roles", "保存角色", roleSave},
	{http.MethodPost, "/roles/delete", "roles", "roles", "删除角色", roleDelete},

	// 计数业务数据
	{http.MethodGet, "/counters", "counters:view", "", "", counterList},
	{http.MethodPost, "/counters/incr", "counters", "counters", "计数自增", counterIncr},
	{http.MethodPost, "/counters/set", "counters", "counters", "设置计数值", counterSet},
	{http.MethodPost, "/counters/delete", "counters", "counters", "删除计数记录", counterDelete},
	{http.MethodPost, "/counters/create", "counters", "counters", "新建计数记录", counterCreate},

	// 日志
	{http.MethodGet, "/logs/login", "logs:view", "", "", logLoginList},
	{http.MethodGet, "/logs/operation", "logs:view", "", "", logOperationList},
	{http.MethodPost, "/logs/clear", "logs", "logs", "清空日志", logClear},

	// 系统设置
	{http.MethodGet, "/configs", "configs:view", "", "", configList},
	{http.MethodPost, "/configs/save", "configs", "configs", "保存系统设置", configSave},
	{http.MethodGet, "/system/info", "system:view", "", "", systemInfo},

	// 小程序端（骨架，待接入）
	{http.MethodGet, "/mp/summary", "mp:view", "", "", mpSummary},
}

// AdminAPIHandler 后台管理接口统一入口，路径前缀 /api/admin
func AdminAPIHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/admin"), "/")
	if path == "" {
		path = "/"
	}

	// 登录接口不需要令牌
	if path == "/auth/login" {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeError(w, &apiError{Status: http.StatusMethodNotAllowed, Msg: "该接口仅支持 POST"})
			return
		}
		authLogin(w, r)
		return
	}

	ctx, err := loadAdminCtx(r)
	if err != nil {
		writeError(w, err)
		return
	}

	route, pathMatched := matchRoute(path, r.Method)
	if route == nil {
		if pathMatched {
			writeError(w, &apiError{Status: http.StatusMethodNotAllowed, Msg: "请求方法 " + r.Method + " 不支持"})
			return
		}
		writeError(w, notFound("接口不存在: "+path))
		return
	}

	if !hasRoutePermission(ctx, route.perm) {
		logOperation(ctx, "auth", "越权访问 "+path, r, false, "权限不足")
		writeError(w, forbidden("当前账号没有该操作权限"))
		return
	}

	data, err := route.handler(ctx, r)
	if err != nil {
		if route.module != "" {
			logOperation(ctx, route.module, route.action, r, false, err.Error())
		}
		writeError(w, err)
		return
	}

	if route.module != "" && route.action != "" {
		logOperation(ctx, route.module, route.action, r, true, "")
	}
	writeOK(w, data)
}

// matchRoute 匹配路由，返回路由与「路径是否命中」
func matchRoute(path, method string) (*apiRoute, bool) {
	pathMatched := false
	for i := range adminRoutes {
		rt := &adminRoutes[i]
		if rt.path != path {
			continue
		}
		pathMatched = true
		if rt.method == method {
			return rt, true
		}
	}
	return nil, pathMatched
}

// hasRoutePermission 校验路由权限
func hasRoutePermission(c *adminCtx, perm string) bool {
	if perm == "" {
		return true
	}
	if strings.HasSuffix(perm, ":view") {
		return c.canView(strings.TrimSuffix(perm, ":view"))
	}
	return c.canWrite(perm)
}
