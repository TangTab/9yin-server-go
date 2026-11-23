package fssoloworld

// skillcooldown.go 玩家冷却表（记录 109 cooldown_rec）的行表镜像与官服写法。
//
// 行 = [类别, 起, 止, 队]，起止为 1601(FILETIME) 纪元毫秒。女号官服抓包
// 逐次复原（tools/pcapskill/cdsim.py 冷却表模拟器）得到三类行：
//
//	个人行  类别 = 静态包 CoolDownCategory，时长 = 逐级 CoolDownTime，
//	        每次受理追加新行；
//	窗口行  类别 1，时长恒 1000ms——首放插入，之后每次只改写起止两格；
//	公共行  时长 = skill_new AddCoolDownTime，按 CoolEffectType 分两形：
//	        1（5414 行）套路间共享：类别 50000/队 = CoolDownTeam，仅换套路
//	          时拦截（同套路连放只按招式自身 CD），每次受理都写；
//	        2（249 行，普攻族）组内共享：类别 = CoolDownTeam/队 = 0xFFFF，
//	          总是拦截，每次受理追加新行。
//
// 线上方言（抓包逐字节）：插入 = 0x11 空行 + 4×0x13 逐格；改写 = 2×0x13
// 只写起止；清扫 = 0x12。清扫对每次被处理的按键都发生（受理、超距拒绝、
// 冷却中静默三者同款——397.17s 那次单发删除就是冷却中被拒的按键留下
// 的），新行只在受理时写。
//
// 官服长会话里刷新公共行用的缓存行号会失效（381s 时写到了别的行上、
// 表里积了三行 50000），本实现始终按类别定位真行。

import (
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/skilldata"
)

const (
	// cooldownWindowCategory 窗口行类别（施法窗，时长恒 cooldownWindowMS）。
	cooldownWindowCategory uint32 = 1
	// cooldownSharedCategory 套路间共享冷却行的类别（CoolEffectType=1）。
	cooldownSharedCategory uint32 = 50000
	// cooldownWindowMS 窗口行时长（抓包全样本恒 1000ms）。
	cooldownWindowMS = 1000
	// cooldownSweepPerCast 单次清扫过期行的上限（抓包每次 0~2 行）。
	cooldownSweepPerCast = 2
	// cooldownRowsHighWater 行表水位：超过后连非本技能类别的过期行也清。
	// 官服只清本技能类别的过期行，长会话下表越积越多；这道有界性保险
	// 对客户端不可见（过期行本就不显示）。
	cooldownRowsHighWater = 16
	// cooldownTeamNone 不分组的队位。
	cooldownTeamNone uint16 = 0xFFFF
	// cooldownFiletimeEpoch FILETIME 纪元偏移：1601 毫秒 = Unix 毫秒 + 此值。
	cooldownFiletimeEpoch int64 = 11644473600000
)

// cooldownRow cooldown_rec 一行的内存镜像。
type cooldownRow struct {
	category uint32
	start    int64
	end      int64
	team     uint16
}

// filetimeMS 官服冷却表时间基准（1601 纪元毫秒）。
func filetimeMS(now time.Time) int64 { return now.UnixMilli() + cooldownFiletimeEpoch }

// publicCooldownRow 本次施放的公共行（AddCoolDownTime<=0 或无分组则无）。
func publicCooldownRow(def skilldata.Definition, nowMS int64) (cooldownRow, bool) {
	if def.PublicCoolDownMS <= 0 || def.CoolDownTeam <= 0 {
		return cooldownRow{}, false
	}
	end := nowMS + int64(def.PublicCoolDownMS)
	if def.CoolEffectType == 2 {
		return cooldownRow{category: uint32(def.CoolDownTeam), start: nowMS, end: end, team: cooldownTeamNone}, true
	}
	return cooldownRow{category: cooldownSharedCategory, start: nowMS, end: end, team: uint16(def.CoolDownTeam & 0xFFFF)}, true
}

// live 现有行表里该类别是否有未过期行。
func cooldownLive(rows []cooldownRow, category uint32, nowMS int64) bool {
	for _, row := range rows {
		if row.category == category && row.end > nowMS {
			return true
		}
	}
	return false
}

// cooldownGate 冷却闸：命中返回原因（空串 = 放行）。在清扫后的行表上判定。
// 官服对冷却中的按键零应答（客户端按冷却表自行拦键——397.08/397.45 两次
// 普攻静默即此）。
func cooldownGate(rows []cooldownRow, lastTaoLu string, def skilldata.Definition, nowMS int64) string {
	if def.CoolDownMS > 0 && cooldownLive(rows, uint32(def.CoolDownCategory), nowMS) {
		return "个人冷却"
	}
	if def.PublicCoolDownMS <= 0 || def.CoolDownTeam <= 0 {
		return ""
	}
	if def.CoolEffectType == 2 {
		if cooldownLive(rows, uint32(def.CoolDownTeam), nowMS) {
			return "组内公共冷却"
		}
		return ""
	}
	if lastTaoLu != def.TaoLu && cooldownLive(rows, cooldownSharedCategory, nowMS) {
		return "换套路共享冷却"
	}
	return ""
}

// sweepCooldowns 清扫过期行，返回 0x12 帧与新行表（不改动入参切片）。
// 优先清本技能会写的类别（个人/公共/共享）的过期行，行号升序，至多
// cooldownSweepPerCast 行；窗口行不清。表超高水位时放宽到任意过期行。
func sweepCooldowns(objectID, ownerID uint32, rows []cooldownRow, def skilldata.Definition, nowMS int64) ([][]byte, []cooldownRow) {
	swept := append([]cooldownRow(nil), rows...)
	own := map[uint32]bool{uint32(def.CoolDownCategory): true, cooldownSharedCategory: true}
	if public, ok := publicCooldownRow(def, nowMS); ok {
		own[public.category] = true
	}
	relaxed := len(swept) > cooldownRowsHighWater
	record := clientdata.MustRecordIndexByName("cooldown_rec")
	var frames [][]byte
	for count := 0; count < cooldownSweepPerCast; count++ {
		index := -1
		for i, row := range swept {
			if row.category != cooldownWindowCategory && row.end <= nowMS && (own[row.category] || relaxed) {
				index = i
				break
			}
		}
		if index < 0 {
			break
		}
		frames = append(frames, entergen.RecordDelRowFrame(objectID, ownerID, record, uint16(index)))
		swept = append(swept[:index], swept[index+1:]...)
	}
	return frames, swept
}

// cooldownRowWriter 行表写入器：帧序 = 调用序，行表随之镜像。
type cooldownRowWriter struct {
	objectID, ownerID uint32
	record            uint16
	rows              []cooldownRow
	frames            [][]byte
}

// insert 在表尾追加一行：0x11 空行 + 四格。
func (w *cooldownRowWriter) insert(row cooldownRow) error {
	index := uint16(len(w.rows))
	frame, err := entergen.RecordInsertRowFrame(w.objectID, w.ownerID, w.record, index)
	if err != nil {
		return err
	}
	w.frames = append(w.frames, frame)
	for _, cell := range []entergen.RecordGridCell{
		entergen.GridInt(0, row.category), entergen.GridInt64(1, row.start),
		entergen.GridInt64(2, row.end), entergen.GridWord(3, row.team),
	} {
		frame, err = entergen.RecordGridFrame(w.objectID, w.ownerID, w.record, index, cell)
		if err != nil {
			return err
		}
		w.frames = append(w.frames, frame)
	}
	w.rows = append(w.rows, row)
	return nil
}

// rewriteSpan 改写某行的起止两格。
func (w *cooldownRowWriter) rewriteSpan(index int, start, end int64) error {
	for _, cell := range []entergen.RecordGridCell{entergen.GridInt64(1, start), entergen.GridInt64(2, end)} {
		frame, err := entergen.RecordGridFrame(w.objectID, w.ownerID, w.record, uint16(index), cell)
		if err != nil {
			return err
		}
		w.frames = append(w.frames, frame)
	}
	w.rows[index].start, w.rows[index].end = start, end
	return nil
}

// find 按类别（可带队）定位首行。
func (w *cooldownRowWriter) find(category uint32, team *uint16) int {
	for i, row := range w.rows {
		if row.category == category && (team == nil || row.team == *team) {
			return i
		}
	}
	return -1
}

// writeCooldownRows 在（已清扫的）行表上写本次受理的三类行，返回新帧与
// 新行表。帧序：个人行 → 窗口行 → 公共行（抓包 197.729/209.389 逐字节）。
func writeCooldownRows(objectID, ownerID uint32, rows []cooldownRow, def skilldata.Definition, nowMS int64) ([][]byte, []cooldownRow, error) {
	w := &cooldownRowWriter{
		objectID: objectID, ownerID: ownerID,
		record: clientdata.MustRecordIndexByName("cooldown_rec"),
		rows:   append([]cooldownRow(nil), rows...),
	}
	// 个人行：每次受理追加新行。
	if def.CoolDownMS > 0 {
		if err := w.insert(cooldownRow{category: uint32(def.CoolDownCategory), start: nowMS,
			end: nowMS + int64(def.CoolDownMS), team: cooldownTeamNone}); err != nil {
			return nil, nil, err
		}
	}
	// 窗口行：有则改写起止，无则插入。
	if index := w.find(cooldownWindowCategory, nil); index >= 0 {
		if err := w.rewriteSpan(index, nowMS, nowMS+cooldownWindowMS); err != nil {
			return nil, nil, err
		}
	} else if err := w.insert(cooldownRow{category: cooldownWindowCategory, start: nowMS,
		end: nowMS + cooldownWindowMS, team: cooldownTeamNone}); err != nil {
		return nil, nil, err
	}
	// 公共行：CoolEffectType=1 沿用未过期的同行（原地改写起止），过期或
	// 不存在则追加；CoolEffectType=2 恒追加（旧行未过期已被闸拦下）。
	if public, ok := publicCooldownRow(def, nowMS); ok {
		reuse := -1
		if def.CoolEffectType != 2 {
			reuse = w.find(public.category, &public.team)
			if reuse >= 0 && w.rows[reuse].end <= nowMS {
				reuse = -1
			}
		}
		if reuse >= 0 {
			if err := w.rewriteSpan(reuse, public.start, public.end); err != nil {
				return nil, nil, err
			}
		} else if err := w.insert(public); err != nil {
			return nil, nil, err
		}
	}
	return w.frames, w.rows, nil
}
