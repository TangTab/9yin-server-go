package fssoloworld

// C2S 自定义消息（GameVisual:CustomSend，0x1E/0x27 TVarList 或 0x0A
// 活动信封携带）接收链——fork 客户端本地不落盘，键位/快捷栏/个性配置
// 全靠「上传 → 落 roleState → 回推确认」闭环收敛。规则移植参考端
// cmd/protocol-probe custom_config.go/shortcut_records.go/custom_c2s.go：
//
//	107 个性配置整串上传（首传闸：登录期默认转储不覆盖存档）；
//	195 键位方案上传 [195, scheme, str]（scheme 0=当前/1=方案一/2=方案二，
//	    同款首传闸）；
//	99 操作模式 [99, mode]（0=3D WASD/1=2.5D 点击）；
//	3 走/跑切换（会话级镜像，极性待真机定版，只记日志）；
//	199 内功栏绑定串 [199, "v;×25;"]（25 格=form_main_shortcut
//	    G_NG_GRID_COUNT）；
//	200/201 快捷栏设置/删除（按可见栏位索引；应答走整表重建方言：
//	    倒序 0x12 清空 + 0x11 整表重放——增量「旧行 DEL+尾行 ADD」在
//	    本客户端上行号口径会漂，真机实证拖出不移除/图标重叠）；
//	240 原生打坐切换（逻辑态机未落地，暂只记日志）。
//
// 帧下发与内存态优先于持久化：写 roleState 后尽力 sendRoleSave
//（store 0x04），失败仅告警不断会话。

import (
	"strconv"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// C2S 自定义消息号（shortcut_game.lua/customizing_manager 直读口径）。
const (
	clientCustomSaveCustomizing int32 = 107
	clientCustomSetOperateMode  int32 = 99
	clientCustomChangeMoveType  int32 = 3
	clientCustomNeiGongBind     int32 = 199
	clientCustomKeybindUpload   int32 = 195
	clientCustomSetShortcut     int32 = 200
	clientCustomRemoveShortcut  int32 = 201
	clientCustomSitCross        int32 = 240
)

// serverCustomPushCustomizing S2C 705（与 entergen.customcfg 同源）。
const serverCustomPushCustomizing int32 = 705

// customizingConfigLimit / keybindValueLimit 引擎侧串宽上限（Java
// handleKeybindUpload 校验同口径）。
const (
	customizingConfigLimit = 65535
	keybindValueLimit      = 4096
)

// neiGongBindSlotCount 内功栏绑定串固定格数（get_svr_ng_bind 按拆段数
// 校验，不是 26 段就整串回落全零）。串形 = 25 个 "v;" + 尾分号。
const neiGongBindSlotCount = 25

// fork 客户端玩家对象恒 1-1（enterContext 同锚；属性帧/记录行挂点）。
const (
	forkPlayerObjectID uint32 = 1
	forkPlayerOwnerID  uint32 = 1
)

// clientCustom op0x1E 自定义消息入口（frame 含 opcode 字节）。
// 解析失败静默忽略（畸形流量不断会话）；无 roleState 的会话（未进场/
// 旧档回退）拒收编辑但保留日志。
func (s *Server) clientCustom(pl *Player, frame []byte) bool {
	message, err := entergen.ParseCustomMessage(frame)
	if err != nil {
		s.trace("custom message parse failed client=%d: %v", pl.Conn1, err)
		return true
	}
	s.dispatchCustom(pl, message)
	return true
}

// clientActivity op0x0A 活动信封入口：67B 位移包先走位置收编（NPC AOI
// 视口对账 + roleState.Position 写回）；21B 头 + TVarList 形态才消费，
// 其余活动流量原样放行（返回 false 交默认分支簿记）。
func (s *Server) clientActivity(pl *Player, frame []byte) bool {
	// NPC 物化就绪屏障：ClientReady 后首条上行活动消息（含 67B 位移包）
	// =stage_main lua 已在跑，触发挂账的物化（幂等）。
	s.npcMaterializeOnActivity(pl)
	if x, y, z, orient, ok := parseMotionPosition(frame); ok {
		s.absorbPlayerPosition(pl, [4]float64{float64(x), float64(y), float64(z), float64(orient)})
		return true
	}
	message, ok, err := entergen.ParseActivityCustomMessage(frame)
	if err != nil || !ok {
		return true
	}
	s.dispatchCustom(pl, message)
	return true
}

// dispatchCustom 首值消息号分发（Type!=int 的畸形首值忽略）。
func (s *Server) dispatchCustom(pl *Player, message entergen.CustomInbound) {
	if len(message.Values) == 0 || message.Values[0].Type != 2 {
		return
	}
	id := message.Values[0].Int32
	switch id {
	case clientCustomSaveCustomizing:
		s.handleCustomizingUpload(pl, message)
	case clientCustomKeybindUpload:
		s.handleKeybindUpload(pl, message)
	case clientCustomSetOperateMode:
		s.handleOperateMode(pl, message)
	case clientCustomNeiGongBind:
		s.handleNeiGongBind(pl, message)
	case clientCustomSetShortcut:
		s.handleShortcutSet(pl, message)
	case clientCustomRemoveShortcut:
		s.handleShortcutRemove(pl, message)
	case clientCustomChangeMoveType:
		s.handleMoveType(pl, message)
	case clientCustomSitCross:
		s.handleSitcross(pl, message)
	case clientCustomMovementUpload:
		s.handleMovementUpload(pl, message)
	case clientCustomTaskMsg:
		s.handleTaskMsg(pl, message)
	case clientCustomPostRead:
		s.handleReadLetter(pl, message)
	case clientCustomPostGetAppendix:
		s.handleGetAppendix(pl, message)
	case clientCustomPostOpenBox, clientCustomPostSelect, clientCustomPostCloseReadBox,
		clientCustomPostSelectSend, clientCustomPostCancel, clientCustomPostSend:
		// 邮件族零动作面：开箱/勾选/关窗/写信族（列表=data_binder 绑
		// 记录行；玩家发信 352 未实现）——trace 簿记。
		s.handlePostTrace(pl, id)
	case clientCustomSelectTarget:
		// msg431 选中目标（点击 NPC/对象）：LastObject/AutoSelectObject
		// 双帧 + m25 选靶授权（2433 selection.go 定版）。
		s.handleNPCSelectTarget(pl, message)
	default:
		s.trace("custom message unhandled client=%d id=%d values=%d", pl.Conn1, id, len(message.Values))
	}
}

// requireRoleState 编辑类消息的前置：无 roleState（未进场/旧档 replay
// 会话）无法归属持久化，拒收。
func (s *Server) requireRoleState(pl *Player, id int32) bool {
	if pl.haveRoleState {
		return true
	}
	s.trace("reject custom %d before role state ready client=%d", id, pl.Conn1)
	return false
}

// pushCustomizing 705 回推（空串零帧）。
func (s *Server) pushCustomizing(pl *Player, config string) error {
	frame, err := entergen.CustomMessageFrame(serverCustomPushCustomizing, entergen.CustomString(config))
	if err != nil {
		return err
	}
	s.sendToClient(pl, frame)
	return nil
}

// pushKeybindScheme 键位域 0x10 单属性帧（198/199/200，串 = 值 + " "
// 尾空格——与 0x0D 档内嵌无尾空格形态不可混用）。
func (s *Server) pushKeybindScheme(pl *Player, scheme int32, value string) error {
	name := map[int32]string{0: "ShortcutKey", 1: "ShortcutKey1", 2: "ShortcutKey2"}[scheme]
	if name == "" {
		return strconv.ErrRange
	}
	frame, err := entergen.ObjectPropertyFrame(forkPlayerObjectID, forkPlayerOwnerID, 0,
		[]entergen.IndexedProperty{entergen.NamedProperty(name, entergen.StringValue(value+" "))})
	if err != nil {
		return err
	}
	s.sendToClient(pl, frame)
	return nil
}

// pushNeiGongBind 0x10 属性 201（同键位域尾空格方言）。
func (s *Server) pushNeiGongBind(pl *Player, value string) error {
	frame, err := entergen.ObjectPropertyFrame(forkPlayerObjectID, forkPlayerOwnerID, 0,
		[]entergen.IndexedProperty{entergen.NamedProperty("NeiGongBind", entergen.StringValue(value+" "))})
	if err != nil {
		return err
	}
	s.sendToClient(pl, frame)
	return nil
}

// pushOperateMode 0x10 属性 223（int32）。
func (s *Server) pushOperateMode(pl *Player, mode int32) error {
	frame, err := entergen.ObjectPropertyFrame(forkPlayerObjectID, forkPlayerOwnerID, 0,
		[]entergen.IndexedProperty{entergen.NamedProperty("OperateMode", entergen.Int32Value(mode))})
	if err != nil {
		return err
	}
	s.sendToClient(pl, frame)
	return nil
}

// handleCustomizingUpload C2S 107 [107, string]。首传闸：客户端登录期
// 会先上传默认键位转储（恢复帧未必已应用）——首传且库存有档时跳过
// 落库存档、回发存档再确认恢复态；后续上传（编辑提交）照常收。
func (s *Server) handleCustomizingUpload(pl *Player, message entergen.CustomInbound) {
	if !s.requireRoleState(pl, clientCustomSaveCustomizing) {
		return
	}
	if len(message.Values) != 2 || message.Values[1].Type != 6 {
		s.trace("customizing expects (107, string), got %d values client=%d", len(message.Values), pl.Conn1)
		return
	}
	config := message.Values[1].Text
	if config == "" {
		return
	}
	if len(config) > customizingConfigLimit {
		s.trace("customizing config length %d exceeds %d client=%d", len(config), customizingConfigLimit, pl.Conn1)
		return
	}
	first := !pl.customizingSeen
	pl.customizingSeen = true
	// 705 echo 应答永禁（2026-11-22 深夜七轮二分定案，debug 分支同款
	// 修复移植）：客户端 on_parser_customizing 每收一帧 705 强制重置
	// 引擎控制对象（game_control_mode/camera_value，2433 参考端
	// custom_config.go:226 注释同证）——登录期应答式回推反复触发重置链，
	// 重登后持续掉帧/操作延时。只落内存+存档，不回推。
	if first && pl.roleState.Customizing != "" {
		s.trace("skipped default customizing dump (%d bytes) client=%d",
			len(config), pl.Conn1)
		return
	}
	pl.roleState.Customizing = config
	s.sendRoleSave(pl)
	s.trace("saved customizing config %d bytes client=%d (echo muted)", len(config), pl.Conn1)
}

// printableKeybindValue 键位串限可打印 ASCII（0x20-0x7E）——真机引擎会
// 发 649B 形态串，自旧 [0-9,;]+ 白名单放宽（Java 同口径）。
func printableKeybindValue(value string) bool {
	for index := 0; index < len(value); index++ {
		if value[index] < 0x20 || value[index] > 0x7E {
			return false
		}
	}
	return true
}

// handleKeybindUpload C2S 195 [195, scheme, string]。首传闸同 107：
// 登录转储不覆盖存档（「重登丢改键」根因——真机日志实证转储 658B 把
// 已入库的编辑覆盖回默认）；非法 scheme/串静默拒（零帧）。
func (s *Server) handleKeybindUpload(pl *Player, message entergen.CustomInbound) {
	if !s.requireRoleState(pl, clientCustomKeybindUpload) {
		return
	}
	if len(message.Values) < 3 {
		s.trace("keybind expects (195, scheme, string), got %d values client=%d", len(message.Values), pl.Conn1)
		return
	}
	scheme, ok := message.Values[1].ExactInt32()
	bind := message.Values[2].Text
	if !ok || message.Values[2].Type != 6 {
		s.trace("keybind scheme/string mismatch client=%d", pl.Conn1)
		return
	}
	if scheme < 0 || scheme > 2 || bind == "" || len(bind) > keybindValueLimit || !printableKeybindValue(bind) {
		s.trace("ignore invalid keybind upload scheme=%d len=%d client=%d", scheme, len(bind), pl.Conn1)
		return
	}
	saved := map[int32]string{
		0: pl.roleState.Keybind, 1: pl.roleState.Keybind1, 2: pl.roleState.Keybind2,
	}[scheme]
	first := !pl.keybindSeen
	pl.keybindSeen = true
	if first && scheme == 0 && saved != "" {
		if err := s.pushKeybindScheme(pl, 0, saved); err != nil {
			s.trace("re-assert keybind failed client=%d: %v", pl.Conn1, err)
			return
		}
		s.trace("skipped login keybind dump (%d bytes), re-asserted saved %d bytes client=%d", len(bind), len(saved), pl.Conn1)
		return
	}
	if err := s.pushKeybindScheme(pl, scheme, bind); err != nil {
		s.trace("echo keybind failed client=%d: %v", pl.Conn1, err)
		return
	}
	switch scheme {
	case 0:
		pl.roleState.Keybind = bind
	case 1:
		pl.roleState.Keybind1 = bind
	case 2:
		pl.roleState.Keybind2 = bind
	}
	s.sendRoleSave(pl)
	s.trace("keybind scheme %d uploaded (%d bytes)%s client=%d", scheme, len(bind),
		map[bool]string{true: " (first)", false: ""}[first], pl.Conn1)
}

// handleOperateMode C2S 99 [99, mode]：回推 0x10 属性 223 同步客户端
// 属性缓存（跨服重连 lua 直接 QueryProp），再落库。非法值静默拒。
func (s *Server) handleOperateMode(pl *Player, message entergen.CustomInbound) {
	if !s.requireRoleState(pl, clientCustomSetOperateMode) {
		return
	}
	if len(message.Values) < 2 {
		s.trace("operate mode expects (99, mode), got %d values client=%d", len(message.Values), pl.Conn1)
		return
	}
	mode, ok := message.Values[1].ExactInt32()
	if !ok || (mode != 0 && mode != 1) {
		s.trace("ignore unknown operate mode client=%d", pl.Conn1)
		return
	}
	if err := s.pushOperateMode(pl, mode); err != nil {
		s.trace("send operate mode failed client=%d: %v", pl.Conn1, err)
		return
	}
	pl.roleState.OperateMode, pl.roleState.OperateModeSet = mode, true
	s.sendRoleSave(pl)
	s.trace("operate mode set %d (%s) client=%d", mode,
		map[bool]string{true: "2.5D click", false: "3D WASD"}[mode == 1], pl.Conn1)
}

// validNeiGongBind 校验内功栏绑定串：恰好 25 个值加尾分号（拆 26 段、
// 末段空），各值 0..99（绑定框 DataSource 序号，0=清除绑定）。不符
// 整串拒收（零帧），防脏串落库把客户端档位校验打回全零。
func validNeiGongBind(value string) bool {
	if !strings.HasSuffix(value, ";") {
		return false
	}
	fields := strings.Split(value, ";")
	if len(fields) != neiGongBindSlotCount+1 || fields[neiGongBindSlotCount] != "" {
		return false
	}
	for _, field := range fields[:neiGongBindSlotCount] {
		if field == "" || len(field) > 2 {
			return false
		}
		number, err := strconv.Atoi(field)
		if err != nil || number < 0 || number > 99 {
			return false
		}
	}
	return true
}

// handleNeiGongBind C2S 199 [199, string]：校验 → 回推 0x10 属性 201 →
// 落库。update_to_srv 靠与属性缓存比对决定是否再上传——不回推则两端
// 永不收敛（表象：内功栏绑定序号重登全零）。
func (s *Server) handleNeiGongBind(pl *Player, message entergen.CustomInbound) {
	if !s.requireRoleState(pl, clientCustomNeiGongBind) {
		return
	}
	if len(message.Values) != 2 || message.Values[1].Type != 6 {
		s.trace("neigong bind expects (199, string), got %d values client=%d", len(message.Values), pl.Conn1)
		return
	}
	value := message.Values[1].Text
	if !validNeiGongBind(value) {
		s.trace("ignore invalid neigong bind len=%d client=%d", len(value), pl.Conn1)
		return
	}
	if err := s.pushNeiGongBind(pl, value); err != nil {
		s.trace("echo neigong bind failed client=%d: %v", pl.Conn1, err)
		return
	}
	pl.roleState.NeiGongBind = value
	s.sendRoleSave(pl)
	bound := 0
	for _, field := range strings.Split(strings.TrimSuffix(value, ";"), ";") {
		if field != "0" {
			bound++
		}
	}
	s.trace("neigong bind uploaded (%d/%d slots bound) client=%d", bound, neiGongBindSlotCount, pl.Conn1)
}

// materializeShortcutBaseline 首次编辑时把生效表（含 starter 打坐条目）
// 落成存档基线——玩家所见即所存；此后内存表是唯一事实源。
func materializeShortcutBaseline(pl *Player) {
	if pl.roleState.ShortcutsSet {
		return
	}
	pl.roleState.Shortcuts = pl.roleState.EffectiveShortcuts()
	pl.roleState.ShortcutsSet = true
}

// rebuildShortcutRecord 快捷栏编辑应答方言：先把 shortcut_rec 从尾行
// 到 0 倒序清空，再按内存序整表重建（倒序删行使行号重编号永不参与；
// 帧序构造见 entergen.ShortcutRebuildFrames）。
func (s *Server) rebuildShortcutRecord(pl *Player, oldRows int) error {
	frames, err := entergen.ShortcutRebuildFrames(forkPlayerObjectID, forkPlayerOwnerID, oldRows, pl.roleState.Shortcuts)
	if err != nil {
		return err
	}
	for _, frame := range frames {
		s.sendToClient(pl, frame)
	}
	return nil
}

// handleShortcutSet C2S 200 [200, int index, string kind, string id]：
// 目标槽位为权威设置；变更后整表重建应答 + 落库。
func (s *Server) handleShortcutSet(pl *Player, message entergen.CustomInbound) {
	if !s.requireRoleState(pl, clientCustomSetShortcut) {
		return
	}
	if len(message.Values) != 4 ||
		message.Values[1].Type != 2 || message.Values[2].Type != 6 || message.Values[3].Type != 6 {
		s.trace("set shortcut expects (200, int, string, string), got %d values client=%d", len(message.Values), pl.Conn1)
		return
	}
	index := message.Values[1].Int32
	kind, id := message.Values[2].Text, message.Values[3].Text
	if index < 0 || index > 4096 {
		s.trace("shortcut index %d out of range client=%d", index, pl.Conn1)
		return
	}
	if kind == "" || id == "" || len(kind) > 64 || len(id) > 256 {
		s.trace("invalid shortcut kind/id lengths %d/%d client=%d", len(kind), len(id), pl.Conn1)
		return
	}
	materializeShortcutBaseline(pl)
	oldRows := len(pl.roleState.Shortcuts)
	replaced := false
	for i, row := range pl.roleState.Shortcuts {
		if row.Index != index {
			continue
		}
		if row.Kind == kind && row.ID == id {
			s.trace("shortcut unchanged index=%d client=%d", index, pl.Conn1)
			return
		}
		pl.roleState.Shortcuts[i] = rolestate.ShortcutRow{Index: index, Kind: kind, ID: id}
		replaced = true
		break
	}
	if !replaced {
		pl.roleState.Shortcuts = append(pl.roleState.Shortcuts, rolestate.ShortcutRow{Index: index, Kind: kind, ID: id})
	}
	if err := s.rebuildShortcutRecord(pl, oldRows); err != nil {
		s.trace("rebuild after set index %d failed client=%d: %v", index, pl.Conn1, err)
		return
	}
	s.sendRoleSave(pl)
	s.trace("shortcut set index=%d kind=%s id=%s replaced=%t client=%d", index, kind, id, replaced, pl.Conn1)
}

// handleShortcutRemove C2S 201 [201, int index]：shortcut_game.lua
// remove_shortcut 全部调用点传可见栏位索引（非记录行号）。删除即删除
// ——starter 条目被删也不复活（基线已物化）。
func (s *Server) handleShortcutRemove(pl *Player, message entergen.CustomInbound) {
	if !s.requireRoleState(pl, clientCustomRemoveShortcut) {
		return
	}
	if len(message.Values) != 2 || message.Values[1].Type != 2 {
		s.trace("remove shortcut expects (201, int), got %d values client=%d", len(message.Values), pl.Conn1)
		return
	}
	index := message.Values[1].Int32
	if index < 0 || index > 4096 {
		s.trace("shortcut index %d out of range client=%d", index, pl.Conn1)
		return
	}
	materializeShortcutBaseline(pl)
	oldRows := len(pl.roleState.Shortcuts)
	for i, row := range pl.roleState.Shortcuts {
		if row.Index != index {
			continue
		}
		pl.roleState.Shortcuts = append(pl.roleState.Shortcuts[:i], pl.roleState.Shortcuts[i+1:]...)
		if err := s.rebuildShortcutRecord(pl, oldRows); err != nil {
			s.trace("rebuild after remove index %d failed client=%d: %v", index, pl.Conn1, err)
			return
		}
		s.sendRoleSave(pl)
		s.trace("shortcut removed index=%d client=%d", index, pl.Conn1)
		return
	}
	s.trace("ignore missing shortcut index=%d client=%d", index, pl.Conn1)
}

// handleMoveType C2S 3 走/跑切换（fxgamelogic 切换键与点击移动都会发）：
// 客户端本地权威，服务端只记录（IsRun 回显曾疑似构成「切换→回显→
// 翻回」回路，2026-11-15 参考端 A/B 撤回）。
func (s *Server) handleMoveType(pl *Player, message entergen.CustomInbound) {
	if len(message.Values) < 2 {
		return
	}
	mode, ok := message.Values[1].ExactInt32()
	if !ok {
		return
	}
	s.trace("move type change -> %d client=%d (walk/run toggle; polarity pending)", mode, pl.Conn1)
}

// handleSitcross C2S 240 原生打坐切换 [240, int startOrStop]。逻辑态机
// （LS_SITCROSS/vital 0x10 + dazuo_old 动作飘字）未落地，暂只记日志。
func (s *Server) handleSitcross(pl *Player, message entergen.CustomInbound) {
	if len(message.Values) < 2 {
		return
	}
	state, ok := message.Values[1].ExactInt32()
	if !ok {
		return
	}
	s.trace("sitcross native toggle -> %d client=%d (logic state pending)", state, pl.Conn1)
}
