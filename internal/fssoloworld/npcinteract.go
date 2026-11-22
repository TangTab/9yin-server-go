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
	"fmt"
	"strings"
	"sync"
	"time"

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

// talkMenuItem cinematic 菜单项（funcID/textID 对）。
type talkMenuItem struct {
	funcID int32
	textID string
}

// npcTalkMenuItems 服务列 → cinematic 菜单项；无服务兜底「交谈」；尾项
// 恒「离开」（600000000/ui_form_talk_leave——2433 定版）。
func npcTalkMenuItems(services []npcService) []talkMenuItem {
	items := make([]talkMenuItem, 0, len(services)+1)
	for _, service := range services {
		items = append(items, talkMenuItem{funcID: npcTalkFuncID(service.mark), textID: npcTalkTextID(service.mark)})
	}
	if len(items) == 0 {
		items = append(items, talkMenuItem{funcID: 860000001, textID: "ui_talk"})
	}
	return append(items, talkMenuItem{funcID: 600000000, textID: "ui_form_talk_leave"})
}

func npcTalkFuncID(mark uint16) int32 {
	switch mark {
	case npcMarkShop:
		return 805000000 // 商店
	case npcMarkDepot:
		return 807000000 // 仓库
	case npcMarkTransport:
		return 860000022 // 传送
	case npcMarkJob:
		return 806000036 // 生活职业
	case npcMarkGem:
		return 806000013 // 宝石服务
	case npcMarkGuild:
		return 809000000 // 帮会
	case npcMarkHome:
		return 881000001 // 回城点
	case npcMarkTask:
		return 100000001 // 任务入口占位
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
	case npcMarkTask:
		return "ui_form_talk_receive_task"
	default:
		return "ui_talk"
	}
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
	frames, err := npcCinematicTalkMenu(spawn.ObjectID, spawn.OwnerID, npcTalkMenuItems(services))
	if err != nil {
		s.trace("npc interact: 菜单编码失败 %s: %v", spawn.ConfigID, err)
		return true
	}
	for _, frame := range frames {
		s.sendToClient(pl, frame)
	}
	npcMenuMu.Lock()
	npcMenuState[pl.Conn1] = &npcActiveMenu{objectID: spawn.ObjectID, ownerID: spawn.OwnerID, at: time.Now()}
	npcMenuMu.Unlock()
	s.trace("npc interact: %s (%s) 菜单 %d 项 client=%d", spawn.ConfigID, spawn.Script, len(services)+1, pl.Conn1)
	return true
}

// handleNPCMenuSelect 菜单第二段：mark 路由。
func (s *Server) handleNPCMenuSelect(pl *Player, req npcObjectRequest) bool {
	npcMenuMu.Lock()
	menu := npcMenuState[pl.Conn1]
	npcMenuMu.Unlock()
	switch req.FuncID {
	case 600000000: // ui_form_talk_leave
		s.sendNPCMenuClose(pl)
		s.clearNPCMenu(pl)
		return true
	case 0:
		s.sendToClient(pl, []byte{0x1D}) // SERVER_CLEAR_MENU
		s.clearNPCMenu(pl)
		return true
	}
	if menu == nil || menu.objectID != req.ObjectID || menu.ownerID != req.OwnerID {
		s.trace("npc interact: 菜单选择无活动菜单 funcID=%d id=%d", req.FuncID, req.ObjectID)
		return true
	}
	// 业务 mark：第一阶段无商店/仓库 View 实现——关菜单并留痕。
	s.sendNPCMenuClose(pl)
	s.clearNPCMenu(pl)
	s.trace("npc interact: 菜单选择 funcID=%d id=%d（业务 View 待接）", req.FuncID, req.ObjectID)
	return true
}

func (s *Server) clearNPCMenu(pl *Player) {
	npcMenuMu.Lock()
	delete(npcMenuState, pl.Conn1)
	npcMenuMu.Unlock()
}

// sendNPCMenuClose cinematic 关闭（msg8）。
func (s *Server) sendNPCMenuClose(pl *Player) {
	if frame, err := entergen.CustomMessageFrame(8); err == nil {
		s.sendToClient(pl, frame)
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

// npcCinematicTalkMenu cinematic 菜单帧族（2433 encodeCinematicTalkMenu
// 定版）：msg4 begin / msg6 [funcID, textID]×N / msg7 [obj] end——该客户端
// 不接受字符串别名，首值必须数字。
func npcCinematicTalkMenu(objectID, ownerID uint32, items []talkMenuItem) ([][]byte, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("talk menu has no items")
	}
	out := make([][]byte, 0, 2+len(items))
	begin, err := entergen.CustomMessageFrame(4)
	if err != nil {
		return nil, err
	}
	out = append(out, begin)
	for _, item := range items {
		pkt, err := entergen.CustomMessageFrame(6, entergen.CustomInt(item.funcID), entergen.CustomString(item.textID))
		if err != nil {
			return nil, err
		}
		out = append(out, pkt)
	}
	end, err := entergen.CustomMessageFrame(7, entergen.CustomObject(objectID, ownerID))
	if err != nil {
		return nil, err
	}
	return append(out, end), nil
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
