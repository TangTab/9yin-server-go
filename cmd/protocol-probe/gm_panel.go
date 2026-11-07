package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/local/9yin-go-server/internal/role"
)

// gmHub 刻意只监听 localhost。命令由所属的游戏会话循环
// 执行，而不是在 HTTP handler 中执行，这样两个 goroutine
// 绝不会向同一个客户端连接交叉写入 FxNet2 帧。
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
	state gmPlayerState
}

type gmPlayerState struct {
	SessionID  uint64        `json:"sessionId"`
	Remote     string        `json:"remote"`
	Name       string        `json:"name"`
	Scene      role.Scene    `json:"scene"`
	Position   role.Position `json:"position"`
	Silver     int32         `json:"silver"`
	SP         int32         `json:"sp"`
	Faction    string        `json:"faction"`
	LastAction string        `json:"lastAction"`
	UpdatedAt  time.Time     `json:"updatedAt"`
}

type gmCommand struct {
	action        string
	silver        int32
	sp            int32
	value         float32
	destination   sceneDestination
	position      role.Position
	faction       string
	code          string // lua_eval 的 Lua 源码
	itemConfig    string // grant_item/add_item 的物品配置 ID
	amount        int32  // grant_item 数量
	wearNow       bool   // grant_item 穿戴形态（否则入背包）
	cardId        int32  // equip_card/equip_hair 皮肤号
	showEquipType int32  // set_showequip_type 0/1/2
	gold          int32  // set_gold（会话级 CapitalType0）
	// reply 带缓冲，使会话循环总能完成其帧的写入，
	// 即便浏览器在等待 HTTP 响应期间断开连接。
	reply chan error
}

func (command gmCommand) complete(err error) {
	if command.reply != nil {
		command.reply <- err
	}
}

func newGMHub() *gmHub { return &gmHub{sessions: make(map[uint64]*gmSession)} }

func (hub *gmHub) attach(id uint64, remote string, active *role.RoleSnapshot, faction string, silver, sp int32) *gmSession {
	session := &gmSession{
		id:     id,
		remote: remote,
		// 刻意使用无缓冲交接：GM 的点击意味着“在游戏循环的
		// 下一轮发送”，而不是“留着等到之后某个场景再发”。
		commands: make(chan gmCommand),
		state: gmPlayerState{
			SessionID: id, Remote: remote, Name: active.Name, Scene: active.Location.Scene,
			Position: active.Location.Position, Silver: silver, SP: sp, Faction: faction, LastAction: "在线", UpdatedAt: time.Now(),
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

func (hub *gmHub) snapshots() []gmPlayerState {
	hub.mu.RLock()
	sessions := make([]*gmSession, 0, len(hub.sessions))
	for _, session := range hub.sessions {
		sessions = append(sessions, session)
	}
	hub.mu.RUnlock()
	states := make([]gmPlayerState, 0, len(sessions))
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

func (session *gmSession) snapshot() gmPlayerState {
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

func (session *gmSession) setSP(value int32, action string) {
	session.mu.Lock()
	session.state.SP = value
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

func serveGM(listen string, hub *gmHub) {
	server := &http.Server{Addr: listen, Handler: newGMMux(hub), ReadHeaderTimeout: 5 * time.Second}
	log.Printf("GM panel listening on http://%s (localhost only)", listen)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("GM panel stopped: %v", err)
	}
}

// newGMMux 组装 GM 面板的全部 HTTP 路由。独立成函数后，
// 测试可以直接把路由挂到 httptest 上验证，而不必真的监听端口。
func newGMMux(hub *gmHub) *http.ServeMux {
	mux := http.NewServeMux()
	// console.css / console-*.js 等静态资源来自编译期嵌入的
	// web/static 目录，页面可以像普通网站一样引用外部文件，
	// 同时保持单二进制分发。
	static, err := fs.Sub(gmWebAssets, "web/static")
	if err != nil {
		log.Fatalf("GM static assets unavailable: %v", err)
	}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(static))))
	mux.HandleFunc("/", gmIndex)
	mux.HandleFunc("/api/sessions", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		writeGMJSON(writer, http.StatusOK, hub.snapshots())
	})
	mux.HandleFunc("/api/fashion-catalog", hub.fashionCatalogHandler)
	mux.HandleFunc("/api/cards", hub.cardsHandler)
	mux.HandleFunc("/api/hair-cards", hub.hairCardsHandler)
	mux.HandleFunc("/api/sessions/", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || !strings.HasSuffix(request.URL.Path, "/command") {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		idText := strings.TrimSuffix(strings.TrimPrefix(request.URL.Path, "/api/sessions/"), "/command")
		id, err := strconv.ParseUint(idText, 10, 64)
		if err != nil {
			writeGMError(writer, http.StatusBadRequest, "invalid session id")
			return
		}
		var body struct {
			Action      string  `json:"action"`
			Silver      int32   `json:"silver"`
			SP          int32   `json:"sp"`
			Value       float32 `json:"value"`
			Destination string  `json:"destination"`
			Preset      string  `json:"preset"`
			Position    string  `json:"position"`
			Faction     string  `json:"faction"`
			Code        string  `json:"code"`
			Type        int32   `json:"type"`
			ItemConfig  string  `json:"itemConfig"`
			Amount      int32   `json:"amount"`
			WearNow     bool    `json:"wearNow"`
			CardId      int32   `json:"cardId"`
			Gold        int32   `json:"gold"`
		}
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			writeGMError(writer, http.StatusBadRequest, "invalid JSON body")
			return
		}
		// 将 HTTP 侧的追踪与下方游戏会话的追踪区分开。
		// 服务器重启后，浏览器可能还保留着旧的面板页面；
		// 没有这条日志，就无法区分 POST 丢失与命令
		// 已入队但未送达活跃会话这两种情况。
		log.Printf("GM API command received: session=%d action=%q", id, body.Action)
		command := gmCommand{action: body.Action, silver: body.Silver, sp: body.SP, value: body.Value, faction: body.Faction, reply: make(chan error, 1)}
		switch body.Action {
		case "set_silver":
			if body.Silver < 0 {
				writeGMError(writer, http.StatusBadRequest, "silver must be non-negative")
				return
			}
		case "set_sp":
			if body.SP < 0 || body.SP > maxPlayerSP {
				writeGMError(writer, http.StatusBadRequest, fmt.Sprintf("SP must be between 0 and %d", maxPlayerSP))
				return
			}
		case "add_sp":
			if body.SP < 1 || body.SP > maxPlayerSP {
				writeGMError(writer, http.StatusBadRequest, fmt.Sprintf("SP addition must be between 1 and %d", maxPlayerSP))
				return
			}
		case "fill_sp":
		case "set_move_speed":
			if math.IsNaN(float64(body.Value)) || math.IsInf(float64(body.Value), 0) || body.Value < 0.1 || body.Value > 100 {
				writeGMError(writer, http.StatusBadRequest, "move speed must be between 0.1 and 100")
				return
			}
		case "set_gravity":
			if math.IsNaN(float64(body.Value)) || math.IsInf(float64(body.Value), 0) || body.Value < 0.1 || body.Value > 100 {
				writeGMError(writer, http.StatusBadRequest, "gravity must be between 0.1 and 100")
				return
			}
		case "switch_scene":
			destination, parseErr := parseSceneDestination(body.Destination)
			if parseErr != nil {
				writeGMError(writer, http.StatusBadRequest, parseErr.Error())
				return
			}
			command.destination = destination
		case "switch_scene_preset":
			destination, presetErr := gmScenePresetDestination(body.Preset)
			if presetErr != nil {
				writeGMError(writer, http.StatusBadRequest, presetErr.Error())
				return
			}
			// 会话循环拥有既有的 switch_scene 执行路径。
			// 预设只是 HTTP 侧的输入校验，必须在命令
			// 到达该循环之前完成转换。
			command.action = "switch_scene"
			command.destination = destination
		case "teleport":
			position, parseErr := parseScenePosition(body.Position)
			if parseErr != nil {
				writeGMError(writer, http.StatusBadRequest, parseErr.Error())
				return
			}
			command.position = position
		case "set_faction":
			if _, factionErr := gmFactionPresetByKey(body.Faction); factionErr != nil {
				writeGMError(writer, http.StatusBadRequest, factionErr.Error())
				return
			}
		case "set_showequip_type":
			if body.Type < 0 || body.Type > 2 {
				writeGMError(writer, http.StatusBadRequest, "show equip type must be 0 (江湖装), 1 (身份装) or 2 (时装)")
				return
			}
			command.showEquipType = body.Type
		case "set_gold":
			if body.Gold < 0 {
				writeGMError(writer, http.StatusBadRequest, "gold must be non-negative")
				return
			}
			command.gold = body.Gold
		case "grant_item", "add_item":
			if strings.TrimSpace(body.ItemConfig) == "" {
				writeGMError(writer, http.StatusBadRequest, "itemConfig is required")
				return
			}
			if body.Amount < 1 {
				body.Amount = 1
			}
			command.itemConfig, command.amount, command.wearNow = strings.TrimSpace(body.ItemConfig), body.Amount, body.WearNow
		case "equip_card", "equip_hair":
			if body.CardId <= 0 {
				writeGMError(writer, http.StatusBadRequest, "cardId must be a positive skin id")
				return
			}
			command.cardId = body.CardId
		case "grant_identity_suits", "grant_all_fashion", "unequip_card", "grant_all_cards":
			// 无负载动作：目录/存档校验在会话循环内完成。
		case "npc_bubble", "drama_prompt", "red_super_armor",
			"action_state_fishing", "action_state_hsqs05", "action_state_sgmd",
			"action_native_fishing", "action_state_stop":
			// 只测试已注册的客户端原生剧情路由。负载由服务器
			// 持有，因此面板无法提交格式错误的自定义值。
		case "lua_eval":
			// Lua 源码在会话循环内经沙盒执行（超时+指令预算），
			// 这里只做长度闸门，16KB 与 internal/luascript 一致。
			if strings.TrimSpace(body.Code) == "" {
				writeGMError(writer, http.StatusBadRequest, "lua_eval requires non-empty code")
				return
			}
			if len(body.Code) > 16<<10 {
				writeGMError(writer, http.StatusBadRequest, "lua_eval code exceeds 16KB")
				return
			}
			command.code = body.Code
		case "lua_reload", "task_rebuild":
		default:
			writeGMError(writer, http.StatusBadRequest, "unsupported action")
			return
		}
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
	})
	return mux
}

func writeGMJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func writeGMError(writer http.ResponseWriter, status int, message string) {
	writeGMJSON(writer, status, map[string]string{"error": message})
}

func gmIndex(writer http.ResponseWriter, request *http.Request) {
	page, known := map[string]string{
		"/":             "web/index.html",
		"/console.html": "web/console.html",
	}[request.URL.Path]
	if !known {
		writer.WriteHeader(http.StatusNotFound)
		return
	}
	index, err := gmWebAssets.ReadFile(page)
	if err != nil {
		http.Error(writer, "GM panel index missing", http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = writer.Write(index)
}

// gmWebAssets 编译期嵌入 GM 面板的全部网页文件（web/index.html、
// web/console.html 与 static/ 样式脚本），运行期不依赖磁盘上的任何前端资源。
//
//go:embed web
var gmWebAssets embed.FS
