package gmapi

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// 管理页「记录服务器」登记册的服务端支撑（docs/gm-service-rpc-split-design.md
// §实施记录）。index.html 的登记册存浏览器 localStorage（Java gm-web 同款），
// 卡片探活经本包的同源代理打到目标游戏服的免鉴权 /internal/v1/health——
// 浏览器跨端口会被 CORS 拦，必须由面板进程代探。
//
// MountManageAPI 由两种面板形态共用：独立面板（gm-panel.exe，绑定
// -game-addr）与游戏服内嵌面板（-gm-listen，绑定 -gm-internal-listen）。

// statusProbeTimeout 是单目标探活超时；管理页每 3s 轮询全部卡片，
// 未响应的服不该拖住整页。
const statusProbeTimeout = 1500 * time.Millisecond

var manageProbeClient = &http.Client{Timeout: statusProbeTimeout}

// validManageHost 与 index.html 的 validHost 同一口径：只收 IP/localhost，
// 不放行任意 URL（探活代理会让面板向该地址发起请求，必须收紧输入面）。
var manageHostPattern = regexp.MustCompile(`(?i)^(localhost|(\d{1,3}\.){3}\d{1,3})$`)

// parseManageAddress 解析 "host:port"；host 仅 IP/localhost，port 1..65535。
func parseManageAddress(s string) (string, bool) {
	host, port, err := net.SplitHostPort(strings.TrimSpace(s))
	if err != nil || !manageHostPattern.MatchString(host) {
		return "", false
	}
	value, err := strconv.Atoi(port)
	if err != nil || value < 1 || value > 65535 {
		return "", false
	}
	return net.JoinHostPort(host, port), true
}

// MountManageAPI 挂两个管理页端点：
//
//	GET /api/manage/status?s=host:port —— Java gm-web 网关同名端点的
//	  面板内对应物。把目标服 health 映射回 Java 的 {online, status:{
//	  running, listen_addr, server_root, connected_clients, uptime_sec,
//	  phase}} 形状，登记册卡片逐字段消费。只打 health，不带 token；
//	  非本系 Go 服（无 internal API）表现为 online:false。
//	GET /api/panel-info —— 登记册据此判断哪张卡片是本面板所连
//	  （进入控制台只对它有意义），其余卡片仅监控。
func MountManageAPI(mux *http.ServeMux, boundGameAddr, version string) {
	mux.HandleFunc("/api/manage/status", func(writer http.ResponseWriter, request *http.Request) {
		manageStatusProxy(writer, request)
	})
	mux.HandleFunc("/api/panel-info", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		WriteJSON(writer, http.StatusOK, map[string]any{
			"gameAddr":     boundGameAddr,
			"panelVersion": version,
		})
	})
}

func manageStatusProxy(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	target, ok := parseManageAddress(request.URL.Query().Get("s"))
	if !ok {
		WriteError(writer, http.StatusBadRequest, "invalid s, expect host:port")
		return
	}
	probe, cancel := context.WithTimeout(request.Context(), statusProbeTimeout)
	defer cancel()
	probeRequest, err := http.NewRequestWithContext(probe, http.MethodGet,
		"http://"+target+"/internal/v1/health", nil)
	if err != nil {
		WriteError(writer, http.StatusBadRequest, "invalid s, expect host:port")
		return
	}
	response, err := manageProbeClient.Do(probeRequest)
	if err != nil {
		WriteJSON(writer, http.StatusOK, map[string]any{"online": false})
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		WriteJSON(writer, http.StatusOK, map[string]any{"online": false})
		return
	}
	var health struct {
		Status     string `json:"status"`
		Version    string `json:"version"`
		UptimeSec  int    `json:"uptimeSec"`
		Sessions   int    `json:"sessions"`
		Running    bool   `json:"running"`
		ListenAddr string `json:"listenAddr"`
		ServerRoot string `json:"serverRoot"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 64<<10)).Decode(&health); err != nil ||
		health.Status != "ok" {
		WriteJSON(writer, http.StatusOK, map[string]any{"online": false})
		return
	}
	WriteJSON(writer, http.StatusOK, map[string]any{
		"online": true,
		"status": map[string]any{
			"running":           true,
			"listen_addr":       health.ListenAddr,
			"server_root":       health.ServerRoot,
			"connected_clients": health.Sessions,
			"uptime_sec":        health.UptimeSec,
			"phase":             "版本 " + health.Version,
		},
	})
}
