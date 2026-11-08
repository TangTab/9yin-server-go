package main

// 命中表现帧族字节级断言。规格来源（双源一致）：
//   - F 盘 native-menu.exe 反汇编（writeSkillHitEvents/fightRelationDamageFrame/
//     writeNPCAttackFrames，报告 nm-combat/combat-data-distribution.md）
//   - Java 仓抓包文档 docs/combat-skill-211.md §4（capture 127-0-0-1-62433）

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"
)

func TestFightRelationDamageFrameLayout(t *testing.T) {
	frame := fightRelationDamageFrame(1, 0x7A5C0B, 344)
	if len(frame) != 21 {
		t.Fatalf("fight relation frame length = %d, want 21", len(frame))
	}
	if frame[0] != 0x13 {
		t.Fatalf("opcode = %#x, want 0x13", frame[0])
	}
	if frame[1] != 0 {
		t.Fatalf("flag byte = %#x, want 0", frame[1])
	}
	if got := binary.LittleEndian.Uint32(frame[2:]); got != uint32(1) {
		t.Fatalf("object = %#x, want 1", got)
	}
	if got := binary.LittleEndian.Uint32(frame[6:]); got != 0x7A5C0B {
		t.Fatalf("owner = %#x, want 0x7A5C0B", got)
	}
	if got := binary.LittleEndian.Uint16(frame[10:]); got != recordFightRelation {
		t.Fatalf("record index = %d, want %d (fight_relation_rec)", got, recordFightRelation)
	}
	if got := binary.LittleEndian.Uint16(frame[12:]); got != 1 {
		t.Fatalf("row = %d, want 1", got)
	}
	if got := binary.LittleEndian.Uint16(frame[14:]); got != 1 {
		t.Fatalf("col = %d, want 1", got)
	}
	if frame[16] != 0 {
		t.Fatalf("pad byte = %#x, want 0", frame[16])
	}
	if got := binary.LittleEndian.Uint32(frame[17:]); got != 344 {
		t.Fatalf("cumulative damage = %d, want 344", got)
	}
}

func TestHurtEffectFrameLayout(t *testing.T) {
	frame := hurtEffectFrame(7, 1, 87)
	if len(frame) != 25 {
		t.Fatalf("hurt effect frame length = %d, want 25", len(frame))
	}
	if frame[0] != 0x2D {
		t.Fatalf("opcode = %#x, want 0x2D", frame[0])
	}
	if frame[1] != 1 || frame[2] != 0 {
		t.Fatalf("leading bytes = %d/%d, want 1/0", frame[1], frame[2])
	}
	if got := binary.LittleEndian.Uint32(frame[3:]); got != 7 {
		t.Fatalf("victim = %d, want 7", got)
	}
	if got := binary.LittleEndian.Uint32(frame[7:]); got != 1 {
		t.Fatalf("owner = %d, want 1", got)
	}
	if got := binary.LittleEndian.Uint16(frame[11:]); got != 2 {
		t.Fatalf("constant @11 = %d, want 2", got)
	}
	if got := binary.LittleEndian.Uint16(frame[13:]); got != 28 {
		t.Fatalf("constant @13 = %d, want 28", got)
	}
	if got := binary.LittleEndian.Uint32(frame[15:]); got != 87 {
		t.Fatalf("hp ratio @15 = %d, want 87", got)
	}
	if got := binary.LittleEndian.Uint16(frame[19:]); got != 112 {
		t.Fatalf("constant @19 = %d, want 112", got)
	}
	if got := binary.LittleEndian.Uint32(frame[21:]); got != 87 {
		t.Fatalf("hp ratio @21 = %d, want 87 (出现两次)", got)
	}
}

// decodeCustomValuesFrame 把 0x1E 数值 custom 帧解回 clientCustomValue
// 序列（复用 C2S 解析器），用于断言 msg24/18/205 的值形态。
func decodeCustomValuesFrame(t *testing.T, frame []byte) []clientCustomValue {
	t.Helper()
	if frame[0] != 0x1E {
		t.Fatalf("outer opcode = %#x, want 0x1E", frame[0])
	}
	message, err := parseClientCustomMessage(frame)
	if err != nil {
		t.Fatalf("parse custom values: %v", err)
	}
	return message.Values
}

func TestSkillHitDamageFrameElevenValues(t *testing.T) {
	frame, err := skillHitDamageFrame(forkNPCID(7), 1, 143, "sfgp01", "SkillLock", "monv030101", "测试侠客")
	if err != nil {
		t.Fatalf("encode skill hit frame: %v", err)
	}
	values := decodeCustomValuesFrame(t, frame)
	if len(values) != 11 {
		t.Fatalf("value count = %d, want 11", len(values))
	}
	wantInts := map[int]int32{0: 24, 2: 17, 3: 143, 4: 0, 5: 0, 10: 0}
	for index, want := range wantInts {
		if values[index].Type != 2 || values[index].Int32 != want {
			t.Fatalf("value[%d] = type %d int %d, want int %d", index, values[index].Type, values[index].Int32, want)
		}
	}
	if values[1].Type != 8 || values[1].Raw != packTestObject(forkNPCID(7), 1) {
		t.Fatalf("value[1] victim object mismatch: %+v", values[1])
	}
	if values[6].Text != "sfgp01" || values[7].Text != "SkillLock" || values[8].Text != "monv030101" {
		t.Fatalf("narrow strings = %q/%q/%q, want sfgp01/SkillLock/monv030101", values[6].Text, values[7].Text, values[8].Text)
	}
	if values[9].Type != 7 || values[9].Text != "测试侠客" {
		t.Fatalf("attacker wide name = type %d %q, want wide 测试侠客", values[9].Type, values[9].Text)
	}
}

func TestNPCAttackHitFrameNineValues(t *testing.T) {
	frame, err := npcAttackHitFrame(playerObjectID, playerOwnerID, 1, forkNPCID(7), 1, "monv030101", "受害者", "野怪")
	if err != nil {
		t.Fatalf("encode NPC attack hit frame: %v", err)
	}
	values := decodeCustomValuesFrame(t, frame)
	if len(values) != 9 {
		t.Fatalf("value count = %d, want 9", len(values))
	}
	wantInts := map[int]int32{0: 24, 2: 7, 3: 1, 8: 20}
	for index, want := range wantInts {
		if values[index].Type != 2 || values[index].Int32 != want {
			t.Fatalf("value[%d] = type %d int %d, want int %d", index, values[index].Type, values[index].Int32, want)
		}
	}
	if values[1].Type != 8 || values[1].Raw != packTestObject(playerObjectID, playerOwnerID) {
		t.Fatalf("value[1] player object mismatch: %+v", values[1])
	}
	if values[4].Type != 8 || values[4].Raw != packTestObject(forkNPCID(7), 1) {
		t.Fatalf("value[4] NPC object mismatch: %+v", values[4])
	}
	if values[5].Text != "monv030101" {
		t.Fatalf("NPC model = %q, want monv030101", values[5].Text)
	}
	if values[6].Type != 7 || values[6].Text != "受害者" || values[7].Type != 7 || values[7].Text != "野怪" {
		t.Fatalf("wide names = %q/%q, want 受害者/野怪", values[6].Text, values[7].Text)
	}
}

func TestSkillHurtAndConfirmFrames(t *testing.T) {
	hurt, err := skillHurtActionFrame(forkNPCID(7), 1, playerObjectID, playerOwnerID)
	if err != nil {
		t.Fatalf("encode hurt frame: %v", err)
	}
	values := decodeCustomValuesFrame(t, hurt)
	if len(values) != 5 || values[0].Int32 != 18 || values[2].Text != "hurt_half" || values[3].Int32 != 0 {
		t.Fatalf("msg18 shape mismatch: %+v", values)
	}
	if values[1].Raw != packTestObject(forkNPCID(7), 1) || values[4].Raw != packTestObject(playerObjectID, playerOwnerID) {
		t.Fatalf("msg18 objects mismatch: %+v", values)
	}

	confirm, err := skillHitConfirmFrame(playerObjectID, playerOwnerID, "sfgp01", forkNPCID(7), 1)
	if err != nil {
		t.Fatalf("encode confirm frame: %v", err)
	}
	values = decodeCustomValuesFrame(t, confirm)
	if len(values) != 4 || values[0].Int32 != 205 || values[2].Text != "sfgp01" {
		t.Fatalf("msg205 shape mismatch: %+v", values)
	}
	if values[1].Raw != packTestObject(playerObjectID, playerOwnerID) || values[3].Raw != packTestObject(forkNPCID(7), 1) {
		t.Fatalf("msg205 objects mismatch: %+v", values)
	}
}

func TestNPCCastPreparationFrameShape(t *testing.T) {
	frame, err := npcCastPreparationFrame(forkNPCID(7), 1, 1.5, 2, 3, 0.25)
	if err != nil {
		t.Fatalf("encode NPC cast preparation: %v", err)
	}
	values := decodeCustomValuesFrame(t, frame)
	if len(values) != 7 || values[0].Int32 != 337 {
		t.Fatalf("msg337 shape mismatch: %+v", values)
	}
	if values[1].Type != 8 || values[1].Raw != packTestObject(forkNPCID(7), 1) {
		t.Fatalf("msg337 object mismatch: %+v", values[1])
	}
	if values[2].Float32 != 1.5 || values[3].Float32 != 2 || values[4].Float32 != 3 || values[5].Float32 != 0.25 {
		t.Fatalf("msg337 coordinates = %v/%v/%v orient %v", values[2].Float32, values[3].Float32, values[4].Float32, values[5].Float32)
	}
	if values[6].Int32 != 0 {
		t.Fatalf("msg337 tail = %d, want 0", values[6].Int32)
	}
}

func TestSkillCooldownFramesDualRows(t *testing.T) {
	definition := combatSkillDefinition{
		id:               "sfgp01",
		cooldownCategory: 8837,
		cooldownTeam:     50369,
		personalCD:       8 * time.Second,
		publicCD:         7 * time.Second,
	}
	now := time.Now()
	personal, public, err := skillCooldownFrames(definition, now)
	if err != nil {
		t.Fatalf("encode cooldown frames: %v", err)
	}
	start := now.UnixMilli()
	assertCooldownRow := func(frame []byte, wantCat uint32, wantTeam uint16, wantEnd int64, label string) {
		t.Helper()
		if frame[0] != 0x11 {
			t.Fatalf("%s opcode = %#x, want 0x11", label, frame[0])
		}
		if got := binary.LittleEndian.Uint16(frame[10:]); got != recordCooldown {
			t.Fatalf("%s record = %d, want %d", label, got, recordCooldown)
		}
		if len(frame) != 38 {
			t.Fatalf("%s length = %d, want 38", label, len(frame))
		}
		if got := binary.LittleEndian.Uint32(frame[16:]); got != wantCat {
			t.Fatalf("%s category = %d, want %d", label, got, wantCat)
		}
		if got := binary.LittleEndian.Uint64(frame[20:]); got != uint64(start) {
			t.Fatalf("%s start = %d, want %d", label, got, start)
		}
		if got := binary.LittleEndian.Uint64(frame[28:]); got != uint64(wantEnd) {
			t.Fatalf("%s end = %d, want %d", label, got, wantEnd)
		}
		if got := binary.LittleEndian.Uint16(frame[36:]); got != wantTeam {
			t.Fatalf("%s team = %d, want %d", label, got, wantTeam)
		}
	}
	assertCooldownRow(personal, 8837, 0xFFFF, start+8000, "personal row")
	assertCooldownRow(public, 50000, 50369, start+7000, "GCD row")
}

// TestSkillCooldownFramesOmitPublicRowWithoutTeamConfig 无我剑诀
// （CS_jh_dgjj，skill_static 无 CoolDownTeam、skill_new/varprop 无
// AddCoolDownTime）只发个人行：官方从不发 team=0 的 50000 公共行——
// 客户端把 team=0 当全组公共 CD，一招出来整条技能栏转 personalCD
// （2026-11-08 修正：CD 严格走配置，publicCD 不再回落 personalCD）。
func TestSkillCooldownFramesOmitPublicRowWithoutTeamConfig(t *testing.T) {
	definition := combatSkillDefinition{
		id:               "CS_jh_dgjj04",
		cooldownCategory: 9054,
		cooldownTeam:     0,
		personalCD:       20 * time.Second,
		publicCD:         0,
	}
	now := time.Now()
	personal, public, err := skillCooldownFrames(definition, now)
	if err != nil {
		t.Fatalf("encode cooldown frames: %v", err)
	}
	if len(personal) != 38 || personal[0] != 0x11 {
		t.Fatalf("personal row = %x", personal)
	}
	if got := binary.LittleEndian.Uint32(personal[16:]); got != 9054 {
		t.Fatalf("personal category = %d, want 9054 (dgjj04 横断明枪)", got)
	}
	if end := int64(binary.LittleEndian.Uint64(personal[28:])); end-now.UnixMilli() != 20000 {
		t.Fatalf("personal duration = %d ms, want 20000 (官方 View40 同值)", end-now.UnixMilli())
	}
	if got := binary.LittleEndian.Uint16(personal[36:]); got != 0xFFFF {
		t.Fatalf("personal team = %#x, want 0xFFFF", got)
	}
	if public != nil {
		t.Fatalf("team-less set must not emit public GCD row: %x", public)
	}
	// 目录真值钉子（夹具含 dgjj 时）：无 team 套路编译后 publicCD 必须
	// 为 0（不回落 personalCD）。
	if definition, ok := combatSkills["CS_jh_dgjj04"]; ok {
		if definition.cooldownTeam != 0 || definition.publicCD != 0 {
			t.Fatalf("dgjj04 catalog cooldownTeam=%d publicCD=%s, want 0/0 (配置无公共CD)",
				definition.cooldownTeam, definition.publicCD)
		}
		if definition.personalCD != 20*time.Second {
			t.Fatalf("dgjj04 personalCD=%s, want 20s (官方 View40 同值)", definition.personalCD)
		}
	}
}

func TestActiveSkillRecordRowRoundTrip(t *testing.T) {
	row := serverRecordAddString(playerObjectID, playerOwnerID, recordActiveSkill, "sfgp01")
	if row[0] != 0x11 {
		t.Fatalf("active skill row opcode = %#x, want 0x11", row[0])
	}
	if got := binary.LittleEndian.Uint16(row[10:]); got != recordActiveSkill {
		t.Fatalf("active skill record = %d, want %d", got, recordActiveSkill)
	}
	if !bytes.Contains(row, append([]byte("sfgp01"), 0)) {
		t.Fatalf("active skill row missing NUL-terminated skill id: % x", row)
	}
	del := serverRecordDelRow(playerObjectID, playerOwnerID, recordActiveSkill, 0)
	if del[0] != 0x12 {
		t.Fatalf("delete row opcode = %#x, want 0x12", del[0])
	}
	if got := binary.LittleEndian.Uint16(del[10:]); got != recordActiveSkill {
		t.Fatalf("delete record = %d, want %d", got, recordActiveSkill)
	}
}
