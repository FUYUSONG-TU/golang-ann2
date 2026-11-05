package model

import "time"

// AdminUser 后台管理账号
type AdminUser struct {
	Id          int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Username    string     `gorm:"column:username;size:64;uniqueIndex:uk_admin_username" json:"username"`
	Password    string     `gorm:"column:password;size:255" json:"-"`
	Nickname    string     `gorm:"column:nickname;size:64" json:"nickname"`
	Email       string     `gorm:"column:email;size:128" json:"email"`
	Phone       string     `gorm:"column:phone;size:32" json:"phone"`
	RoleCode    string     `gorm:"column:roleCode;size:32;index:idx_admin_role" json:"roleCode"`
	Status      int        `gorm:"column:status;default:1" json:"status"`
	Remark      string     `gorm:"column:remark;size:255" json:"remark"`
	MustChange  bool       `gorm:"column:mustChangePwd;default:false" json:"mustChangePwd"`
	LastLoginAt *time.Time `gorm:"column:lastLoginAt" json:"lastLoginAt"`
	LastLoginIp string     `gorm:"column:lastLoginIp;size:64" json:"lastLoginIp"`
	CreatedAt   time.Time  `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt   time.Time  `gorm:"column:updatedAt" json:"updatedAt"`
}

// TableName 指定表名
func (AdminUser) TableName() string { return "admin_user" }

// AdminRole 角色，Permissions 为逗号分隔的权限编码，* 表示全部
type AdminRole struct {
	Id          int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Code        string    `gorm:"column:code;size:32;uniqueIndex:uk_role_code" json:"code"`
	Name        string    `gorm:"column:name;size:64" json:"name"`
	Description string    `gorm:"column:description;size:255" json:"description"`
	Permissions string    `gorm:"column:permissions;size:512" json:"permissions"`
	Builtin     bool      `gorm:"column:builtin;default:false" json:"builtin"`
	Sort        int       `gorm:"column:sort;default:0" json:"sort"`
	CreatedAt   time.Time `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"column:updatedAt" json:"updatedAt"`
}

// TableName 指定表名
func (AdminRole) TableName() string { return "admin_role" }

// AdminLoginLog 登录日志
type AdminLoginLog struct {
	Id        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserId    int64     `gorm:"column:userId;index:idx_login_user" json:"userId"`
	Username  string    `gorm:"column:username;size:64;index:idx_login_name" json:"username"`
	Ip        string    `gorm:"column:ip;size:64" json:"ip"`
	UserAgent string    `gorm:"column:userAgent;size:255" json:"userAgent"`
	Success   bool      `gorm:"column:success" json:"success"`
	Message   string    `gorm:"column:message;size:255" json:"message"`
	CreatedAt time.Time `gorm:"column:createdAt;index:idx_login_time" json:"createdAt"`
}

// TableName 指定表名
func (AdminLoginLog) TableName() string { return "admin_login_log" }

// AdminOperationLog 操作日志
type AdminOperationLog struct {
	Id        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserId    int64     `gorm:"column:userId;index:idx_op_user" json:"userId"`
	Username  string    `gorm:"column:username;size:64" json:"username"`
	Module    string    `gorm:"column:module;size:32;index:idx_op_module" json:"module"`
	Action    string    `gorm:"column:action;size:64" json:"action"`
	Method    string    `gorm:"column:method;size:16" json:"method"`
	Path      string    `gorm:"column:path;size:255" json:"path"`
	Ip        string    `gorm:"column:ip;size:64" json:"ip"`
	Success   bool      `gorm:"column:success" json:"success"`
	Detail    string    `gorm:"column:detail;size:512" json:"detail"`
	CreatedAt time.Time `gorm:"column:createdAt;index:idx_op_time" json:"createdAt"`
}

// TableName 指定表名
func (AdminOperationLog) TableName() string { return "admin_operation_log" }

// SysConfig 站点配置（键值对）
type SysConfig struct {
	Id        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Key       string    `gorm:"column:key;size:64;uniqueIndex:uk_config_key" json:"key"`
	Value     string    `gorm:"column:value;type:text" json:"value"`
	GroupName string    `gorm:"column:groupName;size:32" json:"groupName"`
	Label     string    `gorm:"column:label;size:64" json:"label"`
	Type      string    `gorm:"column:type;size:16" json:"type"`
	Remark    string    `gorm:"column:remark;size:255" json:"remark"`
	UpdatedAt time.Time `gorm:"column:updatedAt" json:"updatedAt"`
}

// TableName 指定表名
func (SysConfig) TableName() string { return "sys_config" }
