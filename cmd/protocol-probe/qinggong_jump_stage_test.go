package main

import (
	"strings"
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/qinggong"
)

// newJumpStageCatalog 构造跳段限制批次的测试目录：qinggong_4/5 带能力
// 门槛（ExistBufID），qinggong_20 大跳双重引用能力 Buff，qinggong_rush
// 是不进跳段集合的对照（Type 7 冲刺）。
func newJumpStageCatalog(t *testing.T) *qinggong.Catalog {
	t.Helper()
	ini := `
[qinggong_4]
Name=凌空踏虚
Type=4
UseType=0
Consume=14
ExistBufID=buf_DHadd_2rd

[qinggong_5]
Name=扶摇步法
Type=5
UseType=0
Consume=15
ExistBufID=buf_DHadd_3rd

[qinggong_20]
Name=高阶轻功：大跳
Type=20
UseType=1
Consume=25
BufferID=buf_DHadd_2rd
ExistBufID=buf_DHadd_2rd

[qinggong_rush]
Name=雁行功对照
Type=7
UseType=0
Consume=15
`
	catalog, err := qinggong.LoadFromReader(strings.NewReader(ini), "test-qgdefine-jump.ini")
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

// 能力门槛：未激活 qinggong_15 → qinggong_4 被拒且不扣点；激活后免费
// 放行（Consume=14 跳过）并挂 buf_DHadd_2rd 能力 Buff。
func TestQingGongJumpCapabilityGateAndFreeUse(t *testing.T) {
	catalog := newJumpStageCatalog(t)
	player := newPlayerActor("测试", 0)
	link := conn()
	base := player.stateSnapshot().QingGongPoint

	if handled, err := handleQingGongCustom(link, player, catalog, useQingGongMessage("qinggong_4", 1), "test"); err != nil || !handled {
		t.Fatalf("rejected begin handled=%t err=%v", handled, err)
	}
	if got := player.stateSnapshot().QingGongPoint; got != base {
		t.Fatalf("points after capability reject=%d, want %d (no charge)", got, base)
	}

	// 激活 qinggong_15 → 挂 buf_DHadd_2rd（槽 17）。
	if changed, err := activateQingGongRecord(link, player, "qinggong_15"); err != nil || !changed {
		t.Fatalf("activate qinggong_15 changed=%t err=%v", changed, err)
	}
	if !player.hasActiveBuff(qgDHadd2rdBuffSlot, 1578, time.Now()) {
		t.Fatal("capability buff buf_DHadd_2rd must be active after activating qinggong_15")
	}

	// 倒回 500ms 重报闸，让 begin 到达能力/扣费分支。
	player.instantQingGongAt["qinggong_4"] = time.Now().Add(-time.Second)
	if handled, err := handleQingGongCustom(link, player, catalog, useQingGongMessage("qinggong_4", 1), "test"); err != nil || !handled {
		t.Fatalf("capability begin handled=%t err=%v", handled, err)
	}
	if got := player.stateSnapshot().QingGongPoint; got != base {
		t.Fatalf("points after capability begin=%d, want %d (free)", got, base)
	}
}

// 三段跳能力：qinggong_5 需激活 qinggong_17 挂 buf_DHadd_3rd。
func TestQingGongThirdJumpNeedsDHadd3rd(t *testing.T) {
	catalog := newJumpStageCatalog(t)
	player := newPlayerActor("测试", 0)
	link := conn()
	activateQingGongRecord(link, player, "qinggong_15")

	if handled, err := handleQingGongCustom(link, player, catalog, useQingGongMessage("qinggong_5", 1), "test"); err != nil || !handled {
		t.Fatalf("reject handled=%t err=%v", handled, err)
	}
	if got := player.stateSnapshot().QingGongPoint; got != player.stateSnapshot().MaxQingGongPoint+player.stateSnapshot().MaxQingGongPointAdd {
		t.Fatalf("third jump without buf_DHadd_3rd must not charge, point=%d", got)
	}

	activateQingGongRecord(link, player, "qinggong_17")
	if !player.hasActiveBuff(qgDHadd3rdBuffSlot, 1579, time.Now()) {
		t.Fatal("capability buff buf_DHadd_3rd must be active after activating qinggong_17")
	}
	player.instantQingGongAt["qinggong_5"] = time.Now().Add(-time.Second)
	if handled, err := handleQingGongCustom(link, player, catalog, useQingGongMessage("qinggong_5", 1), "test"); err != nil || !handled {
		t.Fatalf("accepted third jump handled=%t err=%v", handled, err)
	}
}

// 跳段闸（playerActor 层）：同一腾空同 id 只放行一次，不同 id 各一次，
// 技能重置只给一段，落地（y 回落基准 1m 内）全量重置。
func TestAirJumpStageExhaustGrantAndLandingReset(t *testing.T) {
	player := newPlayerActor("测试", 0)
	now := time.Now()

	// 着地基准校准。
	player.observePositionY(10.0, now)
	if !player.beginAirJump("qinggong_4", now) {
		t.Fatal("first jump of a new airtime must pass")
	}
	if !player.beginAirJump("qinggong_5", now) {
		t.Fatal("different id in same airtime must pass")
	}
	if player.beginAirJump("qinggong_4", now) {
		t.Fatal("repeated id in same airtime must be rejected")
	}

	// 技能重置：恰好再放行一段。
	player.grantAirJumpReset()
	if !player.beginAirJump("qinggong_4", now) {
		t.Fatal("granted stage must pass exactly once")
	}
	if player.beginAirJump("qinggong_4", now) {
		t.Fatal("stage after grant exhaustion must be rejected")
	}

	// 腾空中 y 未回落不重置。
	player.observePositionY(18.0, now.Add(time.Second))
	if player.beginAirJump("qinggong_4", now.Add(time.Second)) {
		t.Fatal("still airborne must stay exhausted")
	}
	// 回落到基准 1m 内 → 落地重置，新循环可再跳。
	player.observePositionY(10.4, now.Add(2*time.Second))
	if !player.beginAirJump("qinggong_4", now.Add(2*time.Second)) {
		t.Fatal("after landing the stage must reset")
	}
}

// 腾空超时兜底：位移包缺失时 qingGongTick 15s 后强制落地重置。
func TestAirJumpLandingTimeoutFallback(t *testing.T) {
	player := newPlayerActor("测试", 0)
	now := time.Now()
	if !player.beginAirJump("qinggong_4", now) {
		t.Fatal("initial jump must pass")
	}
	player.qingGongTick(now.Add(5 * time.Second))
	if player.beginAirJump("qinggong_4", now.Add(5*time.Second)) {
		t.Fatal("5s without motion must not reset the stage")
	}
	player.qingGongTick(now.Add(16 * time.Second))
	if !player.beginAirJump("qinggong_4", now.Add(16*time.Second)) {
		t.Fatal("15s fallback must reset the airtime")
	}
}

// 端到端：能力齐备后，腾空内重复的二段跳在 216 处理器里被跳段闸拒绝，
// 且免费路径不扣点。
func TestQingGongHandlerRejectsRepeatedStageUse(t *testing.T) {
	catalog := newJumpStageCatalog(t)
	player := newPlayerActor("测试", 0)
	link := conn()
	activateQingGongRecord(link, player, "qinggong_15")
	base := player.stateSnapshot().QingGongPoint

	player.instantQingGongAt["qinggong_4"] = time.Now().Add(-time.Second)
	if handled, err := handleQingGongCustom(link, player, catalog, useQingGongMessage("qinggong_4", 1), "test"); err != nil || !handled {
		t.Fatalf("first begin handled=%t err=%v", handled, err)
	}
	if got := player.stateSnapshot().QingGongPoint; got != base {
		t.Fatalf("capability path must be free, point=%d want %d", got, base)
	}

	// 倒回重报闸让第二次到达跳段闸。
	player.instantQingGongAt["qinggong_4"] = time.Now().Add(-time.Second)
	if handled, err := handleQingGongCustom(link, player, catalog, useQingGongMessage("qinggong_4", 1), "test"); err != nil || !handled {
		t.Fatalf("repeat begin handled=%t err=%v", handled, err)
	}
	if got := player.stateSnapshot().QingGongPoint; got != base {
		t.Fatalf("exhausted stage must not charge, point=%d want %d", got, base)
	}

	// 对照：不进跳段集合的冲刺（Type 7）不受跳段闸影响，仅扣费。
	player.instantQingGongAt["qinggong_rush"] = time.Now().Add(-time.Second)
	if handled, err := handleQingGongCustom(link, player, catalog, useQingGongMessage("qinggong_rush", 1), "test"); err != nil || !handled {
		t.Fatalf("rush begin handled=%t err=%v", handled, err)
	}
	if got := player.stateSnapshot().QingGongPoint; got != base-15 {
		t.Fatalf("rush must still charge 15, point=%d want %d", got, base-15)
	}
}

// qinggong_20 大跳：BufferID 与能力 Buff 共槽——使用路径不得在 end 清
// 掉激活挂上的 buf_DHadd_2rd。
func TestQingGongBigJumpKeepsCapabilityBuffOnEnd(t *testing.T) {
	catalog := newJumpStageCatalog(t)
	player := newPlayerActor("测试", 0)
	link := conn()
	activateQingGongRecord(link, player, "qinggong_15")

	player.instantQingGongAt["qinggong_20"] = time.Now().Add(-time.Second)
	if handled, err := handleQingGongCustom(link, player, catalog, useQingGongMessage("qinggong_20", 1), "test"); err != nil || !handled {
		t.Fatalf("big jump begin handled=%t err=%v", handled, err)
	}
	if handled, err := handleQingGongCustom(link, player, catalog, useQingGongMessage("qinggong_20", 0), "test"); err != nil || !handled {
		t.Fatalf("big jump end handled=%t err=%v", handled, err)
	}
	if !player.hasActiveBuff(qgDHadd2rdBuffSlot, 1578, time.Now()) {
		t.Fatal("big jump end must not clear the activation capability buff")
	}
}
