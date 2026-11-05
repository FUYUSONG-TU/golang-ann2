package service

import (
	"net/http"
	"os"
	"strings"
	"time"

	"wxcloudrun-golang/db/dao"
	"wxcloudrun-golang/db/model"
	"wxcloudrun-golang/pkg/security"
)

// 令牌有效期
const (
	tokenTTL         = 12 * time.Hour
	tokenTTLRemember = 7 * 24 * time.Hour
)

// startAt 进程启动时间，用于计算运行时长
var startAt = time.Now()

// InitSecurity 初始化令牌签名密钥。
// 优先 ADMIN_SECRET，其次 ADMIN_TOKEN，最后回退到数据库口令派生的固定串，
// 保证多副本部署时各实例签发的令牌互通。
func InitSecurity() {
	secret := strings.TrimSpace(os.Getenv("ADMIN_SECRET"))
	if secret == "" {
		secret = strings.TrimSpace(os.Getenv("ADMIN_TOKEN"))
	}
	if secret == "" {
		secret = "jjxt-" + os.Getenv("MYSQL_PASSWORD")
	}
	if strings.TrimSpace(secret) == "jjxt-" {
		secret = "jjxt-local-dev-secret"
	}
	security.SetTokenSecret(secret)
}

// adminCtx 单次请求的登录上下文
type adminCtx struct {
	Claims *security.Claims
	User   *model.AdminUser
	Role   *model.AdminRole
	perms  map[string]bool
}

// can 判断是否拥有某个权限编码
func (c *adminCtx) can(code string) bool {
	if c == nil || c.perms == nil {
		return false
	}
	if c.perms["*"] {
		return true
	}
	return c.perms[code]
}

// canView 读权限：模块全权限或模块 .view 权限
func (c *adminCtx) canView(module string) bool {
	return c.can(module) || c.can(module+".view")
}

// canWrite 写权限：必须拥有模块全权限
func (c *adminCtx) canWrite(module string) bool {
	return c.can(module)
}

// permList 返回权限编码列表，超级管理员返回 ["*"]
func (c *adminCtx) permList() []string {
	if c == nil || c.Role == nil {
		return []string{}
	}
	raw := strings.TrimSpace(c.Role.Permissions)
	if raw == "" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// loadAdminCtx 解析请求中的令牌并加载账号，失败返回 401
func loadAdminCtx(r *http.Request) (*adminCtx, error) {
	raw := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(raw), "bearer ") {
		raw = strings.TrimSpace(raw[7:])
	}
	if raw == "" {
		raw = strings.TrimSpace(r.Header.Get("X-Admin-Token"))
	}
	if raw == "" {
		raw = r.URL.Query().Get("token")
	}
	if raw == "" {
		return nil, unauthorized("未登录或登录状态已失效")
	}

	claims, err := security.ParseToken(raw)
	if err != nil {
		return nil, unauthorized(err.Error())
	}

	user, err := dao.FindUserById(claims.Uid)
	if err != nil {
		return nil, unauthorized("账号不存在或已被删除")
	}
	if user.Status != 1 {
		return nil, forbidden("账号已被禁用，请联系管理员")
	}

	role, err := dao.FindRoleByCode(user.RoleCode)
	if err != nil {
		return nil, forbidden("账号角色配置异常，请联系管理员")
	}

	ctx := &adminCtx{Claims: claims, User: user, Role: role, perms: map[string]bool{}}
	for _, p := range splitPerms(role.Permissions) {
		ctx.perms[p] = true
	}
	return ctx, nil
}

// splitPerms 拆分权限串
func splitPerms(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// ---------- 登录相关接口 ----------

// authLogin 账号密码登录
func authLogin(w http.ResponseWriter, r *http.Request) {
	req := struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Remember bool   `json:"remember"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, err)
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		writeError(w, badRequest("请输入账号与密码"))
		return
	}

	ip := clientIP(r)
	ua := truncate(r.UserAgent(), 250)
	now := time.Now()

	user, err := dao.FindUserByUsername(req.Username)
	if err != nil {
		logLogin(0, req.Username, ip, ua, false, "账号不存在")
		writeError(w, unauthorized("账号或密码错误"))
		return
	}
	if !security.VerifyPassword(user.Password, req.Password) {
		logLogin(user.Id, user.Username, ip, ua, false, "密码错误")
		writeError(w, unauthorized("账号或密码错误"))
		return
	}
	if user.Status != 1 {
		logLogin(user.Id, user.Username, ip, ua, false, "账号已禁用")
		writeError(w, forbidden("账号已被禁用，请联系管理员"))
		return
	}

	role, err := dao.FindRoleByCode(user.RoleCode)
	if err != nil {
		logLogin(user.Id, user.Username, ip, ua, false, "角色不存在: "+user.RoleCode)
		writeError(w, forbidden("账号角色配置异常，请联系管理员"))
		return
	}

	ttl := tokenTTL
	if req.Remember {
		ttl = tokenTTLRemember
	}
	token, err := security.SignToken(security.Claims{
		Uid:      user.Id,
		Username: user.Username,
		Role:     user.RoleCode,
		IssuedAt: now.Unix(),
		ExpireAt: now.Add(ttl).Unix(),
	}, ttl)
	if err != nil {
		writeError(w, serverError("签发登录凭证失败: "+err.Error()))
		return
	}

	_ = dao.TouchUserLogin(user.Id, ip, now)
	logLogin(user.Id, user.Username, ip, ua, true, "登录成功")

	writeOK(w, map[string]interface{}{
		"token":       token,
		"expiresAt":   now.Add(ttl),
		"user":        user,
		"role":        role,
		"permissions": splitPerms(role.Permissions),
	})
}

// authProfile 当前登录人信息，附带少量通用运行信息，供布局页使用
func authProfile(c *adminCtx, r *http.Request) (interface{}, error) {
	latency, pingErr := dbLatency()
	values, _ := dao.ConfigMap()
	return map[string]interface{}{
		"user":        c.User,
		"role":        c.Role,
		"permissions": c.permList(),
		"dbOk":        pingErr == nil,
		"dbLatencyMs": latency,
		"version":     Version(),
		"siteName":    values["siteName"],
	}, nil
}

// authLogout 退出登录（客户端丢弃令牌，这里只记录日志）
func authLogout(c *adminCtx, r *http.Request) (interface{}, error) {
	logOperation(c, "auth", "退出登录", r, true, "")
	return map[string]interface{}{"ok": true}, nil
}

// authChangePassword 修改自己的密码
func authChangePassword(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if !security.VerifyPassword(c.User.Password, req.OldPassword) {
		return nil, badRequest("原密码不正确")
	}
	if len(req.NewPassword) < 6 {
		return nil, badRequest("新密码长度不能少于 6 位")
	}
	if req.NewPassword == req.OldPassword {
		return nil, badRequest("新密码不能与原密码相同")
	}
	hash, err := security.HashPassword(req.NewPassword)
	if err != nil {
		return nil, badRequest(err.Error())
	}
	if err := dao.UpdateUserFields(c.User.Id, map[string]interface{}{
		"password":      hash,
		"mustChangePwd": false,
	}); err != nil {
		return nil, serverError("保存失败: " + err.Error())
	}
	logOperation(c, "auth", "修改密码", r, true, "")
	return map[string]interface{}{"ok": true}, nil
}

// ---------- 日志辅助 ----------

func logLogin(userId int64, username, ip, ua string, success bool, message string) {
	_ = dao.AddLoginLog(&model.AdminLoginLog{
		UserId:    userId,
		Username:  username,
		Ip:        ip,
		UserAgent: ua,
		Success:   success,
		Message:   message,
		CreatedAt: time.Now(),
	})
}

func logOperation(c *adminCtx, module, action string, r *http.Request, success bool, detail string) {
	entry := &model.AdminOperationLog{
		Module:    module,
		Action:    action,
		Method:    r.Method,
		Path:      truncate(r.URL.Path, 240),
		Ip:        clientIP(r),
		Success:   success,
		Detail:    truncate(detail, 500),
		CreatedAt: time.Now(),
	}
	if c != nil && c.User != nil {
		entry.UserId = c.User.Id
		entry.Username = c.User.Username
	}
	_ = dao.AddOperationLog(entry)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	cut := max
	for cut > 0 && (s[cut]&0xC0) == 0x80 {
		cut--
	}
	return s[:cut]
}
