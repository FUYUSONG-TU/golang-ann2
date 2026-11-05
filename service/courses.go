package service

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"wxcloudrun-golang/db/dao"
	"wxcloudrun-golang/db/model"
)

// 难度取值
const (
	DifficultyBasic    = 1 // 入门
	DifficultyAdvanced = 2 // 进阶
	DifficultyExpert   = 3 // 高级
)

// courseItem 带专业名与章节数的课程视图
type courseItem struct {
	model.Course
	MajorName    string `json:"majorName"`
	ChapterCount int64  `json:"chapterCount"`
}

// validContentStatus 判断内容状态是否合法
func validContentStatus(status string) bool {
	switch status {
	case model.ContentDraft, model.ContentPending, model.ContentPublished, model.ContentOffline:
		return true
	}
	return false
}

// majorOptions 专业下拉选项
func majorOptions() []map[string]interface{} {
	list, err := dao.ListMajors(dao.MajorQuery{Status: -1})
	if err != nil {
		return []map[string]interface{}{}
	}
	out := make([]map[string]interface{}, 0, len(list))
	for _, item := range list {
		out = append(out, map[string]interface{}{
			"id": item.Id, "name": item.Name, "status": item.Status,
		})
	}
	return out
}

// courseList 分页查询课程
func courseList(c *adminCtx, r *http.Request) (interface{}, error) {
	page, size := pageParams(r)
	list, total, err := dao.ListCourses(dao.CourseQuery{
		MajorId:    queryInt64(r, "majorId", 0),
		Keyword:    strings.TrimSpace(r.URL.Query().Get("keyword")),
		Status:     strings.TrimSpace(r.URL.Query().Get("status")),
		Difficulty: queryInt(r, "difficulty", 0),
		Page:       page,
		Size:       size,
	})
	if err != nil {
		return nil, serverError("查询课程失败: " + err.Error())
	}

	majorNames := map[int64]string{}
	for _, item := range majorOptions() {
		if id, ok := item["id"].(int64); ok {
			if name, ok := item["name"].(string); ok {
				majorNames[id] = name
			}
		}
	}

	ids := make([]int64, 0, len(list))
	for _, item := range list {
		ids = append(ids, item.Id)
	}
	chapterCounts, _ := dao.CountChaptersGroupByCourse(ids)

	items := make([]courseItem, 0, len(list))
	for _, item := range list {
		name := majorNames[item.MajorId]
		if name == "" {
			name = "未分配专业"
		}
		items = append(items, courseItem{
			Course: item, MajorName: name, ChapterCount: chapterCounts[item.Id],
		})
	}

	statusCounts, _ := dao.CountCoursesByStatus()

	return map[string]interface{}{
		"list":         items,
		"total":        total,
		"page":         page,
		"size":         size,
		"majors":       majorOptions(),
		"statusCounts": statusCounts,
	}, nil
}

// courseDetail 课程详情，含章节树
func courseDetail(c *adminCtx, r *http.Request) (interface{}, error) {
	id := queryInt64(r, "id", 0)
	if id <= 0 {
		return nil, badRequest("缺少课程 id")
	}
	course, err := dao.FindCourse(id)
	if err != nil {
		return nil, notFound("课程不存在")
	}

	majorName := ""
	if major, err := dao.FindMajor(course.MajorId); err == nil {
		majorName = major.Name
	}
	chapters, err := dao.ListChapters(id)
	if err != nil {
		return nil, serverError("读取章节失败: " + err.Error())
	}

	return map[string]interface{}{
		"course":   courseItem{Course: *course, MajorName: majorName, ChapterCount: int64(len(chapters))},
		"chapters": buildChapterTree(chapters),
		"majors":   majorOptions(),
	}, nil
}

// courseSave 新建或编辑课程
func courseSave(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id         int64  `json:"id"`
		MajorId    int64  `json:"majorId"`
		Name       string `json:"name"`
		Cover      string `json:"cover"`
		Difficulty int    `json:"difficulty"`
		Summary    string `json:"summary"`
		Hours      int    `json:"hours"`
		Status     string `json:"status"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Summary = strings.TrimSpace(req.Summary)
	req.Cover = strings.TrimSpace(req.Cover)

	if req.Name == "" {
		return nil, badRequest("课程名称不能为空")
	}
	if len([]rune(req.Name)) > 64 {
		return nil, badRequest("课程名称不能超过 64 个字")
	}
	if req.MajorId <= 0 {
		return nil, badRequest("请选择所属专业")
	}
	if _, err := dao.FindMajor(req.MajorId); err != nil {
		return nil, badRequest("所属专业不存在")
	}
	if req.Difficulty == 0 {
		req.Difficulty = DifficultyBasic
	}
	if req.Difficulty < DifficultyBasic || req.Difficulty > DifficultyExpert {
		return nil, badRequest("难度只能是 1（入门）、2（进阶）、3（高级）")
	}
	if req.Hours < 0 || req.Hours > 9999 {
		return nil, badRequest("课时数需在 0-9999 之间")
	}
	if req.Status == "" {
		req.Status = model.ContentDraft
	}
	if !validContentStatus(req.Status) {
		return nil, badRequest("状态取值不合法")
	}

	now := time.Now()
	if req.Id > 0 {
		course, err := dao.FindCourse(req.Id)
		if err != nil {
			return nil, notFound("课程不存在")
		}
		// 从已发布退回草稿时无需校验章节，发布动作统一走 courseStatus
		if req.Status == model.ContentPublished && course.Status != model.ContentPublished {
			count, err := dao.CountChaptersByCourse(course.Id)
			if err != nil {
				return nil, serverError(err.Error())
			}
			if count == 0 {
				return nil, badRequest("课程还没有章节，无法发布")
			}
		}
		fields := map[string]interface{}{
			"majorId":    req.MajorId,
			"name":       req.Name,
			"cover":      req.Cover,
			"difficulty": req.Difficulty,
			"summary":    req.Summary,
			"hours":      req.Hours,
			"status":     req.Status,
			"updatedAt":  now,
		}
		if err := dao.UpdateCourseFields(req.Id, fields); err != nil {
			return nil, serverError("保存失败: " + err.Error())
		}
		updated, _ := dao.FindCourse(req.Id)
		return updated, nil
	}

	item := &model.Course{
		MajorId:    req.MajorId,
		Name:       req.Name,
		Cover:      req.Cover,
		Difficulty: req.Difficulty,
		Summary:    req.Summary,
		Hours:      req.Hours,
		Status:     model.ContentDraft, // 新建一律为草稿，避免空课程直接对外
		Sort:       dao.NextSort("course"),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := dao.CreateCourse(item); err != nil {
		return nil, serverError("创建失败: " + err.Error())
	}
	return item, nil
}

// courseStatus 切换课程发布状态
func courseStatus(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id     int64  `json:"id"`
		Status string `json:"status"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if req.Id <= 0 {
		return nil, badRequest("缺少课程 id")
	}
	if !validContentStatus(req.Status) {
		return nil, badRequest("状态取值不合法")
	}
	if _, err := dao.FindCourse(req.Id); err != nil {
		return nil, notFound("课程不存在")
	}
	if req.Status == model.ContentPublished {
		count, err := dao.CountChaptersByCourse(req.Id)
		if err != nil {
			return nil, serverError(err.Error())
		}
		if count == 0 {
			return nil, badRequest("课程还没有章节，无法发布")
		}
	}
	if err := dao.UpdateCourseFields(req.Id, map[string]interface{}{
		"status": req.Status, "updatedAt": time.Now(),
	}); err != nil {
		return nil, serverError("保存失败: " + err.Error())
	}
	return map[string]interface{}{"id": req.Id, "status": req.Status}, nil
}

// courseDelete 软删除课程，同时软删除其章节
func courseDelete(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id int64 `json:"id"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if req.Id <= 0 {
		return nil, badRequest("缺少课程 id")
	}
	if _, err := dao.FindCourse(req.Id); err != nil {
		return nil, notFound("课程不存在")
	}
	// 被学习路线引用时不允许直接删除，否则路线里会留下打不开的课程
	refs, err := dao.ListPathsByCourse(req.Id)
	if err != nil {
		return nil, serverError("检查路线引用失败: " + err.Error())
	}
	if len(refs) > 0 {
		names := make([]string, 0, len(refs))
		for _, ref := range refs {
			if ref.PathName == "" {
				continue
			}
			names = append(names, "「"+ref.PathName+"」")
		}
		return nil, badRequest("该课程已被 " + strconv.Itoa(len(refs)) + " 条学习路线引用" +
			strings.Join(names, "、") + "，请先从路线中移除")
	}
	if err := dao.DeleteChaptersByCourse(req.Id); err != nil {
		return nil, serverError("删除章节失败: " + err.Error())
	}
	if err := dao.DeleteCourse(req.Id); err != nil {
		return nil, serverError("删除失败: " + err.Error())
	}
	return map[string]interface{}{"id": req.Id}, nil
}

// courseSort 调整课程顺序
func courseSort(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Ids []int64 `json:"ids"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if len(req.Ids) == 0 {
		return nil, badRequest("没有需要排序的课程")
	}
	if err := dao.BatchUpdateSort("course", req.Ids); err != nil {
		return nil, serverError("排序失败: " + err.Error())
	}
	return map[string]interface{}{"sorted": len(req.Ids)}, nil
}
