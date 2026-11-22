package fssoloworld

// msg54 手动格挡回声序列测试（官方女号抓包 192s 段方言）：
// 值变化=翻转，同值重发静默；ON/OFF 各按官方帧序回 m169 + 0x10 属性。

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
)

func TestParryToggleEcho(t *testing.T) {
	s := newNpcTestServer(t)
	pl := &Player{Conn1: 9, State: PlOnline}

	var frames [][]byte
	prev := clientSendObserver
	clientSendObserver = func(pl *Player, frame []byte) { frames = append(frames, frame) }
	t.Cleanup(func() { clientSendObserver = prev })

	send := func(v int64) {
		s.handleParryState(pl, entergen.CustomInbound{Opcode: 0x1A, Values: []entergen.CustomInboundValue{
			{Type: 2, Int32: clientCustomParryState},
			{Type: 3, Int64: v},
		}})
	}

	// 按下：ON 簇（5×0x10 + m169 c1=0 + 0x10 InManualParry）。
	send(5525727475)
	if len(frames) != 7 {
		t.Fatalf("ON 簇帧数 = %d, want 7", len(frames))
	}
	if !bytes.Contains(frames[5], []byte("BuffInParry")) || frames[5][0] != 0x1E {
		t.Fatalf("第 6 帧应为 m169 BuffInParry: % x", frames[5])
	}
	if mid := binary.LittleEndian.Uint32(frames[5][4:]); mid != 169 {
		t.Fatalf("m169 消息号 = %d", mid)
	}
	if c1 := frames[5][9]; c1 != 0 { // n=9 @1，msg id 后首值 tag=02、值低字节
		t.Fatalf("ON 回声 c1 = %d, want 0（添加）", c1)
	}
	if !pl.parryOn {
		t.Error("按下后 parryOn 应为 true")
	}

	// 同值重发（客户端等回声轮询）：静默。
	before := len(frames)
	send(5525727475)
	if len(frames) != before {
		t.Fatalf("同值重发不应再发帧：%d → %d", before, len(frames))
	}

	// 抬起：OFF 簇（m169 c1=1 + 6×0x10）。
	send(5226499786)
	if len(frames) != before+7 {
		t.Fatalf("OFF 簇帧数异常：%d → %d", before, len(frames))
	}
	off := frames[before]
	if off[0] != 0x1E || !bytes.Contains(off, []byte("BuffInParry")) {
		t.Fatalf("OFF 首帧应为 m169: % x", off)
	}
	if c1 := off[9]; c1 != 1 {
		t.Fatalf("OFF 回声 c1 = %d, want 1（移除）", c1)
	}
	if pl.parryOn {
		t.Error("抬起后 parryOn 应为 false")
	}

	// 再按下（同值复用也不误判——值变化才翻转）。
	send(5525727475)
	if !pl.parryOn {
		t.Error("再次按下应翻回 true")
	}
}
