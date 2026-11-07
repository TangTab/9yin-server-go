package main

// qinggong_9 轻身缓降（buf_qg_light）链路测试：msg169 九值挂/摘帧 +
// 槽 19 租约。对照 Java MeridianAndQingGongTest（挂-扣-摘-耗尽强摘）。

import (
	"testing"
	"time"
)

// glideExpiryTolerance 是 msg169 v7（租约到期毫秒）相对 now+10min 的
// 编码/断言容差。
const glideExpiryTolerance = 2 * time.Second

// decodeGlideTimedBuff 解出 msg169 缓降帧并校验公共骨架
// （buff 串/双 obj/双宽名/Category），返回 addRemove 与到期毫秒。
func decodeGlideTimedBuff(t *testing.T, frame []byte) (addRemove int32, endEpochMs int64) {
	t.Helper()
	values := decodeCustomValuesFrame(t, frame)
	if len(values) != 9 || values[0].Type != 2 || values[0].Int32 != 169 {
		t.Fatalf("msg169 nine-value shape mismatch: %d values, v0=%+v", len(values), values[0])
	}
	if values[1].Type != 2 {
		t.Fatalf("addRemove type = %d, want int32", values[1].Type)
	}
	if values[2].Type != 6 || values[2].Text != qingGongGlideBuffID {
		t.Fatalf("buff id = %+v, want narrow %q", values[2], qingGongGlideBuffID)
	}
	if values[3].Type != 8 || values[4].Type != 8 || values[3].Raw != values[4].Raw {
		t.Fatalf("bearer objects = %+v/%+v, want same obj pair", values[3], values[4])
	}
	if values[5].Type != 7 || values[6].Type != 7 || values[5].Text == "" {
		t.Fatalf("wide names = %+v/%+v, want non-empty pair", values[5], values[6])
	}
	if values[7].Type != 3 {
		t.Fatalf("expiry type = %d, want int64(3)", values[7].Type)
	}
	if values[8].Type != 2 || values[8].Int32 != glideBuffCategory {
		t.Fatalf("category = %+v, want %d", values[8], glideBuffCategory)
	}
	return values[1].Int32, values[7].Int64
}

// begin：msg169(挂) 先于 vitalUpdate 槽帧；v7=now+10min、槽 19 挂 218。
func TestGlideEngageSendsTimedBuffThenSlotLease(t *testing.T) {
	catalog := newQingGongEconomyCatalog(t)
	player := newPlayerActor("测试侠客", 0)
	conn := &captureMessageConnection{}
	now := time.Now()

	if handled, err := handleQingGongCustom(conn, player, catalog, useQingGongMessage("qinggong_glide", 1), "test"); err != nil || !handled {
		t.Fatalf("begin handled=%t err=%v", handled, err)
	}
	frames := conn.Frames()
	if len(frames) != 2 {
		t.Fatalf("frames after begin = %d, want 2 (msg169 + vitalUpdate)", len(frames))
	}
	addRemove, endMs := decodeGlideTimedBuff(t, frames[0])
	if addRemove != 0 {
		t.Fatalf("engage addRemove = %d, want 0", addRemove)
	}
	if endMs < now.Add(10*time.Minute-glideExpiryTolerance).UnixMilli() || endMs > now.Add(10*time.Minute+glideExpiryTolerance).UnixMilli() {
		t.Fatalf("engage expiry = %d, want ~now+10min", endMs)
	}
	if frames[1][0] != 0x10 {
		t.Fatalf("second frame opcode = %#x, want 0x10 slot update", frames[1][0])
	}
	if !player.hasActiveBuff(qgLightBuffSlot, glideBuffStaticData, now) {
		t.Fatal("begin must stage buf_qg_light lease on slot 19")
	}
	if expires, ok := player.buffExpiry(qgLightBuffSlot, glideBuffStaticData, now); !ok || expires.UnixMilli() != endMs {
		t.Fatalf("slot lease expiry %v ok=%t must equal msg169 v7 %d", expires, ok, endMs)
	}
}

// end：msg169(摘) v7 取租约原到期 + 空槽帧；秒耗租约与槽一并清空。
func TestGlideReleaseKeepsLeaseExpiryAndClearsSlot(t *testing.T) {
	catalog := newQingGongEconomyCatalog(t)
	player := newPlayerActor("测试侠客", 0)
	beginConn := &captureMessageConnection{}
	now := time.Now()

	if handled, err := handleQingGongCustom(beginConn, player, catalog, useQingGongMessage("qinggong_glide", 1), "test"); err != nil || !handled {
		t.Fatalf("begin handled=%t err=%v", handled, err)
	}
	_, engageEnd := decodeGlideTimedBuff(t, beginConn.Frames()[0])
	endConn := &captureMessageConnection{}
	if handled, err := handleQingGongCustom(endConn, player, catalog, useQingGongMessage("qinggong_glide", 0), "test"); err != nil || !handled {
		t.Fatalf("end handled=%t err=%v", handled, err)
	}
	frames := endConn.Frames()
	if len(frames) != 2 {
		t.Fatalf("frames after end = %d, want 2 (msg169 + vitalUpdate)", len(frames))
	}
	addRemove, endMs := decodeGlideTimedBuff(t, frames[0])
	if addRemove != 1 {
		t.Fatalf("release addRemove = %d, want 1", addRemove)
	}
	if endMs != engageEnd {
		t.Fatalf("release expiry = %d, want lease end %d", endMs, engageEnd)
	}
	if frames[1][0] != 0x10 {
		t.Fatalf("second frame opcode = %#x, want 0x10 slot clear", frames[1][0])
	}
	if player.hasActiveBuff(qgLightBuffSlot, glideBuffStaticData, now) {
		t.Fatal("end must clear the glide slot lease")
	}
	if got := player.engagedDriftID(); got != "" {
		t.Fatalf("lease id after end = %q, want empty", got)
	}
}

// 游离 end（从未 begin）：仍回一帧 msg169(摘) 客户端重同步，无槽帧。
func TestGlideFreeReleaseStillSendsRemoval(t *testing.T) {
	catalog := newQingGongEconomyCatalog(t)
	player := newPlayerActor("测试侠客", 0)
	conn := &captureMessageConnection{}
	before := time.Now().UnixMilli()

	if handled, err := handleQingGongCustom(conn, player, catalog, useQingGongMessage("qinggong_glide", 0), "test"); err != nil || !handled {
		t.Fatalf("free end handled=%t err=%v", handled, err)
	}
	frames := conn.Frames()
	if len(frames) != 1 {
		t.Fatalf("frames after free end = %d, want 1 (msg169 only)", len(frames))
	}
	addRemove, endMs := decodeGlideTimedBuff(t, frames[0])
	if addRemove != 1 {
		t.Fatalf("free release addRemove = %d, want 1", addRemove)
	}
	if endMs < before {
		t.Fatalf("free release expiry = %d, want >= call time %d", endMs, before)
	}
}

// 耗尽强摘：ticker 回传 id 后主循环先摘帧（v7=原到期）再清槽——与
// main.go 连接循环同序（writeGlideBuffRemovalFor 先于 endQingGongBuff）。
func TestGlideExhaustionRemovalBeforeSlotClear(t *testing.T) {
	catalog := newQingGongEconomyCatalog(t)
	player := newPlayerActor("测试侠客", 0)
	beginConn := &captureMessageConnection{}
	player.mu.Lock()
	player.qinggong.Point = 12 // begin(10) + 一次秒耗付不起
	player.mu.Unlock()

	if handled, err := handleQingGongCustom(beginConn, player, catalog, useQingGongMessage("qinggong_glide", 1), "test"); err != nil || !handled {
		t.Fatalf("begin handled=%t err=%v", handled, err)
	}
	_, engageEnd := decodeGlideTimedBuff(t, beginConn.Frames()[0])
	now := time.Now()
	if changed, forceEnded := player.qingGongTick(now); !changed || forceEnded != "qinggong_glide" {
		t.Fatalf("exhausting tick changed=%t forceEnded=%q", changed, forceEnded)
	}
	definition, exists := catalog.Lookup("qinggong_glide")
	if !exists {
		t.Fatal("catalog must know qinggong_glide")
	}
	endConn := &captureMessageConnection{}
	writeGlideBuffRemovalFor(endConn, player, definition, now)
	if err := endQingGongBuff(endConn, player, definition); err != nil {
		t.Fatalf("endQingGongBuff: %v", err)
	}
	frames := endConn.Frames()
	if len(frames) != 2 {
		t.Fatalf("frames after forced release = %d, want 2", len(frames))
	}
	addRemove, endMs := decodeGlideTimedBuff(t, frames[0])
	if addRemove != 1 || endMs != engageEnd {
		t.Fatalf("forced release addRemove=%d expiry=%d, want 1/%d", addRemove, endMs, engageEnd)
	}
	if player.hasActiveBuff(qgLightBuffSlot, glideBuffStaticData, now) {
		t.Fatal("forced release must clear the glide slot lease")
	}
}

// 持续型漂移族（踏波行）不发 msg169 缓降帧——只有 buf_qg_light 走
// 九值时间形态（52795 抓包口径）。
func TestGlideTimedBuffNotSentForOtherFamilies(t *testing.T) {
	catalog := newQingGongEconomyCatalog(t)
	player := newPlayerActor("测试侠客", 0)
	conn := &captureMessageConnection{}

	if handled, err := handleQingGongCustom(conn, player, catalog, useQingGongMessage("qinggong_drift", 1), "test"); err != nil || !handled {
		t.Fatalf("begin handled=%t err=%v", handled, err)
	}
	for _, frame := range conn.Frames() {
		if frame[0] == 0x1E {
			t.Fatalf("drift begin must not emit msg169 timed buff: %x", frame[:8])
		}
	}
}
