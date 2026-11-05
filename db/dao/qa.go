package dao

import (
	"time"

	"gorm.io/gorm"

	"wxcloudrun-golang/db"
	"wxcloudrun-golang/db/model"
)

// QaQuery 问答对查询条件
type QaQuery struct {
	Keyword  string
	Status   string
	Category string
	Source   string
	Sort     string // heat 热度倒序；newest 最新；quality 质量分
	Page     int
	Size     int
}

// qaHeatExpr 热度表达式：被问到、被直接命中、被点赞都算贡献，差评直接扣。
// 排序放在 SQL 里做，避免把整表拉到内存再排。
const qaHeatExpr = "(askCount + hitCount * 2 + upCount * 3 - downCount * 4)"

// ListKbQa 分页查询问答对
func ListKbQa(q QaQuery) ([]model.KbQa, int64, error) {
	list := make([]model.KbQa, 0)
	query := db.Get().Model(&model.KbQa{})
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		query = query.Where("question LIKE ? OR answer LIKE ? OR aliases LIKE ?", kw, kw, kw)
	}
	if q.Status != "" {
		query = query.Where("status = ?", q.Status)
	}
	if q.Category != "" {
		query = query.Where("category = ?", q.Category)
	}
	if q.Source != "" {
		query = query.Where("source = ?", q.Source)
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

	switch q.Sort {
	case "newest":
		query = query.Order("id desc")
	case "quality":
		query = query.Order("quality desc, " + qaHeatExpr + " desc")
	default:
		query = query.Order(qaHeatExpr + " desc, id desc")
	}

	err := query.Offset((q.Page - 1) * q.Size).Limit(q.Size).Find(&list).Error
	return list, total, err
}

// FindKbQa 按主键查询问答对
func FindKbQa(id int64) (*model.KbQa, error) {
	var item model.KbQa
	if err := db.Get().Where("id = ?", id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// CreateKbQa 新建问答对
func CreateKbQa(item *model.KbQa) error {
	return db.Get().Create(item).Error
}

// UpdateKbQaFields 更新问答对指定字段
func UpdateKbQaFields(id int64, fields map[string]interface{}) error {
	fields["updatedAt"] = time.Now()
	return db.Get().Model(&model.KbQa{}).Where("id = ?", id).Updates(fields).Error
}

// DeleteKbQa 软删除问答对
func DeleteKbQa(id int64) error {
	return db.Get().Delete(&model.KbQa{}, id).Error
}

// CountKbQaByStatus 统计各状态的问答对数量
func CountKbQaByStatus() (map[string]int64, error) {
	rows := make([]kbCountRow, 0)
	err := db.Get().Model(&model.KbQa{}).
		Select("status AS k, COUNT(*) AS total").
		Group("status").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := map[string]int64{
		model.QaCandidate: 0,
		model.QaPublished: 0,
		model.QaOffline:   0,
		model.QaMerged:    0,
		model.QaRejected:  0,
	}
	for _, row := range rows {
		out[row.Key] = row.Total
	}
	return out, nil
}

// ListKbQaForMatch 取出参与匹配的问答对。
// 只匹配「已定稿」与「待审核」两种：前者用于直接出答案，后者用于把重复提问归并到同一条，
// 已下架、已合并、已驳回的不参与，避免脏数据污染去重结果。
func ListKbQaForMatch(limit int) ([]model.KbQa, error) {
	list := make([]model.KbQa, 0)
	if limit <= 0 || limit > 20000 {
		limit = 8000
	}
	err := db.Get().
		Select("id, question, answer, aliases, terms, category, status, refs, askCount, hitCount, upCount, downCount").
		Where("status IN ?", []string{model.QaPublished, model.QaCandidate}).
		Order(qaHeatExpr + " desc").
		Limit(limit).Find(&list).Error
	return list, err
}

// 允许自增的计数字段白名单，字段名直接拼进 SQL，必须严格限定
var qaCounterFields = map[string]bool{
	"askCount":  true,
	"hitCount":  true,
	"upCount":   true,
	"downCount": true,
}

// IncrKbQaCounter 计数字段自增，单条 SQL 完成，避免并发下读改写丢计数
func IncrKbQaCounter(id int64, field string, delta int) error {
	if !qaCounterFields[field] {
		return gorm.ErrInvalidField
	}
	return db.Get().Model(&model.KbQa{}).Where("id = ?", id).
		UpdateColumn(field, gorm.Expr(field+" + ?", delta)).Error
}

// QaTotals 问答库规模汇总
type QaTotals struct {
	Total     int64 `gorm:"column:total" json:"total"`
	Hits      int64 `gorm:"column:hits" json:"hits"`
	Asks      int64 `gorm:"column:asks" json:"asks"`
	Published int64 `gorm:"column:published" json:"published"`
}

// SumKbQaTotals 汇总问答库总量、累计命中与提问次数。
// asks 统计「被问到」的总次数，既包含需要模型现答的（askCount），
// 也包含直接命中已有答案的（hitCount）——对学员来说这两者都是提问。
func SumKbQaTotals() (QaTotals, error) {
	var row QaTotals
	err := db.Get().Model(&model.KbQa{}).
		Select("COUNT(*) AS total, " +
			"IFNULL(SUM(hitCount), 0) AS hits, " +
			"(IFNULL(SUM(askCount), 0) + IFNULL(SUM(hitCount), 0)) AS asks, " +
			"SUM(CASE WHEN status = 'published' THEN 1 ELSE 0 END) AS published").
		Scan(&row).Error
	return row, err
}

// ListKbQaCategories 已使用的问答分类
func ListKbQaCategories() ([]string, error) {
	rows := make([]struct {
		Category string `gorm:"column:category"`
	}, 0)
	err := db.Get().Model(&model.KbQa{}).
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
