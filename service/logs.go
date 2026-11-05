package service

import (
	"net/http"
	"strings"

	"wxcloudrun-golang/db/dao"
)

// logLoginList 登录日志
func logLoginList(c *adminCtx, r *http.Request) (interface{}, error) {
	page, size := pageParams(r)
	list, total, err := dao.ListLoginLogs(dao.LogQuery{
		Keyword: strings.TrimSpace(r.URL.Query().Get("keyword")),
		Success: queryInt(r, "success", -1),
		Page:    page,
		Size:    size,
	})
	if err != nil {
		return nil, serverError("查询登录日志失败: " + err.Error())
	}
	return map[string]interface{}{"list": list, "total": total, "page": page, "size": size}, nil
}

// logOperationList 操作日志
func logOperationList(c *adminCtx, r *http.Request) (interface{}, error) {
	page, size := pageParams(r)
	list, total, err := dao.ListOperationLogs(dao.LogQuery{
		Keyword: strings.TrimSpace(r.URL.Query().Get("keyword")),
		Module:  strings.TrimSpace(r.URL.Query().Get("module")),
		Success: queryInt(r, "success", -1),
		Page:    page,
		Size:    size,
	})
	if err != nil {
		return nil, serverError("查询操作日志失败: " + err.Error())
	}
	return map[string]interface{}{"list": list, "total": total, "page": page, "size": size}, nil
}

// logClear 清空日志，scope=login/operation/all
func logClear(c *adminCtx, r *http.Request) (interface{}, error) {
	req := struct {
		Scope string `json:"scope"`
	}{}
	if err := decodeBody(r, &req); err != nil {
		return nil, err
	}
	cleared := make([]string, 0, 2)
	switch req.Scope {
	case "login":
		if err := dao.ClearLoginLogs(); err != nil {
			return nil, serverError("清空登录日志失败: " + err.Error())
		}
		cleared = append(cleared, "login")
	case "operation":
		if err := dao.ClearOperationLogs(); err != nil {
			return nil, serverError("清空操作日志失败: " + err.Error())
		}
		cleared = append(cleared, "operation")
	case "all", "":
		if err := dao.ClearLoginLogs(); err != nil {
			return nil, serverError("清空登录日志失败: " + err.Error())
		}
		if err := dao.ClearOperationLogs(); err != nil {
			return nil, serverError("清空操作日志失败: " + err.Error())
		}
		cleared = append(cleared, "login", "operation")
	default:
		return nil, badRequest("不支持的清空范围: " + req.Scope)
	}
	return map[string]interface{}{"cleared": cleared}, nil
}
