package main

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/role"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

// TestChaseMoveBroadcastsServerAllDest 复刻 2026-11-15 60380 真机会话的
// 追击断流：服务端 transform 一路推进（0x10 坐标帧证据），但客户端一条
// 0x21 ServerAllDest 都没收到——模型钉在出生点、靠视口 reconcile 的
// 0x10/0x1F 硬坐标帧闪现。本测试走生产可见集路径（refreshVisibility，
// 而非 combatTickForTest 的全量直通）验证追击位移帧确实到达连接。
func TestChaseMoveBroadcastsServerAllDest(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	defer lifecycle.close()
	// 生产視口半径（main.go world.radius = npc-radius 默认 120）。
	lifecycle.radius = 120

	npc := testNPCSpawn()
	npc.resolved.ConfigID = "AttackNPC_chase"
	npc.resolved.ScriptClass = "AttackNpc"
	npc.x, npc.y, npc.z = 10, 0, 0
	catalog := []npcSpawn{npc}
	if err := lifecycle.run.populate(catalog); err != nil {
		t.Fatal(err)
	}
	npcID := forkNPCID(0 + 2)

	player := newPlayerActor("tester", 0)
	lifecycle.attachMember(player, role.Position{X: 0, Y: 0, Z: 0}, roleVisual{})
	// 生产路径：ClientReady 后激活成员（scene_lifecycle clientActivity）。
	lifecycle.run.activateMember(lifecycle.wireID)
	// 生产路径：位移上报刷新可见集（main.go updateMemberPosition）。
	lifecycle.updateMemberPosition(role.Position{X: 0, Y: 0, Z: 0}, catalog)
	if !lifecycle.npcVisible(npcID) {
		t.Fatalf("NPC %d not in visible set after refreshVisibility", npcID)
	}

	before, _ := lifecycle.npcTransform(worldcore.EntityID(npcID))
	now := time.Unix(9000, 0)
	for i := 0; i < 3; i++ {
		lifecycle.run.combatTick(now.Add(time.Duration(i) * time.Second))
	}
	after, _ := lifecycle.npcTransform(worldcore.EntityID(npcID))
	if before.X == after.X && before.Z == after.Z {
		t.Fatalf("NPC did not chase: before=%v after=%v", before, after)
	}

	moves := 0
	for _, frame := range conn.Frames() {
		if len(frame) < 7 || frame[0] != 0x21 {
			continue
		}
		count := binary.LittleEndian.Uint16(frame[1:])
		for k := 0; k < int(count); k++ {
			off := 3 + k*44
			if off+8 > len(frame) {
				break
			}
			if binary.LittleEndian.Uint32(frame[off:]) == npcID {
				moves++
			}
		}
	}
	if moves == 0 {
		t.Fatalf("chase moved transform (%v -> %v) but no 0x21 ServerAllDest reached the client", before, after)
	}
}
