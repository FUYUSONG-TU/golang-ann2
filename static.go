package main

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// webDist 前端构建产物（Vite 输出到 web/dist，构建时嵌入二进制）
//
//go:embed web/dist
var webDist embed.FS

// newSPAHandler 返回前端静态资源处理器。
// 命中真实文件则直接返回；其余路径回退到 index.html，交由 vue-router 处理。
func newSPAHandler() (http.Handler, error) {
	sub, err := fs.Sub(webDist, "web/dist")
	if err != nil {
		return nil, err
	}

	index, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		return nil, err
	}

	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 未注册的接口路径返回 404，不要回退成页面
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		upath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if upath == "" || upath == "." {
			writeIndex(w, index)
			return
		}

		if st, statErr := fs.Stat(sub, upath); statErr == nil && !st.IsDir() {
			if strings.HasPrefix(upath, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			fileServer.ServeHTTP(w, r)
			return
		}

		writeIndex(w, index)
	}), nil
}

func writeIndex(w http.ResponseWriter, index []byte) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	w.Write(index)
}
