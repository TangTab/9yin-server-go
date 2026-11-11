package main

// 宝物洗炼（CLIENT_CUSTOMMSG_TERSURE=688）v1：两套界面主链。
//
// 客户端方言（lua64 form_treasure_reset / form_treasure_reset_add +
// custom_handler.lua:5461-5519 定版）：
//
//	C2S sub 0/1  整件锁/解锁(view,index)——LockStatus 属性帧 + 会话绑定
//	C2S sub 6/7  单条属性锁/解锁(view,index,row)——RandomPropRec col2 重写
//	C2S sub 8/9  老界面 属性洗/存(view,index,-1)      → S2C 结果 sub 2/3
//	C2S sub 10/11 老界面 武学洗/存(view,index,0门派/1江湖) → S2C sub 6/7
//	C2S sub 12/13 新界面 主武学洗/存(view,index,2门派/3江湖) → S2C sub 8/9
//	其余子命令（2 满值/5 升级/14-22 娃娃淬炼族/23 套路洗/24-32 副套路族/
//	31 第二套路洗/33-34 目标锁）本批不实现，日志静默。
//
// S2C = serverCustomStringMessage("on_custom_show_tersuer_result",
// customInt(sub))（线名=lua 函数名原样，官方拼写就是 tersuer；loot
// show_hide_drop_view 同款先例）。预览走 TempTersureRec(402) 行：属性行
// [1,propID,value,lock,flag,quality,level,0] 锁定行前置（老界面把前几行
// 渲染进"保留"框），技能行 [0,storeID,value,skill_id,0,0,0,0]；行为参照=
// 前代服日志 artifacts/_r43_full.txt（kept=2 rolled=5，总数不变）。
// 提交（存）重写 RandomPropRec/RandomSkillRec（0x12 清行 + 0x11 加行，
// flag=1 视图寻址）+ 快照 upsert；客户端收结果后自行重发锁 0（隐式重锁
// 受理）。计费见 tersure_cost.go。

import (
	"encoding/binary"
	"fmt"
	"log"
	"strings"

	"github.com/local/9yin-go-server/internal/clientdata"
)

// clientCustomTersure CLIENT_CUSTOMMSG_TERSURE（client_custom_define.lua:452）。
const clientCustomTersure = 688

const recordTempTersure uint16 = 402 // TempTersureRec（官方 447 schema #402，8 列）

// 洗炼界面功能号（on_custom_show_tersure 的 sub_type；NPC 对话开窗方言，
// GM 动作经数字 517 直发）。数字形态（int32→客户端 nx_string 比对）。
const (
	tersureFormResetAddID = int32(89900021) // 新界面（主武学洗 12/13）
	tersureFormResetID    = int32(89900008) // 老界面（属性洗 8/9、武学洗 10/11）
)

// tersureWashSession 一次洗炼会话（放入锁 0 起、取出/换物解锁 1 止）。
type tersureWashSession struct {
	active   bool
	viewID   uint16 // 客户端寻址视图（洗炼界面 get_view_item 读回同源）
	slot     int32
	uid      string
	configID string
	kind     int              // randomKindTreaure / randomKindWawa
	snapshot *itemRandomProps // 会话权威快照（首用时 ensure，存后更新——
	// 锁定期间无并发修改方）
	pendingProp  []randomPropLine // 属性洗候选全量（锁定行前置）
	pendingSkill *randomSkillLine // 武学洗候选
	lockedRows   map[int]bool     // RandomPropRec 行锁（0 基行号）
	tempRows     int              // 已下发 TempTersureRec 行数（清行用）
}

// handleTersureCustom 688 入口（活动信封/直达双路都汇到这里）。
func handleTersureCustom(h *itemSession, custom clientCustomMessage) (bool, error) {
	args := customIntArgs(custom)
	if len(args) < 3 {
		log.Printf("%s: tersure 688 malformed args=%d (need sub,view,index)", h.remote, len(args))
		return true, nil
	}
	sub, view, slot := args[0], args[1], args[2]
	switch sub {
	case 0:
		tersureLockItem(h, uint16(view), slot, true)
	case 1:
		tersureLockItem(h, uint16(view), slot, false)
	case 6, 7:
		if len(args) < 4 {
			log.Printf("%s: tersure lock_prop malformed args=%d", h.remote, len(args))
			return true, nil
		}
		tersureLockPropRow(h, uint16(view), slot, args[3], sub == 6)
	case 8:
		tersureWashProps(h, uint16(view), slot)
	case 9:
		tersureSaveProps(h, uint16(view), slot)
	case 10, 12:
		skillType := int32(-1)
		if len(args) >= 4 {
			skillType = args[3]
		}
		tersureWashSkill(h, uint16(view), slot, int(skillType), sub == 12)
	case 11, 13:
		tersureSaveSkill(h, uint16(view), slot, sub == 13)
	case 14:
		// 娃娃整件锁（custom_sender.lua:4814 custom_newtersure_lock——
		// 新娃娃界面与 0/1 同语义，直接复用锁链）。
		tersureLockItem(h, uint16(view), slot, true)
	case 15:
		tersureLockItem(h, uint16(view), slot, false)
	case 16, 17:
		// 娃娃材料锁（stuff_lock/unlock，16 带 prop_count）——材料位语义
		// 待真机（form_newtreasure_grave），一期静默受理。
		log.Printf("%s: tersure stuff lock sub=%d view=%d index=%d (accepted, phase-2)",
			h.remote, sub, view, slot)
	case 18, 19, 22:
		// 娃娃 grave（埋葬换新）/transfer（词条转移）/repair（修复）——
		// 客户端等回执（10/12/14），语义未定前不回执（防 UI 状态错乱）。
		log.Printf("%s: tersure sub=%d view=%d index=%d not implemented (phase-2)", h.remote, sub, view, slot)
	case 20:
		if len(args) < 5 {
			log.Printf("%s: tersure replace malformed args=%d", h.remote, len(args))
			return true, nil
		}
		tersureWawaReplace(h, uint16(view), slot, args[3], args[4])
	case 21:
		selectRow := int32(-1)
		if len(args) >= 4 {
			selectRow = args[3]
		}
		tersureWawaIntensify(h, uint16(view), slot, selectRow)
	default:
		log.Printf("%s: tersure 688 sub=%d not implemented (view=%d index=%d)", h.remote, sub, view, slot)
	}
	return true, nil
}

// tersureTersureSessionKey 会话归属校验：view+slot 与会话一致。
func tersureSessionMatches(s *tersureWashSession, view uint16, slot int32) bool {
	return s.active && s.viewID == view && s.slot == slot
}

// bagItemAtLocked bagItemAt 的持锁变体（调用方已持有 p.mu；同匹配口径：
// bagViewOf(row.viewID) == view）。
func bagItemAtLocked(p *playerActor, view uint16, slot int32) (bagRow, int, bool) {
	for index := range p.bag {
		if p.bag[index].slot == slot && bagViewOf(p.bag[index].viewID) == view {
			return p.bag[index], index, true
		}
	}
	return bagRow{}, -1, false
}

// tersureLockItem 整件锁/解锁：绑定会话 + LockStatus 属性帧（0x10 isView=1，
// 经脉/内功行帧同款方言）。
func tersureLockItem(h *itemSession, view uint16, slot int32, lock bool) {
	h.player.mu.Lock()
	session := &h.player.tersure
	if lock {
		row, _, ok := bagItemAtLocked(h.player, view, slot)
		if !ok {
			h.player.mu.Unlock()
			log.Printf("%s: tersure lock view=%d slot=%d empty", h.remote, view, slot)
			return
		}
		kind := randomEquipCatalogOf().randomEquipKind(row.configID)
		if kind == randomKindNone {
			h.player.mu.Unlock()
			log.Printf("%s: tersure lock %s not a random equip", h.remote, row.configID)
			return
		}
		// 换物=旧会话作废（客户端 clearform 已发解锁，这里兜底）。
		*session = tersureWashSession{
			active: true, viewID: view, slot: slot,
			uid: row.uid, configID: row.configID, kind: kind,
			lockedRows: map[int]bool{},
		}
	} else {
		if !tersureSessionMatches(session, view, slot) {
			h.player.mu.Unlock()
			return
		}
		session.active = false
		session.pendingProp = nil
		session.pendingSkill = nil
		session.lockedRows = map[int]bool{}
	}
	h.player.mu.Unlock()

	if !lock {
		// 解锁同时清客户端残留预览行（tempRows 记账在 tersureClearTempRows
		// 内部清零，此处补发清行帧）。
		tersureClearTempRows(h, view, slot)
	}
	value := int32(0)
	if lock {
		value = 1
	}
	frame, err := sceneObjectProperties(uint32(view), uint32(slot), 1, []clientdata.IndexedProperty{
		{Index: propLockStatus, Name: "LockStatus", Value: clientdata.Int32Value(value)},
	})
	if err == nil {
		err = h.link.WriteFrame(frame)
	}
	if err != nil {
		log.Printf("%s: tersure lock frame view=%d slot=%d: %v", h.remote, view, slot, err)
	}
	log.Printf("%s: tersure lock=%t view=%d slot=%d", h.remote, lock, view, slot)
}

// tersureLockPropRow 老界面单条属性锁/解锁：会话行锁 + RandomPropRec
// col2 重写（整记录清+加，避免 0x13 逐格视图寻址无实证）+ 结果 sub 4/5。
func tersureLockPropRow(h *itemSession, view uint16, slot int32, row int32, lock bool) {
	h.player.mu.Lock()
	session := &h.player.tersure
	if !tersureSessionMatches(session, view, slot) {
		h.player.mu.Unlock()
		log.Printf("%s: tersure lock_prop without session view=%d slot=%d", h.remote, view, slot)
		return
	}
	session.lockedRows[int(row)] = lock
	h.player.mu.Unlock()

	if lock {
		tersureNotifyResult(h, 4)
	} else {
		tersureNotifyResult(h, 5)
	}
	log.Printf("%s: tersure lock_prop row=%d lock=%t", h.remote, row, lock)
}

// tersureWashProps 老界面属性洗：锁定行保留 + 其余同池重掷（总数不变），
// TempTersureRec 预览（锁定行前置）+ 结果 sub 2。计费先行（不足即拒）。
func tersureWashProps(h *itemSession, view uint16, slot int32) {
	session, snapshot, ok := tersureSessionSnapshot(h, view, slot)
	if !ok {
		return
	}
	keep := make([]randomPropLine, 0, len(snapshot.PropRows))
	rolledCount := 0
	for index, line := range snapshot.PropRows {
		if session.lockedRows[index] {
			keep = append(keep, line)
		} else {
			rolledCount++
		}
	}
	if !h.billTersureWash(0, tersurePropWashCostSilver, "prop wash", session.configID) {
		return
	}
	rolled, err := randomEquipCatalogOf().rollTreasurePropsForWash(session.configID, rolledCount, itemUseRandom())
	if err != nil {
		log.Printf("%s: tersure prop wash %s: %v", h.remote, session.configID, err)
		return
	}
	pending := append(keep, rolled...)

	h.player.mu.Lock()
	h.player.tersure.pendingProp = pending
	h.player.mu.Unlock()

	rows := make([][]recordCell, 0, len(pending))
	for index, line := range pending {
		rows = append(rows, tersureTempPropRow(line, index < len(keep)))
	}
	if !tersurePushTempRows(h, view, slot, rows) {
		return
	}
	tersureNotifyResult(h, 2)
	log.Printf("%s: tersure prop wash %s kept=%d rolled=%d", h.remote, session.configID, len(keep), len(rolled))
}

// tersureSaveProps 老界面属性存：pending 全量写回快照 + RandomPropRec 重写
// + 清预览 + 结果 sub 3（客户端随后隐式重锁）。
func tersureSaveProps(h *itemSession, view uint16, slot int32) {
	session, snapshot, ok := tersureSessionSnapshot(h, view, slot)
	if !ok {
		return
	}
	h.player.mu.Lock()
	pending := h.player.tersure.pendingProp
	h.player.mu.Unlock()
	if pending == nil {
		log.Printf("%s: tersure prop save without pending wash", h.remote)
		return
	}
	snapshot.PropRows = pending
	if !tersureCommitSnapshot(h, session, snapshot) {
		return
	}
	tersureUpdateSessionSnapshot(h, view, slot, snapshot)
	tersureRewriteRandomProps(h, view, slot, snapshot)
	tersureClearTempRows(h, view, slot)
	h.player.mu.Lock()
	h.player.tersure.pendingProp = nil
	h.player.tersure.lockedRows = map[int]bool{}
	h.player.mu.Unlock()
	tersureNotifyResult(h, 3)
	log.Printf("%s: tersure prop save %s rows=%d", h.remote, session.configID, len(pending))
}

// tersureWashSkill 武学洗（老 sub 10 / 新 sub 12）：技能池按门派/江湖过滤
// 重掷主武学条，TempTersureRec 预览（单技能行）+ 结果 sub 6（老）/8（新）。
func tersureWashSkill(h *itemSession, view uint16, slot int32, skillType int, newUI bool) {
	session, _, ok := tersureSessionSnapshot(h, view, slot)
	if !ok {
		return
	}
	line, err := randomEquipCatalogOf().rollTreasureSkillForWash(session.configID, skillType, itemUseRandom())
	if err != nil {
		log.Printf("%s: tersure skill wash %s type=%d: %v", h.remote, session.configID, skillType, err)
		return
	}
	huantai, wen, billed := tersureSkillWashCost(randomEquipCatalogOf().skillStoreTaoLu(line.StoreID))
	if !h.billTersureWash(huantai, wen, "skill wash "+billed, session.configID) {
		return
	}
	h.player.mu.Lock()
	lineCopy := line
	h.player.tersure.pendingSkill = &lineCopy
	h.player.mu.Unlock()

	if !tersurePushTempRows(h, view, slot, [][]recordCell{tersureTempSkillRow(line)}) {
		return
	}
	if newUI {
		tersureNotifyResult(h, 8)
	} else {
		tersureNotifyResult(h, 6)
	}
	log.Printf("%s: tersure skill wash %s type=%d store=%d value=%.0f billed(huantai=%d wen=%d)",
		h.remote, session.configID, skillType, line.StoreID, line.Value, huantai, wen)
}

// tersureSaveSkill 武学存（老 sub 11 / 新 sub 13）：主武学条（Unlock==0）
// 替换 + RandomSkillRec 重写 + 清预览 + 结果 sub 7（老）/9（新）。
func tersureSaveSkill(h *itemSession, view uint16, slot int32, newUI bool) {
	session, snapshot, ok := tersureSessionSnapshot(h, view, slot)
	if !ok {
		return
	}
	h.player.mu.Lock()
	pending := h.player.tersure.pendingSkill
	h.player.mu.Unlock()
	if pending == nil {
		log.Printf("%s: tersure skill save without pending wash", h.remote)
		return
	}
	replaced := false
	for index := range snapshot.SkillRows {
		if snapshot.SkillRows[index].Unlock == 0 {
			snapshot.SkillRows[index] = *pending
			replaced = true
			break
		}
	}
	if !replaced {
		snapshot.SkillRows = append(snapshot.SkillRows, *pending)
	}
	if !tersureCommitSnapshot(h, session, snapshot) {
		return
	}
	tersureUpdateSessionSnapshot(h, view, slot, snapshot)
	tersureRewriteRandomSkills(h, view, slot, snapshot)
	tersureClearTempRows(h, view, slot)
	h.player.mu.Lock()
	h.player.tersure.pendingSkill = nil
	h.player.mu.Unlock()
	if newUI {
		tersureNotifyResult(h, 9)
	} else {
		tersureNotifyResult(h, 7)
	}
	log.Printf("%s: tersure skill save %s store=%d", h.remote, session.configID, pending.StoreID)
}

// ---- 会话/快照/帧 helpers ----

// tersureSessionSnapshot 会话校验 + 会话权威快照（首用时 ensure 并缓存，
// 存后由调用方更新——锁定期间无并发修改方，避免每次操作重查/重掷）。
func tersureSessionSnapshot(h *itemSession, view uint16, slot int32) (*tersureWashSessionFields, itemRandomProps, bool) {
	h.player.mu.Lock()
	session := h.player.tersure
	matched := tersureSessionMatches(&session, view, slot)
	cached := session.snapshot
	h.player.mu.Unlock()
	if !matched {
		log.Printf("%s: tersure wash without session view=%d slot=%d", h.remote, view, slot)
		return nil, itemRandomProps{}, false
	}
	if cached != nil {
		return tersureSessionFieldsOf(&session), *cached, true
	}
	snapshot, err := h.ensureRandomProps(session.configID, session.uid, session.kind)
	if err != nil {
		log.Printf("%s: tersure snapshot %s: %v", h.remote, session.configID, err)
		return nil, itemRandomProps{}, false
	}
	h.player.mu.Lock()
	if tersureSessionMatches(&h.player.tersure, view, slot) && h.player.tersure.snapshot == nil {
		h.player.tersure.snapshot = &snapshot
	}
	h.player.mu.Unlock()
	return tersureSessionFieldsOf(&session), snapshot, true
}

// tersureSessionFieldsOf 会话只读字段（锁外传递；调用方已做过归属校验）。
func tersureSessionFieldsOf(session *tersureWashSession) *tersureWashSessionFields {
	return &tersureWashSessionFields{
		uid: session.uid, configID: session.configID, kind: session.kind,
		lockedRows: session.lockedRows,
	}
}

// tersureUpdateSessionSnapshot 存后更新会话缓存快照。
func tersureUpdateSessionSnapshot(h *itemSession, view uint16, slot int32, snapshot itemRandomProps) {
	h.player.mu.Lock()
	if tersureSessionMatches(&h.player.tersure, view, slot) {
		stored := snapshot
		h.player.tersure.snapshot = &stored
	}
	h.player.mu.Unlock()
}

// tersureWashSessionFields 会话只读快照（锁外传递）。
type tersureWashSessionFields struct {
	uid        string
	configID   string
	kind       int
	lockedRows map[int]bool
}

// tersureCommitSnapshot 快照落库（randomProps 存档在时）。
func tersureCommitSnapshot(h *itemSession, session *tersureWashSessionFields, snapshot itemRandomProps) bool {
	if h.randomProps == nil || session.uid == "" {
		return true
	}
	if err := h.randomProps.Save(h.ctx, h.roleID, session.uid, session.configID, snapshot); err != nil {
		log.Printf("%s: tersure save snapshot %s: %v", h.remote, session.configID, err)
		return false
	}
	return true
}

// tersureNotifyResult 结果通知（线名=lua on_custom_show_tersuer_result）。
func tersureNotifyResult(h *itemSession, sub int32) {
	frame, err := serverCustomStringMessage("on_custom_show_tersuer_result", customInt(sub))
	if err != nil {
		log.Printf("%s: tersure notify encode sub=%d: %v", h.remote, sub, err)
		return
	}
	if err := h.link.WriteFrame(frame); err != nil {
		log.Printf("%s: tersure notify write sub=%d: %v", h.remote, sub, err)
	}
}

// tersurePushTempRows 清旧 TempTersureRec 行 + 追加新行（记账 session.tempRows）。
func tersurePushTempRows(h *itemSession, view uint16, slot int32, rows [][]recordCell) bool {
	tersureClearTempRows(h, view, slot)
	if len(rows) == 0 {
		return true
	}
	frame, err := serverRecordAddRowsView(view, uint16(slot), recordTempTersure, rows)
	if err != nil {
		log.Printf("%s: tersure temp rows encode: %v", h.remote, err)
		return false
	}
	if err := h.link.WriteFrame(frame); err != nil {
		log.Printf("%s: tersure temp rows write: %v", h.remote, err)
		return false
	}
	h.player.mu.Lock()
	h.player.tersure.tempRows = len(rows)
	h.player.mu.Unlock()
	return true
}

// tersureClearTempRows 按记账清空客户端 TempTersureRec 行（0x12 视图寻址）。
func tersureClearTempRows(h *itemSession, view uint16, slot int32) {
	h.player.mu.Lock()
	count := h.player.tersure.tempRows
	h.player.tersure.tempRows = 0
	h.player.mu.Unlock()
	for row := count - 1; row >= 0; row-- {
		if err := h.link.WriteFrame(serverRecordDelRowView(view, uint16(slot), recordTempTersure, uint16(row))); err != nil {
			log.Printf("%s: tersure temp clear row=%d: %v", h.remote, row, err)
			return
		}
	}
}

// tersureRewriteRandomProps RandomPropRec 整记录替换（0x12 清 + 0x11 加）。
func tersureRewriteRandomProps(h *itemSession, view uint16, slot int32, snapshot itemRandomProps) {
	tersureRewriteRecord(h, view, slot, recordRandomProp, len(snapshot.PropRows), randomPropRecRows(snapshot.PropRows))
}

// tersureRewriteRandomSkills RandomSkillRec 整记录替换。
func tersureRewriteRandomSkills(h *itemSession, view uint16, slot int32, snapshot itemRandomProps) {
	count := len(snapshot.SkillRows)
	tersureRewriteRecord(h, view, slot, recordRandomSkill, count, randomSkillRecRows(snapshot.SkillRows))
}

// tersureRewriteRecord 通用整记录替换：清 rowCount 行后追加 rows。
func tersureRewriteRecord(h *itemSession, view uint16, slot int32, record uint16, rowCount int, rows [][]recordCell) {
	for row := rowCount - 1; row >= 0; row-- {
		if err := h.link.WriteFrame(serverRecordDelRowView(view, uint16(slot), record, uint16(row))); err != nil {
			log.Printf("%s: tersure rewrite clear rec=%d row=%d: %v", h.remote, record, row, err)
			return
		}
	}
	if len(rows) == 0 {
		return
	}
	frame, err := serverRecordAddRowsView(view, uint16(slot), record, rows)
	if err != nil {
		log.Printf("%s: tersure rewrite add rec=%d: %v", h.remote, record, err)
		return
	}
	if err := h.link.WriteFrame(frame); err != nil {
		log.Printf("%s: tersure rewrite write rec=%d: %v", h.remote, record, err)
	}
}

// serverRecordDelRowView 0x12 ServerRecordDelRow 的视图对象寻址形态
// （flag=1，寻址布局镜像 serverRecordAddRowsView；场景对象版见
// shortcut_records.go serverRecordDelRow——0x12 末 u16=起始行号，逐行删）。
func serverRecordDelRowView(viewID, objectIndex, recordIndex, row uint16) []byte {
	msg := make([]byte, 16)
	msg[0] = 0x12
	msg[1] = 1 // 视图对象（view+objectIndex 寻址）
	binary.LittleEndian.PutUint32(msg[2:], uint32(viewID))
	binary.LittleEndian.PutUint32(msg[6:], uint32(objectIndex))
	binary.LittleEndian.PutUint16(msg[10:], recordIndex)
	binary.LittleEndian.PutUint16(msg[12:], row)
	binary.LittleEndian.PutUint16(msg[14:], 1) // 删 1 行
	return msg
}

// tersureTempPropRow TempTersureRec 属性行：[1,propID,value,lock,flag,
// quality,level,0]（col7=skill_flag，<1 才展示）。
func tersureTempPropRow(line randomPropLine, locked bool) []recordCell {
	lock := int32(0)
	if locked {
		lock = 1
	}
	return []recordCell{
		recordInt(1),
		recordInt(uint32(line.PropID)),
		recordFloat64(line.Value),
		recordInt(uint32(lock)),
		recordInt(uint32(line.Flag)),
		recordInt(uint32(line.Quality)),
		recordInt(uint32(line.Level)),
		recordInt(0),
	}
}

// tersureTempSkillRow TempTersureRec 技能行：[0,storeID,value,SubSkill下标,
// 0,0,0,0]（col0 type 0=技能；界面 GetSkillText(col1=store, col3=sub, col2=value)）。
func tersureTempSkillRow(line randomSkillLine) []recordCell {
	return []recordCell{
		recordInt(0),
		recordInt(uint32(line.StoreID)),
		recordFloat64(line.Value),
		recordInt(uint32(line.Sub)),
		recordInt(0),
		recordInt(0),
		recordInt(0),
		recordInt(0),
	}
}

// tersureWashOpenFrame 洗炼界面开窗帧（GM 动作直发；NPC 对话链同方言）。
// 数字寻址 517（真机 GM 探针实证：517=on_custom_show_tersure，sub_type
// 899000000/1/8 开精炼/重制/洗炼窗——console.html 探针区注记）；subType
// 传整数，客户端 nx_string() 转字符串后比对功能号。
func tersureWashOpenFrame(form string) ([]byte, error) {
	if !strings.EqualFold(form, "new") && !strings.EqualFold(form, "old") {
		return nil, fmt.Errorf("form must be new or old")
	}
	id := int32(89900021)
	if strings.EqualFold(form, "old") {
		id = int32(89900008)
	}
	return serverModernCustomIntMessage(517, customInt(id))
}

// ---- 娃娃淬炼族（sub 14-22，custom_sender.lua:4814-4878 方言） ----

// tersureWawaIntensify sub 21 娃娃强化（custom_newtersure_intensify(view,
// index, select)）：select 行（-1=全部未锁行）同池重掷 + TempTersureRec
// 预览 + 结果 sub 13（客户端 form_newtreasure_grave show_intensify_result）。
// 升级/品质语义待真机（洗炼近似：同池重掷选中行）。
func tersureWawaIntensify(h *itemSession, view uint16, slot int32, selectRow int32) {
	session, snapshot, ok := tersureSessionSnapshot(h, view, slot)
	if !ok {
		return
	}
	if session.kind != randomKindWawa {
		log.Printf("%s: tersure intensify kind=%d not wawa view=%d slot=%d", h.remote, session.kind, view, slot)
		return
	}
	targets := wawaIntensifyRows(session.lockedRows, snapshot.PropRows, selectRow)
	if len(targets) == 0 {
		log.Printf("%s: tersure intensify no target rows (select=%d locked-all?)", h.remote, selectRow)
		return
	}
	if !h.billTersureWash(0, tersurePropWashCostSilver, "wawa intensify", session.configID) {
		return
	}
	rolled, err := randomEquipCatalogOf().rollTreasurePropsForWash(session.configID, len(targets), itemUseRandom())
	if err != nil {
		log.Printf("%s: tersure intensify %s: %v", h.remote, session.configID, err)
		return
	}
	pending := append([]randomPropLine(nil), snapshot.PropRows...)
	for index, line := range rolled {
		pending[targets[index]] = line
	}
	h.player.mu.Lock()
	h.player.tersure.pendingProp = pending
	h.player.mu.Unlock()

	rows := make([][]recordCell, 0, len(pending))
	for index, line := range pending {
		_, locked := session.lockedRows[index]
		rows = append(rows, tersureTempPropRow(line, locked))
	}
	if !tersurePushTempRows(h, view, slot, rows) {
		return
	}
	tersureNotifyResult(h, 13)
	log.Printf("%s: tersure wawa intensify %s rows=%v", h.remote, session.configID, targets)
}

// tersureWawaReplace sub 20 娃娃词条替换（custom_newtersure_replace(view,
// index, select_1, select_2)）：两行同池重掷 + 预览 + 结果 sub 11（客户端
// show_newtersure_replace_result）。双行语义待真机（替换近似：重掷两行）。
func tersureWawaReplace(h *itemSession, view uint16, slot int32, select1, select2 int32) {
	session, snapshot, ok := tersureSessionSnapshot(h, view, slot)
	if !ok {
		return
	}
	if session.kind != randomKindWawa {
		log.Printf("%s: tersure replace kind=%d not wawa view=%d slot=%d", h.remote, session.kind, view, slot)
		return
	}
	targets := make([]int, 0, 2)
	for _, selectRow := range []int32{select1, select2} {
		index := int(selectRow)
		if index >= 0 && index < len(snapshot.PropRows) {
			targets = append(targets, index)
		}
	}
	if len(targets) == 0 {
		log.Printf("%s: tersure replace no valid rows (%d,%d)", h.remote, select1, select2)
		return
	}
	if !h.billTersureWash(0, tersurePropWashCostSilver, "wawa replace", session.configID) {
		return
	}
	rolled, err := randomEquipCatalogOf().rollTreasurePropsForWash(session.configID, len(targets), itemUseRandom())
	if err != nil {
		log.Printf("%s: tersure replace %s: %v", h.remote, session.configID, err)
		return
	}
	pending := append([]randomPropLine(nil), snapshot.PropRows...)
	for index, line := range rolled {
		pending[targets[index]] = line
	}
	h.player.mu.Lock()
	h.player.tersure.pendingProp = pending
	h.player.mu.Unlock()

	rows := make([][]recordCell, 0, len(pending))
	for index, line := range pending {
		rows = append(rows, tersureTempPropRow(line, session.lockedRows[index]))
	}
	if !tersurePushTempRows(h, view, slot, rows) {
		return
	}
	tersureNotifyResult(h, 11)
	log.Printf("%s: tersure wawa replace %s rows=%v", h.remote, session.configID, targets)
}

// wawaIntensifyRows 强化目标行集：select=-1 全部未锁行，否则单行
//（越界/锁定空集）。
func wawaIntensifyRows(locked map[int]bool, rows []randomPropLine, selectRow int32) []int {
	if selectRow >= 0 {
		index := int(selectRow)
		if index >= len(rows) || locked[index] {
			return nil
		}
		return []int{index}
	}
	targets := make([]int, 0, len(rows))
	for index := range rows {
		if !locked[index] {
			targets = append(targets, index)
		}
	}
	return targets
}
