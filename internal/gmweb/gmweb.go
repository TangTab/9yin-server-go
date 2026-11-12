// Package gmweb 持有 GM 面板的全部网页文件（web/index.html、
// web/console.html、web/console-ax.html 与 static/ 样式脚本），编译期
// 嵌入，运行期不依赖磁盘上的任何前端资源。内嵌面板（cmd/protocol-probe）
// 与独立面板（cmd/gm-panel）共享同一份资产，保证两种部署形态的页面完全一致。
package gmweb

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
)

//go:embed web
var assets embed.FS

// noCacheStatic 给 /static/ 响应加 Cache-Control: no-cache。embed.FS 的
// 文件没有 Last-Modified/ETag，浏览器会按启发式缓存把 JS 模块缓存死——
// 重新部署后页面拿到旧脚本（2026-11-12 实测：修复后的 skill.js 不生效）。
// no-cache 要求浏览器每次回源验证，本地面板往返成本可忽略，正确性优先。
// ax 三件套（4 万行的 ax.esm.js 等）体积大但同源局域网，代价可接受；
// 后续若在意可按文件名细分策略。
func noCacheStatic(files http.FileSystem) http.Handler {
	fileServer := http.StripPrefix("/static/", http.FileServer(files))
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Cache-Control", "no-cache")
		fileServer.ServeHTTP(writer, request)
	})
}

// Mount 把页面路由挂到 mux 上："/" 与已知页面直出 HTML，
// "/static/" 映射嵌入的 web/static 子目录（不是 web 根，与拆分前的
// 行为一致）。
func Mount(mux *http.ServeMux) {
	static, err := fs.Sub(assets, "web/static")
	if err != nil {
		log.Fatalf("GM static assets unavailable: %v", err)
	}
	mux.Handle("/static/", noCacheStatic(http.FS(static)))
	mux.HandleFunc("/", index)
}

// index 直出已知页面；其余路径 404。/console-ax.html 是 AXUI 重写版
// （docs/gm-console-axui-redesign-design.md），与旧版并行存在直至迁移完成。
func index(writer http.ResponseWriter, request *http.Request) {
	page, known := map[string]string{
		"/":                "web/index.html",
		"/console.html":    "web/console.html",
		"/console-ax.html": "web/console-ax.html",
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
	writer.Header().Set("Cache-Control", "no-cache")
	_, _ = writer.Write(content)
}
