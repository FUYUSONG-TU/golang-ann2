package service

import (
	"net/http"
	"runtime"
	"time"

	"wxcloudrun-golang/db"
	"wxcloudrun-golang/db/dao"
	"wxcloudrun-golang/db/model"
)

// appVersion 后台版本号
const appVersion = "3.0.0"

// defaultCounterID 模板默认使用的计数器主键
const defaultCounterID int32 = 1

// dashboardOverview 数据看板
func dashboardOverview(c *adminCtx, r *http.Request) (interface{}, error) {
	todayStart := time.Now().Truncate(24 * time.Hour)
	now := time.Now()
	todayStart = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	userTotal, _ := dao.CountUsers(-1)
	userActive, _ := dao.CountUsers(1)
	userToday, _ := dao.CountUsersCreatedSince(todayStart)

	roleTotal := int64(0)
	if roles, err := dao.ListRoles(); err == nil {
		roleTotal = int64(len(roles))
	}

	loginToday, _ := dao.CountLoginsSince(todayStart)
	loginFailToday, _ := dao.CountLoginFailSince(todayStart)
	opToday, _ := dao.CountOperationsSince(todayStart)

	trend, err := dao.LoginTrend(7)
	if err != nil {
		trend = []dao.DayCount{}
	}

	recentLogins, _ := dao.RecentLoginLogs(6)
	recentOps, _ := dao.RecentOperationLogs(6)

	counterValue := int32(0)
	counterTotal := 0
	if list, err := dao.Imp.ListCounters(); err == nil {
		counterTotal = len(list)
		for _, item := range list {
			if item.Id == defaultCounterID {
				counterValue = item.Count
				break
			}
		}
	}

	// 内容规模：专业与课程，供看板展示
	majorTotal := int64(0)
	if majors, err := dao.ListMajors(dao.MajorQuery{Status: -1}); err == nil {
		majorTotal = int64(len(majors))
	}
	courseTotal, coursePublished := int64(0), int64(0)
	if counts, err := dao.CountCoursesByStatus(); err == nil {
		for _, v := range counts {
			courseTotal += v
		}
		coursePublished = counts[model.ContentPublished]
	}

	latency, pingErr := dbLatency()
	values, _ := dao.ConfigMap()

	return map[string]interface{}{
		"cards": map[string]interface{}{
			"userTotal":       userTotal,
			"userActive":      userActive,
			"userToday":       userToday,
			"roleTotal":       roleTotal,
			"loginToday":      loginToday,
			"loginFailToday":  loginFailToday,
			"opToday":         opToday,
			"counterValue":    counterValue,
			"counterTotal":    counterTotal,
			"majorTotal":      majorTotal,
			"courseTotal":     courseTotal,
			"coursePublished": coursePublished,
		},
		"trend":         trend,
		"recentLogins":  recentLogins,
		"recentOps":     recentOps,
		"announcement":  values["announcement"],
		"dbOk":          pingErr == nil,
		"dbLatencyMs":   latency,
		"dbError":       errText(pingErr),
		"serverTime":    time.Now(),
		"uptimeSeconds": int(time.Since(startAt).Seconds()),
		"version":       appVersion,
	}, nil
}

// systemInfo 运行时信息
func systemInfo(c *adminCtx, r *http.Request) (interface{}, error) {
	latency, pingErr := dbLatency()
	values, _ := dao.ConfigMap()

	var sqlVersion string
	if cli := db.Get(); cli != nil {
		_ = cli.Raw("SELECT VERSION()").Scan(&sqlVersion).Error
	}

	return map[string]interface{}{
		"version":       appVersion,
		"goVersion":     runtime.Version(),
		"os":            runtime.GOOS + "/" + runtime.GOARCH,
		"cpu":           runtime.NumCPU(),
		"goroutines":    runtime.NumGoroutine(),
		"startAt":       startAt,
		"uptimeSeconds": int(time.Since(startAt).Seconds()),
		"serverTime":    time.Now(),
		"timeZone":      timeZoneText(),
		"dbOk":          pingErr == nil,
		"dbLatencyMs":   latency,
		"dbError":       errText(pingErr),
		"dbVersion":     sqlVersion,
		"siteName":      values["siteName"],
		"maintenance":   values["maintenance"] == "true",
		"accounts": map[string]interface{}{
			"current": c.User.Username,
			"role":    c.Role.Name,
		},
	}, nil
}

// mpSummary 小程序端接入骨架
func mpSummary(c *adminCtx, r *http.Request) (interface{}, error) {
	values, _ := dao.ConfigMap()

	appId := values["miniProgramAppId"]
	connected := appId != ""

	modules := []map[string]interface{}{
		{"key": "user", "name": "用户与登录", "desc": "小程序端微信授权、会话与用户档案", "status": "planned", "api": "/api/mp/user/profile"},
		{"key": "banner", "name": "首页轮播", "desc": "首页 banner 图与跳转配置", "status": "planned", "api": "/api/mp/banner/list"},
		{"key": "notice", "name": "公告通知", "desc": "小程序内公告，当前复用后台公告配置", "status": "ready", "api": "/api/mp/notice/list"},
		{"key": "category", "name": "服务分类", "desc": "服务/作品分类树", "status": "planned", "api": "/api/mp/category/list"},
		{"key": "work", "name": "作品案例", "desc": "案例图文、图集与详情", "status": "planned", "api": "/api/mp/work/list"},
		{"key": "booking", "name": "预约工单", "desc": "上门/到店预约与派单", "status": "planned", "api": "/api/mp/booking/create"},
		{"key": "order", "name": "订单支付", "desc": "下单、支付回调与售后", "status": "planned", "api": "/api/mp/order/list"},
		{"key": "message", "name": "订阅消息", "desc": "微信订阅消息模板与推送记录", "status": "planned", "api": "/api/mp/message/push"},
	}

	ready := 0
	for _, m := range modules {
		if m["status"] == "ready" {
			ready++
		}
	}

	return map[string]interface{}{
		"connected":   connected,
		"name":        values["miniProgramName"],
		"appId":       appId,
		"env":         "未配置",
		"modules":     modules,
		"readyCount":  ready,
		"totalCount":  len(modules),
		"apiBase":     "/api/mp",
		"description": "小程序端尚未开发，这里预留了模块清单与接口契约，接入时按模块补齐即可。",
	}, nil
}

// ---------- 内部辅助 ----------

func dbLatency() (int64, error) {
	start := time.Now()
	cli := db.Get()
	if cli == nil {
		return 0, db.ErrNotReady
	}
	sqlDB, err := cli.DB()
	if err != nil {
		return 0, err
	}
	if err := sqlDB.Ping(); err != nil {
		return time.Since(start).Milliseconds(), err
	}
	return time.Since(start).Milliseconds(), nil
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func timeZoneText() string {
	name, offset := time.Now().Zone()
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	return name + " UTC" + sign + pad2(offset/3600) + ":" + pad2((offset%3600)/60)
}

func pad2(n int) string {
	if n < 10 {
		return "0" + string(rune('0'+n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}

// Version 返回后台版本号
func Version() string { return appVersion }
