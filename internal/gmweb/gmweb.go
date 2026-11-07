// Package gmweb 持有 GM 面板的全部网页文件（web/index.html、
// web/console.html 与 static/ 样式脚本），编译期嵌入，运行期不依赖
// 磁盘上的任何前端资源。内嵌面板（cmd/protocol-probe）与独立面板
//（cmd/gm-panel）共享同一份资产，保证两种部署形态的页面完全一致。
package gmweb

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
)

//go:embed web
var assets embed.FS

// Mount 把页面路由挂到 mux 上："/" 与 "/console.html" 直出 HTML，
// "/static/" 映射嵌入的 web/static 子目录（不是 web 根，与拆分前的
// 行为一致）。
func Mount(mux *http.ServeMux) {
	static, err := fs.Sub(assets, "web/static")
	if err != nil {
		log.Fatalf("GM static assets unavailable: %v", err)
	}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(static))))
	mux.HandleFunc("/", index)
}

// index 直出两个已知页面；其余路径 404。
func index(writer http.ResponseWriter, request *http.Request) {
	page, known := map[string]string{
		"/":             "web/index.html",
		"/console.html": "web/console.html",
	}[request.URL.Path]
	if !known {
		writer.WriteHeader(http.StatusNotFound)
		return
	}
	content, err := assets.ReadFile(page)
	if err != nil {
		http.Error(writer, "GM panel index missing", http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = writer.Write(content)
}
