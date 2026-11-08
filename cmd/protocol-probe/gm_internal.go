package main

import (
	"crypto/subtle"
	"log"
	"net/http"
	"time"

	"github.com/local/9yin-go-server/internal/gmapi"
)

// gmInternalVersion 面板健康检查里的版本串。build.bat 经 -ldflags
// 注入编译时间戳，用于面板检测两进程版本不匹配。
var gmInternalVersion = "dev"

// gmHealthInfo 填进 /internal/v1/health 的进程级信息（Java GmRemote
// status RPC 同形：running/listen_addr/server_root/connected_clients/
// uptime_sec/phase），供管理页卡片展示。
type gmHealthInfo struct {
	ListenAddr string // 游戏 TCP 监听（-listen）
	ServerRoot string // NINEYIN_SERVER_ROOT 解析结果
}

// serveGMInternal 起独立面板进程专用的 internal API（docs/gm-service-rpc-split-design.md §5）。
// 刻意只应绑 loopback；鉴权 = Bearer token 常数时间比对。命令仍经由
// hub 的会话命令通道交接，internal API handler 绝不直接写客户端帧
//（单写者纪律）。
func serveGMInternal(listen, token string, hub *gmHub, info gmHealthInfo) {
	server := &http.Server{
		Addr:              listen,
		Handler:           newGMInternalMux(token, hub, info),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("GM internal API listening on http://%s (version %s)", listen, gmInternalVersion)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("GM internal API stopped: %v", err)
	}
}

// newGMInternalMux 组装 /internal/v1 路由。独立成函数供 httptest 直接
// 驱动（鉴权、命令往返、目录端点都有行为级测试，见 gm_internal_test.go）。
func newGMInternalMux(token string, hub *gmHub, info gmHealthInfo) http.Handler {
	mux := http.NewServeMux()
	started := time.Now()
	mux.HandleFunc("/internal/v1/health", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		hub.mu.RLock()
		online := len(hub.sessions)
		hub.mu.RUnlock()
		gmapi.WriteJSON(writer, http.StatusOK, map[string]any{
			"status":     "ok",
			"version":    gmInternalVersion,
			"uptimeSec":  int(time.Since(started).Seconds()),
			"sessions":   online,
			// 以下三字段与 Java GmRemote status RPC 同形，供管理页
			//（index.html 记录服务器卡片）展示；running 恒 true——
			// 能应答即在线。
			"running":           true,
			"listenAddr":        info.ListenAddr,
			"serverRoot":        info.ServerRoot,
		})
	})
	mux.HandleFunc("/internal/v1/sessions", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		gmapi.WriteJSON(writer, http.StatusOK, hub.snapshots())
	})
	mux.HandleFunc("/internal/v1/sessions/", func(writer http.ResponseWriter, request *http.Request) {
		hub.serveSessionCommand("/internal/v1/sessions/", "internal", writer, request)
	})
	// 目录端点与内嵌面板共用同一批 handler：数据源（客户端 form 资源、
	// game_items 名字批查、皮肤表）都在游戏服进程里，独立面板只代理。
	mux.HandleFunc("/internal/v1/catalog/fashion", hub.fashionCatalogHandler)
	mux.HandleFunc("/internal/v1/catalog/cards", hub.cardsHandler)
	mux.HandleFunc("/internal/v1/catalog/hair-cards", hub.hairCardsHandler)
	mux.HandleFunc("/internal/v1/catalog/items", hub.itemsHandler)
	mux.HandleFunc("/internal/v1/catalog/meridians", hub.meridiansHandler)
	return withBearerAuth(token, mux)
}

// withBearerAuth 给整个 mux 套 Bearer 鉴权；常数时间比对防时序侧信道。
// 唯一例外是 /internal/v1/health：探活不带鉴权，面板用它区分“游戏服
// 不在”与“token 错”——前者等，后者找运维。
func withBearerAuth(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/internal/v1/health" {
			next.ServeHTTP(writer, request)
			return
		}
		got := ""
		if header := request.Header.Get("Authorization"); len(header) > 7 && header[:7] == "Bearer " {
			got = header[7:]
		}
		if subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
			gmapi.WriteError(writer, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(writer, request)
	})
}
