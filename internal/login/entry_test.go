package login

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"
)

func TestEntrySceneFrameLayout(t *testing.T) {
	frame := entrySceneFrame(BirthScene)
	if frame[0] != OpEntryScene {
		t.Fatalf("opcode %02X", frame[0])
	}
	u16 := func(off int) uint16 { return binary.LittleEndian.Uint16(frame[off:]) }
	u32 := func(off int) uint32 { return binary.LittleEndian.Uint32(frame[off:]) }
	if got := u32(1); got != 0x1521 {
		t.Fatalf("场景流水 %#x", got)
	}
	if got := u32(5); got != 0x1105C0E3 {
		t.Fatalf("线缆 obj %#x", got)
	}
	if got := u32(9); got != 0x03DEFBAE {
		t.Fatalf("线缆 owner %#x", got)
	}
	if got := u16(13); got != 4 {
		t.Fatalf("属性数 %d", got)
	}

	// 属性段走位：178(常量) → 1468(u16 230) → 7(串) → 8(串)
	off := 15
	if idx := u16(off); idx != 178 {
		t.Fatalf("首属性 idx %d want 178", idx)
	}
	off += 2
	if n := u32(off); n != 8 {
		t.Fatalf("idx178 长度 %d want 8", n)
	}
	off += 4 + 8

	if idx := u16(off); idx != 1468 {
		t.Fatalf("次属性 idx %d want 1468", idx)
	}
	off += 2
	if v := u16(off); v != 230 {
		t.Fatalf("MaxVisuals %d want 230", v)
	}
	off += 2

	strAt := func() (uint16, string) {
		idx := u16(off)
		off += 2
		n := int(u32(off))
		off += 4
		s := string(frame[off : off+n-1])
		off += n
		return idx, s
	}
	if idx, cfg := strAt(); idx != 7 || cfg != BirthScene.Config {
		t.Fatalf("ConfigID 属性 idx=%d %q", idx, cfg)
	}
	if idx, res := strAt(); idx != 8 || res != BirthScene.Resource {
		t.Fatalf("Resource 属性 idx=%d %q", idx, res)
	}
	if off != len(frame) {
		t.Fatalf("走位 %d != 帧长 %d", off, len(frame))
	}

	// 遮蔽常量逐字节
	shade := frame[15+2+4 : 15+2+4+8]
	if !bytes.Equal(shade, []byte{0x1b, 0x54, 0x50, 0x5b, 0x02, 0x58, 0x00, 0x00}) {
		t.Fatalf("idx178 常量 %x", shade)
	}
}

// TestFlowEntrySceneOverGateway 端到端：登录→建角→选角 → 0x0B 进场帧
// （场景串=出生场景）+ OnEnterScene 回调触发。
func TestFlowEntrySceneOverGateway(t *testing.T) {
	flight := capturedFirstFlight(t)
	entered := make(chan *Role, 1)
	flow := &Flow{OnEnterScene: func(_ *Session, r *Role) { entered <- r }}
	g := &Gateway{Addr: "127.0.0.1:0", OnFrame: flow.Handle}
	addr, err := g.Listen()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { g.Close() })
	go g.Serve()

	conn := dialProbe(t, addr.String())
	for _, frame := range [][]byte{flight, buildCreateRoleFrame("小虾米", nil), {OpClientRoleSel}} {
		if err := conn.WriteMsg(frame); err != nil {
			t.Fatal(err)
		}
	}
	// 三应答：0x04（无/有角色）×2 + 0x0B
	var entry []byte
	for i := 0; i < 3; i++ {
		reply, err := conn.ReadMsg()
		if err != nil {
			t.Fatal(err)
		}
		if reply[0] == OpEntryScene {
			entry = reply
		}
	}
	if entry == nil {
		t.Fatal("未收到 0x0B")
	}
	if got := string(entry[15+2+4+8+2+2+2+4 : 15+2+4+8+2+2+2+4+len(BirthScene.Config)]); got != BirthScene.Config {
		t.Fatalf("进场 Config %q", got)
	}
	// 0x0B 写回与 OnEnterScene 同处一个 handler：读到帧时回调可能尚未
	// 执行（写回先于回调），有界等待而非零等待 select。
	var enteredRole *Role
	select {
	case enteredRole = <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("OnEnterScene 未触发")
	}
	if enteredRole.Scene.Resource != "school09" {
		t.Fatalf("回调场景 %q", enteredRole.Scene.Resource)
	}

	// 重复 op04（真机重试方言）：0x0B 仍回发，但物化事件只发一次
	if err := conn.WriteMsg([]byte{OpClientRoleSel}); err != nil {
		t.Fatal(err)
	}
	reply, err := conn.ReadMsg()
	if err != nil || reply[0] != OpEntryScene {
		t.Fatalf("重复选角应答 %x %v", reply, err)
	}
	select {
	case <-entered:
		t.Fatal("重复 op04 不应再次物化")
	default:
	}
}
