package dao

import (
	"wxcloudrun-golang/db"
	"wxcloudrun-golang/db/model"
)

// PathQuery 学习路线查询条件
type PathQuery struct {
	MajorId int64
	Keyword string
	Status  string
	Page    int
	Size    int
}

// ListPaths 分页查询学习路线
func ListPaths(q PathQuery) ([]model.LearningPath, int64, error) {
	list := make([]model.LearningPath, 0)
	query := db.Get().Model(&model.LearningPath{})
	if q.MajorId > 0 {
		query = query.Where("majorId = ? OR majorId = 0", q.MajorId)
	}
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		query = query.Where("name LIKE ? OR summary LIKE ?", kw, kw)
	}
	if q.Status != "" {
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
	err := query.Order("sort asc, id desc").
		Offset((q.Page - 1) * q.Size).Limit(q.Size).Find(&list).Error
	return list, total, err
}

// FindPath 按主键查询学习路线
func FindPath(id int64) (*model.LearningPath, error) {
	var item model.LearningPath
	if err := db.Get().Where("id = ?", id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// CreatePath 新建学习路线
func CreatePath(item *model.LearningPath) error {
	return db.Get().Create(item).Error
}

// UpdatePathFields 更新学习路线指定字段
func UpdatePathFields(id int64, fields map[string]interface{}) error {
	return db.Get().Model(&model.LearningPath{}).Where("id = ?", id).Updates(fields).Error
}

// DeletePath 软删除学习路线
func DeletePath(id int64) error {
	return db.Get().Delete(&model.LearningPath{}, id).Error
}

// ListStages 查询路线下全部阶段
func ListStages(pathId int64) ([]model.PathStage, error) {
	list := make([]model.PathStage, 0)
	err := db.Get().Where("pathId = ?", pathId).
		Order("sort asc, id asc").Find(&list).Error
	return list, err
}

// FindStage 按主键查询阶段
func FindStage(id int64) (*model.PathStage, error) {
	var item model.PathStage
	if err := db.Get().Where("id = ?", id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// CreateStage 新建阶段
func CreateStage(item *model.PathStage) error {
	return db.Get().Create(item).Error
}

// UpdateStageFields 更新阶段指定字段
func UpdateStageFields(id int64, fields map[string]interface{}) error {
	return db.Get().Model(&model.PathStage{}).Where("id = ?", id).Updates(fields).Error
}

// DeleteStage 软删除阶段
func DeleteStage(id int64) error {
	return db.Get().Delete(&model.PathStage{}, id).Error
}

// DeleteStagesByPath 级联软删除路线下全部阶段
func DeleteStagesByPath(pathId int64) error {
	return db.Get().Where("pathId = ?", pathId).Delete(&model.PathStage{}).Error
}

// ListItems 查询路线下全部课程引用
func ListItems(pathId int64) ([]model.PathItem, error) {
	list := make([]model.PathItem, 0)
	err := db.Get().Where("pathId = ?", pathId).
		Order("sort asc, id asc").Find(&list).Error
	return list, err
}

// FindItem 按主键查询课程引用
func FindItem(id int64) (*model.PathItem, error) {
	var item model.PathItem
	if err := db.Get().Where("id = ?", id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// FindItemByCourse 查询路线内是否已引用该课程，用于去重
func FindItemByCourse(pathId, courseId int64) (*model.PathItem, error) {
	var item model.PathItem
	if err := db.Get().Where("pathId = ? AND courseId = ?", pathId, courseId).
		First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// CreateItem 新建课程引用
func CreateItem(item *model.PathItem) error {
	return db.Get().Create(item).Error
}

// UpdateItemFields 更新课程引用指定字段
func UpdateItemFields(id int64, fields map[string]interface{}) error {
	return db.Get().Model(&model.PathItem{}).Where("id = ?", id).Updates(fields).Error
}

// DeleteItem 软删除课程引用
func DeleteItem(id int64) error {
	return db.Get().Delete(&model.PathItem{}, id).Error
}

// DeleteItemsByStage 级联软删除阶段下全部课程引用
func DeleteItemsByStage(stageId int64) error {
	return db.Get().Where("stageId = ?", stageId).Delete(&model.PathItem{}).Error
}

// DeleteItemsByPath 级联软删除路线下全部课程引用
func DeleteItemsByPath(pathId int64) error {
	return db.Get().Where("pathId = ?", pathId).Delete(&model.PathItem{}).Error
}

// CountItemsByStage 统计阶段课程数，用于发布前校验
func CountItemsByStage(stageId int64) (int64, error) {
	var total int64
	err := db.Get().Model(&model.PathItem{}).Where("stageId = ?", stageId).Count(&total).Error
	return total, err
}

// CountItemsByPath 统计路线课程数
func CountItemsByPath(pathId int64) (int64, error) {
	var total int64
	err := db.Get().Model(&model.PathItem{}).Where("pathId = ?", pathId).Count(&total).Error
	return total, err
}

// PathRef 课程被路线引用的信息，用于课程删除前提示
type PathRef struct {
	PathId   int64  `gorm:"column:pathId"`
	PathName string `gorm:"column:pathName"`
}

// ListPathsByCourse 查询引用了指定课程的路线
func ListPathsByCourse(courseId int64) ([]PathRef, error) {
	rows := make([]PathRef, 0)
	err := db.Get().Table("path_item AS i").
		Select("DISTINCT i.pathId AS pathId, p.name AS pathName").
		Joins("LEFT JOIN learning_path p ON p.id = i.pathId AND p.deletedAt IS NULL").
		Where("i.courseId = ? AND i.deletedAt IS NULL", courseId).
		Scan(&rows).Error
	return rows, err
}

// 路线聚合统计行
type pathCountRow struct {
	PathId int64 `gorm:"column:pathId"`
	Total  int64 `gorm:"column:total"`
}

// CountStagesGroupByPath 一次性统计多条路线的阶段数
func CountStagesGroupByPath(pathIds []int64) (map[int64]int64, error) {
	out := make(map[int64]int64, len(pathIds))
	if len(pathIds) == 0 {
		return out, nil
	}
	rows := make([]pathCountRow, 0)
	err := db.Get().Model(&model.PathStage{}).
		Select("pathId, COUNT(*) AS total").
		Where("pathId IN ?", pathIds).
		Group("pathId").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.PathId] = row.Total
	}
	return out, nil
}

// CountItemsGroupByPath 一次性统计多条路线的课程数
func CountItemsGroupByPath(pathIds []int64) (map[int64]int64, error) {
	out := make(map[int64]int64, len(pathIds))
	if len(pathIds) == 0 {
		return out, nil
	}
	rows := make([]pathCountRow, 0)
	err := db.Get().Model(&model.PathItem{}).
		Select("pathId, COUNT(*) AS total").
		Where("pathId IN ?", pathIds).
		Group("pathId").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.PathId] = row.Total
	}
	return out, nil
}

// 路线状态计数行
type pathStatusCount struct {
	Status string `gorm:"column:status"`
	Total  int64  `gorm:"column:total"`
}

// CountPathsByStatus 统计各状态的学习路线数量
func CountPathsByStatus() (map[string]int64, error) {
	rows := make([]pathStatusCount, 0)
	err := db.Get().Model(&model.LearningPath{}).
		Select("status, COUNT(*) AS total").
		Group("status").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := map[string]int64{
		model.ContentDraft:     0,
		model.ContentPending:   0,
		model.ContentPublished: 0,
		model.ContentOffline:   0,
	}
	for _, row := range rows {
		out[row.Status] = row.Total
	}
	return out, nil
}
