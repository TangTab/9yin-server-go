// gm-panel 是 GM 面板的独立进程形态（docs/gm-service-rpc-split-design.md）。
//
// 它自己不持有任何游戏状态：页面与静态资源来自 internal/gmweb 的
// 编译期嵌入资产，/api 请求按契约代理到游戏服的 internal API
// （/internal/v1，Bearer token）。游戏服重启对它只是普通网络故障——
// 代理返回 503，页面顶栏提示，游戏服回来后自动恢复。
//
// 命令路径的语义与内嵌面板逐字一致：面板先用 internal/gmapi 的
// 同一份校验做快速 400（省一次网络往返），游戏服侧仍权威校验；
// 状态码与错误体原样透传，超时绝不被描述成成功。
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/local/9yin-go-server/internal/gmapi"
	"github.com/local/9yin-go-server/internal/gmweb"
)

// panelVersion 面板自身的版本串（build.bat 经 -ldflags 注入编译
// 时间戳）。与游戏服 /internal/v1/health 的 version 比对，两进程
// 应同批部署；不一致只记警告，不阻断服务。
var panelVersion = "dev"

// sessionsCacheTTL 是会话列表的微缓存窗口：概览页与控制台同时轮询
// 时去抖，游戏服不被多个浏览器标签页叠加打挂。
const sessionsCacheTTL = 500 * time.Millisecond

// catalogCacheTTL 目录代理的微缓存窗口（设计文档 §5.1 S5）。目录在
// 游戏服进程内不变，30s 足够长；游戏服重启由 ETag 失配自然纠正。
const catalogCacheTTL = 30 * time.Second

// commandTimeout 覆盖 internal API 的最坏时延（入队 750ms + 等
// 回执 2s + 余量，设计文档 §5.3）。
const commandTimeout = 3 * time.Second

// commandBodyLimit 与游戏服 internal API 的载荷闸一致（lua_eval 源码
// 16KB + JSON 开销）。
const commandBodyLimit = 64 << 10

type panelServer struct {
	gameBase    string // 游戏服 internal API 基址（http://host:port）
	gameAddr    string // game-addr 原值（host:port，面板信息端点展示用）
	token       string
	client      *http.Client
	probeClient *http.Client // 管理页卡片探活：免鉴权 health，短超时

	mu         sync.Mutex
	cachedAt   time.Time
	cachedBody []byte
	cachedCode int
	cachedETag string

	// catalogCache 目录代理微缓存（S5）：key = 目录名+查询串。
	catMu    sync.Mutex
	catCache map[string]catalogCacheEntry
}

// catalogCacheEntry 目录代理缓存条目。code=304 表示后端对某客户端
// INM 回了 304——此时 body 为空、etag 保留，后续同 INM 请求直接 304。
type catalogCacheEntry struct {
	at   time.Time
	code int
	body []byte
	etag string
}

func main() {
	listen := flag.String("listen", "127.0.0.1:19064", "GM panel HTTP listen address")
	gameAddr := flag.String("game-addr", "127.0.0.1:19063", "game server internal API address")
	gameToken := flag.String("game-token", os.Getenv("NINEYIN_GM_TOKEN"), "bearer token for the game server internal API (defaults to NINEYIN_GM_TOKEN)")
	flag.Parse()
	if strings.TrimSpace(*gameToken) == "" {
		log.Fatal("gm-panel requires -game-token (or NINEYIN_GM_TOKEN); the game server internal API rejects unauthenticated callers")
	}
	panel := &panelServer{
		gameBase: "http://" + *gameAddr,
		gameAddr: *gameAddr,
		token:    *gameToken,
		client:   &http.Client{Timeout: commandTimeout},
		catCache: make(map[string]catalogCacheEntry),
	}
	panel.logGameHealth()
	server := &http.Server{Addr: *listen, Handler: panel.mux(), ReadHeaderTimeout: 5 * time.Second}
	log.Printf("GM panel (standalone) listening on http://%s → game internal API at %s", *listen, *gameAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("GM panel stopped: %v", err)
	}
}

// logGameHealth 启动时探一次游戏服并记日志；探不到不阻止启动——
// 面板允许先于游戏服存在。
func (p *panelServer) logGameHealth() {
	response, err := p.client.Get(p.gameBase + "/internal/v1/health")
	if err != nil {
		log.Printf("game server not reachable yet (%v); panel will keep serving and retry per request", err)
		return
	}
	defer response.Body.Close()
	var health struct {
		Version  string `json:"version"`
		Sessions int    `json:"sessions"`
	}
	_ = json.NewDecoder(response.Body).Decode(&health)
	log.Printf("game server internal API up: version=%s sessions=%d", health.Version, health.Sessions)
	if health.Version != "" && panelVersion != "dev" && health.Version != panelVersion {
		log.Printf("WARNING: panel version %s != game server version %s; redeploy both together", panelVersion, health.Version)
	}
}

func (p *panelServer) mux() *http.ServeMux {
	mux := http.NewServeMux()
	gmweb.Mount(mux)
	// 天气目录是契约层的静态闭名单，不经游戏服直出。
	mux.HandleFunc("/api/weather", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		tag := gmapi.WeatherTag()
		etag := gmapi.ETagValue(tag)
		if gmapi.ETagMatches(request.Header.Get("If-None-Match"), etag) {
			gmapi.WriteNotModified(writer)
			return
		}
		gmapi.SetETag(writer, tag)
		gmapi.WriteJSON(writer, http.StatusOK, gmapi.WeatherCatalog())
	})
	mux.HandleFunc("/api/sessions", p.sessionsProxy)
	mux.HandleFunc("/api/sessions/", p.commandProxy)
	// 目录端点依赖游戏服进程内的资源与 game_items 名字，全部代理。
	mux.HandleFunc("/api/fashion-catalog", p.catalogProxy("fashion"))
	mux.HandleFunc("/api/cards", p.catalogProxy("cards"))
	mux.HandleFunc("/api/hair-cards", p.catalogProxy("hair-cards"))
	mux.HandleFunc("/api/items", p.catalogProxy("items"))
	mux.HandleFunc("/api/meridians", p.catalogProxy("meridians"))
	// console 页启动即拉取的补充目录/状态端点（游戏服 gm_catalog_dirs.go
	// 同名 handler 的镜像）；minimal-login 有 POST 切换，走 methodProxy。
	mux.HandleFunc("/api/clones", p.catalogProxy("clones"))
	mux.HandleFunc("/api/scenes", p.catalogProxy("scenes"))
	mux.HandleFunc("/api/giftboxes", p.catalogProxy("giftboxes"))
	mux.HandleFunc("/api/titles", p.catalogProxy("titles"))
	mux.HandleFunc("/api/mounts", p.catalogProxy("mounts"))
	mux.HandleFunc("/api/idle-stances", p.catalogProxy("idle-stances"))
	mux.HandleFunc("/api/recast-affixes", p.catalogProxy("recast-affixes"))
	mux.HandleFunc("/api/combat-dummy/taolus", p.catalogProxy("combat-dummy/taolus"))
	mux.HandleFunc("/api/taolus", p.catalogProxy("taolus"))
	mux.HandleFunc("/api/neigongs", p.catalogProxy("neigongs"))
	mux.HandleFunc("/api/repute/status", p.catalogProxy("repute/status"))
	mux.HandleFunc("/api/origin/status", p.catalogProxy("origin/status"))
	mux.HandleFunc("/api/minimal-login", p.methodProxy("/internal/v1/minimal-login"))
	mux.HandleFunc("/api/catalog/manifest", p.catalogProxy("manifest"))
	// 管理页（index.html 记录服务器登记册）：同源探活代理 + 面板自述。
	// 实现在契约层，与游戏服内嵌面板共用同一份代码。
	gmapi.MountManageAPI(mux, p.gameAddr, panelVersion)
	return mux
}

// gameRequest 发一次带鉴权的 internal API 请求，返回状态码、原始 body
// 与响应 ETag（无则空串）；传输层失败（游戏服不在/超时）以 ok=false
// 报告，由调用方映射 503。inm 非空时作为 If-None-Match 转发给游戏服
//（GET 目录/会话的 304 协商）。
func (p *panelServer) gameRequest(method, path, query string, body []byte, inm string) (int, []byte, string, bool) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	request, err := http.NewRequest(method, p.gameBase+path, reader)
	if err != nil {
		return 0, nil, "", false
	}
	request.Header.Set("Authorization", "Bearer "+p.token)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if inm != "" {
		request.Header.Set("If-None-Match", inm)
	}
	if query != "" {
		request.URL.RawQuery = query
	}
	response, err := p.client.Do(request)
	if err != nil {
		return 0, nil, "", false
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return 0, nil, "", false
	}
	return response.StatusCode, raw, response.Header.Get("ETag"), true
}

func writeProxyJSON(writer http.ResponseWriter, status int, body []byte, etag string) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	if etag != "" {
		writer.Header().Set("ETag", etag)
	}
	writer.WriteHeader(status)
	_, _ = writer.Write(body)
}

// sessionsProxy 代理会话列表，带 sessionsCacheTTL 微缓存。缓存连
// 503 一起缓存：游戏服重启期间，多标签页的轮询不会叠加打到坏连接上。
func (p *panelServer) sessionsProxy(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	inm := request.Header.Get("If-None-Match")
	p.mu.Lock()
	fresh := time.Since(p.cachedAt) < sessionsCacheTTL
	body, code, etag := p.cachedBody, p.cachedCode, p.cachedETag
	p.mu.Unlock()
	if !fresh {
		status, raw, gameETag, ok := p.gameRequest(http.MethodGet, "/internal/v1/sessions", "", nil, inm)
		if !ok {
			status, raw = http.StatusServiceUnavailable, []byte(`{"error":"game server unreachable"}`)
		}
		body, code, etag = raw, status, gameETag
		p.mu.Lock()
		p.cachedAt, p.cachedBody, p.cachedCode, p.cachedETag = time.Now(), body, code, etag
		p.mu.Unlock()
	}
	// 游戏服（或微缓存条目）对客户端 INM 命中 → 304 空体直回。
	if code == http.StatusNotModified || (etag != "" && gmapi.ETagMatches(inm, etag)) {
		gmapi.WriteNotModified(writer)
		return
	}
	writeProxyJSON(writer, code, body, etag)
}

// commandProxy 命令通道：本地快速校验（与游戏服同一份 internal/gmapi
// 实现）→ 原样转发 → 状态码与错误体逐字透传。传输失败 503，绝不报成功。
func (p *panelServer) commandProxy(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost || !strings.HasSuffix(request.URL.Path, "/command") {
		writer.WriteHeader(http.StatusNotFound)
		return
	}
	idText := strings.TrimSuffix(strings.TrimPrefix(request.URL.Path, "/api/sessions/"), "/command")
	id, err := strconv.ParseUint(idText, 10, 64)
	if err != nil {
		gmapi.WriteError(writer, http.StatusBadRequest, "invalid session id")
		return
	}
	raw, err := io.ReadAll(http.MaxBytesReader(writer, request.Body, commandBodyLimit))
	if err != nil {
		gmapi.WriteError(writer, http.StatusBadRequest, "invalid JSON body")
		return
	}
	var body gmapi.CommandBody
	if err := json.Unmarshal(raw, &body); err != nil {
		gmapi.WriteError(writer, http.StatusBadRequest, "invalid JSON body")
		return
	}
	// 预校验只是 UX 快路径；游戏服侧仍会用同一份代码权威校验。
	if _, validateErr := gmapi.Validate(body); validateErr != nil {
		gmapi.WriteError(writer, http.StatusBadRequest, validateErr.Error())
		return
	}
	status, response, _, ok := p.gameRequest(http.MethodPost,
		"/internal/v1/sessions/"+strconv.FormatUint(id, 10)+"/command", "", raw, "")
	if !ok {
		gmapi.WriteError(writer, http.StatusServiceUnavailable, "game server unreachable")
		return
	}
	writeProxyJSON(writer, status, response, "")
}

// catalogProxy 代理目录端点（含查询串透传，/api/items 的 q/view 用）。
// S5：30s 微缓存 + ETag/304 协商——客户端带着 INM 命中缓存（或游戏服
// 304）时空体直回，刷新页面不再全量重拉目录。
func (p *panelServer) catalogProxy(name string) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		inm := request.Header.Get("If-None-Match")
		key := name + "?" + request.URL.RawQuery
		p.catMu.Lock()
		entry, hit := p.catCache[key]
		fresh := hit && time.Since(entry.at) < catalogCacheTTL
		p.catMu.Unlock()
		if !fresh {
			status, raw, etag, ok := p.gameRequest(http.MethodGet, "/internal/v1/catalog/"+name, request.URL.RawQuery, nil, inm)
			if !ok {
				gmapi.WriteError(writer, http.StatusServiceUnavailable, "game server unreachable")
				return
			}
			// 失败响应不缓存；200/304 连同 ETag 入缓存。
			if status == http.StatusOK || status == http.StatusNotModified {
				entry = catalogCacheEntry{at: time.Now(), code: status, body: raw, etag: etag}
				p.catMu.Lock()
				p.catCache[key] = entry
				p.catMu.Unlock()
			} else {
				writeProxyJSON(writer, status, raw, etag)
				return
			}
		}
		if entry.code == http.StatusNotModified || (entry.etag != "" && gmapi.ETagMatches(inm, entry.etag)) {
			gmapi.WriteNotModified(writer)
			return
		}
		writeProxyJSON(writer, entry.code, entry.body, entry.etag)
	}
}

// methodProxy 透传任意方法的 internal 端点（minimal-login 的 GET 读 /
// POST 切换用；请求体直接转发的场景同样适用）。
func (p *panelServer) methodProxy(path string) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		var body []byte
		if request.Body != nil {
			raw, err := io.ReadAll(http.MaxBytesReader(writer, request.Body, commandBodyLimit))
			if err != nil {
				gmapi.WriteError(writer, http.StatusBadRequest, "invalid request body")
				return
			}
			body = raw
		}
		status, raw, etag, ok := p.gameRequest(request.Method, path, request.URL.RawQuery, body, "")
		if !ok {
			gmapi.WriteError(writer, http.StatusServiceUnavailable, "game server unreachable")
			return
		}
		writeProxyJSON(writer, status, raw, etag)
	}
}
