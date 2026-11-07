package main

import (
	"strings"
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/qinggong"
)

// newQingGongEconomyCatalog 构造覆盖三类轻功的测试目录：
// 即时型（雁行功形态）、踏波行（持续+水漂 buff）、轻身缓降（持续+清零）。
func newQingGongEconomyCatalog(t *testing.T) *qinggong.Catalog {
	t.Helper()
	ini := `
[qinggong_rush]
Name=雁行功测试
Type=7
UseType=0
Consume=15

[qinggong_drift]
Name=踏波行测试
Type=2
UseType=1
Consume=10
BufferID=buf_qg_drift

[qinggong_glide]
Name=轻身术测试
Type=9
UseType=1
Consume=10
BufferID=buf_qg_light
`
	catalog, err := qinggong.LoadFromReader(strings.NewReader(ini), "test-qgdefine.ini")
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

func useQingGongMessage(id string, flag int32) clientCustomMessage {
	return clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: 216},
		{Type: 6, Text: id},
		{Type: 2, Int32: flag},
	}}
}

// 即时型 500ms 重报闸：引擎逐帧重发的 begin 只扣一次费。
func TestQingGongInstantRefireGuardChargesOnce(t *testing.T) {
	catalog := newQingGongEconomyCatalog(t)
	player := newPlayerActor("测试", 0)
	conn := &captureMessageConnection{}
	base := player.stateSnapshot().QingGongPoint

	start := time.Now()
	if handled, err := handleQingGongCustom(conn, player, catalog, useQingGongMessage("qinggong_rush", 1), "test"); err != nil || !handled {
		t.Fatalf("first begin handled=%t err=%v", handled, err)
	}
	if got := player.stateSnapshot().QingGongPoint; got != base-15 {
		t.Fatalf("points after first begin=%d, want %d", got, base-15)
	}
	// 引擎重报（<500ms）：不重复扣费、不发帧。
	frames := len(conn.Frames())
	for i := 0; i < 5; i++ {
		if handled, err := handleQingGongCustom(conn, player, catalog, useQingGongMessage("qinggong_rush", 1), "test"); err != nil || !handled {
			t.Fatalf("repeat %d handled=%t err=%v", i, handled, err)
		}
	}
	if got := player.stateSnapshot().QingGongPoint; got != base-15 {
		t.Fatalf("points after engine repeats=%d, want %d", got, base-15)
	}
	if got := len(conn.Frames()); got != frames {
		t.Fatalf("frames after repeats=%d, want %d (suppressed)", got, frames)
	}
	// 超过闸窗口后恢复扣费。
	later := start.Add(600 * time.Millisecond)
	player.instantQingGongAt["qinggong_rush"] = start // 固定闸起点便于推进
	if handled, err := handleQingGongCustom(conn, player, catalog, useQingGongMessage("qinggong_rush", 1), "test"); err != nil || !handled {
		t.Fatalf("post-guard begin handled=%t err=%v", handled, err)
	}
	// handleQingGongCustom 内部用 time.Now()；这里只验证闸本身。
	if !player.instantQingGongRechargeable("qinggong_rush", later) {
		t.Fatal("guard must rearm after 500ms")
	}
}

// 持续型租约：begin 扣开局费 + 挂租约，end 摘租约不再扣费。
func TestQingGongSustainedLeaseBeginDrainRelease(t *testing.T) {
	catalog := newQingGongEconomyCatalog(t)
	player := newPlayerActor("测试", 0)
	conn := &captureMessageConnection{}
	base := player.stateSnapshot().QingGongPoint

	if handled, err := handleQingGongCustom(conn, player, catalog, useQingGongMessage("qinggong_glide", 1), "test"); err != nil || !handled {
		t.Fatalf("begin handled=%t err=%v", handled, err)
	}
	if got := player.stateSnapshot().QingGongPoint; got != base-10 {
		t.Fatalf("points after glide begin=%d, want %d", got, base-10)
	}
	if got := player.engagedDriftID(); got != "qinggong_glide" {
		t.Fatalf("lease id=%q, want qinggong_glide", got)
	}

	// 秒耗租约：每拍 -10（qinggong_9 缓降 CCQGP）。
	now := time.Now()
	if !player.qingGongTick(now) {
		t.Fatal("first drain tick must change the point")
	}
	if got := player.stateSnapshot().QingGongPoint; got != base-20 {
		t.Fatalf("points after one drain tick=%d, want %d", got, base-20)
	}

	// 引擎重发 begin（续约）：不重复扣开局费。
	frames := len(conn.Frames())
	if handled, err := handleQingGongCustom(conn, player, catalog, useQingGongMessage("qinggong_glide", 1), "test"); err != nil || !handled {
		t.Fatalf("renewal begin handled=%t err=%v", handled, err)
	}
	if got := player.stateSnapshot().QingGongPoint; got != base-20 {
		t.Fatalf("points after renewal begin=%d, want %d", got, base-20)
	}
	if got := len(conn.Frames()); got != frames {
		t.Fatalf("frames after renewal=%d, want %d", got, frames)
	}

	// end：摘租约，不扣费。
	if handled, err := handleQingGongCustom(conn, player, catalog, useQingGongMessage("qinggong_glide", 0), "test"); err != nil || !handled {
		t.Fatalf("end handled=%t err=%v", handled, err)
	}
	if got := player.engagedDriftID(); got != "" {
		t.Fatalf("lease id after end=%q, want empty", got)
	}
	if got := player.stateSnapshot().QingGongPoint; got != base-20 {
		t.Fatalf("points after end=%d, want %d (no end charge)", got, base-20)
	}
}

// 租约耗尽强摘 + 回复暂停/恢复。
func TestQingGongLeaseExhaustsAndRecoveryResumes(t *testing.T) {
	catalog := newQingGongEconomyCatalog(t)
	player := newPlayerActor("测试", 0)
	now := time.Now()
	player.mu.Lock()
	player.qinggong.Point = 12 // 只够 begin(10) + 一次秒耗(10 付不起→0)
	player.mu.Unlock()

	if handled, err := handleQingGongCustom(conn(), player, catalog, useQingGongMessage("qinggong_glide", 1), "test"); err != nil || !handled {
		t.Fatalf("begin handled=%t err=%v", handled, err)
	}
	if got := player.stateSnapshot().QingGongPoint; got != 2 {
		t.Fatalf("points after begin=%d, want 2", got)
	}
	if !player.qingGongTick(now) {
		t.Fatal("exhausting tick must report change")
	}
	if got := player.stateSnapshot().QingGongPoint; got != 0 {
		t.Fatalf("points after exhaustion=%d, want 0", got)
	}
	if got := player.engagedDriftID(); got != "" {
		t.Fatalf("lease must be force-released on exhaustion, got %q", got)
	}

	// 租约摘除后回复恢复：三拍一分档（0..50 档 +3）。
	if player.qingGongTick(now) || player.qingGongTick(now) {
		t.Fatal("first two ticks only accumulate the 3s window")
	}
	if !player.qingGongTick(now) {
		t.Fatal("third tick must recover")
	}
	if got := player.stateSnapshot().QingGongPoint; got != 3 {
		t.Fatalf("points after tiered recovery=%d, want 3", got)
	}
}

// 点数不足时 begin 被拒（客户端本地物理不受影响，仅点数闸门）。
func TestQingGongBeginRejectedWhenPointsInsufficient(t *testing.T) {
	catalog := newQingGongEconomyCatalog(t)
	player := newPlayerActor("测试", 0)
	player.mu.Lock()
	player.qinggong.Point = 5
	player.mu.Unlock()

	if handled, err := handleQingGongCustom(conn(), player, catalog, useQingGongMessage("qinggong_rush", 1), "test"); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := player.stateSnapshot().QingGongPoint; got != 5 {
		t.Fatalf("points after rejected begin=%d, want 5", got)
	}
	if got := player.engagedDriftID(); got != "" {
		t.Fatalf("rejected begin must not engage a lease, got %q", got)
	}
}

// qingGongPointFrame 是 0x10 单属性帧（QingGongPoint@692）。
func TestQingGongPointFrameShape(t *testing.T) {
	player := newPlayerActor("测试", 0)
	player.mu.Lock()
	player.qinggong.Point = 77
	player.mu.Unlock()
	frame := player.qingGongPointFrame()
	if len(frame) != 18 || frame[0] != 0x10 || frame[1] != 0 {
		t.Fatalf("frame=%x", frame)
	}
	if got := int(frame[12]) | int(frame[13])<<8; got != 692 {
		t.Fatalf("property index=%d, want 692", got)
	}
	if got := int32(frame[14]) | int32(frame[15])<<8 | int32(frame[16])<<16 | int32(frame[17])<<24; got != 77 {
		t.Fatalf("property value=%d, want 77", got)
	}
}

// 秒耗表与 Java 实证值一致（CCQGP：light=10、drift/climb=7、其余 0）。
func TestQingGongDrainPerSecondMatchesJavaEvidence(t *testing.T) {
	cases := map[string]int32{
		"buf_qg_light": 10,
		"buf_qg_drift": 7,
		"buf_qg_climb": 7,
		"buf_unknown":  0,
	}
	for bufferID, want := range cases {
		if got := qingGongDrainPerSecond(qinggong.Definition{ID: "x", BufferID: bufferID}); got != want {
			t.Fatalf("drain(%s)=%d, want %d", bufferID, got, want)
		}
	}
}

func conn() *captureMessageConnection { return &captureMessageConnection{} }
