package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
)

// apiError 业务错误，携带 HTTP 状态码
type apiError struct {
	Status int
	Msg    string
}

func (e *apiError) Error() string { return e.Msg }

func badRequest(msg string) error   { return &apiError{Status: http.StatusBadRequest, Msg: msg} }
func unauthorized(msg string) error { return &apiError{Status: http.StatusUnauthorized, Msg: msg} }
func forbidden(msg string) error    { return &apiError{Status: http.StatusForbidden, Msg: msg} }
func notFound(msg string) error     { return &apiError{Status: http.StatusNotFound, Msg: msg} }
func serverError(msg string) error {
	return &apiError{Status: http.StatusInternalServerError, Msg: msg}
}

// writeJSON 输出 JSON
func writeJSON(w http.ResponseWriter, status int, res *JsonResult) {
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

// writeOK 输出成功结果
func writeOK(w http.ResponseWriter, data interface{}) {
	writeJSON(w, http.StatusOK, &JsonResult{Code: 0, Data: data})
}

// writeError 输出错误结果
func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	msg := err.Error()
	var ae *apiError
	if errors.As(err, &ae) {
		status = ae.Status
		msg = ae.Msg
	}
	writeJSON(w, status, &JsonResult{Code: status, ErrorMsg: msg})
}

// decodeBody 解析请求体
func decodeBody(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	if r.ContentLength == 0 {
		return badRequest("请求体为空")
	}
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return badRequest("请求体格式错误: " + err.Error())
	}
	return nil
}

// clientIP 取真实客户端 IP，兼容云托管转发
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if ip := strings.TrimSpace(parts[0]); ip != "" {
			return ip
		}
	}
	if xri := r.Header.Get("X-Real-Ip"); xri != "" {
		return xri
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// queryInt 读取整型查询参数
func queryInt(r *http.Request, key string, def int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return n
}

// queryInt64 读取 int64 查询参数
func queryInt64(r *http.Request, key string, def int64) int64 {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return def
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return def
	}
	return n
}

// queryIntAll 读取整型查询参数，支持 -1 表示全部
func queryIntAll(r *http.Request, key string, def int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return n
}

// pageParams 归一化分页参数
func pageParams(r *http.Request) (page, size int) {
	page = queryInt(r, "page", 1)
	size = queryInt(r, "size", 10)
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	if size > 200 {
		size = 200
	}
	return
}
