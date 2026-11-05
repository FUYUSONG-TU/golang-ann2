package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"wxcloudrun-golang/db"
	"wxcloudrun-golang/service"
)

func main() {
	// 令牌签名密钥必须在提供接口之前就绪
	service.InitSecurity()

	if err := db.Init(); err != nil {
		panic(fmt.Sprintf("mysql init failed with %+v", err))
	}

	// 接口路由
	http.HandleFunc("/api/count", service.CounterHandler)
	http.HandleFunc("/api/admin/", service.AdminAPIHandler)
	http.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write([]byte(`{"code":0,"data":{"status":"ok"}}`))
	})

	// 前端（Vue 构建产物已嵌入二进制），未命中的路径交由 vue-router 处理
	spa, err := newSPAHandler()
	if err != nil {
		log.Fatalf("加载前端资源失败: %v", err)
	}
	http.Handle("/", spa)

	// 容器内默认监听 80 端口；本地调试可用 PORT 覆盖
	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}

	log.Printf("机匠学徒后台已启动，监听 :%s，版本 %s", port, service.Version())
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
