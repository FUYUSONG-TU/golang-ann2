package model

import (
	"time"

	"gorm.io/gorm"
)

// LearningPath 学习路线，如「走心机入门 30 天」
// 路线不拥有课程，只通过 path_item 引用，因此同一门课可出现在多条路线里
type LearningPath struct {
	Id         int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Code       string         `gorm:"column:code;size:32;index:idx_path_code" json:"code"`
	Name       string         `gorm:"column:name;size:64" json:"name"`
	Cover      string         `gorm:"column:cover;size:255" json:"cover"`
	Summary    string         `gorm:"column:summary;size:500" json:"summary"`
	MajorId    int64          `gorm:"column:majorId;index:idx_path_major;default:0" json:"majorId"` // 0 表示通用路线
	Difficulty int            `gorm:"column:difficulty;default:1" json:"difficulty"`
	Hours      int            `gorm:"column:hours;default:0" json:"hours"` // 预计学时，0 表示按课程累加
	Status     string         `gorm:"column:status;size:16;default:draft" json:"status"`
	Sort       int            `gorm:"column:sort;default:0" json:"sort"`
	CreatedAt  time.Time      `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt  time.Time      `gorm:"column:updatedAt" json:"updatedAt"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deletedAt;index:idx_path_deleted" json:"-"`
}

// TableName 指定表名
func (LearningPath) TableName() string { return "learning_path" }

// PathStage 路线阶段，如「第一阶段 认识设备」
type PathStage struct {
	Id        int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	PathId    int64          `gorm:"column:pathId;index:idx_stage_path" json:"pathId"`
	Title     string         `gorm:"column:title;size:128" json:"title"`
	Summary   string         `gorm:"column:summary;size:500" json:"summary"`
	Sort      int            `gorm:"column:sort;default:0" json:"sort"`
	CreatedAt time.Time      `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"column:updatedAt" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"column:deletedAt;index:idx_stage_deleted" json:"-"`
}

// TableName 指定表名
func (PathStage) TableName() string { return "path_stage" }

// PathItem 阶段内的课程引用
// UnlockRule 为解锁条件预留字段（当前仅存文本，不做规则引擎），
// Required 标记是否必学，选修课不影响路线完成判定
type PathItem struct {
	Id         int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	PathId     int64          `gorm:"column:pathId;index:idx_item_path" json:"pathId"`
	StageId    int64          `gorm:"column:stageId;index:idx_item_stage" json:"stageId"`
	CourseId   int64          `gorm:"column:courseId;index:idx_item_course" json:"courseId"`
	Sort       int            `gorm:"column:sort;default:0" json:"sort"`
	Required   bool           `gorm:"column:required;default:1" json:"required"`
	UnlockRule string         `gorm:"column:unlockRule;size:255" json:"unlockRule"`
	CreatedAt  time.Time      `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt  time.Time      `gorm:"column:updatedAt" json:"updatedAt"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deletedAt;index:idx_item_deleted" json:"-"`
}

// TableName 指定表名
func (PathItem) TableName() string { return "path_item" }
