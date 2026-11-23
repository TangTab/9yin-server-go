package fssoloworld

// skillrush.go 冲锋技（静态包 CanRush=1，如 CS_jh_cqgf01 长拳 01）的
// 暂存 → 授予 → 落地链（官服女号抓包 206.4~207.8s 逐帧定版）。
//
//	暂存   按压点超出直收带（castRange+0.5）：CurSkillID → CurSkillTarget →
//	      m25[PauseTime,"skill_pause"] → PauseSkill → m707[15]；冷却/MP 此刻不结。
//	授予   客户端接近上报（m15 行走 / m220）进入冲锋带（rush_range.ini
//	      GRushRange+Add）：
//	      m24[自己,61,落点xyz,档1,动作] → CantMove=1 → CantParry=1 → 冷却行 →
//	      MPRatio/MP → [QingGongPoint]
//	      落点 = 目标身前 bind 距离（Lock.BindDistance，缺省 1.5m）、
//	      高度 = 目标 y + 1.0；动作 = Lock.LockGRushAction。
//	落地   客户端 m212 到位申报（锚 = 自报位）：
//	      CantMove/CantParry=0 → 0x1F → CurBreathState=0 → PauseSkill="" → m26 →
//	      0x1F → m337 → LastWGAttackNpc/LastWGHit → m24 目标 40 → m19 →
//	      CantMove/CantParry=1 → 0x1F
//	      随后按出招锁收招（锁定包 LockSrcTime，抓包落地后 1465ms 清位）。
//
// 授予后 rushLandingWatch 内未收到 m212（客户端未冲刺/未到位）：在玩家
// 当前上报位就地释放，不写 bind 落点（防瞬移观感）。

import (
	"math"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/skilldata"
)

const (
	// rushBindDistance c61 落点距目标的缺省贴身距离（Lock.BindDistance 缺席时）。
	rushBindDistance = 1.5
	// rushDestLift c61 落点相对目标高度的抬升（抓包 6.704 = 5.704 + 1.0）。
	rushDestLift = 1.0
	// defaultRushRange 进带阈值缺省（rush_range.ini 缺席时）：GRushRange + GRushRangeAdd。
	defaultRushRange = 6.5
	// defaultRushAction c61 地面冲刺动作缺省。
	defaultRushAction = "qg_fastrun_0h_01"
	// rushDashFlagGround c61 档位：1 = 地面。
	rushDashFlagGround = 1
	// rushLandingWatch 授予后等待 m212 的兜底时长。
	rushLandingWatch = 1500 * time.Millisecond
)

// stagedRush 冲锋暂存态（会话锁护）。
type stagedRush struct {
	def        skilldata.Definition
	seq        int
	target     uint64
	targetPos  [3]float64
	targetName string
	granted    bool
	expiry     *time.Timer
	landing    *time.Timer
}

// clearStagedLocked 暂存收尾（换技/取消/超时共用，调用方持会话锁）：停表，
// 已授予则先还原授予对，再三清 CurSkillID="" → m26 → PauseSkill=""。
func (s *Server) clearStagedLocked(st *combatState) [][]byte {
	stage := st.staged
	if stage == nil {
		return nil
	}
	if stage.expiry != nil {
		stage.expiry.Stop()
	}
	if stage.landing != nil {
		stage.landing.Stop()
	}
	st.staged, st.curSkill, st.locked = nil, "", false
	b := &frameBuilder{}
	if stage.granted {
		b.prop("CantMove", entergen.ByteValue(0))
		b.prop("CantParry", entergen.Int32Value(0))
	}
	b.prop("CurSkillID", entergen.StringValue(""))
	b.custom(26)
	b.prop("PauseSkill", entergen.StringValue(""))
	return b.frames
}

// expireStaged 暂存窗到期（PauseTime，抓包 30000）仍未进带授予：三清收尾。
func (s *Server) expireStaged(pl *Player, seq int) {
	st := &pl.combat
	st.mu.Lock()
	stage := st.staged
	if stage == nil || stage.seq != seq || stage.granted {
		st.mu.Unlock()
		return
	}
	skillID := stage.def.ID
	frames := s.clearStagedLocked(st)
	st.mu.Unlock()
	s.sendFrames(pl, frames)
	s.trace("skill %s staged rush expired seq=%d client=%d", skillID, seq, pl.Conn1)
}

// rushRange 进带阈值（地面）：rush_range.ini 的 GRushRange + GRushRangeAdd。
func (s *Server) rushRange() float64 {
	if catalog := s.skillCatalog(); catalog != nil {
		if r := catalog.RushRange(); r.Ground > 0 {
			return r.Ground + r.GroundAdd
		}
	}
	return defaultRushRange
}

// rushOnReport 位置上报驱动的进带判定（m15 行走 / m220 / 位移包共用）：
// 平面距离进带即授予 c61。无暂存或已授予时直接返回（热路径零开销）。
func (s *Server) rushOnReport(pl *Player, x, z float64) {
	st := &pl.combat
	st.mu.Lock()
	if st.staged == nil || st.staged.granted {
		st.mu.Unlock()
		return
	}
	stage := st.staged
	distance := math.Hypot(stage.targetPos[0]-x, stage.targetPos[2]-z)
	if distance > s.rushRange() {
		st.mu.Unlock()
		return
	}
	nowMS := filetimeMS(combatNow())
	st.ensureVitals(pl)
	def := stage.def
	if def.Consume.MP > st.mp {
		b := &frameBuilder{}
		b.frames = append(b.frames, s.clearStagedLocked(st)...)
		b.custom(707, entergen.CustomInt(useSkillResultBase+useSkillResultMPLess))
		st.mu.Unlock()
		s.sendFrames(pl, b.frames)
		return
	}
	rowFrames, rows, err := writeCooldownRows(forkPlayerObjectID, forkPlayerOwnerID, st.cooldowns, def, nowMS)
	if err != nil {
		st.mu.Unlock()
		s.trace("skill %s rush grant cooldown rows encode failed client=%d: %v", def.ID, pl.Conn1, err)
		return
	}
	// 落点：目标身前 bind 距离，方向取自当前上报位（零向量时沿上报朝向的反向）。
	bind := def.Lock.BindDistance
	if bind <= 0 {
		bind = rushBindDistance
	}
	dx, dz := x-stage.targetPos[0], z-stage.targetPos[2]
	if norm := math.Hypot(dx, dz); norm > 1e-6 {
		dx, dz = dx/norm*bind, dz/norm*bind
	} else {
		dx, dz = 0, bind
	}
	action := def.Lock.GroundRushAction
	if action == "" {
		action = defaultRushAction
	}
	f32 := entergen.CustomFloat32
	b := &frameBuilder{}
	b.custom(24, selfValue(), entergen.CustomInt(61),
		f32(float32(stage.targetPos[0]+dx)), f32(float32(stage.targetPos[1]+rushDestLift)),
		f32(float32(stage.targetPos[2]+dz)), entergen.CustomInt(rushDashFlagGround), entergen.CustomString(action))
	b.prop("CantMove", entergen.ByteValue(1))
	b.prop("CantParry", entergen.Int32Value(1))
	b.frames = append(b.frames, rowFrames...)
	st.cooldowns = rows
	if def.Consume.MP > 0 {
		b.prop("MPRatio", entergen.Int32Value(mpRatio(st.mp-def.Consume.MP, playerMaxMP(pl))))
		b.prop("MP", entergen.Int32Value(int32(st.mp-def.Consume.MP)))
	}
	st.mp -= def.Consume.MP
	if def.Consume.QingGong > 0 && st.qinggong > 0 {
		st.qinggong = max(st.qinggong-def.Consume.QingGong, 0)
		b.prop("QingGongPoint", entergen.Int32Value(int32(st.qinggong)))
	}
	st.lastTaoLu = def.TaoLu
	stage.granted, st.locked = true, true
	seq := stage.seq
	stage.landing = time.AfterFunc(rushLandingWatch, func() { s.rushLandingFallback(pl, seq) })
	st.mu.Unlock()
	s.sendFrames(pl, b.frames)
	s.trace("skill %s rush granted dist=%.2f dest=(%.2f,%.2f,%.2f) client=%d",
		def.ID, distance, stage.targetPos[0]+dx, stage.targetPos[1]+rushDestLift, stage.targetPos[2]+dz, pl.Conn1)
}

// rushLandingFrames 落地释放簇（锚 = 落地位）。调用方持会话锁。
func rushLandingFrames(def skilldata.Definition, stage *stagedRush, pos [4]float32) *frameBuilder {
	orient := towardOrient(float64(pos[0]), float64(pos[2]), stage.targetPos[0], stage.targetPos[2])
	facing := normalizeOrient(orient)
	f32 := entergen.CustomFloat32
	b := &frameBuilder{}
	b.prop("CantMove", entergen.ByteValue(0))
	b.prop("CantParry", entergen.Int32Value(0))
	b.location(pos[0], pos[1], pos[2], facing)
	b.prop("CurBreathState", entergen.Int32Value(0))
	b.prop("PauseSkill", entergen.StringValue(""))
	b.custom(26)
	b.location(pos[0], pos[1], pos[2], facing)
	b.custom(337, selfValue(), f32(pos[0]), f32(pos[1]), f32(pos[2]), f32(facing), entergen.CustomInt(0))
	if stage.targetName != "" {
		b.prop("LastWGAttackNpc", entergen.StringValue(stage.targetName))
	}
	b.prop("LastWGHit", entergen.Float32Value(100))
	b.custom(24, targetValue(stage.target), entergen.CustomInt(40), entergen.CustomInt(1), selfValue())
	cast := skillCast{skill: def.ID, x: pos[0], y: pos[1], z: pos[2],
		targetID: uint32(stage.target), targetOwner: uint32(stage.target >> 32), hasTarget: true}
	b.custom(19, skillLockMessageValues(def, cast, stage.targetPos, orient)...)
	b.prop("CantMove", entergen.ByteValue(1))
	b.prop("CantParry", entergen.Int32Value(1))
	b.location(pos[0], pos[1], pos[2], facing)
	return b
}

// settleRushLanding 落地收尾（调用方持会话锁）：消费暂存态，CurSkillID 保持
// 到出招锁到期由 finishCast 清。
func settleRushLanding(st *combatState, stage *stagedRush) (seq int) {
	if stage.landing != nil {
		stage.landing.Stop()
	}
	if stage.expiry != nil {
		stage.expiry.Stop()
	}
	st.staged, st.curSkill, st.locked, st.castSeq = nil, stage.def.ID, true, stage.seq
	return stage.seq
}

// handleDashLanding C2S m212 冲锋到位申报 [212, x, y, z, 朝向, 尾]：位置收编，
// 有已授予的暂存则发落地簇；无暂存的 212 官服零回应（仅位置收编）。
func (s *Server) handleDashLanding(pl *Player, message entergen.CustomInbound) {
	values := message.Values
	if len(values) < 5 {
		return
	}
	var pos [4]float32
	for i := range pos {
		value, ok := finiteFloat(values[1+i])
		if !ok {
			return
		}
		pos[i] = value
	}
	s.absorbPlayerPosition(pl, [4]float64{float64(pos[0]), float64(pos[1]), float64(pos[2]), float64(pos[3])})
	st := &pl.combat
	st.mu.Lock()
	stage := st.staged
	if stage == nil || !stage.granted {
		st.mu.Unlock()
		return
	}
	b := rushLandingFrames(stage.def, stage, pos)
	if b.err != nil {
		st.mu.Unlock()
		s.trace("skill %s rush landing encode failed client=%d: %v", stage.def.ID, pl.Conn1, b.err)
		return
	}
	def, seq := stage.def, settleRushLanding(st, stage)
	st.mu.Unlock()
	s.sendFrames(pl, b.frames)
	time.AfterFunc(castLockMS(def), func() { s.finishCast(pl, seq, true) })
	s.trace("skill %s rush landed seq=%d client=%d", def.ID, seq, pl.Conn1)
}

// rushLandingFallback 授予后 rushLandingWatch 内未收到 m212：在玩家当前
// 上报位就地释放（不写 bind 落点）。
func (s *Server) rushLandingFallback(pl *Player, seq int) {
	st := &pl.combat
	st.mu.Lock()
	stage := st.staged
	if stage == nil || stage.seq != seq || !stage.granted {
		st.mu.Unlock()
		return
	}
	reported := pl.roleState.Position
	pos := [4]float32{float32(reported[0]), float32(reported[1]), float32(reported[2]), float32(reported[3])}
	b := rushLandingFrames(stage.def, stage, pos)
	if b.err != nil {
		st.mu.Unlock()
		return
	}
	def, landed := stage.def, settleRushLanding(st, stage)
	st.mu.Unlock()
	s.sendFrames(pl, b.frames)
	time.AfterFunc(castLockMS(def), func() { s.finishCast(pl, landed, true) })
	s.trace("skill %s rush landing fallback seq=%d client=%d", def.ID, landed, pl.Conn1)
}

// handleRushApproach C2S m220 冲锋接近上报 [220, x, y, z, 朝向]：位置收编
// （absorbPlayerPosition 尾部驱动进带判定）。
func (s *Server) handleRushApproach(pl *Player, message entergen.CustomInbound) {
	values := message.Values
	if len(values) < 5 {
		return
	}
	var pos [4]float64
	for i := range pos {
		value, ok := customFloat(values[1+i])
		if !ok {
			return
		}
		pos[i] = value
	}
	s.absorbPlayerPosition(pl, pos)
}
