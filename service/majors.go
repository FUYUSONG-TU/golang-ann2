package service

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"wxcloudrun-golang/db/dao"
	"wxcloudrun-golang/db/model"
)

// majorItem 带课程数的专业视图
type majorItem struct {
	model.Major
	CourseCount int64 `json:"courseCount"`
}

// majorList 专业列表
func majorList(c *adminCtx, r *http.Request) (interface{}, error) {
	status := queryInt(r, "status", -1)
	list, err := dao.ListMajors(dao.MajorQuery{
		Keyword: strings.TrimSpace(r.URL.Query().Get("keyword")),
		Status:  status,
	})
	if err != nil {
		return nil, serverError("查询专业失败: " + err.Error())
	}

	counts, _ := dao.CountCoursesGroupByMajor()
	items := make([]majorItem, 0, len(list))
	for _, item := range list {
		items = append(items, majorItem{Major: item, CourseCount: counts[item.Id]})
	}
	return map[string]interface{}{"list": items, "total": len(items)}, nil
}

// majorSave 新建或编辑专业
func majorSave(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id      int64  `json:"id"`
		Code    string `json:"code"`
		Name    string `json:"name"`
		Icon    string `json:"icon"`
		Summary string `json:"summary"`
		Status  *int   `json:"status"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Code = strings.TrimSpace(req.Code)
	req.Icon = strings.TrimSpace(req.Icon)
	req.Summary = strings.TrimSpace(req.Summary)

	if req.Name == "" {
		return nil, badRequest("专业名称不能为空")
	}
	if len([]rune(req.Name)) > 32 {
		return nil, badRequest("专业名称不能超过 32 个字")
	}
	if len(req.Code) > 32 {
		return nil, badRequest("专业编码不能超过 32 个字符")
	}

	// 编码唯一性校验，空编码不参与
	if req.Code != "" {
		exist, err := dao.FindMajorByCode(req.Code)
		if err == nil && exist.Id != req.Id {
			return nil, badRequest("专业编码 " + req.Code + " 已被「" + exist.Name + "」占用")
		}
	}

	status := 1
	if req.Status != nil {
		status = *req.Status
		if status != 0 && status != 1 {
			return nil, badRequest("状态值只能是 0 或 1")
		}
	}

	now := time.Now()
	if req.Id > 0 {
		if _, err := dao.FindMajor(req.Id); err != nil {
			return nil, notFound("专业不存在")
		}
		fields := map[string]interface{}{
			"code":      req.Code,
			"name":      req.Name,
			"icon":      req.Icon,
			"summary":   req.Summary,
			"status":    status,
			"updatedAt": now,
		}
		if err := dao.UpdateMajorFields(req.Id, fields); err != nil {
			return nil, serverError("保存失败: " + err.Error())
		}
		item, _ := dao.FindMajor(req.Id)
		return item, nil
	}

	item := &model.Major{
		Code:      req.Code,
		Name:      req.Name,
		Icon:      req.Icon,
		Summary:   req.Summary,
		Status:    status,
		Sort:      dao.NextSort("major"),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := dao.CreateMajor(item); err != nil {
		return nil, serverError("创建失败: " + err.Error())
	}
	demo := majorItem{Major: *item, CourseCount: 0}
	return demo, nil
}

// majorDelete 删除专业，存在课程时拒绝
func majorDelete(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id int64 `json:"id"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if req.Id <= 0 {
		return nil, badRequest("缺少专业 id")
	}
	if _, err := dao.FindMajor(req.Id); err != nil {
		return nil, notFound("专业不存在")
	}
	total, err := dao.CountCoursesByMajor(req.Id)
	if err != nil {
		return nil, serverError(err.Error())
	}
	if total > 0 {
		return nil, badRequest("该专业下还有 " + strconv.FormatInt(total, 10) + " 门课程，请先移走或删除课程")
	}
	if err := dao.DeleteMajor(req.Id); err != nil {
		return nil, serverError("删除失败: " + err.Error())
	}
	return map[string]interface{}{"id": req.Id}, nil
}

// majorSort 调整专业顺序，按传入 id 顺序重排
func majorSort(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Ids []int64 `json:"ids"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if len(req.Ids) == 0 {
		return nil, badRequest("没有需要排序的专业")
	}
	if err := dao.BatchUpdateSort("major", req.Ids); err != nil {
		return nil, serverError("排序失败: " + err.Error())
	}
	return map[string]interface{}{"sorted": len(req.Ids)}, nil
}
