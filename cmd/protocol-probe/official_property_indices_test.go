package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/local/9yin-go-server/internal/clientdata"
)

// TestPlayerPropertyIndicesMatchOfficialTable 把所有玩家侧属性索引
// 钉死到官方 0x09 属性表的玩家域(同名取末现)名字与线型上。
// 2026-11-10 P2b 起 prop* 常量已改为按名解析的 var（MustPlayerProperty/
// MustSceneProperty，0x09 表方言门控换表自动跟随）——本测试转为校验
// 「所引用表行与语义名/线型一致」：改表行名或线型、或改常量引用名都会
// 在此爆。历史背景：MaxSP 撞 208/209 曾致进世界即崩，位置绑定依旧成立。
func TestPlayerPropertyIndicesMatchOfficialTable(t *testing.T) {
	playerAnchors := []struct {
		name     string
		index    uint16
		wireType clientdata.WireType
	}{
		// player_progress.go:成长/属性段
		{"CurNeiGong", propCurNeiGong, clientdata.WireString},
		{"Faculty", propFaculty, clientdata.WireInt32},
		{"FacultyState", propFacultyState, clientdata.WireInt32},
		{"FacultyStyle", propFacultyStyle, clientdata.WireInt32},
		{"FacultyName", propFacultyName, clientdata.WireString},
		{"CurFillValue", propCurFillValue, clientdata.WireInt32},
		{"TotalFillValue", propTotalFillValue, clientdata.WireInt32},
		{"CurLevel", propCurLevel, clientdata.WireInt32},
		{"FillSpeed", propFillSpeed, clientdata.WireInt32},
		// 历练层（docs/livepoint-conversion.md；修炼界面 5 条历练条数据源）
		{"HoldFaculty", propHoldFaculty, clientdata.WireInt32},
		{"LiveGroove_1", propLiveGroove1, clientdata.WireInt32},
		{"LiveGroove_2", propLiveGroove2, clientdata.WireInt32},
		{"LiveGroove_3", propLiveGroove3, clientdata.WireInt32},
		{"LiveGroove_4", propLiveGroove4, clientdata.WireInt32},
		{"LiveGroove_5", propLiveGroove5, clientdata.WireInt32},
		{"LiveSpeedPower", propLiveSpeedPower, clientdata.WireInt32},
		{"PowerValue", propPowerValue, clientdata.WireInt32},
		{"MaxPowerValue", propMaxPowerValue, clientdata.WireInt32},
		{"SP", propSP, clientdata.WireInt32},
		{"Str", propStr, clientdata.WireInt32},
		{"Sta", propSta, clientdata.WireInt32},
		{"Dex", propDex, clientdata.WireInt32},
		{"Ing", propIng, clientdata.WireInt32},
		{"Spi", propSpi, clientdata.WireInt32},
		{"MeleePower", propMeleePower, clientdata.WireInt32},
		{"MagicPower", propMagicPower, clientdata.WireInt32},
		{"PhyHit", propPhyHit, clientdata.WireInt32},
		{"MagicHit", propMagicHit, clientdata.WireInt32},
		{"Dodge", propDodge, clientdata.WireInt32},
		{"MaxParry", propMaxParry, clientdata.WireInt32},
		{"PhyDef", propPhyDef, clientdata.WireInt32},
		{"StiDef", propStiDef, clientdata.WireInt32},
		{"NegDef", propNegDef, clientdata.WireInt32},
		{"MasDef", propMasDef, clientdata.WireInt32},
		{"JujDef", propJujDef, clientdata.WireInt32},
		{"MinMeleeDamage", propMinMeleeDamage, clientdata.WireInt32},
		{"MaxMeleeDamage", propMaxMeleeDamage, clientdata.WireInt32},
		{"MaxHPAdd", propMaxHPAdd, clientdata.WireInt32},
		{"MaxMPAdd", propMaxMPAdd, clientdata.WireInt32},
		{"MaxSP", propMaxSP, clientdata.WireInt32},
		{"MaxSPAdd", propMaxSPAdd, clientdata.WireInt32},
		{"MaxLevel", propMaxLevel, clientdata.WireInt32},
		{"StaticData", propStaticData, clientdata.WireInt32},
		{"ItemType", propItemType, clientdata.WireInt32},
		{"WuXing", propWuXing, clientdata.WireInt32},
		{"MaxParryAdd", propMaxParryAdd, clientdata.WireInt32},
		{"MinMagicDefAdd", propMinMagicDefAdd, clientdata.WireInt32},
		// player_actor.go:技能/恢复/格挡段
		{"NeigongPKStatus", propNeigongPKStatus, clientdata.WireWord},
		{"Dead", propDead, clientdata.WireByte},
		{"CurSkillID", propCurSkillID, clientdata.WireString},
		{"PauseTime", propPauseTime, clientdata.WireInt32},
		{"HPHeartSpeed", propHPHeartSpeed, clientdata.WireInt32},
		{"HPHeartSpeedAdd", propHPHeartSpeedAdd, clientdata.WireInt32},
		{"MPHeartSpeed", propMPHeartSpeed, clientdata.WireInt32},
		{"MPHeartSpeedAdd", propMPHeartSpeedAdd, clientdata.WireInt32},
		{"HPUpSpeed", propHPUpSpeed, clientdata.WireInt32},
		{"HPUpSpeedAdd", propHPUpSpeedAdd, clientdata.WireInt32},
		{"MPUpSpeed", propMPUpSpeed, clientdata.WireInt32},
		{"MPUpSpeedAdd", propMPUpSpeedAdd, clientdata.WireInt32},
		{"CurSkillTarget", propCurSkillTarget, clientdata.WireObject},
		// 收招清除帧归零通道（exe #2014 六项形态）
		{"CurSkillCanNoLimit", propCurSkillCanNoLimit, clientdata.WireByte},
		{"CantBeSkillLocked", propCantBeSkillLocked, clientdata.WireByte},
		{"CantBeSkillMustLocked", propCantBeSkillMustLocked, clientdata.WireByte},
		{"ModifySkillLockTime", propModifySkillLockTime, clientdata.WireInt32},
		{"CanUse", propSkillCanUse, clientdata.WireByte},
		{"Force", propForce, clientdata.WireString},
		{"NewSchool", propNewSchool, clientdata.WireString},
		{"InParry", propInParry, clientdata.WireInt32},
		{"InManualParry", propInManualParry, clientdata.WireInt32},
		{"ParryValue", propParryValue, clientdata.WireInt32},
		// Buff 快照族
		{"BufferListStr", bufferListPropertyIndex, clientdata.WireString},
		{"BufferInfo1", bufferInfoFirstPropertyIndex, clientdata.WireString},
		// selection.go
		{"AutoSelectObject", propAutoSelectObject, clientdata.WireObject},
		// gm_stats.go:GM 面板域（货币/属性/状态批次）
		{"CapitalType2", propCapitalType2, clientdata.WireInt64},
		{"PowerLevel", propPowerLevel, clientdata.WireInt32},
		{"LevelTitle", propLevelTitle, clientdata.WireString},
		{"CharacterFlag", propCharacterFlag, clientdata.WireInt32},
		{"CharacterValue", propCharacterValue, clientdata.WireInt32},
		{"Justice", propJustice, clientdata.WireInt32},
		{"Evil", propEvil, clientdata.WireInt32},
		{"FreeBase", propFreeBase, clientdata.WireInt32},
		{"PKValue", propPKValue, clientdata.WireInt32},
		{"PKMode", propPKMode, clientdata.WireByte},
		{"Ene", propEne, clientdata.WireInt32},
		{"MaxEne", propMaxEne, clientdata.WireInt32},
		{"ResumeEne", propResumeEne, clientdata.WireInt32},
		{"Sat", propSat, clientdata.WireInt32},
		{"MaxSat", propMaxSat, clientdata.WireInt32},
		{"DrunkLevel", propDrunkLevel, clientdata.WireInt32},
		{"NorFacultyAdd", propNorFacultyAdd, clientdata.WireInt32},
		{"ActDayValue", propActDayValue, clientdata.WireInt32},
		{"ActCostSuiYin", propActCostSuiYin, clientdata.WireInt32},
		{"ActDayValueLast", propActDayValueLast, clientdata.WireInt32},
		{"ActDayValueUsed", propActDayValueUsed, clientdata.WireInt32},
		{"ActDayValueMax", propActDayValueMax, clientdata.WireInt32},
		// gm_stats.go:面板域块（称号/坐骑/兵录/声望；TitleID 归
		// title_system.go 声明，此处一并钉线型）
		{"TitleID", propTitleID, clientdata.WireInt32},
		{"Mount", propMount, clientdata.WireString},
		{"CurRideMount", propCurRideMount, clientdata.WireObject},
		{"PrestigeExp", propPrestigeExp, clientdata.WireInt32},
		{"BingLuType", propBingLuType, clientdata.WireInt32},
		{"MaxBingLuLevel", propMaxBingLuLevel, clientdata.WireInt32},
	}
	for _, anchor := range playerAnchors {
		entry, ok := clientdata.PlayerProperty(anchor.name)
		if !ok {
			t.Errorf("official table has no player property %q", anchor.name)
			continue
		}
		if entry.Index != anchor.index {
			t.Errorf("prop %q index = %d, official player domain = %d", anchor.name, anchor.index, entry.Index)
		}
		if entry.Type != anchor.wireType {
			t.Errorf("prop %q encoded as %s, official table wire type = %s", anchor.name, anchor.wireType, entry.Type)
		}
	}
	// BufferInfo1..24 必须与 bufferInfoSlotCount 连续对齐。
	if entry, _ := clientdata.PlayerProperty("BufferInfo24"); entry.Index != bufferInfoFirstPropertyIndex+bufferInfoSlotCount-1 {
		t.Errorf("BufferInfo24 = %d, want %d", entry.Index, bufferInfoFirstPropertyIndex+bufferInfoSlotCount-1)
	}
}

// TestOfficialDialectMaxSPLonginOnly 官方方言下 MaxSP(580) 只在登录/切图
// 链首个权威 0x10 全量批出现一次(fork 49840 实证:全场 5181 帧仅 #16 一处,
// 121 属性批携带),值 = 基础 80 + 运行内功档位加成折算合一(rageCapWireValue);
// MaxSPAdd(583) 官方从未发送(2026-11-09 docs/official 复核),本服同样
// 不出帧;恢复批/事件批/0x0D 归档都不得再携带二者。塞错位置会让客户端
// 在进世界时崩溃。
func TestOfficialDialectMaxSPLoginOnly(t *testing.T) {
	for _, name := range []string{"MaxSP", "MaxSPAdd"} {
		entry, ok := clientdata.PlayerProperty(name)
		if !ok {
			t.Fatalf("official table has no %q", name)
		}
		if entry.Index != 580 && entry.Index != 583 {
			t.Errorf("%q moved to %d; update this guard", name, entry.Index)
		}
	}
	player := newPlayerActor("本地角色", 0)
	// 登录批:stateProperties 大批 + MaxSP(折算最终值)一次,583 不出帧。
	login, err := player.loginVitalUpdate()
	if err != nil {
		t.Fatal(err)
	}
	loginIndices, walkErr := propertyFrameIndices(login)
	if walkErr != nil {
		t.Fatalf("walk login 0x10 frame: %v", walkErr)
	}
	maxSPCount, maxSPAddCount := 0, 0
	for _, index := range loginIndices {
		if index == 580 {
			maxSPCount++
		}
		if index == 583 {
			maxSPAddCount++
		}
	}
	if maxSPCount != 1 {
		t.Fatalf("login batch carries MaxSP %d times, want exactly 1", maxSPCount)
	}
	if maxSPAddCount != 0 {
		t.Fatalf("login batch carries MaxSPAdd %d times, want 0 (official never sends 583)", maxSPAddCount)
	}
	// MaxSP = 基础 80 + 默认运行内功 ng_wd_001(武当一内 +10) = 90。
	rageCap := player.rageCapWireValue()
	if rageCap != maxPlayerSP+10 {
		t.Fatalf("default wire MaxSP=%d, want %d", rageCap, maxPlayerSP+10)
	}
	maxSPNeedle := []byte{byte(580 & 0xff), byte(580 >> 8), byte(rageCap), 0, 0, 0}
	if !bytes.Contains(login, maxSPNeedle) {
		t.Fatalf("login batch MaxSP slot is not %d: %x", rageCap, login)
	}
	// SP(467) 随同一大批出现且为满值(登录即满 = 有效上限)。
	if !containsIndex(loginIndices, propSP) {
		t.Fatalf("login batch misses SP at index %d", propSP)
	}
	spNeedle := []byte{byte(propSP & 0xff), byte(propSP >> 8), byte(rageCap), 0, 0, 0}
	if !bytes.Contains(login, spNeedle) {
		t.Fatalf("login batch SP slot is not %d: %x", rageCap, login)
	}
	// 常规批(恢复/事件)不得携带 MaxSP/MaxSPAdd。
	regular, err := player.vitalUpdate()
	if err != nil {
		t.Fatal(err)
	}
	regularIndices, walkErr := propertyFrameIndices(regular)
	if walkErr != nil {
		t.Fatalf("walk regular 0x10 frame: %v", walkErr)
	}
	for _, index := range regularIndices {
		if index == 580 || index == 583 {
			t.Fatalf("regular vitals batch carries forbidden index %d (MaxSP/MaxSPAdd)", index)
		}
	}
}

func containsIndex(indices []uint16, want uint16) bool {
	for _, index := range indices {
		if index == want {
			return true
		}
	}
	return false
}

// propertyFrameIndices 按官方 0x09 表线型走帧,返回 0x10 属性帧的全部
// 属性索引。用于测试断言;不合法帧返回错误。
func propertyFrameIndices(frame []byte) ([]uint16, error) {
	if len(frame) < 12 || frame[0] != 0x10 {
		return nil, fmt.Errorf("not a 0x10 property frame")
	}
	count := int(binary.LittleEndian.Uint16(frame[10:12]))
	offset := 12
	indices := make([]uint16, 0, count)
	for i := 0; i < count; i++ {
		if offset+2 > len(frame) {
			return nil, fmt.Errorf("entry %d: truncated index", i)
		}
		index := binary.LittleEndian.Uint16(frame[offset:])
		offset += 2
		entry, ok := clientdata.PropertyByIndex(int(index))
		if !ok || entry.Type == 0 {
			return nil, fmt.Errorf("entry %d: index %d not in official table", i, index)
		}
		switch entry.Type {
		case clientdata.WireByte:
			offset += 1
		case clientdata.WireWord:
			offset += 2
		case clientdata.WireInt32, clientdata.WireFloat32:
			offset += 4
		case clientdata.WireInt64, clientdata.WireObject:
			offset += 8
		case clientdata.WireString, clientdata.WireWideString:
			if offset+4 > len(frame) {
				return nil, fmt.Errorf("entry %d: truncated string length", i)
			}
			length := int(binary.LittleEndian.Uint32(frame[offset:]))
			offset += 4 + length
		}
		if offset > len(frame) {
			return nil, fmt.Errorf("entry %d: value overruns frame", i)
		}
		indices = append(indices, index)
	}
	return indices, nil
}
