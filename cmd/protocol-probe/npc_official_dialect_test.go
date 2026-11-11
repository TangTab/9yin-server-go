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
	// 本测试自带双方言断言：fork 段先显式切回 fork（进程可能以
	// NINEYIN_PROPERTY_TABLE=official2433 启动做官服模式冒烟）。
	clientdata.SwapPropertyTableForTest(t, "fork")
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

// TestOfficialNPCPropertiesBoxGatherCollideGate 宝箱/采集族碰撞方言
// （npc.pcapng 2026-11-11 官服 0x0D 定版）：NpcType=161/153 不发
// CollideRadius，改发尾行 OpenRange=2.0/GatherRange=3.0；ConfigID 前缀
// 兜底判族；常规 NPC 保留 collide 首行且 schema 真值直通。
func TestOfficialNPCPropertiesBoxGatherCollideGate(t *testing.T) {
	clientdata.SwapPropertyTableForTest(t, "fork")
	propByName := func(props []clientdata.IndexedProperty, name string) (clientdata.IndexedProperty, bool) {
		for _, p := range props {
			if p.Name == name {
				return p, true
			}
		}
		return clientdata.IndexedProperty{}, false
	}

	box := testNPCSpawn()
	box.resolved.Properties["NpcType"] = clientdata.PropertyValue{Value: clientdata.Int32Value(161)}
	box.resolved.Properties["ConfigID"] = clientdata.PropertyValue{Value: clientdata.StringValue("BoxNpc10014")}
	props := officialNPCProperties(box)
	if _, ok := propByName(props, "CollideRadius"); ok {
		t.Fatal("宝箱族（NpcType=161）不应下发 CollideRadius")
	}
	if p, ok := propByName(props, "OpenRange"); !ok || p.Value.F32 != 2.0 {
		t.Fatalf("宝箱族应尾行 OpenRange=2.0, got ok=%v v=%v", ok, p.Value.F32)
	}

	// 采集族走 ConfigID 前缀兜底（schema 无 NpcType 列的表）。
	gather := testNPCSpawn()
	delete(gather.resolved.Properties, "NpcType")
	gather.resolved.Properties["ConfigID"] = clientdata.PropertyValue{Value: clientdata.StringValue("GatherClone003001")}
	props = officialNPCProperties(gather)
	if _, ok := propByName(props, "CollideRadius"); ok {
		t.Fatal("采集族（GatherClone*）不应下发 CollideRadius")
	}
	if p, ok := propByName(props, "GatherRange"); !ok || p.Value.F32 != 3.0 {
		t.Fatalf("采集族应尾行 GatherRange=3.0, got ok=%v v=%v", ok, p.Value.F32)
	}

	// 常规 NPC：collide 保持首行且读 schema 真值（官服 0.5 档）。
	normal := testNPCSpawn()
	normal.resolved.Properties["CollideRadius"] = clientdata.PropertyValue{Value: clientdata.Float32Value(0.5)}
	props = officialNPCProperties(normal)
	if len(props) == 0 || props[0].Name != "CollideRadius" {
		t.Fatal("常规 NPC 的 CollideRadius 应保持官服位块首行")
	}
	if props[0].Value.F32 != 0.5 {
		t.Fatalf("CollideRadius schema 真值未直通: %v", props[0].Value.F32)
	}
}
