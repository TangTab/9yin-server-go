package main

import (
	"encoding/json"
	"fmt"
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
	action      string
	silver      int32
	sp          int32
	value       float32
	destination sceneDestination
	position    role.Position
	faction     string
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
	mux := http.NewServeMux()
	mux.HandleFunc("/", gmIndex)
	mux.HandleFunc("/api/sessions", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		writeGMJSON(writer, http.StatusOK, hub.snapshots())
	})
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
				writeGMError(writer, http.StatusBadRequest, "SP must be between 0 and 100")
				return
			}
		case "add_sp":
			if body.SP < 1 || body.SP > maxPlayerSP {
				writeGMError(writer, http.StatusBadRequest, "SP addition must be between 1 and 100")
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
		case "npc_bubble", "drama_prompt", "red_super_armor",
			"action_state_fishing", "action_state_hsqs05", "action_state_sgmd",
			"action_native_fishing", "action_state_stop":
			// 只测试已注册的客户端原生剧情路由。负载由服务器
			// 持有，因此面板无法提交格式错误的自定义值。
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
	server := &http.Server{Addr: listen, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("GM panel listening on http://%s (localhost only)", listen)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("GM panel stopped: %v", err)
	}
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
	if request.URL.Path != "/" {
		writer.WriteHeader(http.StatusNotFound)
		return
	}
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = writer.Write([]byte(gmPanelHTML))
}

const gmPanelHTML = `<!doctype html><meta charset="utf-8"><title>九阴本地 GM</title>
<style>body{font:15px system-ui;margin:28px;max-width:860px;background:#171719;color:#eee}button,input{padding:8px;margin:4px}input{width:360px}section{padding:14px;margin:14px 0;background:#26262a;border-radius:8px}.muted{color:#aaa}pre{white-space:pre-wrap}</style>
<h1>九阴本地 GM</h1><p class="muted">仅监听 127.0.0.1。每次点击会交给游戏主循环直接写入一帧；会话忙碌时会报失败，不会排队保留。</p>
<section><h2>在线角色</h2><pre id="state">读取中…</pre><button onclick="refresh()">刷新</button></section>
<section><h2>碎银</h2><input id="silver" type="number" value="99999" min="0"><button onclick="silver()">设置碎银</button></section>
<section><h2>怒气</h2><p class="muted">服务端权威 <code>SP</code>（0–100）。技能消耗已读取同一数值；战斗获得规则后续也会走同一入口。</p><input id="sp" type="number" value="100" min="0" max="100"><button onclick="setSP()">设置怒气</button><button onclick="addSP(10)">+10</button><button onclick="addSP(50)">+50</button><button onclick="fillSP()">回满</button></section>
<section><h2>移动与下落</h2><p class="muted">移动速度同时更新 MoveSpeed 与 RunSpeed；下落速度更新 Gravity，数值越大下落越快，下一次起跳生效。</p><label>移动速度 <input id="move-speed" type="number" value="25" min="0.1" max="100" step="0.5"></label><button onclick="setMoveSpeed()">应用移动速度</button><br><label>下落重力 <input id="gravity" type="number" value="9.8" min="0.1" max="100" step="0.1"></label><button onclick="setGravity()">应用下落速度</button></section>
<section><h2>主城传送</h2><p class="muted">使用安装资源的场景名和原始门点坐标；按钮走受限预设，避免手输时误切到没有资源的地图。</p><button onclick="preset('yanjing')">燕京</button><button onclick="preset('suzhou')">苏州</button><button onclick="preset('jinling')">金陵</button><button onclick="preset('luoyang')">洛阳</button><button onclick="preset('chengdu')">成都</button><br><button onclick="preset('jimingyi')">鸡鸣驿</button><button onclick="preset('yanyuzhuang')">烟雨庄</button><button onclick="preset('qiandengzhen')">千灯镇</button></section>
<section><h2>八大门派传送</h2><button onclick="preset('jinyiwei')">锦衣卫</button><button onclick="preset('gaibang')">丐帮</button><button onclick="preset('junzitang')">君子堂</button><button onclick="preset('jilegu')">极乐谷</button><br><button onclick="preset('tangmen')">唐门</button><button onclick="preset('emei')">峨眉</button><button onclick="preset('wudang')">武当</button><button onclick="preset('shaolin')">少林</button></section>
<section><h2>江湖势力传送</h2><p class="muted">使用各势力在当前资源中的 Safe HomePoint；徐家庄、金针沈家在成都，万兽山庄在丐帮地图，天轮寺和青衣阁为独立场景。</p><button onclick="preset('yihuagong')">移花宫</button><button onclick="preset('taohuadao')">桃花岛</button><button onclick="preset('wugenmen')">无根门</button><br><button onclick="preset('xujia')">徐家庄</button><button onclick="preset('jinzhen')">金针沈家</button><button onclick="preset('wanshou')">万兽山庄</button><button onclick="preset('tianlun')">天轮寺</button><button onclick="preset('qingyi')">青衣阁</button></section>
<section><h2>隐世宗门传送</h2><p class="muted">全部使用各宗门在安装资源中标记为 Safe=1 的默认出生点；长风镖局在金陵，血刀门在恶人谷，其余为对应宗门地图。</p><button onclick="preset('xuedao')">血刀门</button><button onclick="preset('huashan')">华山派</button><button onclick="preset('gumu')">古墓派</button><button onclick="preset('damo')">达摩派</button><button onclick="preset('shenshui')">神水宫</button><br><button onclick="preset('changfeng')">长风镖局</button><button onclick="preset('nianluo')">念萝坝</button><button onclick="preset('wuxian')">无仙教</button><button onclick="preset('shenji')">神机营</button><button onclick="preset('xingmiao')">星渺阁</button><br><button onclick="preset('tianya')">天涯海阁</button><button onclick="preset('wanghui')">王回洲</button><button onclick="preset('shenjihui')">神机汇</button><button onclick="preset('wuxu')">无虚门</button></section>
<section><h2>选择门派 / 宗门 / 势力</h2><p class="muted">修改角色原生 <code>School</code> 属性并持久化；不自动赠送对应套路或任务线。</p><select id="faction"><optgroup label="八大门派"><option value="shaolin">少林</option><option value="wudang">武当</option><option value="emei">峨眉</option><option value="gaibang">丐帮</option><option value="tangmen">唐门</option><option value="junzitang">君子堂</option><option value="jinyiwei">锦衣卫</option><option value="jilegu">极乐谷</option></optgroup><optgroup label="江湖势力"><option value="yihuagong">移花宫</option><option value="taohuadao">桃花岛</option><option value="wugenmen">无根门</option><option value="xujia">徐家庄</option><option value="wanshou">万兽山庄</option><option value="jinzhen">金针沈家</option><option value="tianlun">天轮寺</option><option value="qingyi">青衣阁</option></optgroup><optgroup label="宗门"><option value="xuedao">血刀门</option><option value="huashan">华山派</option><option value="gumu">古墓派</option><option value="damo">达摩派</option><option value="shenshui">神水宫</option><option value="changfeng">长风镖局</option><option value="nianluo">念萝坝</option><option value="wuxian">无仙教</option><option value="shenji">神机营</option><option value="xingmiao">星渺阁</option><option value="tianya">天涯海阁</option><option value="wanghui">王回洲</option><option value="shenjihui">神机汇</option></optgroup></select><button onclick="setFaction()">应用门派</button></section>
<section><h2>自定义切图</h2><input id="destination" value="ini\scene\city04_LuoYang,city04,243.387,43.277,403.821,5.024"><button onclick="scene(document.querySelector('#destination').value)">按自定义坐标切换</button></section>
<section><h2>场景内瞬移</h2><p class="muted">只发送 ServerLocation（0x1F），不退场、不读图；格式为 X,Y,Z,朝向。先“填入当前坐标”，再只改目标位置。</p><input id="teleport" placeholder="X,Y,Z,朝向"><button onclick="useCurrentPosition()">填入当前坐标</button><button onclick="teleportPlayer()">瞬移到坐标</button></section>
<section><h2>状态 / Buff</h2><p class="muted">红霸体：服务端写入玩家 <code>BufferInfo1</code> 原生 BuffStore 槽；20 秒后移除。</p><button onclick="story('red_super_armor')">施加红霸体（20 秒）</button></section>
<section><h2>人物动作通道测试</h2><p class="muted"><code>State</code> 已验证可以播放有限时长的身体动作。神鬼莫敌·无缺使用单段 <code>hsqs_07</code>（147 帧）；第五招使用 <code>hsqs_05_h</code> 起手并在 0.24 秒后衔接主段。</p><button onclick="story('action_state_sgmd')">播放神鬼莫敌动作</button><button onclick="story('action_state_hsqs05')">播放第五招起手</button><button onclick="story('action_state_fishing')">钓鱼持续姿势对照</button><button onclick="story('action_native_fishing')">字符串 action 对照</button><button onclick="story('action_state_stop')">恢复待机</button></section>
<section><h2>剧情通道验证</h2><p class="muted">仅验证现代客户端原生显示链：气泡对白会显示在当前场景首个 NPC 头顶；章节提示卡使用已安装的 ui_shop 文本 ID，不会写入任务进度。</p><button onclick="story('npc_bubble')">测试 NPC 气泡</button><button onclick="story('drama_prompt')">测试章节提示卡</button></section>
<script>let sessions=[];async function refresh(){sessions=await (await fetch('/api/sessions')).json();document.querySelector('#state').textContent=JSON.stringify(sessions,null,2)}async function command(payload){await refresh();if(!sessions.length)return alert('没有在线角色');let status=document.querySelector('#command-status');if(status)status.textContent='发送中…';let r=await fetch('/api/sessions/'+sessions[0].sessionId+'/command',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(payload)});let j=await r.json();if(!r.ok){if(status)status.textContent='失败：'+(j.error||'命令失败');return alert(j.error||'命令失败')}await refresh();if(status)status.textContent=sessions[0]&&sessions[0].lastAction||'已发送'}function silver(){command({action:'set_silver',silver:+document.querySelector('#silver').value})}function setSP(){command({action:'set_sp',sp:+document.querySelector('#sp').value})}function addSP(sp){command({action:'add_sp',sp})}function fillSP(){command({action:'fill_sp'})}function setMoveSpeed(){command({action:'set_move_speed',value:+document.querySelector('#move-speed').value})}function setGravity(){command({action:'set_gravity',value:+document.querySelector('#gravity').value})}function scene(destination){command({action:'switch_scene',destination})}function preset(key){command({action:'switch_scene_preset',preset:key})}function setFaction(){command({action:'set_faction',faction:document.querySelector('#faction').value})}function useCurrentPosition(){if(!sessions.length)return alert('没有在线角色');let p=sessions[0].position;document.querySelector('#teleport').value=[p.x,p.y,p.z,p.orient].join(',')}function teleportPlayer(){command({action:'teleport',position:document.querySelector('#teleport').value})}function story(action){command({action})}refresh();setInterval(refresh,2500)</script>`
