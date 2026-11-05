package db

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
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
	// 初始化专业分类，已存在则跳过
	if err := seedMajors(); err != nil {
		fmt.Println("seed majors error,err=", err.Error())
	}
	// 演示学员数据，仅在学员表为空且未显式关闭时写入，可在后台一键清除
	if err := seedDemoLearners(); err != nil {
		fmt.Println("seed demo learners error,err=", err.Error())
	}
	// 为存量内置角色补齐新增模块权限，避免升级后运营账号看不到课程菜单
	if err := syncBuiltinRolePermissions(); err != nil {
		fmt.Println("sync role permissions error,err=", err.Error())
	}
	// 修复历史版本在保存设置时把文本配置误写成布尔值的问题，失败不阻断启动
	if err := repairConfigs(); err != nil {
		fmt.Println("repair configs error,err=", err.Error())
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
		&model.Major{},
		&model.Course{},
		&model.Chapter{},
		&model.LearningPath{},
		&model.PathStage{},
		&model.PathItem{},
		&model.KbDocument{},
		&model.KbChunk{},
		&model.KbQa{},
		&model.MpUser{},
		&model.MpLearnRecord{},
	)
}

// 内置角色
var builtinRoles = []model.AdminRole{
	{Code: "super", Name: "超级管理员", Description: "拥有全部权限，可管理账号与角色", Permissions: "*", Builtin: true, Sort: 1},
	{Code: "operator", Name: "运营人员", Description: "日常数据与小程序内容维护，不能管理账号权限", Permissions: "dashboard,users.view,majors.view,majors,courses.view,courses,paths.view,paths,kb.view,kb,qa.view,qa,ai.view,ai,learners.view,learners,counters,mp,logs.view", Builtin: true, Sort: 2},
	{Code: "viewer", Name: "只读访客", Description: "仅可查看数据，所有写操作被拒绝", Permissions: "dashboard,majors.view,courses.view,paths.view,kb.view,qa.view,ai.view,learners.view,counters.view,logs.view", Builtin: true, Sort: 3},
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

// AI 配置默认值。密钥留空，配好之后才能回答。
// 注意：这一组配置不出现在「系统设置」页，统一在「AI 配置」页维护。
var defaultAiConfigs = []model.SysConfig{
	{Key: "aiEnabled", Value: "false", GroupName: "ai", Label: "AI 助手开关", Type: "switch", Remark: "关闭后小程序端与问答接口都不再调用模型"},
	{Key: "aiProvider", Value: "deepseek", GroupName: "ai", Label: "服务商", Type: "text", Remark: "当前支持所有 OpenAI 兼容接口"},
	{Key: "aiBaseUrl", Value: "https://api.deepseek.com", GroupName: "ai", Label: "接口地址", Type: "text", Remark: "DeepSeek 官方地址，自建代理可改成自己的域名"},
	{Key: "aiApiKey", Value: "", GroupName: "ai", Label: "API Key", Type: "secret", Remark: "只保存在数据库，接口返回时脱敏"},
	{Key: "aiChatModel", Value: "deepseek-chat", GroupName: "ai", Label: "对话模型", Type: "text", Remark: "深度思考用 deepseek-reasoner，速度更快用 deepseek-chat"},
	{Key: "aiEmbeddingModel", Value: "", GroupName: "ai", Label: "向量模型", Type: "text", Remark: "留空表示暂用关键词检索"},
	{Key: "aiTemperature", Value: "0.30", GroupName: "ai", Label: "温度", Type: "text", Remark: "0 最稳定，1 以上更发散；工艺类问答建议 0.2 - 0.4"},
	{Key: "aiMaxTokens", Value: "1200", GroupName: "ai", Label: "最大输出长度", Type: "text", Remark: "单次回答的 token 上限"},
	{Key: "aiTopK", Value: "5", GroupName: "ai", Label: "检索条数", Type: "text", Remark: "每次提问带几段资料给模型"},
	{Key: "aiMinScore", Value: "0.15", GroupName: "ai", Label: "最低相关度", Type: "text", Remark: "低于该分数的分块不进入上下文"},
	{Key: "aiSystemPrompt", Value: "", GroupName: "ai", Label: "系统提示词", Type: "textarea", Remark: "留空使用内置的机匠助手指令"},
	{Key: "aiRefuseText", Value: "", GroupName: "ai", Label: "无答案话术", Type: "text", Remark: "检索不到内容时的兜底回复"},
}

// 默认专业分类，仅在 major 表为空时写入
var defaultMajors = []model.Major{
	{Code: "zouxinji", Name: "走心机", Icon: "走", Summary: "走心式车床编程与加工工艺", Sort: 1, Status: 1},
	{Code: "cnc-lathe", Name: "数控车床", Icon: "车", Summary: "数控车削基础、循环指令与典型零件加工", Sort: 2, Status: 1},
	{Code: "machining-center", Name: "加工中心", Icon: "铣", Summary: "立式加工中心铣削编程与工艺", Sort: 3, Status: 1},
}

// seedMajors 初始化专业分类
func seedMajors() error {
	var count int64
	if err := dbInstance.Model(&model.Major{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	now := time.Now()
	for _, item := range defaultMajors {
		item.CreatedAt = now
		item.UpdatedAt = now
		if err := dbInstance.Create(&item).Error; err != nil {
			return err
		}
	}
	log.Printf("已初始化 %d 个默认专业", len(defaultMajors))
	return nil
}

// demoLearner 演示学员模板
type demoLearner struct {
	name      string
	phone     string
	level     int
	progress  []int
	dayOffset []int
}

// seedDemoLearners 写入演示学员与学习记录，让后台在没有小程序数据时也能看到完整页面。
// 只写一次：写过之后在 sys_config 里留标记，即使清空学员表、重启服务也不会再灌一次。
func seedDemoLearners() error {
	if strings.EqualFold(os.Getenv("SEED_DEMO_DATA"), "false") {
		return nil
	}
	var seeded int64
	if err := dbInstance.Model(&model.SysConfig{}).
		Where("`key` = ?", "demoLearnerSeeded").Count(&seeded).Error; err != nil {
		return err
	}
	if seeded > 0 {
		return nil
	}
	var count int64
	if err := dbInstance.Model(&model.MpUser{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		// 已有学员数据，直接记账，避免以后清空后又被重新灌入演示数据
		return markDemoSeeded()
	}

	majors := make([]model.Major, 0)
	if err := dbInstance.Order("sort asc, id asc").Find(&majors).Error; err != nil {
		return err
	}
	courses := make([]model.Course, 0)
	if err := dbInstance.Order("id asc").Limit(6).Find(&courses).Error; err != nil {
		return err
	}

	templates := []demoLearner{
		{"张一鸣", "13800000001", 2, []int{100, 65, 30}, []int{0, 1, 2}},
		{"李国强", "13800000002", 1, []int{80, 40}, []int{1, 3}},
		{"王海涛", "13800000003", 3, []int{100, 100, 75, 55}, []int{0, 0, 2, 5}},
		{"陈小娟", "13800000004", 1, []int{25}, []int{2}},
		{"赵鹏", "13800000005", 2, []int{100, 90}, []int{1, 4}},
		{"周敏", "13800000006", 1, nil, nil},
	}

	now := time.Now()
	for index, tpl := range templates {
		user := &model.MpUser{
			OpenId:     fmt.Sprintf("demo-learner-%02d", index+1),
			Nickname:   tpl.name,
			Phone:      tpl.phone,
			Level:      tpl.level,
			Status:     model.MpUserNormal,
			Remark:     "演示数据，可在学员页一键清理",
			LoginCount: 3 + index,
			CreatedAt:  now.AddDate(0, 0, -(20 - index*3)),
			UpdatedAt:  now.AddDate(0, 0, -(3 + index)),
		}
		if len(majors) > 0 {
			user.MajorId = majors[index%len(majors)].Id
		}
		activeAt := now.Add(-time.Duration(6+index*9) * time.Hour)
		user.LastActiveAt = &activeAt
		if err := dbInstance.Create(user).Error; err != nil {
			return err
		}

		for i, progress := range tpl.progress {
			if i >= len(courses) {
				break
			}
			studyAt := now.AddDate(0, 0, -tpl.dayOffset[i])
			status := model.LearnLearning
			if progress >= 100 {
				status = model.LearnFinished
			}
			record := &model.MpLearnRecord{
				UserId:       user.Id,
				CourseId:     courses[i].Id,
				Progress:     progress,
				StudySeconds: 600 + progress*90,
				Status:       status,
				LastStudyAt:  &studyAt,
				CreatedAt:    user.CreatedAt,
				UpdatedAt:    studyAt,
			}
			if err := dbInstance.Create(record).Error; err != nil {
				return err
			}
		}
	}
	log.Printf("已初始化 %d 名演示学员（可清理）", len(templates))
	return markDemoSeeded()
}

// markDemoSeeded 记录演示数据已处理过，避免清理后重启又灌一遍
func markDemoSeeded() error {
	var count int64
	if err := dbInstance.Model(&model.SysConfig{}).
		Where("`key` = ?", "demoLearnerSeeded").Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return dbInstance.Create(&model.SysConfig{
		Key:       "demoLearnerSeeded",
		Value:     "true",
		GroupName: "system",
		Label:     "演示学员已初始化",
		Type:      "hidden",
		Remark:    "内部标记，不对外展示",
		UpdatedAt: time.Now(),
	}).Error
}

// syncBuiltinRolePermissions 为已存在的内置角色补齐新增模块的默认权限。
// 只追加角色当前完全没有的权限码，管理员已手工调整过的其它权限不受影响。
func syncBuiltinRolePermissions() error {
	defaults := map[string][]string{
		"operator": {"majors.view", "majors", "courses.view", "courses", "paths.view", "paths", "kb.view", "kb", "qa.view", "qa", "ai.view", "ai", "learners.view", "learners"},
		"viewer":   {"majors.view", "courses.view", "paths.view", "kb.view", "qa.view", "ai.view", "learners.view"},
	}
	for code, modules := range defaults {
		var role model.AdminRole
		if err := dbInstance.Where("code = ? AND builtin = ?", code, true).First(&role).Error; err != nil {
			continue
		}
		has := map[string]bool{}
		for _, item := range strings.Split(role.Permissions, ",") {
			if item = strings.TrimSpace(item); item != "" {
				has[item] = true
			}
		}
		if has["*"] {
			continue
		}
		missing := make([]string, 0, len(modules))
		for _, m := range modules {
			if !has[m] {
				missing = append(missing, m)
			}
		}
		if len(missing) == 0 {
			continue
		}
		merged := strings.TrimSpace(role.Permissions)
		if merged != "" {
			merged += ","
		}
		merged += strings.Join(missing, ",")
		if err := dbInstance.Model(&model.AdminRole{}).Where("id = ?", role.Id).
			Update("permissions", merged).Error; err != nil {
			return err
		}
		log.Printf("已为内置角色 %s 补充权限: %s", code, strings.Join(missing, ","))
	}
	return nil
}

// repairConfigs 修复历史缺陷：保存设置时会把所有文本配置写成 "true"/"false"。
// 只处理内置的文本类配置，且仅当当前值恰好是布尔字符串时才回退到默认值，
// 因此对正常编辑过的配置没有影响，重复执行也安全。
func repairConfigs() error {
	repaired := 0
	for _, def := range defaultConfigs {
		if def.Type == "switch" {
			continue
		}
		var item model.SysConfig
		if err := dbInstance.Where("`key` = ?", def.Key).First(&item).Error; err != nil {
			continue
		}
		value := strings.TrimSpace(item.Value)
		if value != "true" && value != "false" {
			continue
		}
		if err := dbInstance.Model(&model.SysConfig{}).Where("id = ?", item.Id).
			Update("value", def.Value).Error; err != nil {
			return err
		}
		repaired++
	}
	if repaired > 0 {
		log.Printf("已修复 %d 项被误写为布尔值的站点配置", repaired)
	}
	return nil
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

	for _, cfg := range append(append([]model.SysConfig{}, defaultConfigs...), defaultAiConfigs...) {
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
