package dao

import (
	"strings"

	"gorm.io/gorm"

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

// ReplaceKbChunks 用新分块整体替换旧分块。
// 分块是资料的派生数据，重建比逐条比对简单可靠，因此在事务里先删后插。
func ReplaceKbChunks(docId int64, chunks []model.KbChunk) error {
	return db.Get().Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("docId = ?", docId).Delete(&model.KbChunk{}).Error; err != nil {
			return err
		}
		if len(chunks) == 0 {
			return nil
		}
		// 分批写入，避免一次插入上千行超出单条 SQL 的长度限制
		for start := 0; start < len(chunks); start += 100 {
			end := start + 100
			if end > len(chunks) {
				end = len(chunks)
			}
			if err := tx.Create(chunks[start:end]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// DeleteKbChunksByDoc 删除某份资料的全部分块
func DeleteKbChunksByDoc(docId int64) error {
	return db.Get().Where("docId = ?", docId).Delete(&model.KbChunk{}).Error
}

// ListKbChunks 查询某份资料的分块，按序号升序
func ListKbChunks(docId int64) ([]model.KbChunk, error) {
	list := make([]model.KbChunk, 0)
	err := db.Get().Where("docId = ?", docId).Order("seq asc").Find(&list).Error
	return list, err
}

// CountKbChunks 统计全部分块数量
func CountKbChunks() (int64, error) {
	var total int64
	err := db.Get().Model(&model.KbChunk{}).Count(&total).Error
	return total, err
}

// SearchKbChunks 按关键词粗召回。
// 只检索「已发布且解析成功」的资料，避免草稿或解析失败的内容被 AI 引用；
// 关键词之间是 OR 关系，扩大召回范围，精排交给上层打分。
func SearchKbChunks(terms []string, limit int) ([]model.KbChunk, error) {
	list := make([]model.KbChunk, 0)
	query := db.Get().Table("kb_chunk AS c").
		Select("c.*").
		Joins("JOIN kb_document AS d ON d.id = c.docId").
		Where("d.status = ? AND d.parseStatus = ? AND d.deletedAt IS NULL",
			model.ContentPublished, model.ParseSuccess)

	if len(terms) > 0 {
		conds := make([]string, 0, len(terms))
		args := make([]interface{}, 0, len(terms))
		for _, term := range terms {
			conds = append(conds, "c.content LIKE ?")
			args = append(args, "%"+term+"%")
		}
		query = query.Where(strings.Join(conds, " OR "), args...)
	}

	if limit <= 0 || limit > 2000 {
		limit = 800
	}
	err := query.Order("c.docId asc, c.seq asc").Limit(limit).Find(&list).Error
	return list, err
}

// CountSearchableDocs 统计可被检索的资料数（已发布且解析成功）
func CountSearchableDocs() (int64, error) {
	var total int64
	err := db.Get().Model(&model.KbDocument{}).
		Where("status = ? AND parseStatus = ?", model.ContentPublished, model.ParseSuccess).
		Count(&total).Error
	return total, err
}
