package service

import (
	"net/http"
	"strings"
	"time"

	"wxcloudrun-golang/db/dao"
	"wxcloudrun-golang/db/model"
)

// chapterNode 章节树节点，两级：章 → 知识点
type chapterNode struct {
	model.Chapter
	Children []*chapterNode `json:"children"`
}

// buildChapterTree 把扁平列表组装成两级树，父节点缺失时降级为顶层
func buildChapterTree(list []model.Chapter) []*chapterNode {
	nodes := make(map[int64]*chapterNode, len(list))
	for _, item := range list {
		nodes[item.Id] = &chapterNode{Chapter: item, Children: []*chapterNode{}}
	}
	roots := make([]*chapterNode, 0, len(list))
	for _, item := range list {
		node := nodes[item.Id]
		if item.ParentId == 0 {
			roots = append(roots, node)
			continue
		}
		parent, ok := nodes[item.ParentId]
		if !ok {
			roots = append(roots, node)
			continue
		}
		parent.Children = append(parent.Children, node)
	}
	return roots
}

// chapterList 查询课程下的章节树
func chapterList(c *adminCtx, r *http.Request) (interface{}, error) {
	courseId := queryInt64(r, "courseId", 0)
	if courseId <= 0 {
		return nil, badRequest("缺少课程 id")
	}
	list, err := dao.ListChapters(courseId)
	if err != nil {
		return nil, serverError("查询章节失败: " + err.Error())
	}
	return map[string]interface{}{
		"list":  buildChapterTree(list),
		"total": len(list),
	}, nil
}

// chapterSave 新建或编辑章节
func chapterSave(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id       int64  `json:"id"`
		CourseId int64  `json:"courseId"`
		ParentId int64  `json:"parentId"`
		Title    string `json:"title"`
		Summary  string `json:"summary"`
		Duration int    `json:"duration"`
		Status   string `json:"status"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}

	req.Title = strings.TrimSpace(req.Title)
	req.Summary = strings.TrimSpace(req.Summary)

	if req.CourseId <= 0 {
		return nil, badRequest("缺少课程 id")
	}
	if req.Title == "" {
		return nil, badRequest("章节标题不能为空")
	}
	if len([]rune(req.Title)) > 64 {
		return nil, badRequest("章节标题不能超过 64 个字")
	}
	if req.Duration < 0 || req.Duration > 9999 {
		return nil, badRequest("时长需在 0-9999 分钟之间")
	}
	if _, err := dao.FindCourse(req.CourseId); err != nil {
		return nil, badRequest("课程不存在")
	}

	// 只允许两级：父节点必须属于同一课程，且自身是顶层
	if req.ParentId > 0 {
		parent, err := dao.FindChapter(req.ParentId)
		if err != nil {
			return nil, badRequest("上级章节不存在")
		}
		if parent.CourseId != req.CourseId {
			return nil, badRequest("上级章节不属于当前课程")
		}
		if parent.ParentId != 0 {
			return nil, badRequest("章节最多两级，知识点下不能再建子项")
		}
		if req.Id > 0 && req.ParentId == req.Id {
			return nil, badRequest("上级章节不能是自己")
		}
	}

	if req.Status == "" {
		req.Status = model.ContentPublished
	}
	if !validContentStatus(req.Status) {
		return nil, badRequest("状态取值不合法")
	}

	now := time.Now()
	if req.Id > 0 {
		chapter, err := dao.FindChapter(req.Id)
		if err != nil {
			return nil, notFound("章节不存在")
		}
		if chapter.CourseId != req.CourseId {
			return nil, badRequest("章节不属于当前课程")
		}
		// 带子节点的章不允许改成某个章的知识点，否则层级会乱
		if req.ParentId > 0 && chapter.ParentId == 0 {
			siblings, err := dao.ListChapters(req.CourseId)
			if err == nil {
				for _, item := range siblings {
					if item.ParentId == chapter.Id {
						return nil, badRequest("该章节下还有知识点，不能移动为子级")
					}
				}
			}
		}
		fields := map[string]interface{}{
			"parentId":  req.ParentId,
			"title":     req.Title,
			"summary":   req.Summary,
			"duration":  req.Duration,
			"status":    req.Status,
			"updatedAt": now,
		}
		if err := dao.UpdateChapterFields(req.Id, fields); err != nil {
			return nil, serverError("保存失败: " + err.Error())
		}
		updated, _ := dao.FindChapter(req.Id)
		return updated, nil
	}

	item := &model.Chapter{
		CourseId:  req.CourseId,
		ParentId:  req.ParentId,
		Title:     req.Title,
		Summary:   req.Summary,
		Duration:  req.Duration,
		Status:    req.Status,
		Sort:      dao.NextSort("chapter"),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := dao.CreateChapter(item); err != nil {
		return nil, serverError("创建失败: " + err.Error())
	}
	return item, nil
}

// chapterDelete 删除章节及其下的知识点
func chapterDelete(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id int64 `json:"id"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if req.Id <= 0 {
		return nil, badRequest("缺少章节 id")
	}
	if _, err := dao.FindChapter(req.Id); err != nil {
		return nil, notFound("章节不存在")
	}
	if err := dao.DeleteChapterWithChildren(req.Id); err != nil {
		return nil, serverError("删除失败: " + err.Error())
	}
	return map[string]interface{}{"id": req.Id}, nil
}

// chapterSort 调整同层章节顺序
func chapterSort(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Ids []int64 `json:"ids"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if len(req.Ids) == 0 {
		return nil, badRequest("没有需要排序的章节")
	}
	if err := dao.BatchUpdateSort("chapter", req.Ids); err != nil {
		return nil, serverError("排序失败: " + err.Error())
	}
	return map[string]interface{}{"sorted": len(req.Ids)}, nil
}
