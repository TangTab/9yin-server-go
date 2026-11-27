package fssoloworld

// faculty.go 修炼族（C2S 1006 FACULTY_INFO）+ 内修历练层 + 潜能赎回
//（C2S 235 LIVEPOINT_INFO）。状态机移植 2433 线 player_progress.go
// facultyReady/facultyBegin/facultyExit/advanceFaculty/awardActiveFaculty/
// livepointTick（form_faculty.lua 官方公式钉死）：
//
//	1006 子令  1 选书(string)——先刷所选 View43 行级增量帧再发资源批
//	         （set_control_state 立即解析武学对象读 Level；此前只在 21
//	          之后刷行是「先点一次演武才解锁内修按钮」的根因）
//	         11/12 普通内修开始/停（State=2、FillSpeed=200/min）
//	         21/22 演武开始/停（FillSpeed=0——选卡后才结算；开始回
//	           m191(22,0)）
//	         23 选卡(card 0..2, payment 1|2|4 双 double 精确整型)——
//	           fill+=20000、升级循环、行级增量+MaxSP 专帧+资源批；升级
//	           回 m191(14,满级?)，关页 m191(23,0)、继续 m191(24,0)
//	         24 重开脉冲——先发 FacultyName 清空帧再恢复（表单绑定器
//	           错过初值变更的补偿）
//	         31/32 经脉流转上报——受理只回资源批
//	tick     facultyTick 1s：整分钟结算 fill=FillSpeed×分钟、耗修为
//	         (fill+1)/2（钳到余额），余数留 FacultyLastAdvance；升级时
//	         Total 取 wuxuelevelinfo 曲线（缺行 +200 自然曲线）、
//	         MaxPower+20；满级 fill=0 自动停修
//	历练层   livepointTick 100s 步进：+1000 毫进首空槽（5×999000 上限）
//	         → 每槽转化 full=base×(100+LiveSpeedPower)/100，普通档 70%
//	         进修为、30% 进潜能 HoldFaculty（黄金按 500/点赎回）
//	235      1 询价 → [235,1,500]；2 赎回 N 黄金×500 潜能转修为
//	         （黄金 CapitalType2 未建模，余额恒 0 → 静默拒）；3 修为兑换
//	         受理不实现
//
// 行级增量帧（0x10 isView=1 view=43）见 entergen.BookUpdateFrame。

import (
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// C2S 消息号。
const (
	clientCustomFaculty   int32 = 1006 // 修炼族 [1006, sub, …]
	clientCustomLivepoint int32 = 235  // 潜能赎回 [235, sub, …]
)

// 1006 子令。
const (
	facultySubConvertReady int32 = 1
	facultySubNormalBegin  int32 = 11
	facultySubNormalExit   int32 = 12
	facultySubActBegin     int32 = 21
	facultySubActExit      int32 = 22
	facultySubActPay       int32 = 23
	facultySubPulse        int32 = 24
)

// 修炼态常量（form_faculty.lua：控件只渲染状态 0 空闲与 2 转换中）。
const (
	facultyStateNone    int32 = 0
	facultyStateConvert int32 = 2
	facultyStyleNormal  int32 = 1
	facultyStyleAct     int32 = 2

	normalFacultyFillPerMinute int32 = 200 // 普通内修每分钟修炼度
	activeFacultyCardReward    int32 = 20_000
	facultyNaturalTotalStep    int32 = 200 // 曲线缺行时每级 +200 自然曲线

	livepointPrice       int32 = 500 // 潜能赎回价：N 黄金 → N×500 潜能转修为
	livepointStepSeconds       = 100
	livepointAccrueMillis int32 = 1_000
	grooveCount                  = 5
	grooveCap             int32 = 999_000
)

// grooveBaseSpeed 各槽基础转化速度（毫/步）。
var grooveBaseSpeed = [grooveCount]int32{10, 20, 50, 100, 200}

// handleFaculty C2S 1006 修炼族（文件头子令表）。
func (s *Server) handleFaculty(pl *Player, message entergen.CustomInbound) {
	if !s.requireRoleState(pl, clientCustomFaculty) {
		return
	}
	if len(message.Values) < 2 || message.Values[1].Type != 2 {
		s.trace("faculty: 1006 需要整型子令 client=%d", pl.Conn1)
		return
	}
	st := &pl.roleState
	switch message.Values[1].Int32 {
	case facultySubConvertReady: // 1 选书
		if len(message.Values) < 3 || message.Values[2].Type != 6 || message.Values[2].Text == "" {
			s.trace("faculty: 1006/1 需要 ConfigID string client=%d", pl.Conn1)
			return
		}
		configID := message.Values[2].Text
		slot := facultySlotOf(st, configID)
		if slot == 0 {
			// View43 切换其它武学面板期间的选择流量：非内功 ConfigID
			// 不是会话损坏，静默（2433 线同款不拆线）。
			return
		}
		st.FacultyName = configID
		st.FacultyState = facultyStateNone
		st.FacultyStyle = facultyStyleNormal
		st.FacultyFillSpeed = 0
		st.FacultyLastAdvance = time.Now()
		s.sendFacultyBookUpdate(pl, slot)
		s.sendFacultyState(pl)
	case facultySubNormalBegin: // 11 普通内修开始
		if !s.facultyBegin(pl, facultyStyleNormal) {
			return
		}
		s.sendFacultyState(pl)
	case facultySubNormalExit: // 12 停
		s.facultyExit(pl)
		s.sendFacultyState(pl)
	case facultySubActBegin: // 21 演武开始
		if !s.facultyBegin(pl, facultyStyleAct) {
			return
		}
		s.sendFacultyState(pl)
		if frame, err := entergen.FacultyMessageFrame(22, 0); err == nil {
			s.sendToClient(pl, frame)
		}
	case facultySubActExit: // 22 演武停
		s.facultyExit(pl)
		s.sendFacultyState(pl)
	case facultySubActPay: // 23 选卡结算
		s.handleFacultyActPay(pl, message)
	case facultySubPulse: // 24 重开脉冲
		if st.FacultyName != "" {
			if frame, err := playerPropertyFrame(pl,
				entergen.NamedProperty("FacultyName", entergen.StringValue(""))); err == nil {
				s.sendToClient(pl, frame)
			}
		}
		s.sendFacultyState(pl)
	case 31, 32: // 经脉流转上报：受理只回资源批
		s.sendFacultyState(pl)
	default:
		s.trace("faculty: 未支持子令 %d client=%d", message.Values[1].Int32, pl.Conn1)
	}
}

// handleFacultyActPay 1006/23 选卡：card 0..2、payment 1|2|4（现网客户端
// 以 double 发枚举值，只接受精确整型）；结算一张 20000 点卡。
func (s *Server) handleFacultyActPay(pl *Player, message entergen.CustomInbound) {
	st := &pl.roleState
	if len(message.Values) < 4 {
		return
	}
	card, cardOK := exactCustomInt(message.Values[2])
	payment, paymentOK := exactCustomInt(message.Values[3])
	if !cardOK || !paymentOK || card < 0 || card > 2 ||
		(payment != 1 && payment != 2 && payment != 4) {
		return
	}
	slot, awarded, levelUp, maxLevel := s.awardActiveFaculty(pl)
	if awarded {
		s.sendFacultyBookUpdate(pl, slot)
		// 升级可能抬运行内功档位：怒气分母用 215 同款专帧刷新。
		if frame, err := playerPropertyFrame(pl, entergen.NamedProperty("MaxSP",
			entergen.Int32Value(entergen.RageCapOf(st.CurNeiGong)))); err == nil {
			s.sendToClient(pl, frame)
		}
	}
	s.sendFacultyState(pl)
	if levelUp {
		// 战斗面板单帧族（书 Add 随等级变化）先于原生升级通知。
		for _, spec := range entergen.CombatAttributeValues(*st) {
			if frame, err := playerPropertyFrame(pl,
				entergen.NamedProperty(spec.Name, entergen.Int32Value(spec.Value))); err == nil {
				s.sendToClient(pl, frame)
			}
		}
		if frame, err := entergen.FacultyMessageFrame(14, boolToInt(maxLevel)); err == nil {
			s.sendToClient(pl, frame)
		}
	}
	if levelUp || maxLevel {
		if frame, err := entergen.FacultyMessageFrame(23, 0); err == nil {
			s.sendToClient(pl, frame)
		}
		return
	}
	if frame, err := entergen.FacultyMessageFrame(24, 0); err == nil {
		s.sendToClient(pl, frame)
	}
}

// handleLivepoint C2S 235 潜能赎回。
func (s *Server) handleLivepoint(pl *Player, message entergen.CustomInbound) {
	if !s.requireRoleState(pl, clientCustomLivepoint) {
		return
	}
	if len(message.Values) < 2 || message.Values[1].Type != 2 {
		return
	}
	st := &pl.roleState
	switch message.Values[1].Int32 {
	case livepointSubGetInfo: // 1 询价
		if frame, err := entergen.CustomMessageFrame(clientCustomLivepoint,
			entergen.CustomInt(livepointSubGetInfo), entergen.CustomInt(livepointPrice)); err == nil {
			s.sendToClient(pl, frame)
		}
	case livepointSubBuy: // 2 赎回：N 黄金 → N×price 潜能转修为
		if len(message.Values) < 3 || message.Values[2].Type != 2 {
			return
		}
		count := message.Values[2].Int32
		// 越界静默（count≤0/黄金不足/潜能不足；客户端同校验双保险）。
		if count <= 0 || st.Gold < int64(count) || st.HoldFaculty < count*livepointPrice {
			return
		}
		st.Gold -= int64(count)
		st.HoldFaculty -= count * livepointPrice
		st.Faculty += count * livepointPrice
		if frame, err := playerPropertyFrame(pl,
			entergen.NamedProperty("CapitalType0", entergen.Int64Value(st.Gold))); err == nil {
			s.sendToClient(pl, frame)
		}
		s.sendFacultyState(pl)
		s.sendRoleSave(pl)
	case 3: // 修为兑换（form_faculty_exchange）：受理不实现
	default:
		s.trace("livepoint: 未支持子令 %d client=%d", message.Values[1].Int32, pl.Conn1)
	}
}

// livepoint 子令常量。
const (
	livepointSubGetInfo int32 = 1
	livepointSubBuy     int32 = 2
)

// facultySlotOf 修炼书的 View43 行号（1 基；0=未学）。
func facultySlotOf(st *rolestate.RoleState, configID string) uint16 {
	for i := range st.NeiGong {
		if st.NeiGong[i].ConfigID == configID {
			return uint16(i + 1)
		}
	}
	return 0
}

// facultyBegin 开始修炼（普通 200/min；演武 0——选卡后才结算）。
// 未选书/书未学拒收返回 false。
func (s *Server) facultyBegin(pl *Player, style int32) bool {
	st := &pl.roleState
	if facultySlotOf(st, st.FacultyName) == 0 {
		s.trace("faculty: 无已选书拒收 begin client=%d", pl.Conn1)
		return false
	}
	if style != facultyStyleAct {
		style = facultyStyleNormal
	}
	st.FacultyState = facultyStateConvert
	st.FacultyStyle = style
	st.FacultyFillSpeed = normalFacultyFillPerMinute
	if style == facultyStyleAct {
		st.FacultyFillSpeed = 0
	}
	st.FacultyLastAdvance = time.Now()
	return true
}

// facultyExit 停止修炼（保留进度，只停时钟）。
func (s *Server) facultyExit(pl *Player) {
	st := &pl.roleState
	st.FacultyState = facultyStateNone
	st.FacultyStyle = facultyStyleNormal
	st.FacultyFillSpeed = 0
	st.FacultyLastAdvance = time.Now()
}

// awardActiveFaculty 恰好结算一张演武卡（fill+=20000、升级循环；满级
// fill=0）。返回（行号、是否发放、是否升级、是否满级）。
func (s *Server) awardActiveFaculty(pl *Player) (slot uint16, awarded, levelUp, maxLevel bool) {
	st := &pl.roleState
	if st.FacultyState != facultyStateConvert || st.FacultyStyle != facultyStyleAct {
		// UI 队列里延迟到达的过期选卡：静默（2433 线同款不拆线）。
		return 0, false, false, false
	}
	for i := range st.NeiGong {
		book := &st.NeiGong[i]
		if book.ConfigID != st.FacultyName {
			continue
		}
		if book.Level >= book.MaxLevel {
			return uint16(i + 1), false, false, true
		}
		previous := book.Level
		book.Fill += int(activeFacultyCardReward)
		s.facultyLevelUpLoop(st, book)
		if book.Level >= book.MaxLevel {
			book.Fill = 0
		}
		return uint16(i + 1), true, book.Level != previous, book.Level >= book.MaxLevel
	}
	return 0, false, false, false
}

// facultyLevelUpLoop 升级循环（Total 随级取曲线；MaxPower+20、Power 顶满）。
func (s *Server) facultyLevelUpLoop(st *rolestate.RoleState, book *rolestate.NeiGongBook) {
	catalog := s.neiGongCatalogOf()
	for book.Fill >= book.Total && book.Level < book.MaxLevel {
		book.Fill -= book.Total
		book.Level++
		if total, ok := catalog.LevelTotal(book.ConfigID, int32(book.Level)); ok {
			book.Total = int(total)
		} else {
			book.Total += int(facultyNaturalTotalStep)
		}
		book.MaxPower += 20
		book.Power = book.MaxPower
	}
	// varprop 档随等级推进：五维 Add 与 buff 档位重取（档缺保持原值）。
	if row, ok := catalog.VarPropRow(int32(book.StaticData), int32(book.Level)); ok {
		book.StrAdd, book.StaAdd, book.DexAdd = int(row.StrAdd), int(row.StaAdd), int(row.DexAdd)
		book.IngAdd, book.SpiAdd = int(row.IngAdd), int(row.SpiAdd)
		if row.BufferID != "" {
			book.BuffID, book.BuffLevel = row.BufferID, int(row.BufferLevel)
		}
	}
}

// advanceFaculty 整分钟结算（tick 每秒调；余数留 FacultyLastAdvance——
// 重连/重启不丢部分时间）。返回（行号、是否有推进）。
func (s *Server) advanceFaculty(pl *Player, now time.Time) (uint16, bool) {
	st := &pl.roleState
	if st.FacultyState != facultyStateConvert || st.FacultyStyle != facultyStyleNormal || st.FacultyFillSpeed <= 0 {
		st.FacultyLastAdvance = now
		return 0, false
	}
	if st.FacultyLastAdvance.IsZero() || now.Before(st.FacultyLastAdvance) {
		st.FacultyLastAdvance = now
		return 0, false
	}
	minutes := int32(now.Sub(st.FacultyLastAdvance) / time.Minute)
	if minutes <= 0 {
		return 0, false
	}
	st.FacultyLastAdvance = st.FacultyLastAdvance.Add(time.Duration(minutes) * time.Minute)
	for i := range st.NeiGong {
		book := &st.NeiGong[i]
		if book.ConfigID != st.FacultyName {
			continue
		}
		fill := int64(st.FacultyFillSpeed) * int64(minutes)
		cost := (fill + 1) / 2
		if cost > int64(st.Faculty) {
			cost = int64(st.Faculty)
			fill = cost * 2
		}
		if fill <= 0 {
			// 修为耗尽：自动停修（时钟复位防离线白攒）。
			st.FacultyState = facultyStateNone
			st.FacultyFillSpeed = 0
			return 0, false
		}
		st.Faculty -= int32(cost)
		book.Fill += int(fill)
		s.facultyLevelUpLoop(st, book)
		if book.Level >= book.MaxLevel {
			book.Fill = 0
			st.FacultyState = facultyStateNone
			st.FacultyFillSpeed = 0
		}
		return uint16(i + 1), true
	}
	return 0, false
}

// facultyTick 修炼结算 tick（timerCheck 每秒）：普通内修整分钟结算，
// 推进时发行级增量帧 + 资源批。
func (s *Server) facultyTick() {
	now := time.Now()
	for _, pl := range s.players.Snapshot() {
		if pl.State != PlOnline || !pl.haveRoleState {
			continue
		}
		slot, advanced := s.advanceFaculty(pl, now)
		if !advanced {
			continue
		}
		s.sendFacultyBookUpdate(pl, slot)
		s.sendFacultyState(pl)
		s.sendRoleSave(pl)
	}
}

// livepointTick 内修历练层 tick（timerCheck 每秒、100s 步进）：累积
// +1000 毫进首空槽 → 每槽转化（普通档 70% 修为 / 30% 潜能）。变化时
// 推 LiveGroove_N 单属性帧 + 资源批。
func (s *Server) livepointTick() {
	now := time.Now()
	for _, pl := range s.players.Snapshot() {
		if pl.State != PlOnline || !pl.haveRoleState {
			continue
		}
		st := &pl.roleState
		if st.LivepointLast.IsZero() || now.Before(st.LivepointLast) {
			st.LivepointLast = now
			continue
		}
		steps := int(now.Sub(st.LivepointLast) / (livepointStepSeconds * time.Second))
		if steps <= 0 {
			continue
		}
		st.LivepointLast = st.LivepointLast.Add(time.Duration(steps) * livepointStepSeconds * time.Second)
		grooveChanged := false
		for step := 0; step < steps; step++ {
			grooveChanged = accrueLivepoint(st) || grooveChanged
			grooveChanged = convertLivepoint(st) || grooveChanged
		}
		if !grooveChanged {
			continue
		}
		id := pl.entergenIdentity()
		for index := 1; index <= grooveCount; index++ {
			if frame, err := entergen.TaskLiveGrooveFrame(id, index, st.LiveGrooves[index]); err == nil {
				s.sendToClient(pl, frame)
			}
		}
		s.sendFacultyState(pl)
		s.sendRoleSave(pl)
	}
}

// accrueLivepoint 在线累积：+1000 毫进第一个未满槽（全满丢弃）。
func accrueLivepoint(st *rolestate.RoleState) bool {
	for i := 1; i <= grooveCount; i++ {
		if st.LiveGrooves[i] < grooveCap {
			st.LiveGrooves[i] = min(grooveCap, st.LiveGrooves[i]+livepointAccrueMillis)
			return true
		}
	}
	return false
}

// convertLivepoint 转化：每槽按官方公式扣毫历练产修为；普通档 70% 进
// 修为、30% 进潜能。
func convertLivepoint(st *rolestate.RoleState) bool {
	changed := false
	for i := 1; i <= grooveCount; i++ {
		if st.LiveGrooves[i] <= 0 {
			continue
		}
		full := int64(grooveBaseSpeed[i-1]) * int64(100+st.LiveSpeedPower) / 100
		actual := int32(min(int64(st.LiveGrooves[i]), full))
		st.LiveGrooves[i] -= actual
		gain := actual * 7 / 10
		st.Faculty += gain
		st.HoldFaculty += actual - gain
		changed = true
	}
	return changed
}

// sendFacultyState 修炼资源批（0x10 多属性：FacultyName/Faculty/
// FacultyState/FacultyStyle/HoldFaculty）。
func (s *Server) sendFacultyState(pl *Player) {
	st := &pl.roleState
	frame, err := playerPropertyFrame(pl,
		entergen.NamedProperty("FacultyName", entergen.StringValue(st.FacultyName)),
		entergen.NamedProperty("Faculty", entergen.Int32Value(st.Faculty)),
		entergen.NamedProperty("FacultyState", entergen.Int32Value(st.FacultyState)),
		entergen.NamedProperty("FacultyStyle", entergen.Int32Value(st.FacultyStyle)),
		entergen.NamedProperty("HoldFaculty", entergen.Int32Value(st.HoldFaculty)),
	)
	if err != nil {
		s.trace("faculty: 资源批编码失败: %v", err)
		return
	}
	s.sendToClient(pl, frame)
}

// sendFacultyBookUpdate View43 行级增量帧（档行 NeiGongLevel 随等级）。
func (s *Server) sendFacultyBookUpdate(pl *Player, slot uint16) {
	st := &pl.roleState
	if slot == 0 || int(slot) > len(st.NeiGong) {
		return
	}
	book := st.NeiGong[slot-1]
	var neiGongLevel int32
	if row, ok := s.neiGongCatalogOf().VarPropRow(int32(book.StaticData), int32(book.Level)); ok {
		neiGongLevel = row.NeiGongLevel
	}
	frame, err := entergen.BookUpdateFrame(slot, book, neiGongLevel)
	if err != nil {
		s.trace("faculty: 行级增量帧编码失败 slot=%d: %v", slot, err)
		return
	}
	s.sendToClient(pl, frame)
}

// exactCustomInt double 值的精确整型判定（现网客户端把枚举以 double
// 发出；非精确整型或非 double/int 型返回 false）。
func exactCustomInt(value entergen.CustomInboundValue) (int32, bool) {
	switch value.Type {
	case 5: // double
		if value.Float64 == float64(int32(value.Float64)) {
			return int32(value.Float64), true
		}
		return 0, false
	case 2: // int32
		return value.Int32, true
	default:
		return 0, false
	}
}

// boolToInt 布尔转 m191 整值（facultyBool 同款）。
func boolToInt(v bool) int32 {
	if v {
		return 1
	}
	return 0
}
