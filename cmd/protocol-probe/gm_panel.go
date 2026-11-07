package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/local/9yin-go-server/internal/gmapi"
	"github.com/local/9yin-go-server/internal/gmweb"
	"github.com/local/9yin-go-server/internal/role"
)

// gmHub 刻意只监听 localhost。命令由所属的游戏会话循环
// 执行，而不是在 HTTP handler 中执行，这样两个 goroutine
// 绝不会向同一个客户端连接交叉写入 FxNet2 帧。
//
// 输入校验与线类型在 internal/gmapi（与独立面板进程共享的契约层），
// 页面资产在 internal/gmweb；本文件只保留会话表与命令交接。
type gmHub struct {
	mu       sync.RWMutex
	sessions map[uint64]*gmSession
	// itemNames 是 GET 目录端点的中文名解析钩子（game_items 批查；
	// main() 装配，nil 时条目名回退 ConfigID——httptest 无 DB 也可用）。
	itemNames func(configIDs []string) map[string]string
}

type gmSession struct {
	id       uint64
	remote   string
	commands chan gmCommand

	mu    sync.RWMutex
	state gmapi.PlayerState
}

// gmPlayerState 保留原名引用契约层结构（快照字段即 /api/sessions
// 的 JSON 行）。
type gmPlayerState = gmapi.PlayerState

// gmCommand 在契约层的纯数据之上只加一件事：带缓冲的回执通道，
// 使会话循环总能完成其帧的写入，即便浏览器在等待 HTTP 响应期间
// 断开连接。
type gmCommand struct {
	gmapi.Command
	reply chan error
}

func (command gmCommand) complete(err error) {
	if command.reply != nil {
		command.reply <- err
	}
}

func newGMHub() *gmHub { return &gmHub{sessions: make(map[uint64]*gmSession)} }

func (hub *gmHub) attach(id uint64, remote string, active *role.RoleSnapshot, faction string, silver, guanYin, gold, sp, faculty int32) *gmSession {
	session := &gmSession{
		id:     id,
		remote: remote,
		// 刻意使用无缓冲交接：GM 的点击意味着“在游戏循环的
		// 下一轮发送”，而不是“留着等到之后某个场景再发”。
		commands: make(chan gmCommand),
		state: gmapi.PlayerState{
			SessionID: id, Remote: remote, Name: active.Name, Scene: active.Location.Scene,
			Position: active.Location.Position, Silver: silver, GuanYin: guanYin, Gold: gold,
			SP: sp, Faculty: faculty, Faction: faction, LastAction: "在线", UpdatedAt: time.Now(),
		},
	}
	hub.mu.Lock()
	hub.sessions[id] = session
	hub.mu.Unlock()
	return session
}

func (hub *gmHub) detach(id uint64) {
	hub.mu.Lock()
	delete(hub.sessions, id)
	hub.mu.Unlock()
}

func (hub *gmHub) snapshots() []gmapi.PlayerState {
	hub.mu.RLock()
	sessions := make([]*gmSession, 0, len(hub.sessions))
	for _, session := range hub.sessions {
		sessions = append(sessions, session)
	}
	hub.mu.RUnlock()
	states := make([]gmapi.PlayerState, 0, len(sessions))
	for _, session := range sessions {
		states = append(states, session.snapshot())
	}
	return states
}

func (hub *gmHub) enqueue(id uint64, command gmCommand) error {
	hub.mu.RLock()
	session, exists := hub.sessions[id]
	hub.mu.RUnlock()
	if !exists {
		return fmt.Errorf("session %d is not online", id)
	}
	timer := time.NewTimer(750 * time.Millisecond)
	defer timer.Stop()
	select {
	case session.commands <- command:
		return nil
	case <-timer.C:
		return fmt.Errorf("session %d is busy; command was not sent", id)
	}
}

func (session *gmSession) snapshot() gmapi.PlayerState {
	session.mu.RLock()
	defer session.mu.RUnlock()
	return session.state
}

func (session *gmSession) setAction(action string) {
	session.mu.Lock()
	session.state.LastAction = action
	session.state.UpdatedAt = time.Now()
	session.mu.Unlock()
}

func (session *gmSession) setSilver(value int32) {
	session.mu.Lock()
	session.state.Silver = value
	session.state.LastAction = fmt.Sprintf("碎银设为 %d", value)
	session.state.UpdatedAt = time.Now()
	session.mu.Unlock()
}

func (session *gmSession) setGuanYin(value int32) {
	session.mu.Lock()
	session.state.GuanYin = value
	session.state.LastAction = fmt.Sprintf("官银设为 %d", value)
	session.state.UpdatedAt = time.Now()
	session.mu.Unlock()
}

func (session *gmSession) setGold(value int32) {
	session.mu.Lock()
	session.state.Gold = value
	session.state.LastAction = fmt.Sprintf("金币设为 %d（会话级）", value)
	session.state.UpdatedAt = time.Now()
	session.mu.Unlock()
}

func (session *gmSession) setFaculty(value int32) {
	session.mu.Lock()
	session.state.Faculty = value
	session.state.UpdatedAt = time.Now()
	session.mu.Unlock()
}

func (session *gmSession) setSP(value int32, action string) {
	session.mu.Lock()
	session.state.SP = value
	session.state.LastAction = action
	session.state.UpdatedAt = time.Now()
	session.mu.Unlock()
}

func (session *gmSession) setRageRecovery(value int32, action string) {
	session.mu.Lock()
	session.state.RageRecovery = value
	session.state.LastAction = action
	session.state.UpdatedAt = time.Now()
	session.mu.Unlock()
}

func (session *gmSession) setFaction(faction, label string) {
	session.mu.Lock()
	session.state.Faction = faction
	session.state.LastAction = "门派设为 " + label
	session.state.UpdatedAt = time.Now()
	session.mu.Unlock()
}

func (session *gmSession) setLocation(location role.Location, action string) {
	session.mu.Lock()
	session.state.Scene = location.Scene
	session.state.Position = location.Position
	session.state.LastAction = action
	session.state.UpdatedAt = time.Now()
	session.mu.Unlock()
}

// serveGM 起内嵌面板。manageBoundAddr 是本进程 internal API 的监听地址
//（未启用则空串）：管理页登记册据此给对应卡片打「本面板所连」标记，
// 并让内嵌面板同样能探活其它已登记的 Go 服。
func serveGM(listen string, hub *gmHub, manageBoundAddr string) {
	mux := newGMMux(hub)
	gmapi.MountManageAPI(mux, manageBoundAddr, gmInternalVersion)
	server := &http.Server{Addr: listen, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("GM panel listening on http://%s (localhost only)", listen)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("GM panel stopped: %v", err)
	}
}

// newGMMux 组装 GM 面板的全部 HTTP 路由。独立成函数后，
// 测试可以直接把路由挂到 httptest 上验证，而不必真的监听端口。
func newGMMux(hub *gmHub) *http.ServeMux {
	mux := http.NewServeMux()
	// 页面与静态资源来自 internal/gmweb 的编译期嵌入资产，
	// 与独立面板进程共用同一份文件。
	gmweb.Mount(mux)
	mux.HandleFunc("/api/sessions", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		writeGMJSON(writer, http.StatusOK, hub.snapshots())
	})
	mux.HandleFunc("/api/items", hub.itemsHandler)
	mux.HandleFunc("/api/fashion-catalog", hub.fashionCatalogHandler)
	mux.HandleFunc("/api/cards", hub.cardsHandler)
	mux.HandleFunc("/api/hair-cards", hub.hairCardsHandler)
	mux.HandleFunc("/api/weather", gmWeatherCatalogHandler)
	mux.HandleFunc("/api/sessions/", func(writer http.ResponseWriter, request *http.Request) {
		hub.serveSessionCommand("/api/sessions/", "panel", writer, request)
	})
	return mux
}

// gmCommandBodyLimit 是命令 POST 的载荷上限：lua_eval 源码闸 16KB，
// 加上 JSON 开销后留足余量（契约层再校验 16KB 语义）。
const gmCommandBodyLimit = 64 << 10

// serveSessionCommand 处理 POST <prefix>/{id}/command。内嵌面板与
// internal API（gm_internal.go）共用同一份实现：路径解析、契约层校验、
// 入队与回执等待的语义完全一致，只是日志来源标签不同。
func (hub *gmHub) serveSessionCommand(prefix, source string, writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost || !strings.HasSuffix(request.URL.Path, "/command") {
		writer.WriteHeader(http.StatusNotFound)
		return
	}
	idText := strings.TrimSuffix(strings.TrimPrefix(request.URL.Path, prefix), "/command")
	id, err := strconv.ParseUint(idText, 10, 64)
	if err != nil {
		writeGMError(writer, http.StatusBadRequest, "invalid session id")
		return
	}
	var body gmapi.CommandBody
	if err := json.NewDecoder(http.MaxBytesReader(writer, request.Body, gmCommandBodyLimit)).Decode(&body); err != nil {
		writeGMError(writer, http.StatusBadRequest, "invalid JSON body")
		return
	}
	// 将 HTTP 侧的追踪与下方游戏会话的追踪区分开。
	// 服务器重启后，浏览器可能还保留着旧的面板页面；
	// 没有这条日志，就无法区分 POST 丢失与命令
	// 已入队但未送达活跃会话这两种情况。
	log.Printf("GM API command received (%s): session=%d action=%q", source, id, body.Action)
	// 校验的单一权威实现在契约层（internal/gmapi），
	// 与独立面板进程的预校验共用同一份代码。
	validated, validateErr := gmapi.Validate(body)
	if validateErr != nil {
		writeGMError(writer, http.StatusBadRequest, validateErr.Error())
		return
	}
	command := gmCommand{Command: validated, reply: make(chan error, 1)}
	if err := hub.enqueue(id, command); err != nil {
		writeGMError(writer, http.StatusConflict, err.Error())
		return
	}
	select {
	case commandErr := <-command.reply:
		if commandErr != nil {
			writeGMError(writer, http.StatusConflict, commandErr.Error())
			return
		}
		writeGMJSON(writer, http.StatusOK, map[string]string{"status": "sent"})
	case <-time.After(2 * time.Second):
		// 命令已被游戏循环接受，但帧写入未能
		// 及时完成。不要把这种情况描述为已排队或成功。
		writeGMError(writer, http.StatusGatewayTimeout, "game session did not confirm the frame send")
	}
}

func writeGMJSON(writer http.ResponseWriter, status int, value any) {
	gmapi.WriteJSON(writer, status, value)
}

func writeGMError(writer http.ResponseWriter, status int, message string) {
	gmapi.WriteError(writer, status, message)
}

// gmWeatherCatalogHandler 服务面板天气下拉；数据是契约层的静态
// 闭名单，独立面板进程不经游戏服直出同一份目录。
func gmWeatherCatalogHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeGMJSON(writer, http.StatusOK, gmapi.WeatherCatalog())
}
