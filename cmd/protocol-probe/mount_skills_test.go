package main

import (
	"strings"
	"testing"
)

// 骑乘技能受理簇（mount_skills.go）：官服 official_players 抓包方言。
// 表链用 scanINISections 流式内核直喂字符串夹具（绕过资源仓）；簇帧
// 断言走 captureMessageConnection。

const rideSkillChainFixture = `
[riding_chongci]
StaticData=1558
script=SkillNormal

[riding_jiben]
StaticData=25
script=SkillNormal

[1558]
MinVarPropNo=31141

[25]
MinVarPropNo=481

[31141]
Level=1
Flow=439
CoolDownTime=3000

[481]
Level=1
Flow=1216
CoolDownTime=60000

[439]
HitEffect=att_bossclone002_shouji01

[1216]
PrepareAction=ride_endlong
`

func TestWriteRideSkillCastClusterChongci(t *testing.T) {
	conn := &captureMessageConnection{}
	dialect := rideSkillDialect{hitEffect: "att_bossclone002_shouji01", cooldown: 3000, flow: 439}
	if err := writeRideSkillCastCluster(conn, "riding_chongci", dialect); err != nil {
		t.Fatalf("cluster: %v", err)
	}
	frames := conn.Frames()
	if len(frames) != 4 { // 置目标 + msg24 54 + 205 + 清目标
		t.Fatalf("frames=%d, want 4", len(frames))
	}
	if !containsPropertyIndex(string(frames[0]), propCurSkillTarget) {
		t.Fatalf("frame0 missing CurSkillTarget set: %x", frames[0])
	}
	if !strings.Contains(string(frames[1]), "hit") ||
		!strings.Contains(string(frames[1]), "att_bossclone002_shouji01") {
		t.Fatalf("frame1 missing hit/effect: %x", frames[1])
	}
	if !strings.Contains(string(frames[2]), "riding_chongci") {
		t.Fatalf("frame2 missing skill id: %x", frames[2])
	}
	if !containsPropertyIndex(string(frames[3]), propCurSkillTarget) {
		t.Fatalf("frame3 missing CurSkillTarget clear: %x", frames[3])
	}
}

func TestWriteRideSkillCastClusterJiben(t *testing.T) {
	conn := &captureMessageConnection{}
	// jiben flow 1216 无 HitEffect → 官服三验均无 msg24 54 帧。
	dialect := rideSkillDialect{cooldown: 60000, flow: 1216}
	if err := writeRideSkillCastCluster(conn, "riding_jiben", dialect); err != nil {
		t.Fatalf("cluster: %v", err)
	}
	frames := conn.Frames()
	if len(frames) != 3 { // 置目标 + 205 + 清目标
		t.Fatalf("frames=%d, want 3", len(frames))
	}
	if !strings.Contains(string(frames[1]), "riding_jiben") {
		t.Fatalf("frame1 missing skill id: %x", frames[1])
	}
}

func TestHandleSkillCustomRideSkillCast(t *testing.T) {
	session, conn, player := newTestMountSession(t)
	player.restoreMounts([]mountEntry{{ConfigID: "MOUNT_A", PackID: 100}})
	if _, err := handleRideCustom(session, rideCustomMessage(1)); err != nil {
		t.Fatalf("mount: %v", err)
	}
	if !player.ridingMount() {
		t.Fatalf("expected riding")
	}

	// 骑乘中发白名单非 dismount 技能（riding_spur，stub 目录）→ 受理簇
	// 三帧（CurSkillTarget 置 / 205 / 清；spur 不在表链 → 无 54 帧）。
	conn.resetFrames()
	handled, err := handleSkillCustom(conn, player, nil, session, useSkillMessage("riding_spur"), "test")
	if !handled || err != nil {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
	frames := conn.Frames()
	if len(frames) != 3 {
		t.Fatalf("cast frames=%d, want 3", len(frames))
	}
	if !strings.Contains(string(frames[1]), "riding_spur") {
		t.Fatalf("205 frame missing skill id: %x", frames[1])
	}

	// 骑乘中发 dismount 技能 → 转下马（属性组 + 242），不发受理簇。
	conn.resetFrames()
	if _, err = handleSkillCustom(conn, player, nil, session, useSkillMessage("riding_dismount"), "test"); err != nil {
		t.Fatalf("dismount cast: %v", err)
	}
	if player.ridingMount() {
		t.Fatalf("still riding after dismount skill")
	}
	tail := conn.Frames()
	if len(tail) != 2 { // 骑乘态 0x10 + 242
		t.Fatalf("dismount frames=%d, want 2", len(tail))
	}

	// 步行发骑乘技能 → 闸拒绝零帧。
	conn.resetFrames()
	if _, err = handleSkillCustom(conn, player, nil, session, useSkillMessage("riding_spur"), "test"); err != nil {
		t.Fatalf("walk cast: %v", err)
	}
	if frames = conn.Frames(); len(frames) != 0 {
		t.Fatalf("walk-state ride skill frames=%d, want 0", len(frames))
	}
}

func TestHandleSkillCustomRidingStartSummons(t *testing.T) {
	session, conn, player := newTestMountSession(t)
	player.restoreMounts([]mountEntry{{ConfigID: "MOUNT_A", PackID: 100}})

	// 步行按骑术精通（主技能栏召唤技）→ soleOwned 自动选中 + 上马。
	handled, err := handleSkillCustom(conn, player, nil, session, useSkillMessage("riding_start"), "test")
	if !handled || err != nil {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
	if !player.ridingMount() {
		t.Fatalf("expected riding after riding_start")
	}
	frames := conn.Frames()
	if len(frames) != 4 { // 整推×2 + 属性组 + 239
		t.Fatalf("summon frames=%d, want 4", len(frames))
	}
	// 239 双 int [pack, pack]（序基双覆盖）。
	ride := frames[3]
	if ride[0] != 0x1E || ride[3] != 2 {
		t.Fatalf("frame3 not 0x1E int custom: %x", ride)
	}
	if n := strings.Count(string(ride), "\x02\x64\x00\x00\x00"); n != 2 { // tag02 int 100 ×2
		t.Fatalf("239 frame pack int count=%d, want 2: %x", n, ride)
	}

	// 骑乘中再按 → 无操作零帧。
	conn.resetFrames()
	if _, err = handleSkillCustom(conn, player, nil, session, useSkillMessage("riding_start"), "test"); err != nil {
		t.Fatalf("riding-start while mounted: %v", err)
	}
	if frames = conn.Frames(); len(frames) != 0 {
		t.Fatalf("riding-start while mounted frames=%d, want 0", len(frames))
	}
}

func TestRideSkillDialectChain(t *testing.T) {
	// 四表链共用一套夹具小节（riding_* / static / varprop / flow 段名
	// 不冲突），走生产 resolveRideSkillDialects 内核。
	sections, err := scanINISections(strings.NewReader(rideSkillChainFixture))
	if err != nil {
		t.Fatalf("scan fixture: %v", err)
	}
	dialects := resolveRideSkillDialects(sections, sections, sections, sections)
	if got := dialects["riding_chongci"]; got.hitEffect != "att_bossclone002_shouji01" ||
		got.cooldown != 3000 || got.flow != 439 {
		t.Fatalf("chongci dialect=%+v", got)
	}
	if got := dialects["riding_jiben"]; got.hitEffect != "" || got.cooldown != 60000 {
		t.Fatalf("jiben dialect=%+v (want no hit effect)", got)
	}
}
