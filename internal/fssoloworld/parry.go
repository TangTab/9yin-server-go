package fssoloworld

// C2S 54 手动格挡（2026-11-23 女号官服抓包「初始账号登陆-完整引导-女」
// 192s 段字节级定版；此前 2433 参考端 protocol-probe 只在战斗态接线，
// 非战斗态 m54 全部落 unhandled——真机表象：非战斗按格挡无任何反馈）。
//
// 官方方言：
//   - 客户端按下/抬起各发一个 u64 值（同值 ~35ms 重发直到回声；ON 值 =
//     OFF 值 + 0x11D5DA29，值本身是客户端计时器，逐对缓变，不可靠分类）；
//   - 服务端只按「值变化 = 一次翻转」应答（重复值静默；重复 OFF 官方
//     也常不回，客户端容忍持续重发）；
//   - ON 应答序（逐帧单属性 0x10）：MoveSpeed=1.17 / RunSpeedAdd=-2.36 /
//     InParry=1 / ParryUpSpeedAdd=-1 / BufferInfo2="11CE,<own>,<obj>,1,0,0"
//     → m169 [169,0,"BuffInParry",obj,obj,宽串×2,0,0] → InManualParry=1；
//   - OFF 应答序：m169 c1=1（移除）→ BufferInfo2="" / MoveSpeed=3.9 /
//     RunSpeedAdd=0 / InParry=0 / ParryUpSpeedAdd=0 / InManualParry=0。
//     （c1 极性经 InParry 转换逐帧相关验证：0=添加 1=移除。）
//   - 宽串两枚官方恒为 UTF-16 "致文"（7e 82 87 65 00 00，计时器余脉），
//     照抄保形。

import (
	"fmt"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
)

// clientCustomParryState C2S msg54 格挡状态 [54, int64]。
const clientCustomParryState int32 = 54

// 格挡官方常量（女号抓包逐帧对齐）。
const (
	parryBuffName     = "BuffInParry"
	parryBufferSerial = "11CE" // BufferInfo2 串首段（buff 槽序号域）
	parryMoveSpeedOn  = 1.17   // 3.9×0.3 格挡减速
	parryRunSpeedAdd  = -2.36
	parryMoveSpeedOff = 3.9
	parryWideResidue  = "\u827e\u6587" // 官方宽串余脉字节 7e 82 87 65
)

// handleParryState C2S 54：值变化 = 翻转一次；按官方帧序回声。
func (s *Server) handleParryState(pl *Player, message entergen.CustomInbound) {
	if len(message.Values) < 2 || message.Values[1].Type != 3 {
		return
	}
	value := message.Values[1].Int64
	pl.parryMu.Lock()
	if value == pl.parryLastValue {
		pl.parryMu.Unlock()
		return // 同值重发（客户端等回声的 ~35ms 轮询）
	}
	pl.parryLastValue = value
	on := !pl.parryOn
	pl.parryOn = on
	pl.parryMu.Unlock()

	if on {
		s.sendParryProps(pl,
			entergen.NamedProperty("MoveSpeed", entergen.Float32Value(parryMoveSpeedOn)),
			entergen.NamedProperty("RunSpeedAdd", entergen.Float32Value(parryRunSpeedAdd)),
			entergen.NamedProperty("InParry", entergen.Int32Value(1)),
			entergen.NamedProperty("ParryUpSpeedAdd", entergen.Int32Value(-1)),
			entergen.NamedProperty("BufferInfo2", entergen.StringValue(parryBufferInfo())),
		)
		s.sendParryEcho(pl, 0)
		s.sendParryProps(pl,
			entergen.NamedProperty("InManualParry", entergen.Int32Value(1)),
		)
		s.trace("parry on client=%d value=%d", pl.Conn1, value)
		return
	}
	s.sendParryEcho(pl, 1)
	s.sendParryProps(pl,
		entergen.NamedProperty("BufferInfo2", entergen.StringValue("")),
		entergen.NamedProperty("MoveSpeed", entergen.Float32Value(parryMoveSpeedOff)),
		entergen.NamedProperty("RunSpeedAdd", entergen.Float32Value(0)),
		entergen.NamedProperty("InParry", entergen.Int32Value(0)),
		entergen.NamedProperty("ParryUpSpeedAdd", entergen.Int32Value(0)),
		entergen.NamedProperty("InManualParry", entergen.Int32Value(0)),
	)
	s.trace("parry off client=%d value=%d", pl.Conn1, value)
}

// sendParryProps 逐属性单帧 0x10（官方每帧一属性；地址=fork 玩家对象）。
func (s *Server) sendParryProps(pl *Player, props ...entergen.IndexedProperty) {
	for _, prop := range props {
		frame, err := entergen.ObjectPropertyFrame(forkPlayerObjectID, forkPlayerOwnerID, 0,
			[]entergen.IndexedProperty{prop})
		if err != nil {
			s.trace("parry prop %s encode failed client=%d: %v", prop.Name, pl.Conn1, err)
			return
		}
		s.sendToClient(pl, frame)
	}
}

// sendParryEcho m169 BuffInParry 加/除（addRemove：0=添加 1=移除）。
func (s *Server) sendParryEcho(pl *Player, addRemove int32) {
	frame, err := entergen.CustomMessageFrame(169,
		entergen.CustomInt(addRemove),
		entergen.CustomString(parryBuffName),
		entergen.CustomObject(forkPlayerObjectID, forkPlayerOwnerID),
		entergen.CustomObject(forkPlayerObjectID, forkPlayerOwnerID),
		entergen.CustomWideString(parryWideResidue),
		entergen.CustomWideString(parryWideResidue),
		entergen.CustomInt64(0),
		entergen.CustomInt(0),
	)
	if err != nil {
		s.trace("parry echo encode failed client=%d: %v", pl.Conn1, err)
		return
	}
	s.sendToClient(pl, frame)
}

// parryBufferInfo ON 态 BufferInfo2 串（官方 "11CE,350000F3,102A,1,0,0"
// = 序号,own,obj,1,0,0；fork 玩家对象 1/1）。
func parryBufferInfo() string {
	return fmt.Sprintf("%s,%X,%X,1,0,0", parryBufferSerial, forkPlayerOwnerID, forkPlayerObjectID)
}
