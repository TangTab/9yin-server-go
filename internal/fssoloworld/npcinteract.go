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
// 服务列两级绑定：①模板行业务列（ShopID/DepotID/TransFuncID/GemGameNpcID/
// GuildFunc/HomePoint/Job——2433 modernNPCServices 同源语义；本档位全树无
// 此类列，为 2433 modern 数据预留）②NpcType 结构域回落（客户端
// lua/share/npc_type_define.lua 类型表定版：61 武馆=技能训练、62~68 商铺
// 族、45 马车船只、109/121~137 生活职业族——2026-11-23 实测档位唯一逐
// NPC 功能源）。菜单文案键与 funcID 映射照抄 2433 modernTalkFuncID/
// TextID 定版（Trainer 行取 legacyTalkFuncID(47) 对）。

import (
	"encoding/binary"
	"strconv"
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
	// services 开窗时的服务列（npcshop.go mark 路由用：805000000 商店
	// 取 ShopID 值、807000000 仓库——npcService.value 携带绑定真值）。
	services []npcService
	// pendingTask 接取窗在途任务（npctask.go：100 首次开窗记录、窗内
	// 「接受任务」同 funcid 二次到达=真接取；换行/关窗清除）。
	pendingTask int
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
	npcMarkTrainer   uint16 = 0x1009
)

// npcTypeServiceMarks NpcType → 服务 mark（lua/share/npc_type_define.lua：
// 45 马车船只 / 61 武馆 / 62 铁匠铺 63 绸缎庄 64 首饰行 65 杂货店
// 66 药堂 67 钱庄 68 特产商 / 109 生活职业NPC / 121~137 矿工..乞丐十六
// 生活职业）。表外类型（怪物/传送门/机关/采集/世界域 NPC）无对话服务
// 语义，不派生。
var npcTypeServiceMarks = map[uint16]uint16{
	45:  npcMarkTransport,
	61:  npcMarkTrainer,
	62:  npcMarkShop,
	63:  npcMarkShop,
	64:  npcMarkShop,
	65:  npcMarkShop,
	66:  npcMarkShop,
	67:  npcMarkShop,
	68:  npcMarkShop,
	109: npcMarkJob,
	121: npcMarkJob, 122: npcMarkJob, 123: npcMarkJob, 124: npcMarkJob,
	125: npcMarkJob, 126: npcMarkJob, 127: npcMarkJob, 128: npcMarkJob,
	129: npcMarkJob, 130: npcMarkJob, 131: npcMarkJob, 132: npcMarkJob,
	133: npcMarkJob, 134: npcMarkJob, 135: npcMarkJob, 136: npcMarkJob,
	137: npcMarkJob,
}

// npcServicesOf 模板行两级服务提取：业务列（非空且非 "0" 才算配置，
// 2433 modern 数据方言；npcgen 2026-11-23 已折叠商店ID/仓库ID 等七列）
// → NpcType 结构域回落（仅当零业务列——显式数据权威，防钱庄
// DepotID+NpcType67 启发式叠出虚构商店行）。value：业务列取列值；
// NpcType 派生取类型号（审计留痕）。
func npcServicesOf(tpl map[string]string) []npcService {
	add := func(items []npcService, mark uint16, key string) []npcService {
		if v := strings.TrimSpace(tpl[key]); v != "" && v != "0" {
			return append(items, npcService{mark: mark, label: npcServiceLabel(mark), value: v})
		}
		return items
	}
	items := make([]npcService, 0, 6)
	items = add(items, npcMarkShop, "ShopID")
	items = add(items, npcMarkDepot, "DepotID")
	items = add(items, npcMarkTransport, "TransFuncID")
	items = add(items, npcMarkGem, "GemGameNpcID")
	items = add(items, npcMarkGuild, "GuildFunc")
	items = add(items, npcMarkHome, "HomePoint")
	items = add(items, npcMarkJob, "Job")
	if len(items) > 0 {
		return items // 显式业务列权威，NpcType 回落不启用
	}
	typeRaw := strings.TrimSpace(tpl["NpcType"])
	if mark, ok := npcTypeMarkOf(typeRaw); ok {
		items = append(items, npcService{
			mark:  mark,
			label: npcServiceLabel(mark),
			value: typeRaw,
		})
	}
	return items
}

// npcTypeMarkOf NpcType 字符串 → mark（脏值/表外类型零值回落）。
func npcTypeMarkOf(typeRaw string) (uint16, bool) {
	if typeRaw == "" {
		return 0, false
	}
	v, err := strconv.Atoi(typeRaw)
	if err != nil {
		return 0, false
	}
	mark, ok := npcTypeServiceMarks[uint16(v)]
	return mark, ok
}

// npcServiceLabel mark → 0x1C 菜单中文标签（npcServerMenu 域）。
func npcServiceLabel(mark uint16) string {
	switch mark {
	case npcMarkShop:
		return "商店"
	case npcMarkDepot:
		return "仓库"
	case npcMarkTransport:
		return "传送"
	case npcMarkJob:
		return "生活职业"
	case npcMarkGem:
		return "宝石服务"
	case npcMarkGuild:
		return "帮会"
	case npcMarkHome:
		return "回城点"
	case npcMarkTrainer:
		return "技能训练"
	}
	return "交谈"
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
	// 先回选中双帧（npc.pcapng 点击时序定版：op07 首段应答=0x10
	// LastObject+AutoSelectObject 双帧先行——C++ 对话流程依赖选中态，
	// 缺它 m4 聚合链不跑=窗不开）。
	s.sendSelectTargetFrames(pl, spawn.ObjectID, spawn.OwnerID)
	services := npcServicesOf(spawn.Template)
	s.sendNPCTalkMenu(pl, spawn, services)
	npcMenuMu.Lock()
	npcMenuState[pl.Conn1] = &npcActiveMenu{
		objectID: spawn.ObjectID, ownerID: spawn.OwnerID, at: time.Now(),
		services: services,
	}
	npcMenuMu.Unlock()
	pauseNpcTalk(spawn.ObjectID) // 行人对话停走（菜单关闭恢复）
	return true
}

// sendNPCTalkMenu talk 帧族下发（m4/m5/m6×N/m7）：服务行 + 任务行 +
// 通用行。任务行来自 taskdata 快照绑定（npctask.go——客户端不自行生成
// 任务行，服务端不发即无任务入口）。services 由调用方计算（菜单簿记
// 共用一份，npcshop.go mark 路由取绑定值）。
func (s *Server) sendNPCTalkMenu(pl *Player, spawn npcfeed.Spawn, services []npcService) {
	tasks := s.npcTaskLines(s.npcTaskCatalogOf(), pl, spawn.ConfigID, npcTaskMenuLimit())
	if frames, err := npcCinematicTalkMenu(spawn.ObjectID, spawn.OwnerID, spawn.ConfigID, services, tasks); err == nil {
		for _, frame := range frames {
			s.sendToClient(pl, frame)
		}
	} else {
		s.trace("npc interact: talk 数据编码失败 %s: %v", spawn.ConfigID, err)
	}
	s.trace("npc interact: %s (%s) 菜单 %d 项（任务行 %d）client=%d",
		spawn.ConfigID, spawn.Script, len(services)+3, len(tasks), pl.Conn1)
}

// handleNPCMenuSelect 菜单第二段：0x1C SERVER_MENU 选择的 mark 路由
// （mark=0 离开 → m8×2 清菜单；任务域 mark 100/200 → 接取/提交受理
// （npctask.go，不关窗刷新菜单）；其余业务 mark → 本阶段清菜单 + trace
// 留痕，业务 View 待接）。
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
	// 任务域 mark：接取（100）/提交（200/201）/剧情（500/50N/60N）受理
	//（引导抓包实证 + lua 推断，npctask.go——接取后刷新菜单，提交/剧情
	// 中断后 m8×2 关窗，剧情段链尾翻任务行）。
	if req.FuncID != 0 {
		if _, _, ok := npcTaskMarkSplit(req.FuncID); ok {
			s.handleNPCTaskMark(pl, menu, req.FuncID)
			return true
		}
	}
	// 服务域 mark：关对话 → 业务 View（2433 selectCinematicTalkLocked
	// 定序：closeCinematicTalk → clearActiveMenu → openSelectedService）。
	if svc, ok := npcMenuServiceByFuncID(menu.services, req.FuncID); ok {
		switch req.FuncID {
		case 805000000: // 商店
			s.closeNPCTalk(pl)
			s.clearNPCMenu(pl)
			if svc.value == "" {
				s.trace("npc shop: %s 商店服务无 ShopID（npcshop.go 数据源待补）", svc.label)
				return true
			}
			s.openNPCShopView(pl, svc.value)
			return true
		case 807000000: // 仓库
			s.closeNPCTalk(pl)
			s.clearNPCMenu(pl)
			s.openNPCDepotView(pl)
			return true
		}
	}
	// 关菜单走官方 m8 对话结束 ×2（女号抓包 102.4s / npc.pcapng 32.0s
	// 双源：结束簇恒两条；全程无 0x1C/0x1D）。
	s.closeNPCTalk(pl)
	s.clearNPCMenu(pl)
	if req.FuncID == 0 {
		s.trace("npc interact: 菜单关闭 id=%d client=%d", req.ObjectID, pl.Conn1)
		return true
	}
	s.trace("npc interact: 菜单选择 %s（mark=%d）id=%d client=%d（业务 View 待接）",
		npcServiceNameByFuncID(req.FuncID), req.FuncID, req.ObjectID, pl.Conn1)
	return true
}

// closeNPCTalk 对话结束 m8×2（女号抓包/npc.pcapng 双源定版：结束簇恒两条）。
func (s *Server) closeNPCTalk(pl *Player) {
	for i := 0; i < 2; i++ {
		if end, err := entergen.CustomMessageFrame(8); err == nil {
			s.sendToClient(pl, end)
		}
	}
}

// npcMenuServiceByFuncID 活动菜单服务列按回执 funcid 匹配（funcid 域
// 见 npcTalkFuncID）。
func npcMenuServiceByFuncID(services []npcService, funcID int32) (npcService, bool) {
	for _, svc := range services {
		if npcTalkFuncID(svc.mark) == funcID {
			return svc, true
		}
	}
	return npcService{}, false
}

// npcServiceNameByFuncID 菜单回执 funcid → 服务名（trace 观测面；业务
// View 接入后此处替换为各服务受理）。
func npcServiceNameByFuncID(funcID int32) string {
	switch funcID {
	case 805000000:
		return "商店"
	case 807000000:
		return "仓库"
	case 860000022:
		return "传送"
	case 806000036:
		return "生活职业"
	case 806000013:
		return "宝石服务"
	case 809000000:
		return "帮会"
	case 881000001:
		return "回城点"
	case 860000047:
		return "技能训练"
	}
	return "未知服务"
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

// npcCinematicTalkMenu talk 数据帧族（npc.pcapng #2377-2382 + 女号官服
// 抓包 88.2s 段双源定版）：msg4 [4] begin / **msg5 [5, "task_<链键>"]**
// / msg6 [6, funcid, ui文本ID]×N（funcid 用 form_talk_data g_menu_id 域，
// 官方离开项 600000000+"ui_exit" 实证同系）/ msg7 [7, obj] end。
// msg5 为开窗必备（两份官服抓包每一窗必带：引导 NPC
// "task_funnpcborn003031"、世界 NPC "task_WorldNpc01230"、功能 NPC
// "task_FuncNpc00107"——键形恒 "task_"+ConfigID，客户端按本地任务链
// 表解析后聚合 form_talk 开窗；此前缺 m5 = 点击 NPC 无窗）。曾发 NPC
// 名 UTF-8 / "ui_form_talk" 乱码是键形错误，非 m5 本身不可发。
// tasks 任务行（npctask.go 绑定派生）置于服务行之前——客户端
// talk_data_add_save_menu 按 flag 域分区聚合，任务行归任务区置顶。
func npcCinematicTalkMenu(objectID, ownerID uint32, configID string, services []npcService, tasks []npcTaskLine) ([][]byte, error) {
	out := make([][]byte, 0, 3+len(services)+len(tasks))
	begin, err := entergen.CustomMessageFrame(4)
	if err != nil {
		return nil, err
	}
	out = append(out, begin)
	menu, err := entergen.CustomMessageFrame(5, entergen.CustomString("task_"+configID))
	if err != nil {
		return nil, err
	}
	out = append(out, menu)
	taskFrames, err := npcTaskMenuFrames(tasks)
	if err != nil {
		return nil, err
	}
	out = append(out, taskFrames...)
	for _, service := range services {
		pkt, err := entergen.CustomMessageFrame(6, entergen.CustomInt(npcTalkFuncID(service.mark)), entergen.CustomString(npcTalkTextID(service.mark)))
		if err != nil {
			return nil, err
		}
		out = append(out, pkt)
	}
	// 世界 NPC 通用行（npc.pcapng 30.275s 官方字节模板：赠送 + 侠恶查询
	// ——898000024/898000023 为客户端 ui_* 通用键，每窗必带；侠恶行带
	// 第三参 int 1000。真机 64344 会话实证：无内容行（仅 ui_exit）时
	// 对话数据虽被受理（客户端回 m298 查询）但窗体不渲染）。
	present, err := entergen.CustomMessageFrame(6, entergen.CustomInt(898000024), entergen.CustomString("ui_present_to_npc_item"))
	if err != nil {
		return nil, err
	}
	out = append(out, present)
	karma, err := entergen.CustomMessageFrame(6, entergen.CustomInt(898000023), entergen.CustomString("ui_karma_query"), entergen.CustomInt(1000))
	if err != nil {
		return nil, err
	}
	out = append(out, karma)
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
// 定版：func_menu_shop=805000000/func_menu_depot=807000000/…；Trainer
// 取 2433 legacyTalkFuncID(47)=860000047/ui_skill_trainer 对）。
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
	case npcMarkTrainer:
		return 860000047
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
	case npcMarkTrainer:
		return "ui_skill_trainer"
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
// sendSelectTargetFrames 选中双帧（LastObject+AutoSelectObject 两个 0x10
// 属性帧，npc.pcapng 点击应答时序的首步）。
func (s *Server) sendSelectTargetFrames(pl *Player, targetID, targetOwner uint32) {
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
}

func (s *Server) handleNPCSelectTarget(pl *Player, message entergen.CustomInbound) {
	if len(message.Values) < 2 {
		return
	}
	packed := message.Values[1].Raw
	targetID := binary.LittleEndian.Uint32(packed[:])
	targetOwner := binary.LittleEndian.Uint32(packed[4:])
	s.sendSelectTargetFrames(pl, targetID, targetOwner)
	// 记最近选靶：m211 不带目标时回落（客户端先选靶后施放）。
	pl.combat.mu.Lock()
	pl.combat.selected = uint64(targetID) | uint64(targetOwner)<<32
	pl.combat.mu.Unlock()
	if targetID == 0 {
		s.trace("npc select: 清除目标 client=%d", pl.Conn1)
		return
	}
	pause, err := entergen.CustomMessageFrame(25, entergen.CustomInt(30000), entergen.CustomString("skill_pause"))
	if err == nil {
		s.sendToClient(pl, pause)
		// m26 看门狗：5s 后解锁（官服 m25 恒 0.4~4.6s 配对；经 entry 转发，
		// 玩家离线由 entry 丢弃，安全）。冲锋暂存中不发——暂停归暂存链
		// 所有，由落地/取消/超时解除；此处抢发会让客户端提前脱离暂存。
		time.AfterFunc(5*time.Second, func() {
			pl.combat.mu.Lock()
			staged := pl.combat.staged != nil
			pl.combat.mu.Unlock()
			if staged {
				return
			}
			if clear, cerr := entergen.CustomMessageFrame(26); cerr == nil {
				s.sendToClient(pl, clear)
			}
		})
	}
	s.trace("npc select: 目标 id=%d owner=%d client=%d（m25 已授权 5s 看门狗）", targetID, targetOwner, pl.Conn1)
}
