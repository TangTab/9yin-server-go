package main

import (
	"testing"
)

// TestMirrorJumpTimesOfficialDialect 官方 JumpTimes 飞行计数镜像（技能4
// .pcapng 62 帧定版）：起飞授 1、空中二跳（state6 再上行 >0.3m）耗 0、
// 空中水平段（state7 腾空期）耗 0、落地（state4/5）静默复位后再次起飞
// 再授 1。我方不发该属性=客户端空中动作配额耗尽后不复活（2026-11-16
// 「穿云纵过去后卡状态」根因，58540 会话原地垂直弹跳循环铁证）。
func TestMirrorJumpTimesOfficialDialect(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("踏云侠", 0)

	frameValue := func(idx int) (int32, bool) {
		frames := conn.Frames()
		if len(frames) <= idx {
			return 0, false
		}
		f := frames[idx]
		// 0x10 单属性帧：值在帧尾 4B（Int32）——按 objectSingleProperty
		// 形状解析尾 int32。
		if f[0] != 0x10 || len(f) < 4 {
			return 0, false
		}
		var v int32
		for _, b := range f[len(f)-4:] {
			v = v<<8 | int32(b)
		}
		// 小端还原
		v = int32(uint32(f[len(f)-4]) | uint32(f[len(f)-3])<<8 | uint32(f[len(f)-2])<<16 | uint32(f[len(f)-1])<<24)
		return v, true
	}

	// 地面态（state4）→ 无帧。
	player.mirrorJumpTimes(conn, 4, 30.5)
	if got := len(conn.Frames()); got != 0 {
		t.Fatalf("ground report emitted %d frames, want 0", got)
	}
	// 起跳（state6）→ 授 1。
	player.mirrorJumpTimes(conn, 6, 31.0)
	if got := len(conn.Frames()); got != 1 {
		t.Fatalf("takeoff emitted %d frames, want 1", got)
	}
	if v, ok := frameValue(0); !ok || v != 1 {
		t.Fatalf("takeoff JumpTimes = %d ok=%t, want 1", v, ok)
	}
	// 腾空水平段（state7）→ 耗 0。
	player.mirrorJumpTimes(conn, 7, 34.5)
	if got := len(conn.Frames()); got != 2 {
		t.Fatalf("air dash emitted %d frames, want 1 more", got)
	}
	if v, _ := frameValue(1); v != 0 {
		t.Fatalf("air dash JumpTimes = %d, want 0", v)
	}
	// 已耗尽后的重复上报 → 静默。
	player.mirrorJumpTimes(conn, 8, 32.0)
	player.mirrorJumpTimes(conn, 7, 31.0)
	if got := len(conn.Frames()); got != 2 {
		t.Fatalf("post-consume reports emitted frames, want silence (got %d)", got)
	}
	// 落地（state5）→ 静默复位。
	player.mirrorJumpTimes(conn, 5, 30.5)
	if got := len(conn.Frames()); got != 2 {
		t.Fatalf("landing emitted frames, want silence (got %d)", got)
	}
	// 再次起跳 → 再授 1（官服每次飞行都重授——本修复核心）。
	player.mirrorJumpTimes(conn, 6, 30.9)
	if got := len(conn.Frames()); got != 3 {
		t.Fatalf("second takeoff emitted %d frames, want 1", got)
	}
	if v, _ := frameValue(2); v != 1 {
		t.Fatalf("second takeoff JumpTimes = %d, want 1", v)
	}
}
