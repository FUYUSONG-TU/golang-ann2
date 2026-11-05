package model

import (
	"time"

	"gorm.io/gorm"
)

// 资料解析状态。解析由后台在保存或手动触发时执行，
// 后续接入真正的文档解析服务后，这里的状态流转不需要改表结构。
const (
	ParsePending = "pending" // 待解析
	ParseRunning = "parsing" // 解析中
	ParseSuccess = "success" // 解析成功
	ParseFailed  = "failed"  // 解析失败
)

// 版权归属，AI 检索需要按版权做过滤，来源不明的资料不允许发布
const (
	CopyrightOwn        = "own"        // 自有
	CopyrightAuthorized = "authorized" // 已授权
	CopyrightPublic     = "public"     // 公开资料
	CopyrightUnknown    = "unknown"    // 待确认
)

// KbDocument AI 知识库资料。一条记录对应一份可被检索的资料，
// 正文以纯文本保存，分块结果后续写入 kb_chunk。
type KbDocument struct {
	Id          int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Title       string         `gorm:"column:title;size:128;index:idx_kb_title" json:"title"`
	Category    string         `gorm:"column:category;size:32;index:idx_kb_category" json:"category"`
	Source      string         `gorm:"column:source;size:64" json:"source"`
	SourceUrl   string         `gorm:"column:sourceUrl;size:255" json:"sourceUrl"`
	Copyright   string         `gorm:"column:copyright;size:16;default:unknown" json:"copyright"`
	Tags        string         `gorm:"column:tags;size:255" json:"tags"`
	FileName    string         `gorm:"column:fileName;size:128" json:"fileName"`
	FileKey     string         `gorm:"column:fileKey;size:255" json:"fileKey"`
	FileSize    int64          `gorm:"column:fileSize;default:0" json:"fileSize"`
	MimeType    string         `gorm:"column:mimeType;size:64" json:"mimeType"`
	Content     string         `gorm:"column:content;type:longtext" json:"content"`
	CharCount   int            `gorm:"column:charCount;default:0" json:"charCount"`
	ChunkCount  int            `gorm:"column:chunkCount;default:0" json:"chunkCount"`
	ParseStatus string         `gorm:"column:parseStatus;size:16;default:pending;index:idx_kb_parse" json:"parseStatus"`
	ParseError  string         `gorm:"column:parseError;size:255" json:"parseError"`
	ParsedAt    *time.Time     `gorm:"column:parsedAt" json:"parsedAt"`
	Status      string         `gorm:"column:status;size:16;default:draft;index:idx_kb_status" json:"status"`
	Sort        int            `gorm:"column:sort;default:0" json:"sort"`
	CreatedAt   time.Time      `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt   time.Time      `gorm:"column:updatedAt" json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deletedAt;index:idx_kb_deleted" json:"-"`
}

// TableName 指定表名
func (KbDocument) TableName() string { return "kb_document" }

// KbChunk 资料分块，是 AI 检索的最小单位。
// 一份资料解析后会被切成若干块，docTitle 与 category 冗余存储，
// 这样检索命中后不需要再回表就能展示出处。
type KbChunk struct {
	Id        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	DocId     int64     `gorm:"column:docId;index:idx_chunk_doc" json:"docId"`
	DocTitle  string    `gorm:"column:docTitle;size:128" json:"docTitle"`
	Category  string    `gorm:"column:category;size:32" json:"category"`
	Seq       int       `gorm:"column:seq;default:0" json:"seq"`
	Content   string    `gorm:"column:content;type:text" json:"content"`
	CharCount int       `gorm:"column:charCount;default:0" json:"charCount"`
	CreatedAt time.Time `gorm:"column:createdAt" json:"createdAt"`
}

// TableName 指定表名
func (KbChunk) TableName() string { return "kb_chunk" }

// 问答库状态。问答对是「学员真实提问」沉淀下来的资产，
// 只有定稿的才允许直接命中，避免模型的错误答案被反复引用。
const (
	QaCandidate = "candidate" // 待审核（AI 生成后自动入池）
	QaPublished = "published" // 已定稿，可被直接命中
	QaOffline   = "offline"   // 已下架，暂停命中
	QaMerged    = "merged"    // 已并入另一条问答对
	QaRejected  = "rejected"  // 已驳回，不再展示
)

// 问答对来源
const (
	QaSourceAi     = "ai"     // 学员提问后由模型作答沉淀
	QaSourceManual = "manual" // 后台手工录入
)

// KbQa 问答对，知识库的第二层结构。
// 文档块解决「资料里写了什么」，问答对解决「这个问题该怎么答」——
// 前者是原文，后者是已经被验证过、可以直接复用的结论。
// terms 缓存问句切好的检索词元，避免每次匹配都重新切词。
type KbQa struct {
	Id        int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Question  string         `gorm:"column:question;size:255;index:idx_qa_question" json:"question"`
	Answer    string         `gorm:"column:answer;type:longtext" json:"answer"`
	Aliases   string         `gorm:"column:aliases;type:text" json:"aliases"`
	Terms     string         `gorm:"column:terms;size:512" json:"-"`
	Category  string         `gorm:"column:category;size:32;index:idx_qa_category" json:"category"`
	Source    string         `gorm:"column:source;size:16;default:ai" json:"source"`
	Refs      string         `gorm:"column:refs;type:text" json:"refs"`
	AskCount  int            `gorm:"column:askCount;default:0" json:"askCount"`
	HitCount  int            `gorm:"column:hitCount;default:0" json:"hitCount"`
	UpCount   int            `gorm:"column:upCount;default:0" json:"upCount"`
	DownCount int            `gorm:"column:downCount;default:0" json:"downCount"`
	Quality   float64        `gorm:"column:quality;default:0" json:"quality"`
	Status    string         `gorm:"column:status;size:16;default:candidate;index:idx_qa_status" json:"status"`
	MergedTo  int64          `gorm:"column:mergedTo;default:0" json:"mergedTo"`
	Remark    string         `gorm:"column:remark;size:255" json:"remark"`
	CreatedAt time.Time      `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"column:updatedAt" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"column:deletedAt;index:idx_qa_deleted" json:"-"`
}

// TableName 指定表名
func (KbQa) TableName() string { return "kb_qa" }
