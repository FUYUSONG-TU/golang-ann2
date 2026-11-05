package model

import (
	"time"

	"gorm.io/gorm"
)

// 内容发布状态，全站内容模块共用同一套取值，避免各模块自定义
const (
	ContentDraft     = "draft"     // 草稿
	ContentPending   = "pending"   // 审核中
	ContentPublished = "published" // 已发布
	ContentOffline   = "offline"   // 已下架
)

// Major 专业分类，如走心机、数控车床、加工中心
type Major struct {
	Id        int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Code      string         `gorm:"column:code;size:32;index:idx_major_code" json:"code"`
	Name      string         `gorm:"column:name;size:64" json:"name"`
	Icon      string         `gorm:"column:icon;size:32" json:"icon"`
	Summary   string         `gorm:"column:summary;size:255" json:"summary"`
	Sort      int            `gorm:"column:sort;default:0" json:"sort"`
	Status    int            `gorm:"column:status;default:1" json:"status"`
	CreatedAt time.Time      `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"column:updatedAt" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"column:deletedAt;index:idx_major_deleted" json:"-"`
}

// TableName 指定表名
func (Major) TableName() string { return "major" }

// Course 课程，隶属于专业，可被多条学习路线引用
type Course struct {
	Id         int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	MajorId    int64          `gorm:"column:majorId;index:idx_course_major" json:"majorId"`
	Name       string         `gorm:"column:name;size:128" json:"name"`
	Cover      string         `gorm:"column:cover;size:255" json:"cover"`
	Difficulty int            `gorm:"column:difficulty;default:1" json:"difficulty"`
	Summary    string         `gorm:"column:summary;size:500" json:"summary"`
	Hours      int            `gorm:"column:hours;default:0" json:"hours"`
	Status     string         `gorm:"column:status;size:16;default:draft" json:"status"`
	Sort       int            `gorm:"column:sort;default:0" json:"sort"`
	CreatedAt  time.Time      `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt  time.Time      `gorm:"column:updatedAt" json:"updatedAt"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deletedAt;index:idx_course_deleted" json:"-"`
}

// TableName 指定表名
func (Course) TableName() string { return "course" }

// Chapter 章节，支持两级：parentId 为 0 表示章，大于 0 表示该章下的知识点
type Chapter struct {
	Id        int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	CourseId  int64          `gorm:"column:courseId;index:idx_chapter_course" json:"courseId"`
	ParentId  int64          `gorm:"column:parentId;index:idx_chapter_parent" json:"parentId"`
	Title     string         `gorm:"column:title;size:128" json:"title"`
	Summary   string         `gorm:"column:summary;size:500" json:"summary"`
	Duration  int            `gorm:"column:duration;default:0" json:"duration"`
	Sort      int            `gorm:"column:sort;default:0" json:"sort"`
	Status    string         `gorm:"column:status;size:16;default:published" json:"status"`
	CreatedAt time.Time      `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"column:updatedAt" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"column:deletedAt;index:idx_chapter_deleted" json:"-"`
}

// TableName 指定表名
func (Chapter) TableName() string { return "chapter" }
