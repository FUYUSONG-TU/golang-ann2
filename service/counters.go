package service

import (
	"net/http"
	"time"

	"wxcloudrun-golang/db/dao"
	"wxcloudrun-golang/db/model"
)

// counterList 列出全部计数记录
func counterList(c *adminCtx, r *http.Request) (interface{}, error) {
	list, err := dao.Imp.ListCounters()
	if err != nil {
		return nil, serverError("读取计数记录失败: " + err.Error())
	}
	return map[string]interface{}{"list": list, "total": len(list)}, nil
}

// counterIncr 指定记录自增
func counterIncr(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id   *int32 `json:"id"`
		Step *int32 `json:"step"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}

	id := defaultCounterID
	if req.Id != nil {
		id = *req.Id
	}
	step := int32(1)
	if req.Step != nil {
		step = *req.Step
	}

	counter, err := dao.Imp.GetCounter(id)
	notFoundRec := dao.IsNotFound(err)
	if err != nil && !notFoundRec {
		return nil, serverError(err.Error())
	}

	now := time.Now()
	if notFoundRec {
		if id != defaultCounterID {
			return nil, notFound("计数记录不存在")
		}
		counter = &model.CounterModel{Id: id, Count: step, CreatedAt: now, UpdatedAt: now}
	} else {
		counter.Count += step
		counter.UpdatedAt = now
	}

	if err := dao.Imp.UpsertCounter(counter); err != nil {
		return nil, serverError("写入失败: " + err.Error())
	}
	return counter, nil
}

// counterSet 把记录设置为固定值
func counterSet(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id    *int32 `json:"id"`
		Count *int32 `json:"count"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if req.Count == nil {
		return nil, badRequest("缺少 count 参数")
	}
	if *req.Count < 0 {
		return nil, badRequest("count 不能为负数")
	}

	id := defaultCounterID
	if req.Id != nil {
		id = *req.Id
	}

	counter, err := dao.Imp.GetCounter(id)
	now := time.Now()
	if err != nil {
		if !dao.IsNotFound(err) {
			return nil, serverError(err.Error())
		}
		if id != defaultCounterID {
			return nil, notFound("计数记录不存在")
		}
		counter = &model.CounterModel{Id: id, CreatedAt: now}
	}

	counter.Count = *req.Count
	counter.UpdatedAt = now
	if err := dao.Imp.UpsertCounter(counter); err != nil {
		return nil, serverError("写入失败: " + err.Error())
	}
	return counter, nil
}

// counterDelete 删除记录
func counterDelete(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Id *int32 `json:"id"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	if req.Id == nil {
		return nil, badRequest("缺少 id 参数")
	}
	if err := dao.Imp.ClearCounter(*req.Id); err != nil {
		return nil, serverError("删除失败: " + err.Error())
	}
	return map[string]interface{}{"id": *req.Id}, nil
}

// counterCreate 新建记录
func counterCreate(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Count *int32 `json:"count"`
	}{}
	// 允许空请求体，此时按 count=0 创建
	if r.ContentLength != 0 {
		if err := decodeBody(r, &req); err != nil {
			return nil, err
		}
	}

	count := int32(0)
	if req.Count != nil {
		count = *req.Count
	}
	if count < 0 {
		return nil, badRequest("count 不能为负数")
	}

	now := time.Now()
	counter := &model.CounterModel{Count: count, CreatedAt: now, UpdatedAt: now}
	if err := dao.Imp.UpsertCounter(counter); err != nil {
		return nil, serverError("创建失败: " + err.Error())
	}
	return counter, nil
}
