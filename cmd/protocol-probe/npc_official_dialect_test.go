package main

import (
	"bytes"
	"testing"

	"github.com/local/9yin-go-server/internal/clientdata"
)

// TestOfficialNPCPropertiesFollowActiveTable 换表跟随实证：officialNPCProperties
// 全部按名解析（PlayerProperty 末现），fork 模式落 fork 位（1590/1595/469…），
// official2433 模式自动落官服位（1597/1602/471…，c.pcapng #1777 原号）。
func TestOfficialNPCPropertiesFollowActiveTable(t *testing.T) {
	npc := testNPCSpawn()

	forkProps := officialNPCProperties(npc)
	forkIdx := map[string]uint16{}
	for _, p := range forkProps {
		forkIdx[p.Name] = p.Index
	}
	for name, want := range map[string]uint16{
		"BossLevel": 1590, "IDCard": 1595, "HPRatio": 469,
		"ParryRatio": 478, "HitHPRatio": 482, "State": 183, "Type": 181,
	} {
		if got := forkIdx[name]; got != want {
			t.Fatalf("fork dialect %s=%d, want %d", name, got, want)
		}
	}

	clientdata.SwapPropertyTableForTest(t, "official2433")
	officialProps := officialNPCProperties(npc)
	officialIdx := map[string]uint16{}
	for _, p := range officialProps {
		officialIdx[p.Name] = p.Index
	}
	// 官服 #1777 16-prop 原号（NpcType 不发：fork schema 场景位已携带）。
	for name, want := range map[string]uint16{
		"CollideRadius": 179, "State": 183, "Type": 181, "ActionSpeed": 214,
		"BossLevel": 1597, "IDCard": 1602,
		"BornAction": 1604, "WeaponMode": 1606,
		"HPRatio": 471, "ParryRatio": 480, "HitHPRatio": 484,
	} {
		if got := officialIdx[name]; got != want {
			t.Fatalf("official2433 dialect %s=%d, want %d", name, got, want)
		}
	}

	// 官服模式下 0x0D 帧逐字节可对官服表重解：IDCard 行原号 1602=0x0642。
	msg, err := npcAddObject(2, npc, 0)
	if err != nil {
		t.Fatal(err)
	}
	idcard := append([]byte{0x42, 0x06, 0x19, 0x00, 0x00, 0x00}, []byte("null,null,null,null,null")...)
	if !bytes.Contains(msg, idcard) {
		t.Fatal("official2433 NPC AddObject missing IDCard row at official index 1602")
	}
	// 官服模式下 schema 场景行数随头块换血缩水（OrderedProperties 按首现
	// 解析，fork 场景域名在官服表缺席）——总计数不固定，只断言官服位行在。
}
