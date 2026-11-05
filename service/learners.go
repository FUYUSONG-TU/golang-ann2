package service

import (
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"wxcloudrun-golang/db/dao"
	"wxcloudrun-golang/db/model"
)

// 学员水平文案，与课程难度共用一套取值
var learnerLevelText = map[int]string{
	DifficultyBasic:    "入门",
	DifficultyAdvanced: "进阶",
	DifficultyExpert:   "高级",
}

// learnerItem 学员列表视图，附带学习汇总
type learnerItem struct {
	model.MpUser
	MajorName     string     `json:"majorName"`
	LevelText     string     `json:"levelText"`
	CourseCount   int64      `json:"courseCount"`
	FinishedCount int64      `json:"finishedCount"`
	AvgProgress   int        `json:"avgProgress"`
	StudyHours    float64    `json:"studyHours"`
	LastStudyAt   *time.Time `json:"lastStudyAt"`
	Demo          bool       `json:"demo"`
}

// learnRecordView 学习记录视图
type learnRecordView struct {
	Id           int64      `json:"id"`
	CourseId     int64      `json:"courseId"`
	CourseName   string     `json:"courseName"`
	ChapterId    int64      `json:"chapterId"`
	ChapterTitle string     `json:"chapterTitle"`
	Progress     int        `json:"progress"`
	StudySeconds int        `json:"studySeconds"`
	Status       string     `json:"status"`
	StatusText   string     `json:"statusText"`
	LastStudyAt  *time.Time `json:"lastStudyAt"`
}

// mpUserList 学员列表，支持按专业 / 水平 / 状态 / 关键词筛选
func mpUserList(c *adminCtx, r *http.Request) (interface{}, error) {
	page, size := pageParams(r)
	status := queryIntAll(r, "status", -1)
	list, total, err := dao.ListMpUsers(dao.MpUserQuery{
		Keyword: strings.TrimSpace(r.URL.Query().Get("keyword")),
		MajorId: queryInt64(r, "majorId", 0),
		Level:   queryInt(r, "level", 0),
		Status:  status,
		Page:    page,
		Size:    size,
	})
	if err != nil {
		return nil, serverError("查询学员失败: " + err.Error())
	}

	ids := make([]int64, 0, len(list))
	for _, item := range list {
		ids = append(ids, item.Id)
	}
	aggs, _ := dao.LearnAggByUsers(ids)

	majorNames := map[int64]string{}
	for _, item := range majorOptions() {
		if id, ok := item["id"].(int64); ok {
			if name, ok := item["name"].(string); ok {
				majorNames[id] = name
			}
		}
	}

	items := make([]learnerItem, 0, len(list))
	for _, item := range list {
		agg := aggs[item.Id]
		row := learnerItem{
			MpUser:        item,
			MajorName:     majorNames[item.MajorId],
			LevelText:     learnerLevelText[item.Level],
			CourseCount:   agg.Total,
			FinishedCount: agg.Finished,
			AvgProgress:   int(math.Round(agg.AvgProgress)),
			StudyHours:    math.Round(float64(agg.StudySeconds)/3600*10) / 10,
			LastStudyAt:   agg.LastStudyAt,
			Demo:          strings.HasPrefix(item.OpenId, "demo-"),
		}
		if row.MajorName == "" {
			row.MajorName = "未选择专业"
		}
		if row.LevelText == "" {
			row.LevelText = "入门"
		}
		items = append(items, row)
	}

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	normalCount, _ := dao.CountMpUsers(model.MpUserNormal)
	disabledCount, _ := dao.CountMpUsers(model.MpUserDisabled)
	newToday, _ := dao.CountMpUsersCreatedSince(todayStart)
	activeToday, _ := dao.CountMpUsersActiveSince(todayStart)
	levelCounts, _ := dao.CountMpUsersByLevel()
	demoCount, _ := dao.CountDemoMpUsers()

	return map[string]interface{}{
		"list":          items,
		"total":         total,
		"page":          page,
		"size":          size,
		"majors":        majorOptions(),
		"levelCounts":   levelCounts,
		"normalCount":   normalCount,
		"disabledCount": disabledCount,
		"newToday":      newToday,
		"activeToday":   activeToday,
		"demoCount":     demoCount,
	}, nil
}

// mpUserDetail 学员详情：档案 + 学习进度 + 逐门课程记录
func mpUserDetail(c *adminCtx, r *http.Request) (interface{}, error) {
	id := queryInt64(r, "id", 0)
	if id <= 0 {
		return nil, badRequest("缺少学员 id")
	}
	user, err := dao.FindMpUser(id)
	if err != nil {
		return nil, notFound("学员不存在")
	}
	records, err := dao.ListRecordsByUser(id)
	if err != nil {
		return nil, serverError("读取学习记录失败: " + err.Error())
	}

	courseIds := make([]int64, 0, len(records))
	chapterIds := make([]int64, 0, len(records))
	for _, item := range records {
		courseIds = append(courseIds, item.CourseId)
		if item.ChapterId > 0 {
			chapterIds = append(chapterIds, item.ChapterId)
		}
	}
	courses, _ := dao.ListCoursesByIds(courseIds)
	courseMap := make(map[int64]model.Course, len(courses))
	for _, item := range courses {
		courseMap[item.Id] = item
	}
	chapters, _ := dao.ListChaptersByIds(chapterIds)
	chapterMap := make(map[int64]string, len(chapters))
	for _, item := range chapters {
		chapterMap[item.Id] = item.Title
	}
	majorNames := map[int64]string{}
	for _, item := range majorOptions() {
		if mid, ok := item["id"].(int64); ok {
			if name, ok := item["name"].(string); ok {
				majorNames[mid] = name
			}
		}
	}

	aggs, _ := dao.LearnAggByUsers([]int64{id})
	agg := aggs[id]

	views := make([]learnRecordView, 0, len(records))
	for _, item := range records {
		view := learnRecordView{
			Id:           item.Id,
			CourseId:     item.CourseId,
			ChapterId:    item.ChapterId,
			ChapterTitle: chapterMap[item.ChapterId],
			Progress:     item.Progress,
			StudySeconds: item.StudySeconds,
			Status:       item.Status,
			StatusText:   learnStatusText(item.Status),
			LastStudyAt:  item.LastStudyAt,
		}
		if course, ok := courseMap[item.CourseId]; ok {
			view.CourseName = course.Name
		} else {
			view.CourseName = "课程已删除"
		}
		views = append(views, view)
	}

	majorName := majorNames[user.MajorId]
	if majorName == "" {
		majorName = "未选择专业"
	}
	levelText := learnerLevelText[user.Level]
	if levelText == "" {
		levelText = "入门"
	}

	return map[string]interface{}{
		"user": learnerItem{
			MpUser:        *user,
			MajorName:     majorName,
			LevelText:     levelText,
			CourseCount:   agg.Total,
			FinishedCount: agg.Finished,
			AvgProgress:   int(math.Round(agg.AvgProgress)),
			StudyHours:    math.Round(float64(agg.StudySeconds)/3600*10) / 10,
			LastStudyAt:   agg.LastStudyAt,
			Demo:          strings.HasPrefix(user.OpenId, "demo-"),
		},
		"records": views,
		"stats": map[string]interface{}{
			"courseCount":   agg.Total,
			"finishedCount": agg.Finished,
			"avgProgress":   int(math.Round(agg.AvgProgress)),
			"studyHours":    math.Round(float64(agg.StudySeconds)/3600*10) / 10,
			"lastStudyAt":   agg.LastStudyAt,
			"finishRate":    finishRate(agg.Finished, agg.Total),
		},
		"majors":       majorOptions(),
		"levelOptions": learnerLevelOptions(),
	}, nil
}

// mpUserSave 新建或维护学员档案。真实场景下学员由小程序授权自动建档，
// 这里保留录入能力，方便小程序未上线时造测试数据、或人工修正档案。
func mpUserSave(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id       int64  `json:"id"`
		OpenId   string `json:"openId"`
		Nickname string `json:"nickname"`
		Phone    string `json:"phone"`
		MajorId  int64  `json:"majorId"`
		Level    int    `json:"level"`
		Status   int    `json:"status"`
		Remark   string `json:"remark"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	req.Nickname = strings.TrimSpace(req.Nickname)
	req.Phone = strings.TrimSpace(req.Phone)
	req.Remark = strings.TrimSpace(req.Remark)
	req.OpenId = strings.TrimSpace(req.OpenId)

	if req.Nickname == "" {
		return nil, badRequest("学员昵称不能为空")
	}
	if len([]rune(req.Nickname)) > 64 {
		return nil, badRequest("昵称不能超过 64 个字")
	}
	if req.Phone != "" && !validPhone(req.Phone) {
		return nil, badRequest("手机号格式不正确")
	}
	if len([]rune(req.Remark)) > 200 {
		return nil, badRequest("备注不能超过 200 个字")
	}
	if req.Level == 0 {
		req.Level = DifficultyBasic
	}
	if _, ok := learnerLevelText[req.Level]; !ok {
		return nil, badRequest("学员水平只能是 1（入门）、2（进阶）、3（高级）")
	}
	if req.MajorId > 0 {
		if _, err := dao.FindMajor(req.MajorId); err != nil {
			return nil, badRequest("所选专业不存在")
		}
	}
	if req.Status != model.MpUserNormal && req.Status != model.MpUserDisabled {
		req.Status = model.MpUserNormal
	}

	now := time.Now()
	if req.Id > 0 {
		if _, err := dao.FindMpUser(req.Id); err != nil {
			return nil, notFound("学员不存在")
		}
		if err := dao.UpdateMpUserFields(req.Id, map[string]interface{}{
			"nickname":  req.Nickname,
			"phone":     req.Phone,
			"majorId":   req.MajorId,
			"level":     req.Level,
			"status":    req.Status,
			"remark":    req.Remark,
			"updatedAt": now,
		}); err != nil {
			return nil, serverError("保存失败: " + err.Error())
		}
		return dao.FindMpUser(req.Id)
	}

	openId := req.OpenId
	if openId == "" {
		openId = fmt.Sprintf("manual-%d", now.UnixNano())
	} else if _, err := dao.FindMpUserByOpenId(openId); err == nil {
		return nil, badRequest("openId 已存在，不能重复建档")
	}
	item := &model.MpUser{
		OpenId:    openId,
		Nickname:  req.Nickname,
		Phone:     req.Phone,
		MajorId:   req.MajorId,
		Level:     req.Level,
		Status:    req.Status,
		Remark:    req.Remark,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := dao.CreateMpUser(item); err != nil {
		return nil, serverError("创建失败: " + err.Error())
	}
	return item, nil
}

// mpUserStatus 启用 / 禁用学员
func mpUserStatus(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id     int64 `json:"id"`
		Status int   `json:"status"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if req.Id <= 0 {
		return nil, badRequest("缺少学员 id")
	}
	if req.Status != model.MpUserNormal && req.Status != model.MpUserDisabled {
		return nil, badRequest("状态只能是 0（禁用）或 1（正常）")
	}
	if _, err := dao.FindMpUser(req.Id); err != nil {
		return nil, notFound("学员不存在")
	}
	if err := dao.UpdateMpUserFields(req.Id, map[string]interface{}{
		"status": req.Status, "updatedAt": time.Now(),
	}); err != nil {
		return nil, serverError("保存失败: " + err.Error())
	}
	return map[string]interface{}{"id": req.Id, "status": req.Status}, nil
}

// mpUserDelete 删除学员，连同其学习记录
func mpUserDelete(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id int64 `json:"id"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if req.Id <= 0 {
		return nil, badRequest("缺少学员 id")
	}
	if _, err := dao.FindMpUser(req.Id); err != nil {
		return nil, notFound("学员不存在")
	}
	if err := dao.DeleteRecordsByUser(req.Id); err != nil {
		return nil, serverError("删除学习记录失败: " + err.Error())
	}
	if err := dao.DeleteMpUser(req.Id); err != nil {
		return nil, serverError("删除失败: " + err.Error())
	}
	return map[string]interface{}{"id": req.Id}, nil
}

// mpDemoClean 清除演示学员，只删 openId 以 demo- 开头的记录
func mpDemoClean(c *adminCtx, r *http.Request) (interface{}, error) {
	removed, err := dao.PurgeDemoMpUsers()
	if err != nil {
		return nil, serverError("清理演示数据失败: " + err.Error())
	}
	if removed == 0 {
		return nil, badRequest("没有演示学员需要清理")
	}
	return map[string]interface{}{"removed": removed}, nil
}

// learnerLevelOptions 水平下拉选项
func learnerLevelOptions() []map[string]interface{} {
	return []map[string]interface{}{
		{"value": DifficultyBasic, "label": "入门"},
		{"value": DifficultyAdvanced, "label": "进阶"},
		{"value": DifficultyExpert, "label": "高级"},
	}
}

// learnStatusText 学习状态文案
func learnStatusText(status string) string {
	switch status {
	case model.LearnFinished:
		return "已学完"
	case model.LearnLearning:
		return "学习中"
	}
	return status
}

// finishRate 完成率百分比，整数
func finishRate(finished, total int64) int {
	if total <= 0 {
		return 0
	}
	return int(math.Round(float64(finished) / float64(total) * 100))
}

// validPhone 简单手机号校验：11 位数字或以 + 开头的国际号码
func validPhone(phone string) bool {
	if len(phone) < 6 || len(phone) > 20 {
		return false
	}
	for i, ch := range phone {
		if i == 0 && ch == '+' {
			continue
		}
		if ch < '0' || ch > '9' {
			if ch != '-' && ch != ' ' {
				return false
			}
		}
	}
	return true
}
