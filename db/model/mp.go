package model

import (
	"time"

	"gorm.io/gorm"
)

// 学员账号状态
const (
	MpUserDisabled = 0 // 已禁用
	MpUserNormal   = 1 // 正常
)

// 学习记录状态
const (
	LearnLearning = "learning" // 学习中
	LearnFinished = "finished" // 已学完
)

// MpUser 小程序学员。数据由小程序端上报，后台只做查看与有限维护。
type MpUser struct {
	Id           int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OpenId       string         `gorm:"column:openId;size:64;uniqueIndex:uk_mp_openid" json:"openId"`
	UnionId      string         `gorm:"column:unionId;size:64" json:"unionId"`
	Nickname     string         `gorm:"column:nickname;size:64" json:"nickname"`
	Avatar       string         `gorm:"column:avatar;size:255" json:"avatar"`
	Phone        string         `gorm:"column:phone;size:20" json:"phone"`
	MajorId      int64          `gorm:"column:majorId;index:idx_mp_major;default:0" json:"majorId"`
	Level        int            `gorm:"column:level;default:1;index:idx_mp_level" json:"level"`
	Status       int            `gorm:"column:status;default:1" json:"status"`
	Remark       string         `gorm:"column:remark;size:255" json:"remark"`
	LoginCount   int            `gorm:"column:loginCount;default:0" json:"loginCount"`
	LastActiveAt *time.Time     `gorm:"column:lastActiveAt" json:"lastActiveAt"`
	CreatedAt    time.Time      `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt    time.Time      `gorm:"column:updatedAt" json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deletedAt;index:idx_mp_deleted" json:"-"`
}

// TableName 指定表名
func (MpUser) TableName() string { return "mp_user" }

// MpLearnRecord 学习记录，一条记录对应「某学员在某门课上的进度」。
// 同一学员同一门课只保留一条，学完后再学只更新最后学习时间。
type MpLearnRecord struct {
	Id           int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserId       int64          `gorm:"column:userId;index:idx_record_user" json:"userId"`
	CourseId     int64          `gorm:"column:courseId;index:idx_record_course" json:"courseId"`
	ChapterId    int64          `gorm:"column:chapterId;default:0" json:"chapterId"` // 最近学到的章节，0 表示未细分
	Progress     int            `gorm:"column:progress;default:0" json:"progress"`   // 0-100
	StudySeconds int            `gorm:"column:studySeconds;default:0" json:"studySeconds"`
	Status       string         `gorm:"column:status;size:16;default:learning" json:"status"`
	LastStudyAt  *time.Time     `gorm:"column:lastStudyAt;index:idx_record_time" json:"lastStudyAt"`
	CreatedAt    time.Time      `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt    time.Time      `gorm:"column:updatedAt" json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deletedAt;index:idx_record_deleted" json:"-"`
}

// TableName 指定表名
func (MpLearnRecord) TableName() string { return "mp_learn_record" }
