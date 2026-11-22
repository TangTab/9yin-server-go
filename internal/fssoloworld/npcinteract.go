package fssoloworld

// npcinteract.go NPC 交互链（2026-11-22 移植 2433 selection.go/scene_lifecycle
// cinematic talk）：
//
//	点击 NPC   C2S op0x07 33B object request（Sequence@17/Obj@21/Own@25/
//	           FuncID@29，含 op 字节）——FuncID=0 首段：talk 类回 cinematic
//	           菜单帧族（msg4 begin/msg6 [funcID,textID]/msg7 [obj] end，
//	           0x1E 自定义消息构造）；attack/boss 类仅选中不弹菜单。
//	选中目标   C2S msg431 [431, obj] → LastObject+AutoSelectObject 双 0x10
//	           属性帧 + msg25[30000,"skill_pause"] 选靶授权（m26 看门狗 5s
//	           配对——裸 m25 锁客户端输入 30s，2433 真机定案）。
//	菜单选择   第二段 op0x07 FuncID≠0：600000000=关闭（msg8）、0=清菜单
//	           （S2C 0x1D）、服务 mark（805000000 商店/807000000 仓库/
//	           860000022 传送…）→ 业务 View 待接（本阶段 trace 留痕）。
//
// 服务列从模板行提取（ShopID/DepotID/TransFuncID/GemGameNpcID/GuildFunc/
// HomePoint/Job——2433 modernNPCServices 同源语义）；菜单文案键与 funcID
// 映射照抄 2433 modernTalkFuncID/TextID 定版。

import (
	"encoding/binary"
	"strings"
	"sync"
	"time"
	"unicode/utf16"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/npcfeed"
)

// ---- C2S op0x07 object request（33B 定长，2433 client_object_request 同构） ----

type npcObjectRequest struct {
	Sequence uint32
	ObjectID uint32
	OwnerID  uint32
	FuncID   int32
}

// parseNPCObjectRequest frame 为完整用户面帧（[0]=0x07）。恒 33B；身份
// 字段全零视为非本形态（与复活等其它 op0x07 用法分流）。
func parseNPCObjectRequest(frame []byte) (npcObjectRequest, bool) {
	if len(frame) < 33 || frame[0] != 0x07 {
		return npcObjectRequest{}, false
	}
	req := npcObjectRequest{
		Sequence: binary.LittleEndian.Uint32(frame[17:]),
		ObjectID: binary.LittleEndian.Uint32(frame[21:]),
		OwnerID:  binary.LittleEndian.Uint32(frame[25:]),
		FuncID:   int32(binary.LittleEndian.Uint32(frame[29:])),
	}
	if req.ObjectID == 0 || req.OwnerID == 0 {
		return npcObjectRequest{}, false
	}
	return req, true
}

// ---- 菜单簿记（Server 级，key=连接号） ----

type npcActiveMenu struct {
	objectID uint32
	ownerID  uint32
	at       time.Time
}

var (
	npcMenuMu    sync.Mutex
	npcMenuState = map[uint32]*npcActiveMenu{}
)

// ---- 服务列（2433 modernNPCServices 同源，mark 族照抄） ----

type npcService struct {
	mark  uint16
	label string
	value string
}

const (
	npcMarkShop      uint16 = 0x1001
	npcMarkDepot     uint16 = 0x1002
	npcMarkTransport uint16 = 0x1003
	npcMarkJob       uint16 = 0x1004
	npcMarkGem       uint16 = 0x1005
	npcMarkGuild     uint16 = 0x1006
	npcMarkHome      uint16 = 0x1007
	npcMarkTask      uint16 = 0x1008
)

// npcServicesOf 模板行服务列提取（非空且非 "0" 才算配置）。
func npcServicesOf(tpl map[string]string) []npcService {
	add := func(items []npcService, mark uint16, label, key string) []npcService {
		if v := strings.TrimSpace(tpl[key]); v != "" && v != "0" {
			return append(items, npcService{mark: mark, label: label, value: v})
		}
		return items
	}
	items := make([]npcService, 0, 6)
	items = add(items, npcMarkShop, "商店", "ShopID")
	items = add(items, npcMarkDepot, "仓库", "DepotID")
	items = add(items, npcMarkTransport, "传送", "TransFuncID")
	items = add(items, npcMarkGem, "宝石服务", "GemGameNpcID")
	items = add(items, npcMarkGuild, "帮会", "GuildFunc")
	items = add(items, npcMarkHome, "回城点", "HomePoint")
	items = add(items, npcMarkJob, "生活职业", "Job")
	return items
}

// ---- 物化时机簿记（2433 clientActivity 定案）----
// NPC 物化必须等客户端 stage_main lua scene_obj:RegisterCallBack 注册
// 完——就绪屏障=ClientReady 后**首条上行活动消息**（op0x0A/0x1E 到达
// 即 stage_main 已在跑）；固定延时不可靠（场景资源加载时长不定）。
// 30s 兜底防客户端哑巴永不物化。

var (
	npcPendingMu  sync.Mutex
	npcPendingSet = map[uint32]*Player{} // connID → 待物化玩家
)

// npcMaterializeArm clientReady 挂账（30s 兜底强制物化）。
func (s *Server) npcMaterializeArm(pl *Player) {
	npcPendingMu.Lock()
	npcPendingSet[pl.Conn1] = pl
	npcPendingMu.Unlock()
	time.AfterFunc(30*time.Second, func() {
		npcPendingMu.Lock()
		_, pending := npcPendingSet[pl.Conn1]
		delete(npcPendingSet, pl.Conn1)
		npcPendingMu.Unlock()
		if !pending || pl.State != PlOnline {
			return
		}
		s.npcMaterializeSend(pl, "fallback-30s")
	})
}

// npcMaterializeOnActivity 首条上行活动消息触发（幂等；已物化直通）。
func (s *Server) npcMaterializeOnActivity(pl *Player) {
	npcPendingMu.Lock()
	_, pending := npcPendingSet[pl.Conn1]
	delete(npcPendingSet, pl.Conn1)
	npcPendingMu.Unlock()
	if !pending {
		return
	}
	s.npcMaterializeSend(pl, "activity")
}

func (s *Server) npcMaterializeSend(pl *Player, trigger string) {
	if pl.State != PlOnline || !pl.haveRoleState {
		return
	}
	npcFrames := s.sceneNPCEnter(pl, pl.roleState.Resource, pl.roleState.Position)
	for _, frame := range npcFrames {
		s.sendToClient(pl, frame)
	}
	s.trace("npc materialize (%s): %d frames client=%d", trigger, len(npcFrames), pl.Conn1)
}

// ---- 主入口 ----

// handleNPCObjectRequest op0x07 点击对象：FuncID=0 首段（弹菜单/仅选中），
// FuncID≠0 第二段（菜单选择路由）。
func (s *Server) handleNPCObjectRequest(pl *Player, req npcObjectRequest) bool {
	if req.FuncID != 0 {
		return s.handleNPCMenuSelect(pl, req)
	}
	pl.npcMu.Lock()
	scene := pl.npcScene
	pl.npcMu.Unlock()
	spawn, ok := s.findNPCSpawn(scene, req.ObjectID)
	if !ok || spawn.OwnerID != req.OwnerID {
		s.trace("npc interact: 未知对象 id=%d owner=%d scene=%s", req.ObjectID, req.OwnerID, scene)
		return true
	}
	// attack/boss 类：战斗选靶通道（LastObject），绝不弹虚构对话。
	switch strings.ToLower(strings.TrimSpace(spawn.Script)) {
	case "attacknpc", "bossnpc":
		s.trace("npc interact: %s (%s) 战斗类仅选中", spawn.ConfigID, spawn.Script)
		return true
	}
	services := npcServicesOf(spawn.Template)
	// talk 数据帧族 msg4 begin/msg6×N(funcid,ui文本ID)/msg7 [obj]——
	// npc.pcapng #2377-2382 官方字节级模板；C++ 聚合调 form_talk.
	// show_window 开窗+渲染菜单（单链路自足）。funcid 用 form_talk_data
	// g_menu_id 官方域。⚠️ 不再叠发 0x1C——0x1C 触发 open_npc_talk 与
	// msg4 触发的 show_window 双开窗竞态（真机实证 form_talk.lua:71
	// add_menu 处 nx_destroy need entity）。
	if frames, err := npcCinematicTalkMenu(spawn.ObjectID, spawn.OwnerID, services); err == nil {
		for _, frame := range frames {
			s.sendToClient(pl, frame)
		}
	} else {
		s.trace("npc interact: talk 数据编码失败 %s: %v", spawn.ConfigID, err)
	}
	npcMenuMu.Lock()
	npcMenuState[pl.Conn1] = &npcActiveMenu{objectID: spawn.ObjectID, ownerID: spawn.OwnerID, at: time.Now()}
	npcMenuMu.Unlock()
	pauseNpcTalk(spawn.ObjectID) // 行人对话停走（菜单关闭恢复）
	s.trace("npc interact: %s (%s) 菜单 %d 项 client=%d", spawn.ConfigID, spawn.Script, len(services)+1, pl.Conn1)
	return true
}

// handleNPCMenuSelect 菜单第二段：0x1C SERVER_MENU 选择的 mark 路由
// （mark=0 离开 → 0x1D 清菜单；mark 0x1001..0x1008 业务 → 本阶段清菜单
// + trace 留痕，业务 View 待接）。
func (s *Server) handleNPCMenuSelect(pl *Player, req npcObjectRequest) bool {
	npcMenuMu.Lock()
	menu := npcMenuState[pl.Conn1]
	npcMenuMu.Unlock()
	if menu == nil || menu.objectID != req.ObjectID || menu.ownerID != req.OwnerID {
		// 无活动菜单的 mark（如兜底「交谈」mark=1）：仅留痕。
		if req.FuncID != 0 {
			s.trace("npc interact: 菜单选择无活动菜单 mark=%d id=%d", req.FuncID, req.ObjectID)
		}
		return true
	}
	// 关菜单统一走 SERVER_CLEAR_MENU（0x1D）。
	s.sendToClient(pl, []byte{0x1D})
	s.clearNPCMenu(pl)
	if req.FuncID == 0 {
		s.trace("npc interact: 菜单关闭 id=%d client=%d", req.ObjectID, pl.Conn1)
		return true
	}
	s.trace("npc interact: 菜单选择 mark=%d id=%d（业务 View 待接）", req.FuncID, req.ObjectID)
	return true
}

func (s *Server) clearNPCMenu(pl *Player) {
	npcMenuMu.Lock()
	menu := npcMenuState[pl.Conn1]
	delete(npcMenuState, pl.Conn1)
	npcMenuMu.Unlock()
	if menu != nil {
		resumeNpcTalk(menu.objectID) // 菜单关闭恢复巡逻
	}
}

// findNPCSpawn 场景目录内按 objectID 查（点击频度低，线性可接受）。
func (s *Server) findNPCSpawn(scene string, objectID uint32) (npcfeed.Spawn, bool) {
	if scene == "" {
		return npcfeed.Spawn{}, false
	}
	for _, spawn := range s.sceneNPCCatalog(scene).spawns {
		if spawn.ObjectID == objectID {
			return spawn, true
		}
	}
	return npcfeed.Spawn{}, false
}

// npcCinematicTalkMenu talk 数据帧族（npc.pcapng #2377-2382 官方字节级
// 模板）：msg4 [4] begin / msg6 [6, funcid, ui文本ID]×N（funcid 用
// form_talk_data g_menu_id 域，官方离开项 600000000+"ui_exit" 实证同系）
// / msg7 [7, obj] end。C++ 聚合调 form_talk.show_window；NPC 名由客户
// 端按 ConfigID 本地查（get_npcname_by_id），标题键（官方发
// "task_WorldNpc01230" 形的文本表键）普通对话场景无合适键——不发 msg5
// （曾发 NPC 名 UTF-8→乱码、发 "ui_form_talk"→原文上屏，均实证 m5 串
// 走 C++ 文本转换而非直显）。
func npcCinematicTalkMenu(objectID, ownerID uint32, services []npcService) ([][]byte, error) {
	out := make([][]byte, 0, 2+len(services))
	begin, err := entergen.CustomMessageFrame(4)
	if err != nil {
		return nil, err
	}
	out = append(out, begin)
	for _, service := range services {
		pkt, err := entergen.CustomMessageFrame(6, entergen.CustomInt(npcTalkFuncID(service.mark)), entergen.CustomString(npcTalkTextID(service.mark)))
		if err != nil {
			return nil, err
		}
		out = append(out, pkt)
	}
	// 尾项「离开」（官方实证 600000000+"ui_exit"）。
	leave, err := entergen.CustomMessageFrame(6, entergen.CustomInt(600000000), entergen.CustomString("ui_exit"))
	if err != nil {
		return nil, err
	}
	out = append(out, leave)
	end, err := entergen.CustomMessageFrame(7, entergen.CustomObject(objectID, ownerID))
	if err != nil {
		return nil, err
	}
	return append(out, end), nil
}

// npcTalkFuncID/npcTalkTextID g_menu_id 官方域映射（form_talk_data.lua
// 定版：func_menu_shop=805000000/func_menu_depot=807000000/…）。
func npcTalkFuncID(mark uint16) int32 {
	switch mark {
	case npcMarkShop:
		return 805000000
	case npcMarkDepot:
		return 807000000
	case npcMarkTransport:
		return 860000022
	case npcMarkJob:
		return 806000036
	case npcMarkGem:
		return 806000013
	case npcMarkGuild:
		return 809000000
	case npcMarkHome:
		return 881000001
	default:
		return 860000000 + int32(mark)
	}
}

func npcTalkTextID(mark uint16) string {
	switch mark {
	case npcMarkShop:
		return "ui_shop"
	case npcMarkDepot:
		return "ui_depot"
	case npcMarkTransport:
		return "ui_transport"
	case npcMarkJob:
		return "ui_life_job"
	case npcMarkGem:
		return "ui_equip_gem"
	case npcMarkGuild:
		return "ui_guild"
	case npcMarkHome:
		return "ui_home_point"
	default:
		return "ui_talk"
	}
}

// npcServerMenu 原生 0x1C SERVER_MENU（2433 messages.go serverMenu 定版）：
// [0x1C][obj u32][own u32][u16 n]{item: [u8 type=0][u16 mark][u32 含NUL
// UTF16LE 字节数][UTF16LE+NUL]}。服务列 → {mark, 中文标签}；空服务兜底
// 「交谈」mark=1；尾项恒「离开」mark=0。
func npcServerMenu(objectID, ownerID uint32, services []npcService) ([]byte, error) {
	type item struct {
		mark    uint16
		content string
	}
	items := make([]item, 0, len(services)+1)
	for _, service := range services {
		items = append(items, item{mark: service.mark, content: service.label})
	}
	if len(items) == 0 {
		items = append(items, item{mark: 1, content: "交谈"})
	}
	items = append(items, item{mark: 0, content: "离开"})

	msg := make([]byte, 11, 11+len(items)*24)
	msg[0] = 0x1C
	binary.LittleEndian.PutUint32(msg[1:], objectID)
	binary.LittleEndian.PutUint32(msg[5:], ownerID)
	binary.LittleEndian.PutUint16(msg[9:], uint16(len(items)))
	for _, it := range items {
		units := utf16.Encode([]rune(it.content))
		msg = append(msg, 0) // type
		msg = binary.LittleEndian.AppendUint16(msg, it.mark)
		msg = binary.LittleEndian.AppendUint32(msg, uint32((len(units)+1)*2))
		for _, unit := range units {
			msg = binary.LittleEndian.AppendUint16(msg, unit)
		}
		msg = binary.LittleEndian.AppendUint16(msg, 0)
	}
	return msg, nil
}

// ---- msg431 选中目标 ----

const clientCustomSelectTarget int32 = 431

// handleNPCSelectTarget msg431 [431, obj(packed)]：回 LastObject+
// AutoSelectObject 双 0x10 属性帧 + m25[30000] 选靶授权；5s 后 m26 解锁
// 看门狗（官服 m25 恒 0.4~4.6s 内配对 m26，裸 m25 锁输入 30s）。
func (s *Server) handleNPCSelectTarget(pl *Player, message entergen.CustomInbound) {
	if len(message.Values) < 2 {
		return
	}
	packed := message.Values[1].Raw
	targetID := binary.LittleEndian.Uint32(packed[:])
	targetOwner := binary.LittleEndian.Uint32(packed[4:])
	// 双帧各自成帧（LastObject 与 AutoSelectObject 是两个 0x10 属性帧）。
	for _, name := range []string{"LastObject", "AutoSelectObject"} {
		frame, err := entergen.ObjectPropertyFrame(1, 1, 0, []entergen.IndexedProperty{
			entergen.NamedProperty(name, entergen.ObjectValue(targetID, targetOwner)),
		})
		if err != nil {
			s.trace("npc select: %s 帧编码失败: %v", name, err)
			return
		}
		s.sendToClient(pl, frame)
	}
	if targetID == 0 {
		s.trace("npc select: 清除目标 client=%d", pl.Conn1)
		return
	}
	pause, err := entergen.CustomMessageFrame(25, entergen.CustomInt(30000), entergen.CustomString("skill_pause"))
	if err == nil {
		s.sendToClient(pl, pause)
		// m26 看门狗：5s 后解锁（官服 m25 恒 0.4~4.6s 配对；经 entry 转发，
		// 玩家离线由 entry 丢弃，安全）。
		time.AfterFunc(5*time.Second, func() {
			if clear, cerr := entergen.CustomMessageFrame(26); cerr == nil {
				s.sendToClient(pl, clear)
			}
		})
	}
	s.trace("npc select: 目标 id=%d owner=%d client=%d（m25 已授权 5s 看门狗）", targetID, targetOwner, pl.Conn1)
}
