package dao

import (
	"time"

	"wxcloudrun-golang/db"
	"wxcloudrun-golang/db/model"
)

// MpUserQuery 学员查询条件
type MpUserQuery struct {
	Keyword string
	MajorId int64
	Level   int // 0 表示不限
	Status  int // -1 表示不限
	Page    int
	Size    int
}

// ListMpUsers 分页查询学员
func ListMpUsers(q MpUserQuery) ([]model.MpUser, int64, error) {
	list := make([]model.MpUser, 0)
	query := db.Get().Model(&model.MpUser{})
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		query = query.Where("nickname LIKE ? OR phone LIKE ? OR openId LIKE ?", kw, kw, kw)
	}
	if q.MajorId > 0 {
		query = query.Where("majorId = ?", q.MajorId)
	}
	if q.Level > 0 {
		query = query.Where("level = ?", q.Level)
	}
	if q.Status >= 0 {
		query = query.Where("status = ?", q.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.Size <= 0 {
		q.Size = 20
	}
	err := query.Order("id desc").
		Offset((q.Page - 1) * q.Size).Limit(q.Size).Find(&list).Error
	return list, total, err
}

// FindMpUser 按主键查询学员
func FindMpUser(id int64) (*model.MpUser, error) {
	var item model.MpUser
	if err := db.Get().Where("id = ?", id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// FindMpUserByOpenId 按 openId 查询学员，小程序登录时复用
func FindMpUserByOpenId(openId string) (*model.MpUser, error) {
	var item model.MpUser
	if err := db.Get().Where("openId = ?", openId).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// CreateMpUser 新建学员
func CreateMpUser(item *model.MpUser) error {
	return db.Get().Create(item).Error
}

// UpdateMpUserFields 更新学员指定字段
func UpdateMpUserFields(id int64, fields map[string]interface{}) error {
	return db.Get().Model(&model.MpUser{}).Where("id = ?", id).Updates(fields).Error
}

// DeleteMpUser 软删除学员
func DeleteMpUser(id int64) error {
	return db.Get().Delete(&model.MpUser{}, id).Error
}

// CountMpUsers 统计学员总数，status 传 -1 表示不限
func CountMpUsers(status int) (int64, error) {
	var total int64
	query := db.Get().Model(&model.MpUser{})
	if status >= 0 {
		query = query.Where("status = ?", status)
	}
	err := query.Count(&total).Error
	return total, err
}

// CountMpUsersCreatedSince 统计某时间之后新增的学员
func CountMpUsersCreatedSince(t time.Time) (int64, error) {
	var total int64
	err := db.Get().Model(&model.MpUser{}).Where("createdAt >= ?", t).Count(&total).Error
	return total, err
}

// CountMpUsersActiveSince 统计某时间之后活跃过的学员
func CountMpUsersActiveSince(t time.Time) (int64, error) {
	var total int64
	err := db.Get().Model(&model.MpUser{}).Where("lastActiveAt >= ?", t).Count(&total).Error
	return total, err
}

// 学员分布统计行
type levelCountRow struct {
	Level int   `gorm:"column:level"`
	Total int64 `gorm:"column:total"`
}

// CountMpUsersByLevel 统计各水平学员数量
func CountMpUsersByLevel() (map[int]int64, error) {
	rows := make([]levelCountRow, 0)
	err := db.Get().Model(&model.MpUser{}).
		Select("level, COUNT(*) AS total").
		Group("level").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[int]int64, len(rows))
	for _, row := range rows {
		out[row.Level] = row.Total
	}
	return out, nil
}

// ListRecordsByUser 查询学员全部学习记录
func ListRecordsByUser(userId int64) ([]model.MpLearnRecord, error) {
	list := make([]model.MpLearnRecord, 0)
	err := db.Get().Where("userId = ?", userId).
		Order("lastStudyAt desc, id desc").Find(&list).Error
	return list, err
}

// DeleteRecordsByUser 级联软删除学员的学习记录
func DeleteRecordsByUser(userId int64) error {
	return db.Get().Where("userId = ?", userId).Delete(&model.MpLearnRecord{}).Error
}

// CreateLearnRecord 写入学习记录，供小程序上报与演示数据写入
func CreateLearnRecord(item *model.MpLearnRecord) error {
	return db.Get().Create(item).Error
}

// LearnAgg 学员学习记录汇总
type LearnAgg struct {
	UserId       int64      `gorm:"column:userId"`
	Total        int64      `gorm:"column:total"`
	Finished     int64      `gorm:"column:finished"`
	AvgProgress  float64    `gorm:"column:avgProgress"`
	StudySeconds int64      `gorm:"column:studySeconds"`
	LastStudyAt  *time.Time `gorm:"column:lastStudyAt"`
}

// LearnAggByUsers 一次性统计多学员的学习进度，列表页避免 N+1 查询
func LearnAggByUsers(userIds []int64) (map[int64]LearnAgg, error) {
	out := make(map[int64]LearnAgg, len(userIds))
	if len(userIds) == 0 {
		return out, nil
	}
	rows := make([]LearnAgg, 0, len(userIds))
	err := db.Get().Model(&model.MpLearnRecord{}).
		Select("userId, COUNT(*) AS total, "+
			"SUM(CASE WHEN status = 'finished' THEN 1 ELSE 0 END) AS finished, "+
			"AVG(progress) AS avgProgress, SUM(studySeconds) AS studySeconds, "+
			"MAX(lastStudyAt) AS lastStudyAt").
		Where("userId IN ?", userIds).
		Group("userId").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.UserId] = row
	}
	return out, nil
}

// CountDemoMpUsers 统计演示学员数量，后台据此显示清理入口
func CountDemoMpUsers() (int64, error) {
	var total int64
	err := db.Get().Model(&model.MpUser{}).Where("openId LIKE ?", "demo-%").Count(&total).Error
	return total, err
}

// PurgeDemoMpUsers 彻底清除演示学员及其学习记录。// 演示数据 openId 统一以 demo- 开头，与真实微信 openId 不会冲突。
func PurgeDemoMpUsers() (int64, error) {
	ids := make([]int64, 0)
	if err := db.Get().Unscoped().Model(&model.MpUser{}).
		Where("openId LIKE ?", "demo-%").Pluck("id", &ids).Error; err != nil {
		return 0, err
	}
	if len(ids) > 0 {
		if err := db.Get().Unscoped().Where("userId IN ?", ids).
			Delete(&model.MpLearnRecord{}).Error; err != nil {
			return 0, err
		}
	}
	res := db.Get().Unscoped().Where("openId LIKE ?", "demo-%").Delete(&model.MpUser{})
	return res.RowsAffected, res.Error
}
