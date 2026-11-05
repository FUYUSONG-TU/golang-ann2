package db

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"wxcloudrun-golang/db/model"
	"wxcloudrun-golang/pkg/security"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

var dbInstance *gorm.DB

// Init 初始化数据库
func Init() error {

	source := "%s:%s@tcp(%s)/%s?readTimeout=1500ms&writeTimeout=1500ms&charset=utf8mb4&loc=Local&&parseTime=true"
	user := os.Getenv("MYSQL_USERNAME")
	pwd := os.Getenv("MYSQL_PASSWORD")
	addr := os.Getenv("MYSQL_ADDRESS")
	dataBase := os.Getenv("MYSQL_DATABASE")
	if dataBase == "" {
		dataBase = "golang_demo"
	}
	source = fmt.Sprintf(source, user, pwd, addr, dataBase)
	// 日志里只保留连接目标，避免把数据库密码打印到云托管日志中
	safeTarget := fmt.Sprintf("%s@tcp(%s)/%s", user, addr, dataBase)
	fmt.Println("start init mysql with ", safeTarget)

	db, err := gorm.Open(mysql.Open(source), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, // use singular table name, table for `User` would be `user` with this option enabled
		}})
	if err != nil {
		fmt.Println("DB Open error,err=", err.Error())
		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		fmt.Println("DB Init error,err=", err.Error())
		return err
	}

	// 用于设置连接池中空闲连接的最大数量
	sqlDB.SetMaxIdleConns(100)
	// 设置打开数据库连接的最大数量
	sqlDB.SetMaxOpenConns(200)
	// 设置了连接可复用的最大时间
	sqlDB.SetConnMaxLifetime(time.Hour)

	dbInstance = db

	if err := migrateTables(); err != nil {
		fmt.Println("AutoMigrate error,err=", err.Error())
		return err
	}
	if err := seedData(); err != nil {
		fmt.Println("seed data error,err=", err.Error())
		return err
	}

	fmt.Println("finish init mysql with ", safeTarget)
	return nil
}

// migrateTables 自动建表，已存在则只补字段
func migrateTables() error {
	return dbInstance.AutoMigrate(
		&model.CounterModel{},
		&model.AdminUser{},
		&model.AdminRole{},
		&model.AdminLoginLog{},
		&model.AdminOperationLog{},
		&model.SysConfig{},
	)
}

// 内置角色
var builtinRoles = []model.AdminRole{
	{Code: "super", Name: "超级管理员", Description: "拥有全部权限，可管理账号与角色", Permissions: "*", Builtin: true, Sort: 1},
	{Code: "operator", Name: "运营人员", Description: "日常数据与小程序内容维护，不能管理账号权限", Permissions: "dashboard,users.view,counters,mp,logs.view", Builtin: true, Sort: 2},
	{Code: "viewer", Name: "只读访客", Description: "仅可查看数据，所有写操作被拒绝", Permissions: "dashboard,counters.view,logs.view", Builtin: true, Sort: 3},
}

// 默认站点配置
var defaultConfigs = []model.SysConfig{
	{Key: "siteName", Value: "机匠学徒", GroupName: "basic", Label: "站点名称", Type: "text", Remark: "显示在登录页与侧边栏"},
	{Key: "siteSlogan", Value: "把手艺沉进系统里", GroupName: "basic", Label: "站点标语", Type: "text", Remark: "登录页副标题"},
	{Key: "siteLogo", Value: "匠", GroupName: "basic", Label: "Logo 文字", Type: "text", Remark: "单字或短词"},
	{Key: "copyright", Value: "© 2026 机匠学徒", GroupName: "basic", Label: "版权信息", Type: "text", Remark: "页脚展示"},
	{Key: "icp", Value: "", GroupName: "basic", Label: "备案号", Type: "text", Remark: "选填"},
	{Key: "announcement", Value: "后台骨架已就绪，小程序端接口待接入。", GroupName: "basic", Label: "后台公告", Type: "textarea", Remark: "登录后首页展示"},
	{Key: "maintenance", Value: "false", GroupName: "basic", Label: "维护模式", Type: "switch", Remark: "开启后前台接口返回维护提示"},
	{Key: "miniProgramName", Value: "机匠学徒", GroupName: "mini", Label: "小程序名称", Type: "text", Remark: "小程序端展示名"},
	{Key: "miniProgramAppId", Value: "", GroupName: "mini", Label: "AppID", Type: "text", Remark: "小程序 AppID，接入后填写"},
}

// seedData 首次启动写入内置数据，已存在则跳过
func seedData() error {
	for _, role := range builtinRoles {
		var count int64
		if err := dbInstance.Model(&model.AdminRole{}).Where("code = ?", role.Code).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		role.CreatedAt = time.Now()
		role.UpdatedAt = time.Now()
		if err := dbInstance.Create(&role).Error; err != nil {
			return err
		}
	}

	for _, cfg := range defaultConfigs {
		var count int64
		if err := dbInstance.Model(&model.SysConfig{}).Where("`key` = ?", cfg.Key).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		cfg.UpdatedAt = time.Now()
		if err := dbInstance.Create(&cfg).Error; err != nil {
			return err
		}
	}

	var userCount int64
	if err := dbInstance.Model(&model.AdminUser{}).Count(&userCount).Error; err != nil {
		return err
	}
	if userCount > 0 {
		return nil
	}

	initPwd := os.Getenv("ADMIN_INIT_PASSWORD")
	if initPwd == "" {
		initPwd = "admin123"
	}
	hash, err := security.HashPassword(initPwd)
	if err != nil {
		return err
	}
	now := time.Now()
	admin := &model.AdminUser{
		Username:   "admin",
		Password:   hash,
		Nickname:   "超级管理员",
		RoleCode:   "super",
		Status:     1,
		Remark:     "系统初始化创建",
		MustChange: true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := dbInstance.Create(admin).Error; err != nil {
		return err
	}
	log.Printf("已创建初始管理员账号 admin，初始密码 %s（首次登录请立即修改）", initPwd)
	return nil
}

// Get ...
func Get() *gorm.DB {
	return dbInstance
}

// ErrNotReady 数据库尚未初始化
var ErrNotReady = errors.New("数据库未初始化")
