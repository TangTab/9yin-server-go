package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/local/9yin-go-server/internal/clientdata"
)

// TestPlayerPropertyIndicesMatchOfficialTable 把所有玩家侧属性索引常量
// 钉死到官方 0x09 属性表(internal/clientdata/official_property_table.txt,
// 2370 项)的玩家域(同名取末现)位置与线型上。0x09 表按位置绑定,任何
// 常量与表漂移都会让客户端按错位索引解析属性——历史上 MaxSP 撞 208/209
// 曾导致进世界即崩。改常量必须同步改这张表(或反之),本测试是唯一闸门。
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
		{"MaxLevel", propMaxLevel, clientdata.WireInt32},
		{"StaticData", propStaticData, clientdata.WireInt32},
		{"ItemType", propItemType, clientdata.WireInt32},
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
// 121 属性批携带);恢复批/事件批/0x0D 归档都不得再携带,MaxSPAdd(583) 则
// 从不发送。塞错位置会让客户端在进世界时崩溃。
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
	// 登录批:stateProperties 大批 + MaxSP 一次(值=maxPlayerSP)。
	login, err := player.loginVitalUpdate()
	if err != nil {
		t.Fatal(err)
	}
	loginIndices, walkErr := propertyFrameIndices(login)
	if walkErr != nil {
		t.Fatalf("walk login 0x10 frame: %v", walkErr)
	}
	maxSPCount := 0
	for _, index := range loginIndices {
		if index == 580 {
			maxSPCount++
		}
		if index == 583 {
			t.Fatal("MaxSPAdd(583) is never sent on the wire")
		}
	}
	if maxSPCount != 1 {
		t.Fatalf("login batch carries MaxSP %d times, want exactly 1", maxSPCount)
	}
	maxSPNeedle := []byte{byte(580 & 0xff), byte(580 >> 8), byte(maxPlayerSP), 0, 0, 0}
	if !bytes.Contains(login, maxSPNeedle) {
		t.Fatalf("login batch MaxSP slot is not %d: %x", maxPlayerSP, login)
	}
	// SP(467) 随同一大批出现且为满值(登录即满)。
	if !containsIndex(loginIndices, propSP) {
		t.Fatalf("login batch misses SP at index %d", propSP)
	}
	spNeedle := []byte{byte(propSP & 0xff), byte(propSP >> 8), byte(maxPlayerSP), 0, 0, 0}
	if !bytes.Contains(login, spNeedle) {
		t.Fatalf("login batch SP slot is not %d: %x", maxPlayerSP, login)
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
