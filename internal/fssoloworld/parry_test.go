package fssoloworld

// msg54 手动格挡双方言测试（参考端 820f79a 真机定版语义）：
// 0/1 直判（0=按下）；官服族状态上报（基调=首条、≠基调=格挡中）；
// 无变迁零帧；应答=三属性权威帧+LogicState 撑持。

import (
	"bytes"
	"encoding/binary"
	"testing"

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

// 官服族状态上报（我方 B 线登录的真机形态）：恒 1=基调闲时上报零帧。
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
	// 值≠基调 = 按下：三属性帧 + LogicState=1。
	sendM54(s, pl, 3, 5525727475)
	if len(*frames) != 2 {
		t.Fatalf("按下应发 2 帧（三属性+LogicState），实发 %d", len(*frames))
	}
	if !bytes.Contains((*frames)[0], []byte("ParryValue")) && !hasPropIdx((*frames)[0]) {
		t.Fatalf("首帧应三属性 0x10: % x", (*frames)[0])
	}
	if !pl.parrying {
		t.Error("按下后应进入格挡态")
	}
	// 回基调 = 松开：三属性帧 + LogicState=0。
	sendM54(s, pl, 3, 1)
	if len(*frames) != 4 {
		t.Fatalf("松开应再发 2 帧，实发 %d", len(*frames))
	}
	if pl.parrying {
		t.Error("松开后应退出格挡态")
	}
}

// fork 二值方言兼容：基调=1 时 0=按下（异值分支覆盖）。
func TestParryBinaryDialect(t *testing.T) {
	s, pl, frames := newParryTestServer(t)
	sendM54(s, pl, 2, 1) // 基线采集（闲时）
	sendM54(s, pl, 2, 0) // 0=按下（异值）
	if !pl.parrying || len(*frames) != 2 {
		t.Fatalf("0=按下应受理: parrying=%t frames=%d", pl.parrying, len(*frames))
	}
	sendM54(s, pl, 2, 1) // 回基调=松开
	if pl.parrying || len(*frames) != 4 {
		t.Fatalf("回基调应松开: parrying=%t frames=%d", pl.parrying, len(*frames))
	}
	sendM54(s, pl, 2, 1) // 重复基调：零帧
	if len(*frames) != 4 {
		t.Fatalf("重复基调应零帧，实发 %d", len(*frames))
	}
}

func hasPropIdx(frame []byte) bool { return len(frame) >= 14 && frame[0] == 0x10 }

func TestParryValueTotalFallback(t *testing.T) {
	s, pl, _ := newParryTestServer(t)
	if got := s.parryValueTotal(pl); got != 72 {
		t.Fatalf("无存档回退招架值 = %d, want 72", got)
	}
	_ = binary.LittleEndian
}
