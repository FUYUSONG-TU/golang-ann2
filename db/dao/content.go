package dao

import (
	"wxcloudrun-golang/db"
	"wxcloudrun-golang/db/model"
)

// MajorQuery 专业查询条件
type MajorQuery struct {
	Keyword string
	Status  int // -1 表示不限
}

// ListMajors 查询专业列表
func ListMajors(q MajorQuery) ([]model.Major, error) {
	list := make([]model.Major, 0)
	query := db.Get().Model(&model.Major{})
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		query = query.Where("name LIKE ? OR code LIKE ?", kw, kw)
	}
	if q.Status >= 0 {
		query = query.Where("status = ?", q.Status)
	}
	err := query.Order("sort asc, id asc").Find(&list).Error
	return list, err
}

// FindMajor 按主键查询专业
func FindMajor(id int64) (*model.Major, error) {
	var item model.Major
	if err := db.Get().Where("id = ?", id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// FindMajorByCode 按编码查询专业，用于唯一性校验
func FindMajorByCode(code string) (*model.Major, error) {
	var item model.Major
	if err := db.Get().Where("code = ?", code).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// CreateMajor 新建专业
func CreateMajor(item *model.Major) error {
	return db.Get().Create(item).Error
}

// UpdateMajorFields 更新专业指定字段
func UpdateMajorFields(id int64, fields map[string]interface{}) error {
	return db.Get().Model(&model.Major{}).Where("id = ?", id).Updates(fields).Error
}

// DeleteMajor 软删除专业
func DeleteMajor(id int64) error {
	return db.Get().Delete(&model.Major{}, id).Error
}

// CountCoursesByMajor 统计专业下的课程数，用于删除前保护
func CountCoursesByMajor(majorId int64) (int64, error) {
	var total int64
	err := db.Get().Model(&model.Course{}).Where("majorId = ?", majorId).Count(&total).Error
	return total, err
}

// 专业课程数聚合行
type majorCourseCount struct {
	MajorId int64 `gorm:"column:majorId"`
	Total   int64 `gorm:"column:total"`
}

// CountCoursesGroupByMajor 一次性统计各专业的课程数
func CountCoursesGroupByMajor() (map[int64]int64, error) {
	rows := make([]majorCourseCount, 0)
	err := db.Get().Model(&model.Course{}).
		Select("majorId, COUNT(*) AS total").
		Group("majorId").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[int64]int64, len(rows))
	for _, row := range rows {
		out[row.MajorId] = row.Total
	}
	return out, nil
}

// CourseQuery 课程查询条件
type CourseQuery struct {
	MajorId    int64
	Keyword    string
	Status     string
	Difficulty int
	Page       int
	Size       int
}

// ListCourses 分页查询课程
func ListCourses(q CourseQuery) ([]model.Course, int64, error) {
	list := make([]model.Course, 0)
	query := db.Get().Model(&model.Course{})
	if q.MajorId > 0 {
		query = query.Where("majorId = ?", q.MajorId)
	}
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		query = query.Where("name LIKE ? OR summary LIKE ?", kw, kw)
	}
	if q.Status != "" {
		query = query.Where("status = ?", q.Status)
	}
	if q.Difficulty > 0 {
		query = query.Where("difficulty = ?", q.Difficulty)
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

// FindCourse 按主键查询课程
func FindCourse(id int64) (*model.Course, error) {
	var item model.Course
	if err := db.Get().Where("id = ?", id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// CreateCourse 新建课程
func CreateCourse(item *model.Course) error {
	return db.Get().Create(item).Error
}

// UpdateCourseFields 更新课程指定字段
func UpdateCourseFields(id int64, fields map[string]interface{}) error {
	return db.Get().Model(&model.Course{}).Where("id = ?", id).Updates(fields).Error
}

// DeleteCourse 软删除课程
func DeleteCourse(id int64) error {
	return db.Get().Delete(&model.Course{}, id).Error
}

// ListChapters 查询课程下全部章节，按层级与排序返回
func ListChapters(courseId int64) ([]model.Chapter, error) {
	list := make([]model.Chapter, 0)
	err := db.Get().Where("courseId = ?", courseId).
		Order("parentId asc, sort asc, id asc").Find(&list).Error
	return list, err
}

// FindChapter 按主键查询章节
func FindChapter(id int64) (*model.Chapter, error) {
	var item model.Chapter
	if err := db.Get().Where("id = ?", id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// CreateChapter 新建章节
func CreateChapter(item *model.Chapter) error {
	return db.Get().Create(item).Error
}

// UpdateChapterFields 更新章节指定字段
func UpdateChapterFields(id int64, fields map[string]interface{}) error {
	return db.Get().Model(&model.Chapter{}).Where("id = ?", id).Updates(fields).Error
}

// DeleteChapter 软删除单个章节
func DeleteChapter(id int64) error {
	return db.Get().Delete(&model.Chapter{}, id).Error
}

// DeleteChaptersByCourse 级联软删除课程下全部章节
func DeleteChaptersByCourse(courseId int64) error {
	return db.Get().Where("courseId = ?", courseId).Delete(&model.Chapter{}).Error
}

// DeleteChapterWithChildren 删除章节及其下级知识点
func DeleteChapterWithChildren(id int64) error {
	return db.Get().Where("id = ? OR parentId = ?", id, id).Delete(&model.Chapter{}).Error
}

// CountChaptersByCourse 统计课程章节数
func CountChaptersByCourse(courseId int64) (int64, error) {
	var total int64
	err := db.Get().Model(&model.Chapter{}).Where("courseId = ?", courseId).Count(&total).Error
	return total, err
}

// 课程章节数聚合行
type courseChapterCount struct {
	CourseId int64 `gorm:"column:courseId"`
	Total    int64 `gorm:"column:total"`
}

// CountChaptersGroupByCourse 一次性统计多条课程的章节数
func CountChaptersGroupByCourse(courseIds []int64) (map[int64]int64, error) {
	out := make(map[int64]int64, len(courseIds))
	if len(courseIds) == 0 {
		return out, nil
	}
	rows := make([]courseChapterCount, 0)
	err := db.Get().Model(&model.Chapter{}).
		Select("courseId, COUNT(*) AS total").
		Where("courseId IN ?", courseIds).
		Group("courseId").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.CourseId] = row.Total
	}
	return out, nil
}

// BatchUpdateSort 批量更新排序号，表名为 major / course / chapter
func BatchUpdateSort(table string, ids []int64) error {
	for index, id := range ids {
		if err := db.Get().Table(table).Where("id = ?", id).
			Update("sort", index+1).Error; err != nil {
			return err
		}
	}
	return nil
}

// NextSort 取表内最大排序号加一，新建记录默认排在末尾
func NextSort(table string) int {
	row := db.Get().Table(table).Select("IFNULL(MAX(sort), 0)").Row()
	if row == nil {
		return 1
	}
	var maxSort int
	if err := row.Scan(&maxSort); err != nil {
		return 1
	}
	return maxSort + 1
}

// 课程状态计数行
type courseStatusCount struct {
	Status string `gorm:"column:status"`
	Total  int64  `gorm:"column:total"`
}

// CountCoursesByStatus 统计各状态的课程数量，用于列表页概览
func CountCoursesByStatus() (map[string]int64, error) {
	rows := make([]courseStatusCount, 0)
	err := db.Get().Model(&model.Course{}).
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
