package dao

import (
	"strings"
	"time"

	"wxcloudrun-golang/db"
	"wxcloudrun-golang/db/model"
)

// LogQuery 日志列表查询条件
type LogQuery struct {
	Keyword string
	Module  string
	Success int // -1 表示不限
	Page    int
	Size    int
}

// DayCount 按天聚合的结果
type DayCount struct {
	Day   string `json:"day"`
	Total int64  `json:"total"`
}

// AddLoginLog 写入登录日志
func AddLoginLog(l *model.AdminLoginLog) error {
	if l.CreatedAt.IsZero() {
		l.CreatedAt = time.Now()
	}
	return db.Get().Create(l).Error
}

// AddOperationLog 写入操作日志
func AddOperationLog(l *model.AdminOperationLog) error {
	if l.CreatedAt.IsZero() {
		l.CreatedAt = time.Now()
	}
	return db.Get().Create(l).Error
}

// ListLoginLogs 分页查询登录日志
func ListLoginLogs(q LogQuery) ([]model.AdminLoginLog, int64, error) {
	cli := db.Get().Model(&model.AdminLoginLog{})
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		like := "%" + kw + "%"
		cli = cli.Where("username LIKE ? OR ip LIKE ?", like, like)
	}
	if q.Success >= 0 {
		cli = cli.Where("success = ?", q.Success == 1)
	}
	var total int64
	if err := cli.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	list := make([]model.AdminLoginLog, 0)
	err := cli.Order("id desc").Offset((q.Page - 1) * q.Size).Limit(q.Size).Find(&list).Error
	return list, total, err
}

// ListOperationLogs 分页查询操作日志
func ListOperationLogs(q LogQuery) ([]model.AdminOperationLog, int64, error) {
	cli := db.Get().Model(&model.AdminOperationLog{})
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		like := "%" + kw + "%"
		cli = cli.Where("username LIKE ? OR action LIKE ? OR path LIKE ?", like, like, like)
	}
	if q.Module != "" {
		cli = cli.Where("module = ?", q.Module)
	}
	if q.Success >= 0 {
		cli = cli.Where("success = ?", q.Success == 1)
	}
	var total int64
	if err := cli.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	list := make([]model.AdminOperationLog, 0)
	err := cli.Order("id desc").Offset((q.Page - 1) * q.Size).Limit(q.Size).Find(&list).Error
	return list, total, err
}

// ClearLoginLogs 清空登录日志
func ClearLoginLogs() error {
	return db.Get().Where("1 = 1").Delete(&model.AdminLoginLog{}).Error
}

// ClearOperationLogs 清空操作日志
func ClearOperationLogs() error {
	return db.Get().Where("1 = 1").Delete(&model.AdminOperationLog{}).Error
}

// CountLoginsSince 统计某时间点之后的登录成功次数
func CountLoginsSince(since time.Time) (int64, error) {
	var n int64
	err := db.Get().Model(&model.AdminLoginLog{}).
		Where("createdAt >= ? AND success = ?", since, true).Count(&n).Error
	return n, err
}

// CountLoginFailSince 统计某时间点之后的登录失败次数
func CountLoginFailSince(since time.Time) (int64, error) {
	var n int64
	err := db.Get().Model(&model.AdminLoginLog{}).
		Where("createdAt >= ? AND success = ?", since, false).Count(&n).Error
	return n, err
}

// CountOperationsSince 统计某时间点之后的操作次数
func CountOperationsSince(since time.Time) (int64, error) {
	var n int64
	err := db.Get().Model(&model.AdminOperationLog{}).Where("createdAt >= ?", since).Count(&n).Error
	return n, err
}

// LoginTrend 返回最近 days 天（含今天）每天的成功登录次数
func LoginTrend(days int) ([]DayCount, error) {
	since := time.Now().AddDate(0, 0, -(days - 1))
	since = time.Date(since.Year(), since.Month(), since.Day(), 0, 0, 0, 0, since.Location())

	type row struct {
		Day   string
		Total int64
	}
	var rows []row
	err := db.Get().Model(&model.AdminLoginLog{}).
		Select("DATE_FORMAT(createdAt, '%Y-%m-%d') as day, count(*) as total").
		Where("createdAt >= ? AND success = ?", since, true).
		Group("day").Order("day asc").Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	filled := make(map[string]int64, len(rows))
	for _, r := range rows {
		filled[r.Day] = r.Total
	}

	out := make([]DayCount, 0, days)
	for i := 0; i < days; i++ {
		d := since.AddDate(0, 0, i).Format("2006-01-02")
		out = append(out, DayCount{Day: d, Total: filled[d]})
	}
	return out, nil
}

// RecentLoginLogs 最近若干条登录日志
func RecentLoginLogs(limit int) ([]model.AdminLoginLog, error) {
	list := make([]model.AdminLoginLog, 0)
	err := db.Get().Order("id desc").Limit(limit).Find(&list).Error
	return list, err
}

// RecentOperationLogs 最近若干条操作日志
func RecentOperationLogs(limit int) ([]model.AdminOperationLog, error) {
	list := make([]model.AdminOperationLog, 0)
	err := db.Get().Order("id desc").Limit(limit).Find(&list).Error
	return list, err
}
