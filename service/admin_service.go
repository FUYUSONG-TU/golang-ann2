package service

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"wxcloudrun-golang/db"
	"wxcloudrun-golang/db/dao"
	"wxcloudrun-golang/db/model"

	"gorm.io/gorm"
)

// adminVersion 后台管理接口版本
const adminVersion = "2.0.0"

// defaultCounterID 模板默认使用的计数器主键
const defaultCounterID int32 = 1

var adminStartAt = time.Now()

// AdminAPIHandler 后台管理接口的统一入口，路径前缀 /api/admin
func AdminAPIHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/admin"), "/")
	if path == "" {
		path = "/"
	}

	if !adminAuthorized(r) {
		writeAdminJSON(w, http.StatusUnauthorized, &JsonResult{Code: 401, ErrorMsg: "未授权：请提供正确的管理密钥"})
		return
	}

	switch path {
	case "/overview":
		if !requireAdminMethod(w, r, http.MethodGet) {
			return
		}
		adminOverview(w, r)
	case "/counters":
		switch r.Method {
		case http.MethodGet:
			adminListCounters(w, r)
		case http.MethodPost:
			adminCreateCounter(w, r)
		default:
			writeAdminJSON(w, http.StatusMethodNotAllowed, &JsonResult{Code: 405, ErrorMsg: "请求方法 " + r.Method + " 不支持"})
		}
	case "/counters/incr":
		if !requireAdminMethod(w, r, http.MethodPost) {
			return
		}
		adminIncrCounter(w, r)
	case "/counters/set":
		if !requireAdminMethod(w, r, http.MethodPost) {
			return
		}
		adminSetCounter(w, r)
	case "/counters/delete":
		if !requireAdminMethod(w, r, http.MethodPost) {
			return
		}
		adminDeleteCounter(w, r)
	default:
		writeAdminJSON(w, http.StatusNotFound, &JsonResult{Code: 404, ErrorMsg: "接口不存在: " + path})
	}
}

// adminOverview 概览数据
func adminOverview(w http.ResponseWriter, r *http.Request) {
	list, err := dao.Imp.ListCounters()
	if err != nil {
		writeAdminJSON(w, http.StatusInternalServerError, &JsonResult{Code: 500, ErrorMsg: "读取计数记录失败: " + err.Error()})
		return
	}

	current := 0
	for _, item := range list {
		if item.Id == defaultCounterID {
			current = int(item.Count)
			break
		}
	}

	latency, pingErr := adminDBLatency()
	dbOK := pingErr == nil

	var updatedAt interface{}
	if len(list) > 0 {
		latest := list[0]
		for _, item := range list {
			if item.UpdatedAt.After(latest.UpdatedAt) {
				latest = item
			}
		}
		updatedAt = latest.UpdatedAt
	}

	writeAdminJSON(w, http.StatusOK, &JsonResult{Code: 0, Data: map[string]interface{}{
		"count":            current,
		"total":            len(list),
		"lastUpdatedAt":    updatedAt,
		"dbOk":             dbOK,
		"dbLatencyMs":      latency,
		"dbError":          adminErrText(pingErr),
		"serverTime":       time.Now(),
		"timeZone":         adminTimeZone(),
		"uptimeSeconds":    int(time.Since(adminStartAt).Seconds()),
		"version":          adminVersion,
		"authRequired":     adminToken() != "",
		"defaultCounterId": defaultCounterID,
	}})
}

// adminListCounters 列出全部计数记录
func adminListCounters(w http.ResponseWriter, r *http.Request) {
	list, err := dao.Imp.ListCounters()
	if err != nil {
		writeAdminJSON(w, http.StatusInternalServerError, &JsonResult{Code: 500, ErrorMsg: "读取计数记录失败: " + err.Error()})
		return
	}
	writeAdminJSON(w, http.StatusOK, &JsonResult{Code: 0, Data: map[string]interface{}{
		"list":  list,
		"total": len(list),
	}})
}

// adminIncrCounter 指定记录自增
func adminIncrCounter(w http.ResponseWriter, r *http.Request) {
	req := struct {
		Id   *int32 `json:"id"`
		Step *int32 `json:"step"`
	}{}
	if err := adminDecode(r, &req); err != nil {
		writeAdminJSON(w, http.StatusBadRequest, &JsonResult{Code: 400, ErrorMsg: err.Error()})
		return
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
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		writeAdminJSON(w, http.StatusInternalServerError, &JsonResult{Code: 500, ErrorMsg: err.Error()})
		return
	}

	now := time.Now()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if id != defaultCounterID {
			writeAdminJSON(w, http.StatusNotFound, &JsonResult{Code: 404, ErrorMsg: "记录不存在"})
			return
		}
		counter = &model.CounterModel{Id: id, Count: step, CreatedAt: now, UpdatedAt: now}
	} else {
		counter.Count += step
		counter.UpdatedAt = now
	}

	if err := dao.Imp.UpsertCounter(counter); err != nil {
		writeAdminJSON(w, http.StatusInternalServerError, &JsonResult{Code: 500, ErrorMsg: "写入失败: " + err.Error()})
		return
	}
	writeAdminJSON(w, http.StatusOK, &JsonResult{Code: 0, Data: counter})
}

// adminSetCounter 把指定记录设置为固定值
func adminSetCounter(w http.ResponseWriter, r *http.Request) {
	req := struct {
		Id    *int32 `json:"id"`
		Count *int32 `json:"count"`
	}{}
	if err := adminDecode(r, &req); err != nil {
		writeAdminJSON(w, http.StatusBadRequest, &JsonResult{Code: 400, ErrorMsg: err.Error()})
		return
	}
	if req.Count == nil {
		writeAdminJSON(w, http.StatusBadRequest, &JsonResult{Code: 400, ErrorMsg: "缺少 count 参数"})
		return
	}
	if *req.Count < 0 {
		writeAdminJSON(w, http.StatusBadRequest, &JsonResult{Code: 400, ErrorMsg: "count 不能为负数"})
		return
	}

	id := defaultCounterID
	if req.Id != nil {
		id = *req.Id
	}

	counter, err := dao.Imp.GetCounter(id)
	now := time.Now()
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			writeAdminJSON(w, http.StatusInternalServerError, &JsonResult{Code: 500, ErrorMsg: err.Error()})
			return
		}
		if id != defaultCounterID {
			writeAdminJSON(w, http.StatusNotFound, &JsonResult{Code: 404, ErrorMsg: "记录不存在"})
			return
		}
		counter = &model.CounterModel{Id: id, CreatedAt: now}
	}

	counter.Count = *req.Count
	counter.UpdatedAt = now
	if err := dao.Imp.UpsertCounter(counter); err != nil {
		writeAdminJSON(w, http.StatusInternalServerError, &JsonResult{Code: 500, ErrorMsg: "写入失败: " + err.Error()})
		return
	}
	writeAdminJSON(w, http.StatusOK, &JsonResult{Code: 0, Data: counter})
}

// adminDeleteCounter 删除指定记录
func adminDeleteCounter(w http.ResponseWriter, r *http.Request) {
	req := struct {
		Id *int32 `json:"id"`
	}{}
	if err := adminDecode(r, &req); err != nil {
		writeAdminJSON(w, http.StatusBadRequest, &JsonResult{Code: 400, ErrorMsg: err.Error()})
		return
	}
	if req.Id == nil {
		writeAdminJSON(w, http.StatusBadRequest, &JsonResult{Code: 400, ErrorMsg: "缺少 id 参数"})
		return
	}
	if err := dao.Imp.ClearCounter(*req.Id); err != nil {
		writeAdminJSON(w, http.StatusInternalServerError, &JsonResult{Code: 500, ErrorMsg: "删除失败: " + err.Error()})
		return
	}
	writeAdminJSON(w, http.StatusOK, &JsonResult{Code: 0, Data: map[string]interface{}{"id": *req.Id}})
}

// adminCreateCounter 新建一条计数记录
func adminCreateCounter(w http.ResponseWriter, r *http.Request) {
	req := struct {
		Count *int32 `json:"count"`
	}{}
	// 允许空请求体，此时按 count=0 创建
	if r.ContentLength != 0 {
		if err := adminDecode(r, &req); err != nil {
			writeAdminJSON(w, http.StatusBadRequest, &JsonResult{Code: 400, ErrorMsg: err.Error()})
			return
		}
	}

	count := int32(0)
	if req.Count != nil {
		count = *req.Count
	}
	if count < 0 {
		writeAdminJSON(w, http.StatusBadRequest, &JsonResult{Code: 400, ErrorMsg: "count 不能为负数"})
		return
	}

	now := time.Now()
	counter := &model.CounterModel{Count: count, CreatedAt: now, UpdatedAt: now}
	if err := dao.Imp.UpsertCounter(counter); err != nil {
		writeAdminJSON(w, http.StatusInternalServerError, &JsonResult{Code: 500, ErrorMsg: "创建失败: " + err.Error()})
		return
	}
	writeAdminJSON(w, http.StatusOK, &JsonResult{Code: 0, Data: counter})
}

// ---------- 内部辅助 ----------

func requireAdminMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		writeAdminJSON(w, http.StatusMethodNotAllowed, &JsonResult{Code: 405, ErrorMsg: "请求方法 " + r.Method + " 不支持"})
		return false
	}
	return true
}

func adminToken() string {
	return strings.TrimSpace(os.Getenv("ADMIN_TOKEN"))
}

// adminAuthorized 未配置 ADMIN_TOKEN 时开放访问，配置后必须携带密钥
func adminAuthorized(r *http.Request) bool {
	token := adminToken()
	if token == "" {
		return true
	}
	provided := r.Header.Get("X-Admin-Token")
	if provided == "" {
		provided = r.URL.Query().Get("token")
	}
	if provided == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(token)) == 1
}

func adminDecode(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	if r.ContentLength == 0 {
		return errors.New("请求体为空")
	}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(v); err != nil {
		return errors.New("请求体格式错误: " + err.Error())
	}
	return nil
}

func adminDBLatency() (int64, error) {
	start := time.Now()
	cli := db.Get()
	if cli == nil {
		return 0, errors.New("数据库未初始化")
	}
	sqlDB, err := cli.DB()
	if err != nil {
		return 0, err
	}
	if err := sqlDB.Ping(); err != nil {
		return time.Since(start).Milliseconds(), err
	}
	return time.Since(start).Milliseconds(), nil
}

func adminTimeZone() string {
	name, offset := time.Now().Zone()
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	return fmt.Sprintf("%s UTC%s%02d:%02d", name, sign, offset/3600, (offset%3600)/60)
}

func adminErrText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func writeAdminJSON(w http.ResponseWriter, status int, res *JsonResult) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	msg, err := json.Marshal(res)
	if err != nil {
		fmt.Fprint(w, `{"code":500,"errorMsg":"结果序列化失败"}`)
		return
	}
	w.Write(msg)
}
