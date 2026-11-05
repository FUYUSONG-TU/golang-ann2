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
	if err := db.Init(); err != nil {
		panic(fmt.Sprintf("mysql init failed with %+v", err))
	}

	http.HandleFunc("/", service.IndexHandler)
	http.HandleFunc("/api/count", service.CounterHandler)

	// 后台管理页与接口
	http.HandleFunc("/admin", service.AdminPageHandler)
	http.HandleFunc("/admin/", service.AdminPageHandler)
	http.HandleFunc("/api/admin/", service.AdminAPIHandler)

	// 容器内默认监听 80 端口；本地调试可用 PORT 覆盖
	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}

	log.Printf("server listening on :%s, admin console at /admin", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
