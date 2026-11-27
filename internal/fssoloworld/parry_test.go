package fssoloworld

// msg54 手动格挡双方言测试（参考端 820f79a 真机定版语义 + 2026-11-23
// 官方女号抓包 197 条全量复核）：0/1 直判（0=按下）；官服族令牌上报
//（基调=首条、按压=基调+K、长按重发零帧、轮换异值采纳）；
// 无变迁零帧；应答=三属性权威帧，全程不碰 LogicState（官服非战斗
// 按压簇无 LogicState 帧；旧撑持帧致「格挡瞬时进入战斗状态」）。

import (
	"encoding/binary"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
	"github.com/lizhanyin/9yin-server-go/internal/entergen"
)

func newParryTestServer(t *testing.T) (*Server, *Player, *[][]byte) {
	t.Helper()
	s := newNpcTestServer(t)
	pl := &Player{Conn1: 9, State: PlOnline}
	var frames [][]byte
	prev := clientSendObserver
	clientSendObserver = func(pl *Player, frame []byte) { frames = append(frames, frame) }
	t.Cleanup(func() { clientSendObserver = prev })
	return s, pl, &frames
}

func sendM54(s *Server, pl *Player, tag byte, value int64) {
	v := entergen.CustomInboundValue{Type: tag}
	if tag == 2 {
		v.Int32 = int32(value)
	} else {
		v.Int64 = value
	}
	s.handleParryState(pl, entergen.CustomInbound{Opcode: 0x0A, Values: []entergen.CustomInboundValue{
		{Type: 2, Int32: clientCustomParryState}, v,
	}})
}

// 官服族状态上报（我方 B 线登录的真机形态）：恒 1=基调闲时上报零帧；
// 按压=基调+K（官服女号抓包六纪元同差 0x11D5DA29）。
func TestParryStateReportBaseline(t *testing.T) {
	s, pl, frames := newParryTestServer(t)
	sendM54(s, pl, 3, 1)
	sendM54(s, pl, 3, 1)
	sendM54(s, pl, 3, 1)
	if len(*frames) != 0 {
		t.Fatalf("基调上报应零帧（回帧自激成洪流），实发 %d 帧", len(*frames))
	}
	if pl.parrying {
		t.Error("基调上报不应进入格挡态")
	}
	// 按压 = 基调+K：仅三属性帧。
	sendM54(s, pl, 3, 1+parryTokenDelta)
	if len(*frames) != 1 {
		t.Fatalf("按下应只发 1 帧三属性，实发 %d", len(*frames))
	}
	if !hasPropIdx((*frames)[0]) {
		t.Fatalf("首帧应三属性 0x10: % x", (*frames)[0])
	}
	if !pl.parrying {
		t.Error("按下后应进入格挡态")
	}
	// 长按重发按压令牌（官服抓包 E3 纪元 11s 长按 9 连发）：零帧。
	sendM54(s, pl, 3, 1+parryTokenDelta)
	if len(*frames) != 1 || !pl.parrying {
		t.Fatalf("长按重发应零帧: frames=%d parrying=%t", len(*frames), pl.parrying)
	}
	// 回基调 = 松开：再发 1 帧三属性；松开重发零帧。
	sendM54(s, pl, 3, 1)
	if len(*frames) != 2 {
		t.Fatalf("松开应再发 1 帧，实发 %d", len(*frames))
	}
	if pl.parrying {
		t.Error("松开后应退出格挡态")
	}
	sendM54(s, pl, 3, 1)
	if len(*frames) != 2 {
		t.Fatalf("松开重发应零帧，实发 %d 帧", len(*frames))
	}
	// 官服非战斗按压簇不含 LogicState：撑持帧会让客户端瞬时进入战斗
	// 状态（2026-11-24 真机），按下/松开全程不得下发该属性。
	logic := uint16(clientdata.MustPlayerProperty("LogicState").Index)
	for _, frame := range *frames {
		for _, idx := range framePropIndexes(frame) {
			if idx == logic {
				t.Fatalf("格挡应答不得带 LogicState: % x", frame)
			}
		}
	}
}

// fork 二值方言兼容：值域 {0,1} 直判（0=按下），与基调锁存值无关。
func TestParryBinaryDialect(t *testing.T) {
	s, pl, frames := newParryTestServer(t)
	sendM54(s, pl, 2, 1) // 基线采集（闲时）
	sendM54(s, pl, 2, 0) // 0=按下（异值）
	if !pl.parrying || len(*frames) != 1 {
		t.Fatalf("0=按下应受理: parrying=%t frames=%d", pl.parrying, len(*frames))
	}
	sendM54(s, pl, 2, 1) // 回基调=松开
	if pl.parrying || len(*frames) != 2 {
		t.Fatalf("回基调应松开: parrying=%t frames=%d", pl.parrying, len(*frames))
	}
	sendM54(s, pl, 2, 1) // 重复基调：零帧
	if len(*frames) != 2 {
		t.Fatalf("重复基调应零帧，实发 %d 帧", len(*frames))
	}
	// 基调先锁 0 的反判防护：首报即 0（按下态上报）按二值方言直判按下。
	s2, pl2, frames2 := newParryTestServer(t)
	sendM54(s2, pl2, 2, 0) // 首报 0：锚定空闲 1、直判按下（1 帧）
	if !pl2.parrying || len(*frames2) != 1 {
		t.Fatalf("首报 0 应判按下: parrying=%t frames=%d", pl2.parrying, len(*frames2))
	}
	sendM54(s2, pl2, 2, 0) // 仍 0：无变迁零帧
	if !pl2.parrying || len(*frames2) != 1 {
		t.Fatalf("持续按下应零帧: parrying=%t frames=%d", pl2.parrying, len(*frames2))
	}
	sendM54(s2, pl2, 2, 1) // 1=松开
	if pl2.parrying || len(*frames2) != 2 {
		t.Fatalf("1 应松开: parrying=%t frames=%d", pl2.parrying, len(*frames2))
	}
}

// 官服族基调轮换（2026-11-23 官方女号抓包：令牌按副本切景轮换、14 分钟
// 8 纪元）：既非基调也非基调+K 的异值=新纪元基调，采纳零帧并复位；
// 新纪元按压=新基调+K。
func TestParryBaselineRotation(t *testing.T) {
	s, pl, frames := newParryTestServer(t)
	sendM54(s, pl, 3, 1)          // 纪元 A 基调采集
	sendM54(s, pl, 3, 5226499786) // 切景轮换：异值非 K 形 → 采纳，零帧
	if pl.parrying || len(*frames) != 0 {
		t.Fatalf("轮换异值应采纳零帧: parrying=%t frames=%d", pl.parrying, len(*frames))
	}
	if pl.parryBaseline != 5226499786 {
		t.Fatalf("基调应更新为 5226499786，实为 %d", pl.parryBaseline)
	}
	sendM54(s, pl, 3, 5226499786) // 新基调心跳：零帧
	if len(*frames) != 0 {
		t.Fatalf("新基调心跳应零帧，实发 %d 帧", len(*frames))
	}
	// 新纪元按压（基调+0x11D5DA29）→ 长按重发零帧 → 松开回新基调。
	sendM54(s, pl, 3, 5525727475)
	if !pl.parrying || len(*frames) != 1 {
		t.Fatalf("新纪元按压应受理: parrying=%t frames=%d", pl.parrying, len(*frames))
	}
	sendM54(s, pl, 3, 5525727475) // 长按重发：零帧
	if len(*frames) != 1 {
		t.Fatalf("长按重发应零帧，实发 %d", len(*frames))
	}
	sendM54(s, pl, 3, 5226499786)
	if pl.parrying || len(*frames) != 2 {
		t.Fatalf("回新基调应松开: parrying=%t frames=%d", pl.parrying, len(*frames))
	}
	// 轮换发生在格挡中：新基调采纳并复位（推松开帧）。
	sendM54(s, pl, 3, 5525727475) // 再按压（新基调+K）
	if !pl.parrying || len(*frames) != 3 {
		t.Fatalf("再按压应受理: parrying=%t frames=%d", pl.parrying, len(*frames))
	}
	sendM54(s, pl, 3, 4667437540) // 持按中切景轮换：采纳 + 复位
	if pl.parrying || len(*frames) != 4 {
		t.Fatalf("持按轮换应采纳并复位: parrying=%t frames=%d", pl.parrying, len(*frames))
	}
}

func hasPropIdx(frame []byte) bool { return len(frame) >= 14 && frame[0] == 0x10 }

// framePropIndexes 0x10 属性帧的属性号列表（[idx u16][值]…，值宽按
// 官方属性表线型走）。
func framePropIndexes(frame []byte) []uint16 {
	if !hasPropIdx(frame) {
		return nil
	}
	count := int(binary.LittleEndian.Uint16(frame[10:]))
	body := frame[12:]
	var out []uint16
	for i := 0; i < count; i++ {
		if len(body) < 2 {
			break
		}
		idx := int(binary.LittleEndian.Uint16(body))
		entry, ok := clientdata.PropertyByIndex(idx)
		if !ok {
			break
		}
		out = append(out, uint16(idx))
		var width int
		switch entry.Type {
		case clientdata.WireByte:
			width = 1
		case clientdata.WireWord:
			width = 2
		case clientdata.WireInt32, clientdata.WireFloat32:
			width = 4
		case clientdata.WireInt64, clientdata.WireObject:
			width = 8
		default:
			return out // 串型值宽不固定，格挡应答帧用不到
		}
		body = body[2+width:]
	}
	return out
}

func TestParryValueTotalFallback(t *testing.T) {
	s, pl, _ := newParryTestServer(t)
	if got := s.parryValueTotal(pl); got != 72 {
		t.Fatalf("无存档回退招架值 = %d, want 72", got)
	}
	_ = binary.LittleEndian
}

// TestParryCombatCluster 战斗上下文官服簇序（docs/official/初始女号抓包
// m54手动格挡全解.md §四）：按压 = MoveSpeed → RunSpeedAdd → InParry →
// ParryUpSpeedAdd → BufferInfo2 槽串 → m169(0,"BuffInParry") →
// InManualParry；松开 = m169(1) 簇首 → BufferInfo2 清空 → 全清零。
func TestParryCombatCluster(t *testing.T) {
	s, pl, frames := newParryTestServer(t)
	pl.Name = "格挡侠"
	// 战斗上下文：live.ready + logicFighting。
	pl.combat.mu.Lock()
	pl.combat.live.ready = true
	pl.combat.live.logic = logicFighting
	pl.combat.mu.Unlock()

	sendM54(s, pl, 3, 1)                    // 基调采集（零帧）
	sendM54(s, pl, 3, 1+parryTokenDelta)    // 按压
	if len(*frames) != 7 {
		t.Fatalf("按压簇应 7 帧，实发 %d: ops=% x", len(*frames), opOf(*frames))
	}
	press := *frames
	for i := 0; i < 5; i++ { // 前五帧单属性，m169 在下标 5、簇尾 6
		if press[i][0] != 0x10 {
			t.Fatalf("按压簇第 %d 帧应单属性 0x10: % x", i, press[i])
		}
	}
	if !frameContains(press[4:5], []byte("11CE,")) {
		t.Fatalf("BufferInfo2 槽串应携 11CE 前缀: % x", press[4])
	}
	if press[6][0] != 0x10 {
		t.Fatalf("簇尾应 InManualParry 0x10: % x", press[6])
	}
	if press[5][0] != 0x1E || !frameContains(press[5:6], []byte("BuffInParry")) ||
		!frameContains(press[5:6], []byte("格挡侠")) {
		t.Fatalf("m169 挂帧异常: % x", press[5])
	}

	*frames = nil
	sendM54(s, pl, 3, 1) // 松开
	release := *frames
	if len(release) != 7 {
		t.Fatalf("松开簇应 7 帧，实发 %d: ops=% x", len(release), opOf(release))
	}
	if release[0][0] != 0x1E || !frameContains(release[0:1], []byte("BuffInParry")) {
		t.Fatalf("松开簇首应 m169 撤帧: % x", release[0])
	}
	for i := 1; i < 7; i++ {
		if release[i][0] != 0x10 {
			t.Fatalf("松开簇第 %d 帧应 0x10: % x", i, release[i])
		}
	}
	if !frameContains(release[1:2], mustHex(t, "0100000000")) {
		t.Fatalf("BufferInfo2 应清空（空串 len=1）: % x", release[1])
	}
}
