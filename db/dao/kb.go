package dao

import (
	"wxcloudrun-golang/db"
	"wxcloudrun-golang/db/model"
)

// KbQuery 知识库资料查询条件
type KbQuery struct {
	Keyword     string
	Category    string
	Copyright   string
	ParseStatus string
	Status      string
	Page        int
	Size        int
}

// ListKbDocuments 分页查询资料，列表不返回正文，避免一次拉回几十万字
func ListKbDocuments(q KbQuery) ([]model.KbDocument, int64, error) {
	list := make([]model.KbDocument, 0)
	query := db.Get().Model(&model.KbDocument{})
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		query = query.Where("title LIKE ? OR source LIKE ? OR tags LIKE ?", kw, kw, kw)
	}
	if q.Category != "" {
		query = query.Where("category = ?", q.Category)
	}
	if q.Copyright != "" {
		query = query.Where("copyright = ?", q.Copyright)
	}
	if q.ParseStatus != "" {
		query = query.Where("parseStatus = ?", q.ParseStatus)
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
	err := query.Omit("content").Order("sort asc, id desc").
		Offset((q.Page - 1) * q.Size).Limit(q.Size).Find(&list).Error
	return list, total, err
}

// FindKbDocument 按主键查询资料，包含正文
func FindKbDocument(id int64) (*model.KbDocument, error) {
	var item model.KbDocument
	if err := db.Get().Where("id = ?", id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// CreateKbDocument 新建资料
func CreateKbDocument(item *model.KbDocument) error {
	return db.Get().Create(item).Error
}

// UpdateKbDocumentFields 更新资料指定字段
func UpdateKbDocumentFields(id int64, fields map[string]interface{}) error {
	return db.Get().Model(&model.KbDocument{}).Where("id = ?", id).Updates(fields).Error
}

// DeleteKbDocument 软删除资料
func DeleteKbDocument(id int64) error {
	return db.Get().Delete(&model.KbDocument{}, id).Error
}

// 资料状态计数行
type kbCountRow struct {
	Key   string `gorm:"column:k"`
	Total int64  `gorm:"column:total"`
}

// CountKbByParseStatus 统计各解析状态的资料数量
func CountKbByParseStatus() (map[string]int64, error) {
	rows := make([]kbCountRow, 0)
	err := db.Get().Model(&model.KbDocument{}).
		Select("parseStatus AS k, COUNT(*) AS total").
		Group("parseStatus").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := map[string]int64{
		model.ParsePending: 0,
		model.ParseRunning: 0,
		model.ParseSuccess: 0,
		model.ParseFailed:  0,
	}
	for _, row := range rows {
		out[row.Key] = row.Total
	}
	return out, nil
}

// CountKbByStatus 统计各发布状态的资料数量
func CountKbByStatus() (map[string]int64, error) {
	rows := make([]kbCountRow, 0)
	err := db.Get().Model(&model.KbDocument{}).
		Select("status AS k, COUNT(*) AS total").
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
		out[row.Key] = row.Total
	}
	return out, nil
}

// KbTotals 资料规模汇总
type KbTotals struct {
	Chars  int64 `gorm:"column:chars" json:"chars"`
	Chunks int64 `gorm:"column:chunks" json:"chunks"`
	Files  int64 `gorm:"column:files" json:"files"`
}

// SumKbTotals 汇总全库资料字数、分块数与文件数
func SumKbTotals() (KbTotals, error) {
	var row KbTotals
	err := db.Get().Model(&model.KbDocument{}).
		Select("IFNULL(SUM(charCount), 0) AS chars, IFNULL(SUM(chunkCount), 0) AS chunks, " +
			"SUM(CASE WHEN fileKey <> '' THEN 1 ELSE 0 END) AS files").
		Scan(&row).Error
	return row, err
}

// ListKbCategories 已使用过的资料分类，供筛选下拉复用
func ListKbCategories() ([]string, error) {
	rows := make([]struct {
		Category string `gorm:"column:category"`
	}, 0)
	err := db.Get().Model(&model.KbDocument{}).
		Select("DISTINCT category").
		Where("category <> ''").
		Order("category asc").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.Category)
	}
	return out, nil
}
