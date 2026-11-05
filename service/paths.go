package service

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"wxcloudrun-golang/db/dao"
	"wxcloudrun-golang/db/model"
)

// pathItemView 带课程信息的路线条目
type pathItemView struct {
	Id           int64  `json:"id"`
	CourseId     int64  `json:"courseId"`
	Name         string `json:"name"`
	MajorName    string `json:"majorName"`
	Difficulty   int    `json:"difficulty"`
	Hours        int    `json:"hours"`
	Status       string `json:"status"`
	ChapterCount int64  `json:"chapterCount"`
	Required     bool   `json:"required"`
	UnlockRule   string `json:"unlockRule"`
	Sort         int    `json:"sort"`
}

// pathStageView 阶段及其课程
type pathStageView struct {
	Id         int64          `json:"id"`
	PathId     int64          `json:"pathId"`
	Title      string         `json:"title"`
	Summary    string         `json:"summary"`
	Sort       int            `json:"sort"`
	CourseNum  int            `json:"courseNum"`
	TotalHours int            `json:"totalHours"`
	Items      []pathItemView `json:"items"`
}

// pathListItem 列表视图，带阶段数与课程数
type pathListItem struct {
	model.LearningPath
	MajorName   string `json:"majorName"`
	StageCount  int64  `json:"stageCount"`
	CourseCount int64  `json:"courseCount"`
}

// pathMajorNames 专业 id 到名称映射
func pathMajorNames() map[int64]string {
	out := map[int64]string{}
	for _, item := range majorOptions() {
		if id, ok := item["id"].(int64); ok {
			if name, ok := item["name"].(string); ok {
				out[id] = name
			}
		}
	}
	return out
}

// pathList 学习路线列表
func pathList(c *adminCtx, r *http.Request) (interface{}, error) {
	page, size := pageParams(r)
	list, total, err := dao.ListPaths(dao.PathQuery{
		MajorId: queryInt64(r, "majorId", 0),
		Keyword: strings.TrimSpace(r.URL.Query().Get("keyword")),
		Status:  strings.TrimSpace(r.URL.Query().Get("status")),
		Page:    page,
		Size:    size,
	})
	if err != nil {
		return nil, serverError("查询学习路线失败: " + err.Error())
	}

	ids := make([]int64, 0, len(list))
	for _, item := range list {
		ids = append(ids, item.Id)
	}
	stageCounts, _ := dao.CountStagesGroupByPath(ids)
	courseCounts, _ := dao.CountItemsGroupByPath(ids)
	majorNames := pathMajorNames()

	items := make([]pathListItem, 0, len(list))
	for _, item := range list {
		name := majorNames[item.MajorId]
		if item.MajorId == 0 {
			name = "通用"
		} else if name == "" {
			name = "未分配专业"
		}
		items = append(items, pathListItem{
			LearningPath: item,
			MajorName:    name,
			StageCount:   stageCounts[item.Id],
			CourseCount:  courseCounts[item.Id],
		})
	}

	statusCounts, _ := dao.CountPathsByStatus()
	return map[string]interface{}{
		"list":         items,
		"total":        total,
		"page":         page,
		"size":         size,
		"majors":       majorOptions(),
		"statusCounts": statusCounts,
	}, nil
}

// pathDetail 路线详情：阶段树 + 课程 + 统计
func pathDetail(c *adminCtx, r *http.Request) (interface{}, error) {
	id := queryInt64(r, "id", 0)
	if id <= 0 {
		return nil, badRequest("缺少路线 id")
	}
	path, err := dao.FindPath(id)
	if err != nil {
		return nil, notFound("学习路线不存在")
	}
	stages, err := dao.ListStages(id)
	if err != nil {
		return nil, serverError("读取阶段失败: " + err.Error())
	}
	items, err := dao.ListItems(id)
	if err != nil {
		return nil, serverError("读取课程失败: " + err.Error())
	}

	courseIds := make([]int64, 0, len(items))
	seen := map[int64]bool{}
	for _, item := range items {
		if !seen[item.CourseId] {
			seen[item.CourseId] = true
			courseIds = append(courseIds, item.CourseId)
		}
	}
	courses, _ := dao.ListCoursesByIds(courseIds)
	courseMap := make(map[int64]model.Course, len(courses))
	for _, item := range courses {
		courseMap[item.Id] = item
	}
	chapterCounts, _ := dao.CountChaptersGroupByCourse(courseIds)
	majorNames := pathMajorNames()

	grouped := map[int64][]pathItemView{}
	totalHours := 0
	requiredNum := 0
	for _, item := range items {
		course, ok := courseMap[item.CourseId]
		if !ok {
			// 课程已被删除，条目保留但不展示脏数据
			continue
		}
		majorName := majorNames[course.MajorId]
		if majorName == "" {
			majorName = "未分配专业"
		}
		grouped[item.StageId] = append(grouped[item.StageId], pathItemView{
			Id:           item.Id,
			CourseId:     item.CourseId,
			Name:         course.Name,
			MajorName:    majorName,
			Difficulty:   course.Difficulty,
			Hours:        course.Hours,
			Status:       course.Status,
			ChapterCount: chapterCounts[item.CourseId],
			Required:     item.Required,
			UnlockRule:   item.UnlockRule,
			Sort:         item.Sort,
		})
		totalHours += course.Hours
		if item.Required {
			requiredNum++
		}
	}

	stageViews := make([]pathStageView, 0, len(stages))
	for _, stage := range stages {
		list := grouped[stage.Id]
		hours := 0
		for _, one := range list {
			hours += one.Hours
		}
		stageViews = append(stageViews, pathStageView{
			Id:         stage.Id,
			PathId:     stage.PathId,
			Title:      stage.Title,
			Summary:    stage.Summary,
			Sort:       stage.Sort,
			CourseNum:  len(list),
			TotalHours: hours,
			Items:      list,
		})
	}

	majorName := majorNames[path.MajorId]
	if path.MajorId == 0 {
		majorName = "通用"
	}
	return map[string]interface{}{
		"path":   pathListItem{LearningPath: *path, MajorName: majorName, StageCount: int64(len(stages)), CourseCount: int64(len(items))},
		"stages": stageViews,
		"stats": map[string]interface{}{
			"stageCount":  len(stages),
			"courseCount": len(items),
			"requiredNum": requiredNum,
			"totalHours":  totalHours,
		},
		"majors": majorOptions(),
	}, nil
}

// pathSave 新建或编辑学习路线
func pathSave(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id         int64  `json:"id"`
		Code       string `json:"code"`
		Name       string `json:"name"`
		Cover      string `json:"cover"`
		Summary    string `json:"summary"`
		MajorId    int64  `json:"majorId"`
		Difficulty int    `json:"difficulty"`
		Hours      int    `json:"hours"`
		Status     string `json:"status"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Code = strings.TrimSpace(req.Code)
	req.Cover = strings.TrimSpace(req.Cover)
	req.Summary = strings.TrimSpace(req.Summary)

	if req.Name == "" {
		return nil, badRequest("路线名称不能为空")
	}
	if len([]rune(req.Name)) > 64 {
		return nil, badRequest("路线名称不能超过 64 个字")
	}
	if len([]rune(req.Summary)) > 200 {
		return nil, badRequest("路线简介不能超过 200 个字")
	}
	if len(req.Code) > 32 {
		return nil, badRequest("路线编码不能超过 32 个字符")
	}
	if req.MajorId > 0 {
		if _, err := dao.FindMajor(req.MajorId); err != nil {
			return nil, badRequest("所选专业不存在")
		}
	}
	if req.Difficulty == 0 {
		req.Difficulty = DifficultyBasic
	}
	if req.Difficulty < DifficultyBasic || req.Difficulty > DifficultyExpert {
		return nil, badRequest("难度只能是 1（入门）、2（进阶）、3（高级）")
	}
	if req.Hours < 0 || req.Hours > 9999 {
		return nil, badRequest("预计学时需在 0-9999 之间")
	}
	if req.Status == "" {
		req.Status = model.ContentDraft
	}
	if !validContentStatus(req.Status) {
		return nil, badRequest("状态取值不合法")
	}

	now := time.Now()
	if req.Id > 0 {
		if _, err := dao.FindPath(req.Id); err != nil {
			return nil, notFound("学习路线不存在")
		}
		fields := map[string]interface{}{
			"code":       req.Code,
			"name":       req.Name,
			"cover":      req.Cover,
			"summary":    req.Summary,
			"majorId":    req.MajorId,
			"difficulty": req.Difficulty,
			"hours":      req.Hours,
			"updatedAt":  now,
		}
		// 发布状态的校验只在显式提交发布时做，避免编辑基本信息被拦
		if req.Status == model.ContentPublished {
			if err := checkPathPublishable(req.Id); err != nil {
				return nil, err
			}
			fields["status"] = req.Status
		} else if req.Status != model.ContentPublished {
			fields["status"] = req.Status
		}
		if err := dao.UpdatePathFields(req.Id, fields); err != nil {
			return nil, serverError("保存失败: " + err.Error())
		}
		updated, _ := dao.FindPath(req.Id)
		return updated, nil
	}

	item := &model.LearningPath{
		Code:       req.Code,
		Name:       req.Name,
		Cover:      req.Cover,
		Summary:    req.Summary,
		MajorId:    req.MajorId,
		Difficulty: req.Difficulty,
		Hours:      req.Hours,
		Status:     model.ContentDraft, // 新建一律草稿，配好阶段后再发布
		Sort:       dao.NextSort("learning_path"),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := dao.CreatePath(item); err != nil {
		return nil, serverError("创建失败: " + err.Error())
	}
	return item, nil
}

// checkPathPublishable 发布前校验：至少一个阶段，且每个阶段都有课程
func checkPathPublishable(pathId int64) error {
	stages, err := dao.ListStages(pathId)
	if err != nil {
		return serverError("读取阶段失败: " + err.Error())
	}
	if len(stages) == 0 {
		return badRequest("路线还没有阶段，无法发布")
	}
	for _, stage := range stages {
		count, err := dao.CountItemsByStage(stage.Id)
		if err != nil {
			return serverError(err.Error())
		}
		if count == 0 {
			return badRequest("阶段「" + stage.Title + "」还没有课程，无法发布")
		}
	}
	return nil
}

// pathStatus 切换路线发布状态
func pathStatus(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id     int64  `json:"id"`
		Status string `json:"status"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if req.Id <= 0 {
		return nil, badRequest("缺少路线 id")
	}
	if !validContentStatus(req.Status) {
		return nil, badRequest("状态取值不合法")
	}
	if _, err := dao.FindPath(req.Id); err != nil {
		return nil, notFound("学习路线不存在")
	}
	if req.Status == model.ContentPublished {
		if err := checkPathPublishable(req.Id); err != nil {
			return nil, err
		}
	}
	if err := dao.UpdatePathFields(req.Id, map[string]interface{}{
		"status": req.Status, "updatedAt": time.Now(),
	}); err != nil {
		return nil, serverError("保存失败: " + err.Error())
	}
	return map[string]interface{}{"id": req.Id, "status": req.Status}, nil
}

// pathDelete 删除路线，级联删除阶段与课程引用
func pathDelete(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id int64 `json:"id"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if req.Id <= 0 {
		return nil, badRequest("缺少路线 id")
	}
	if _, err := dao.FindPath(req.Id); err != nil {
		return nil, notFound("学习路线不存在")
	}
	if err := dao.DeleteItemsByPath(req.Id); err != nil {
		return nil, serverError("删除课程引用失败: " + err.Error())
	}
	if err := dao.DeleteStagesByPath(req.Id); err != nil {
		return nil, serverError("删除阶段失败: " + err.Error())
	}
	if err := dao.DeletePath(req.Id); err != nil {
		return nil, serverError("删除失败: " + err.Error())
	}
	return map[string]interface{}{"id": req.Id}, nil
}

// pathSort 调整路线顺序
func pathSort(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Ids []int64 `json:"ids"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if len(req.Ids) == 0 {
		return nil, badRequest("没有需要排序的路线")
	}
	if err := dao.BatchUpdateSort("learning_path", req.Ids); err != nil {
		return nil, serverError("排序失败: " + err.Error())
	}
	return map[string]interface{}{"sorted": len(req.Ids)}, nil
}

// pathStageSave 新建或编辑阶段
func pathStageSave(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id      int64  `json:"id"`
		PathId  int64  `json:"pathId"`
		Title   string `json:"title"`
		Summary string `json:"summary"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Summary = strings.TrimSpace(req.Summary)

	if req.Title == "" {
		return nil, badRequest("阶段名称不能为空")
	}
	if len([]rune(req.Title)) > 64 {
		return nil, badRequest("阶段名称不能超过 64 个字")
	}
	if len([]rune(req.Summary)) > 200 {
		return nil, badRequest("阶段说明不能超过 200 个字")
	}
	if req.PathId <= 0 {
		return nil, badRequest("缺少所属路线")
	}
	if _, err := dao.FindPath(req.PathId); err != nil {
		return nil, notFound("学习路线不存在")
	}

	now := time.Now()
	if req.Id > 0 {
		stage, err := dao.FindStage(req.Id)
		if err != nil {
			return nil, notFound("阶段不存在")
		}
		if stage.PathId != req.PathId {
			return nil, badRequest("阶段不属于该路线")
		}
		if err := dao.UpdateStageFields(req.Id, map[string]interface{}{
			"title": req.Title, "summary": req.Summary, "updatedAt": now,
		}); err != nil {
			return nil, serverError("保存失败: " + err.Error())
		}
		updated, _ := dao.FindStage(req.Id)
		return updated, nil
	}

	item := &model.PathStage{
		PathId:    req.PathId,
		Title:     req.Title,
		Summary:   req.Summary,
		Sort:      dao.NextSortBy("path_stage", "pathId = ?", req.PathId),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := dao.CreateStage(item); err != nil {
		return nil, serverError("创建失败: " + err.Error())
	}
	return item, nil
}

// pathStageDelete 删除阶段，连同其下课程引用
func pathStageDelete(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id int64 `json:"id"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if req.Id <= 0 {
		return nil, badRequest("缺少阶段 id")
	}
	stage, err := dao.FindStage(req.Id)
	if err != nil {
		return nil, notFound("阶段不存在")
	}
	count, _ := dao.CountItemsByStage(req.Id)
	if err := dao.DeleteItemsByStage(req.Id); err != nil {
		return nil, serverError("删除阶段课程失败: " + err.Error())
	}
	if err := dao.DeleteStage(req.Id); err != nil {
		return nil, serverError("删除失败: " + err.Error())
	}
	return map[string]interface{}{"id": stage.Id, "removedItems": count}, nil
}

// pathStageSort 调整路线内阶段顺序
func pathStageSort(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Ids []int64 `json:"ids"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if len(req.Ids) == 0 {
		return nil, badRequest("没有需要排序的阶段")
	}
	if err := dao.BatchUpdateSort("path_stage", req.Ids); err != nil {
		return nil, serverError("排序失败: " + err.Error())
	}
	return map[string]interface{}{"sorted": len(req.Ids)}, nil
}

// pathItemAdd 向阶段批量添加课程，自动跳过重复引用
func pathItemAdd(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		PathId    int64   `json:"pathId"`
		StageId   int64   `json:"stageId"`
		CourseIds []int64 `json:"courseIds"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if req.StageId <= 0 {
		return nil, badRequest("请先选择阶段")
	}
	stage, err := dao.FindStage(req.StageId)
	if err != nil {
		return nil, notFound("阶段不存在")
	}
	if req.PathId > 0 && stage.PathId != req.PathId {
		return nil, badRequest("阶段不属于该路线")
	}
	if len(req.CourseIds) == 0 {
		return nil, badRequest("请选择要添加的课程")
	}

	added, skipped := 0, []string{}
	now := time.Now()
	for _, courseId := range req.CourseIds {
		course, err := dao.FindCourse(courseId)
		if err != nil {
			skipped = append(skipped, strconv.FormatInt(courseId, 10))
			continue
		}
		if _, err := dao.FindItemByCourse(stage.PathId, courseId); err == nil {
			// 同一路线内不允许重复课程，否则学习进度会算两遍
			skipped = append(skipped, course.Name+"（已在路线中）")
			continue
		}
		item := &model.PathItem{
			PathId:    stage.PathId,
			StageId:   stage.Id,
			CourseId:  courseId,
			Required:  true,
			Sort:      dao.NextSortBy("path_item", "stageId = ?", stage.Id),
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := dao.CreateItem(item); err != nil {
			return nil, serverError("添加「" + course.Name + "」失败: " + err.Error())
		}
		added++
	}
	if added == 0 {
		msg := "没有新增课程"
		if len(skipped) > 0 {
			msg = "所选课程均未添加：" + strings.Join(skipped, "、")
		}
		return nil, badRequest(msg)
	}
	return map[string]interface{}{"added": added, "skipped": skipped}, nil
}

// pathItemUpdate 修改条目属性（必学标记、解锁条件）
func pathItemUpdate(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id         int64   `json:"id"`
		Required   *bool   `json:"required"`
		UnlockRule *string `json:"unlockRule"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if req.Id <= 0 {
		return nil, badRequest("缺少条目 id")
	}
	if _, err := dao.FindItem(req.Id); err != nil {
		return nil, notFound("路线条目不存在")
	}
	fields := map[string]interface{}{"updatedAt": time.Now()}
	if req.Required != nil {
		fields["required"] = *req.Required
	}
	if req.UnlockRule != nil {
		rule := strings.TrimSpace(*req.UnlockRule)
		if len([]rune(rule)) > 120 {
			return nil, badRequest("解锁条件不能超过 120 个字")
		}
		fields["unlockRule"] = rule
	}
	if err := dao.UpdateItemFields(req.Id, fields); err != nil {
		return nil, serverError("保存失败: " + err.Error())
	}
	updated, _ := dao.FindItem(req.Id)
	return updated, nil
}

// pathItemRemove 从阶段移除课程
func pathItemRemove(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id int64 `json:"id"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if req.Id <= 0 {
		return nil, badRequest("缺少条目 id")
	}
	if _, err := dao.FindItem(req.Id); err != nil {
		return nil, notFound("路线条目不存在")
	}
	if err := dao.DeleteItem(req.Id); err != nil {
		return nil, serverError("移除失败: " + err.Error())
	}
	return map[string]interface{}{"id": req.Id}, nil
}

// pathItemSort 调整阶段内课程顺序
func pathItemSort(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Ids []int64 `json:"ids"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if len(req.Ids) == 0 {
		return nil, badRequest("没有需要排序的课程")
	}
	if err := dao.BatchUpdateSort("path_item", req.Ids); err != nil {
		return nil, serverError("排序失败: " + err.Error())
	}
	return map[string]interface{}{"sorted": len(req.Ids)}, nil
}
