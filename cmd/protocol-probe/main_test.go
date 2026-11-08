package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf16"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/luascript"
	"github.com/local/9yin-go-server/internal/sim"
	"github.com/local/9yin-go-server/internal/taskdata"
	"github.com/local/9yin-go-server/internal/transport"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

func TestStarterNPCUsesClientResourceIdentity(t *testing.T) {
	npc := testNPCSpawn()
	msg, err := npcAddObject(2, npc, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(msg, []byte("FuncNpc01608\x00")) {
		t.Fatal("NPC AddObject does not contain the client resource ConfigID")
	}
	if !bytes.Contains(msg, []byte{34, 0, 229, 0, 0, 0}) {
		t.Fatal("NPC AddObject does not contain int32 NpcType=229")
	}
	if got := binary.LittleEndian.Uint16(msg[0x3d:]); got != 7 {
		t.Fatalf("NPC property count=%d, want 7", got)
	}
	for offset := 0x19; offset < 0x3d; offset++ {
		if msg[offset] != 0 {
			t.Fatalf("speculative motion-header byte at %#x = %#x, want zero", offset, msg[offset])
		}
	}
	for _, value := range []string{"hand10_1", "FuncNpc01608", `npc\worldnpc288`} {
		if !bytes.Contains(msg, []byte(value+"\x00")) {
			t.Fatalf("NPC AddObject does not contain %q", value)
		}
	}
}

// TestOfficialPropertyTableFrameIsVerbatim 官方 0x09 表(2370 项)逐行
// 透传:计数、NpcType@34 官方即 Int32(旧 ModernV1 协商 hack 退役)、
// 玩家域关键锚点全部按位置出现。帧内容必须与官方方言逐字节一致。
func TestOfficialPropertyTableFrameIsVerbatim(t *testing.T) {
	msg := officialPropertyTableFrame()
	if msg[0] != 0x09 {
		t.Fatalf("property table opcode=%#x", msg[0])
	}
	if got := binary.LittleEndian.Uint16(msg[1:]); got != 2370 {
		t.Fatalf("official property count=%d, want 2370", got)
	}
	npcType := append([]byte("NpcType"), 0, byte(clientdata.WireInt32))
	if !bytes.Contains(msg, npcType) {
		t.Fatal("official table does not negotiate NpcType as int32")
	}
	sp := append([]byte("SP"), 0, byte(clientdata.WireInt32))
	if !bytes.Contains(msg, sp) {
		t.Fatal("official table does not declare SP as int32")
	}
	maxSP := append([]byte("MaxSP"), 0, byte(clientdata.WireInt32))
	if !bytes.Contains(msg, maxSP) {
		t.Fatal("official table does not declare MaxSP")
	}
	inParry := append([]byte("InParry"), 0, byte(clientdata.WireInt32))
	if !bytes.Contains(msg, inParry) {
		t.Fatal("official table does not declare InParry as int32")
	}
	autoSelect := append([]byte("AutoSelectObject"), 0, byte(clientdata.WireObject))
	if !bytes.Contains(msg, autoSelect) {
		t.Fatal("official table does not declare AutoSelectObject as object")
	}
	// 行位置即索引:重走一遍帧体,断言第 467 项是 SP、第 463 项是
	// LogicState(0x09 表位置绑定的硬语义)。
	offset := 3
	for position := 0; position < 2370; position++ {
		endByte := bytes.IndexByte(msg[offset:], 0)
		if endByte < 0 {
			t.Fatalf("entry %d has no NUL terminator", position)
		}
		name := string(msg[offset : offset+endByte])
		code := msg[offset+endByte+1]
		switch position {
		case 463:
			if name != "LogicState" || code != byte(clientdata.WireByte) {
				t.Fatalf("entry 463 = (%q,%d), want (LogicState,byte)", name, code)
			}
		case 467:
			if name != "SP" || code != byte(clientdata.WireInt32) {
				t.Fatalf("entry 467 = (%q,%d), want (SP,int32)", name, code)
			}
		case 580:
			if name != "MaxSP" {
				t.Fatalf("entry 580 = %q, want MaxSP", name)
			}
		}
		offset = offset + endByte + 2
	}
	if offset != len(msg) {
		t.Fatalf("frame walk consumed %d of %d bytes", offset, len(msg))
	}
}

func testNPCSpawn() npcSpawn {
	properties := map[string]clientdata.PropertyValue{}
	for name, value := range map[string]clientdata.Value{
		"Type":       clientdata.ByteValue(4),
		"ConfigID":   clientdata.StringValue("FuncNpc01608"),
		"Resource":   clientdata.StringValue(`npc\worldnpc288`),
		"PosiX":      clientdata.Float32Value(1),
		"NpcType":    clientdata.Int32Value(229),
		"Name":       clientdata.WideStringValue("武当门派指引人"),
		"WeaponMode": clientdata.StringValue("hand10_1"),
	} {
		properties[name] = clientdata.PropertyValue{Value: value}
	}
	return npcSpawn{resolved: clientdata.ResolvedNPC{
		ConfigID:    "FuncNpc01608",
		ScriptClass: "CommonNpc",
		Properties:  properties,
		Extensions: map[string]string{
			"template.ShopID": "Shop_yaopin_00100",
		},
	}, x: 1}
}

func TestEventNPCIsNotMaterializedWithoutTaskOwnership(t *testing.T) {
	ordinary := testNPCSpawn()
	event := testNPCSpawn()
	event.resolved.ConfigID = "EventNpc_gmpzyb_028"
	event.resolved.ScriptClass = "EventNpc"
	event.x = 2

	if npcSpawnEligible(event) {
		t.Fatal("EventNpc without local task ownership must be excluded")
	}
	if !npcSpawnEligible(ordinary) {
		t.Fatal("ordinary NPC must remain eligible")
	}

	// 共享场景的准入闸门在目录装载层：EventNpc 绝不进入世界。
	lifecycle := newSceneLifecycle("test", discardMessageConnection{})
	if err := lifecycle.begin("scene", "resource", []npcSpawn{ordinary, event}); err != nil {
		t.Fatal(err)
	}
	if _, ok := lifecycle.npcMetaByID(forkNPCID(2)); !ok {
		t.Fatal("ordinary NPC was not registered into the run")
	}
	if _, ok := lifecycle.npcMetaByID(forkNPCID(3)); ok {
		t.Fatal("EventNpc entered the shared scene world")
	}
	// 不注册即无 Vitals/Brain——任务怪（触发任务才刷新）永不进入
	// 死亡→重生循环，重生批次不对其做任何事。
	if _, ok := lifecycle.run.combat.Vitals(worldcore.EntityID(forkNPCID(3))); ok {
		t.Fatal("EventNpc must never gain combat vitals or a respawn cycle")
	}
}

// TestSubSceneCombatNPCIsNotMaterialized 子场景闸门：creator 条目带
// SubSceneNo 的战斗脚本类（AttackNpc/BossNpc）是活动/分线态内容（成都
// AttackNpc.xml 135/219 条，如夜探守卫 newworld_nangong_yetan_*
// SubSceneNo=8961），官服只在子场景激活时物化——基础场景不注册、不
// 物化、无 Vitals/Brain。平民带 SubSceneNo 照常注册（站桩无害）。
func TestSubSceneCombatNPCIsNotMaterialized(t *testing.T) {
	combat := testNPCSpawn()
	combat.resolved.ConfigID = "newworld_nangong_yetan_002"
	combat.resolved.ScriptClass = "AttackNpc"
	combat.resolved.Extensions["creator.SubSceneNo"] = "8961"

	civilian := testNPCSpawn()
	civilian.resolved.Extensions["creator.SubSceneNo"] = "8961"

	boss := testNPCSpawn()
	boss.resolved.ScriptClass = "BossNpc"
	boss.resolved.Extensions["creator.SubSceneNo"] = "50"

	wild := testNPCSpawn() // 野外怪无 SubSceneNo，保持可注册
	wild.resolved.ScriptClass = "AttackNpc"

	if npcSpawnEligible(combat) || npcSpawnEligible(boss) {
		t.Fatal("sub-scene combat NPCs must be excluded from the base scene")
	}
	if !npcSpawnEligible(civilian) {
		t.Fatal("sub-scene civilians stay materialized")
	}
	if !npcSpawnEligible(wild) {
		t.Fatal("wilderness combat NPCs without SubSceneNo must stay eligible")
	}

	// 共享场景准入同闸门：子场景战斗 NPC 不进世界（ID 间隙与 EventNpc
	// 同语义，目录下标→wire ID 映射不受影响）。
	lifecycle := newSceneLifecycle("test", discardMessageConnection{})
	if err := lifecycle.begin("scene", "resource", []npcSpawn{combat, civilian, wild}); err != nil {
		t.Fatal(err)
	}
	if _, ok := lifecycle.npcMetaByID(forkNPCID(2)); ok {
		t.Fatal("sub-scene combat NPC entered the shared scene world")
	}
	if _, ok := lifecycle.npcMetaByID(forkNPCID(3)); !ok {
		t.Fatal("sub-scene civilian must stay registered")
	}
	if _, ok := lifecycle.npcMetaByID(forkNPCID(4)); !ok {
		t.Fatal("wilderness combat NPC must stay registered")
	}
	if _, ok := lifecycle.run.combat.Brain(worldcore.EntityID(forkNPCID(2))); ok {
		t.Fatal("sub-scene combat NPC must never gain a brain")
	}
}

func TestSceneLifecycleWaitsForPostReadyClientActivity(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	config := transport.DefaultConnectionConfig()
	config.MaxFrameSize = 1024
	link, err := transport.NewConnection(server, config)
	if err != nil {
		t.Fatal(err)
	}
	world := newSceneLifecycle("pipe", link)
	if err := world.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	want := []byte{0x0D, 2, 0, 0, 0}
	entity := testWorldEntity(2)
	if err := world.run.scene.Add(entity); err != nil {
		t.Fatal(err)
	}
	world.spawnTestNPC(2, sim.NewVitals(10, 10), worldcore.Transform{}, sceneEntity{id: 2, payload: want}, nil)

	if err := world.clientReady(); err != nil {
		t.Fatalf("ClientReady arm: %v", err)
	}
	if !world.armed {
		t.Fatal("ClientReady must arm rather than materialize the localhost object")
	}
	done := make(chan error, 1)
	go func() { done <- world.clientActivity() }()
	reader, err := transport.NewFrameReader(client, 1024)
	if err != nil {
		t.Fatal(err)
	}
	got, err := reader.ReadFrame(transport.InitialKey)
	if err != nil {
		t.Fatalf("read scene object: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("scene object=%x, want %x", got, want)
	}
	if err := <-done; err != nil {
		t.Fatalf("post-ready client activity: %v", err)
	}
	if delta := world.viewport.Plan(world.run.scene.Snapshot()); len(delta.Adds)+len(delta.Updates)+len(delta.Removes) != 0 {
		t.Fatalf("scene object was not committed to viewport: %#v", delta)
	}
}

func testWorldEntity(id uint32) worldcore.Entity {
	return worldcore.Entity{ID: worldcore.EntityID(id), OwnerID: 1, Kind: worldcore.EntityNPC}
}

type discardMessageConnection struct{}

func (discardMessageConnection) WriteFrame([]byte) error { return nil }

func TestSceneLifecycleRejectsDuplicateIdentityWithoutPanic(t *testing.T) {
	lifecycle := newSceneLifecycle("test", discardMessageConnection{})
	if err := lifecycle.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	npc := testNPCSpawn()
	if err := lifecycle.registerNPCOnRun(2, npc); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.registerNPCOnRun(2, npc); !errors.Is(err, worldcore.ErrDuplicateEntity) {
		t.Fatalf("duplicate register error = %v", err)
	}
}

func TestParseMotionPosition(t *testing.T) {
	msg := make([]byte, 0x26, 67)
	msg[0] = 0x0A
	for _, value := range []float32{481.25, 87.545, 624.5, 3.12} {
		msg = append(msg, 4)
		msg = binary.LittleEndian.AppendUint32(msg, math.Float32bits(value))
	}
	msg = append(msg, make([]byte, 67-len(msg))...)
	x, y, z, orient, ok := parseMotionPosition(msg)
	if !ok || x != 481.25 || y != float32(87.545) || z != 624.5 || orient != float32(3.12) {
		t.Fatalf("position=(%v,%v,%v,%v) ok=%v", x, y, z, orient, ok)
	}
}

func TestParseMotionPositionRejectsLongDestinationPacket(t *testing.T) {
	msg := make([]byte, 82)
	if _, _, _, _, ok := parseMotionPosition(msg); ok {
		t.Fatal("long destination packet must not be persisted as a position update")
	}
}

func TestParseClientObjectRequest(t *testing.T) {
	msg := make([]byte, 33)
	msg[0] = 0x07
	binary.LittleEndian.PutUint32(msg[17:], 0x167)
	binary.LittleEndian.PutUint32(msg[21:], forkNPCID(3494))
	binary.LittleEndian.PutUint32(msg[25:], 1)
	request, ok := parseClientObjectRequest(msg)
	if !ok {
		t.Fatal("captured object request was rejected")
	}
	if request.Sequence != 0x167 || request.ObjectID != forkNPCID(3494) || request.OwnerID != 1 {
		t.Fatalf("request=%+v", request)
	}
}

func TestServerMovingLayout(t *testing.T) {
	msg := serverMoving(forkNPCID(3494), 1, MotionState{DestX: 1.25, DestY: 2.5, DestZ: 3.75, DestOrient: 4, Extra: 7})
	if len(msg) != 45 || msg[0] != 0x20 {
		t.Fatalf("moving header len=%d opcode=%#x", len(msg), msg[0])
	}
	if got := binary.LittleEndian.Uint32(msg[1:]); got != forkNPCID(3494) {
		t.Fatalf("object=%d", got)
	}
	if got := math.Float32frombits(binary.LittleEndian.Uint32(msg[9:])); got != 1.25 {
		t.Fatalf("dest x=%v", got)
	}
	if got := binary.LittleEndian.Uint32(msg[41:]); got != 7 {
		t.Fatalf("extra=%d", got)
	}
}

func TestObjectRequestSetsPlayerLastObjectAndConfiguredTalkMenu(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.registerNPCOnRun(3494, testNPCSpawn()); err != nil {
		t.Fatal(err)
	}
	if _, err := lifecycle.objectRequest(clientObjectRequest{Sequence: 1, ObjectID: forkNPCID(3494), OwnerID: 1}); err != nil {
		t.Fatal(err)
	}
	frames := conn.Frames()
	// LastObject + AutoSelectObject 双 0x10 + 数值型 custom begin/add/end 序列。
	if len(frames) != 6 {
		t.Fatalf("selection frames=%d, want LastObject+AutoSelectObject + cinematic menu: %x", len(frames), frames)
	}
	frame := frames[0]
	if len(frame) != 22 || frame[0] != 0x10 || binary.LittleEndian.Uint32(frame[2:]) != playerObjectID || binary.LittleEndian.Uint16(frame[12:]) != 414 { // LastObject(官方玩家域)
		t.Fatalf("LastObject frame=%x", frame)
	}
	if got := binary.LittleEndian.Uint64(frame[14:]); got != uint64(forkNPCID(3494))|uint64(1)<<32 {
		t.Fatalf("LastObject reference=%#x", got)
	}
	if auto := frames[1]; len(auto) != 22 || binary.LittleEndian.Uint16(auto[12:]) != propAutoSelectObject ||
		binary.LittleEndian.Uint64(auto[14:]) != uint64(forkNPCID(3494))|uint64(1)<<32 {
		t.Fatalf("AutoSelectObject frame=%x", auto)
	}
	if got := binary.LittleEndian.Uint32(frames[2][4:]); frames[2][0] != 0x1E || frames[2][3] != 2 || got != 4 {
		t.Fatalf("cinematic begin frame=%x", frames[2])
	}
	if got := binary.LittleEndian.Uint32(frames[len(frames)-1][4:]); frames[len(frames)-1][0] != 0x1E || frames[len(frames)-1][3] != 2 || got != 7 {
		t.Fatalf("cinematic end frame=%x", frames[len(frames)-1])
	}
	if _, err := lifecycle.objectRequest(clientObjectRequest{Sequence: 2, ObjectID: forkNPCID(3494), OwnerID: 1}); err != nil {
		t.Fatal(err)
	}
	if got := len(conn.Frames()); got != 6 {
		t.Fatalf("duplicate selection frames=%d, want no duplicate menu", got)
	}
}

func TestObjectRequestSelectsCombatNPCWithoutDialogue(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	npc := testNPCSpawn()
	npc.resolved.ConfigID = "AttackNPC_test"
	npc.resolved.ScriptClass = "AttackNpc"
	if err := lifecycle.registerNPCOnRun(4455, npc); err != nil {
		t.Fatal(err)
	}
	meta4455, _ := lifecycle.npcMeta(worldcore.EntityID(forkNPCID(4455)))
	if got := meta4455.interaction; got != npcInteractionCombat {
		t.Fatalf("combat interaction=%s", got)
	}
	if _, err := lifecycle.objectRequest(clientObjectRequest{Sequence: 1, ObjectID: forkNPCID(4455), OwnerID: 1}); err != nil {
		t.Fatal(err)
	}
	frames := conn.Frames()
	// 0x07 点击回 LastObject(100) + AutoSelectObject(230) 双帧
	//（与 msg431 同路径；49840 抓包 414/415 对应）。
	if len(frames) != 2 || frames[0][0] != 0x10 || frames[1][0] != 0x10 {
		t.Fatalf("combat click frames=%x; want LastObject+AutoSelectObject", frames)
	}
	if got := binary.LittleEndian.Uint64(frames[0][14:]); got != uint64(forkNPCID(4455))|uint64(1)<<32 {
		t.Fatalf("combat LastObject=%#x", got)
	}
	if got := binary.LittleEndian.Uint16(frames[1][12:]); got != propAutoSelectObject {
		t.Fatalf("combat AutoSelectObject property index=%d", got)
	}
	if got := binary.LittleEndian.Uint64(frames[1][14:]); got != uint64(forkNPCID(4455))|uint64(1)<<32 {
		t.Fatalf("combat AutoSelectObject=%#x", got)
	}
	if lifecycle.lastMenuObject != 0 {
		t.Fatalf("combat click opened menu for object=%d", lifecycle.lastMenuObject)
	}
	if targetID, targetOwner, ok := lifecycle.selectedCombatTarget(); !ok || targetID != forkNPCID(4455) || targetOwner != 1 {
		t.Fatalf("combat target=%d-%d ok=%t", targetID, targetOwner, ok)
	}
}

func TestObjectRequestDoesNotFabricateDialogueForUnconfiguredCivilian(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	npc := testNPCSpawn()
	npc.resolved.Extensions = nil
	if err := lifecycle.registerNPCOnRun(4456, npc); err != nil {
		t.Fatal(err)
	}
	meta4456, _ := lifecycle.npcMeta(worldcore.EntityID(forkNPCID(4456)))
	if got := meta4456.interaction; got != npcInteractionSelectOnly {
		t.Fatalf("civilian interaction=%s", got)
	}
	if _, err := lifecycle.objectRequest(clientObjectRequest{Sequence: 1, ObjectID: forkNPCID(4456), OwnerID: 1}); err != nil {
		t.Fatal(err)
	}
	frames := conn.Frames()
	if len(frames) != 2 || frames[0][0] != 0x10 || frames[1][0] != 0x10 {
		t.Fatalf("unconfigured civilian click frames=%x; want LastObject+AutoSelectObject", frames)
	}
	if _, _, ok := lifecycle.selectedCombatTarget(); ok {
		t.Fatal("civilian selection became a combat target")
	}
}

func TestObjectRequestOpensDepotViewAfterCinematicSelection(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.registerNPCOnRun(3494, testNPCSpawn()); err != nil {
		t.Fatal(err)
	}
	entity, _ := lifecycle.npcMeta(worldcore.EntityID(forkNPCID(3494)))
	entity.services = []npcService{{mark: 0x1002, label: "仓库", value: "1", source: "DepotID"}}
	lifecycle.writeNpcMeta(worldcore.EntityID(forkNPCID(3494)), entity)

	if _, err := lifecycle.objectRequest(clientObjectRequest{Sequence: 1, ObjectID: forkNPCID(3494), OwnerID: 1}); err != nil {
		t.Fatal(err)
	}
	frames := conn.Frames()
	if len(frames) != 6 || frames[0][0] != 0x10 || frames[2][0] != 0x1E {
		t.Fatalf("depot click frames=%d, want LastObject+AutoSelectObject + cinematic talk: %x", len(frames), frames)
	}
	if err := lifecycle.selectNPCMenu(clientObjectRequest{Sequence: 2, ObjectID: forkNPCID(3494), OwnerID: 1, FuncID: 807000000}); err != nil {
		t.Fatal(err)
	}
	frames = conn.Frames()
	if got := frames[7]; len(got) != 7 || got[0] != 0x15 || binary.LittleEndian.Uint16(got[1:]) != 4 || binary.LittleEndian.Uint16(got[3:]) != 18 {
		t.Fatalf("depot CreateView=%x", got)
	}
	if got := binary.LittleEndian.Uint32(frames[6][4:]); frames[6][0] != 0x1E || frames[6][3] != 2 || got != 8 {
		t.Fatalf("depot close frame=%x", frames[6])
	}
}

func TestObjectRequestOpensShopViewAfterCinematicSelection(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.registerNPCOnRun(3494, testNPCSpawn()); err != nil {
		t.Fatal(err)
	}
	entity, _ := lifecycle.npcMeta(worldcore.EntityID(forkNPCID(3494)))
	entity.services = []npcService{{mark: 0x1001, label: "商店", value: "Shop_yaopin_00100", source: "ShopID"}}
	lifecycle.writeNpcMeta(worldcore.EntityID(forkNPCID(3494)), entity)

	if _, err := lifecycle.objectRequest(clientObjectRequest{Sequence: 1, ObjectID: forkNPCID(3494), OwnerID: 1}); err != nil {
		t.Fatal(err)
	}
	frames := conn.Frames()
	if len(frames) != 6 || frames[0][0] != 0x10 || frames[2][0] != 0x1E {
		t.Fatalf("shop click frames=%d, want LastObject+AutoSelectObject + cinematic talk: %x", len(frames), frames)
	}
	if err := lifecycle.selectNPCMenu(clientObjectRequest{Sequence: 2, ObjectID: forkNPCID(3494), OwnerID: 1, FuncID: 805000000}); err != nil {
		t.Fatal(err)
	}
	frames = conn.Frames()
	if len(frames) != 14 || frames[6][0] != 0x1E {
		t.Fatalf("shop selection frames=%d, want cinematic close + shop view: %x", len(frames), frames)
	}
	shop := frames[7]
	if shop[0] != 0x15 || binary.LittleEndian.Uint16(shop[1:]) != 61 || binary.LittleEndian.Uint16(shop[3:]) != 100 || binary.LittleEndian.Uint16(shop[5:]) != 4 {
		t.Fatalf("shop CreateView header=%x", shop)
	}
	if !bytes.Contains(shop, []byte("Shop_yaopin_00100\x00")) {
		t.Fatalf("shop CreateView missing ShopID: %x", shop)
	}
	for i, configID := range []string{"yaopin_00100", "yaopin_00300", "yaopin_00500", "yaopin_10100", "yaopin_10300", "yaopin_10500"} {
		row := frames[i+8]
		if row[0] != 0x18 || binary.LittleEndian.Uint16(row[1:]) != 61 || binary.LittleEndian.Uint16(row[3:]) != uint16(i+1) || binary.LittleEndian.Uint16(row[5:]) != 6 {
			t.Fatalf("shop row %d header=%x", i, row)
		}
		if !bytes.Contains(row, []byte(configID+"\x00")) {
			t.Fatalf("shop row %d missing ConfigID %s: %x", i, configID, row)
		}
	}
}

func TestMultiServiceNPCUsesNativeMenuThenSelectedShopView(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.registerNPCOnRun(3494, testNPCSpawn()); err != nil {
		t.Fatal(err)
	}
	entity, _ := lifecycle.npcMeta(worldcore.EntityID(forkNPCID(3494)))
	entity.services = []npcService{
		{mark: markShop, label: "商店", value: "Shop_yaopin_00100", source: "ShopID"},
		{mark: markDepot, label: "仓库", value: "1", source: "DepotID"},
	}
	lifecycle.writeNpcMeta(worldcore.EntityID(forkNPCID(3494)), entity)

	if _, err := lifecycle.objectRequest(clientObjectRequest{Sequence: 1, ObjectID: forkNPCID(3494), OwnerID: 1}); err != nil {
		t.Fatal(err)
	}
	frames := conn.Frames()
	if len(frames) != 7 || frames[0][0] != 0x10 || frames[2][0] != 0x1E {
		t.Fatalf("multi-service click must send LastObject+AutoSelectObject + cinematic menu: %x", frames)
	}
	if err := lifecycle.selectNPCMenu(clientObjectRequest{Sequence: 2, ObjectID: forkNPCID(3494), OwnerID: 1, FuncID: 805000000}); err != nil {
		t.Fatal(err)
	}
	frames = conn.Frames()
	if len(frames) != 15 || frames[7][0] != 0x1E || frames[8][0] != 0x15 || binary.LittleEndian.Uint16(frames[8][1:]) != 61 {
		t.Fatalf("multi-service shop selection did not close cinematic menu and create View 61: %x", frames)
	}
}

func TestServerMenuUsesCurrentClientLayout(t *testing.T) {
	msg, err := serverMenu(forkNPCID(3494), 1, []serverMenuItem{{Type: 7, Mark: 0x1234, Content: "测试"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(msg) != 24 || msg[0] != 0x1C {
		t.Fatalf("menu len/header=%d/%x", len(msg), msg)
	}
	if got := binary.LittleEndian.Uint32(msg[1:]); got != forkNPCID(3494) {
		t.Fatalf("object=%d", got)
	}
	if got := binary.LittleEndian.Uint32(msg[5:]); got != 1 {
		t.Fatalf("owner=%d", got)
	}
	if got := binary.LittleEndian.Uint16(msg[9:]); got != 1 {
		t.Fatalf("count=%d", got)
	}
	if msg[11] != 7 || binary.LittleEndian.Uint16(msg[12:]) != 0x1234 || binary.LittleEndian.Uint32(msg[14:]) != 6 {
		t.Fatalf("item prefix=%x", msg[11:18])
	}
	if got := string(utf16.Decode([]uint16{binary.LittleEndian.Uint16(msg[18:]), binary.LittleEndian.Uint16(msg[20:])})); got != "测试" {
		t.Fatalf("content=%q", got)
	}
	if binary.LittleEndian.Uint16(msg[22:]) != 0 {
		t.Fatalf("missing UTF-16 NUL: %x", msg)
	}
}

func TestModernNPCServiceMenuUsesTemplateBusinessFields(t *testing.T) {
	npc := testNPCSpawn()
	npc.resolved.Extensions = map[string]string{
		"template.ShopID":      "Shop_yaopin_00100",
		"template.DepotID":     "1",
		"template.TransFuncID": "37",
	}
	services := modernNPCServices(npc, nil)
	items := npcServiceMenu(services, nil)
	if len(items) != 4 || items[0].Content != "商店" || items[1].Content != "仓库" || items[2].Content != "传送" {
		t.Fatalf("modern services menu = %#v", items)
	}
}

func TestModernNPCServiceCreatorOverrideWinsOverTemplate(t *testing.T) {
	npc := testNPCSpawn()
	npc.resolved.Extensions = map[string]string{
		"template.ShopID": "Shop_yaopin_00100",
		"creator.ShopID":  "Shop_chengdu_002",
	}
	services := modernNPCServices(npc, nil)
	if len(services) != 1 || services[0].value != "Shop_chengdu_002" || services[0].source != "creator.ShopID" {
		t.Fatalf("creator override lost: %#v", services)
	}
}

func TestModernNPCServicesDeriveTaskServiceFromCatalog(t *testing.T) {
	catalog := taskdata.NewCatalog(
		map[int]taskdata.Task{75730: {Title: "虎啸疑云"}},
		[]taskdata.Binding{
			{TaskID: 75731, NPCConfigID: "FuncNpc01608", Source: "storytasklist"},
			{TaskID: 75730, NPCConfigID: "funcnpc01608", Source: "storytasklist"},
		}, nil)
	npc := testNPCSpawn()
	npc.resolved.Extensions = nil // testNPCSpawn 自带 ShopID 演示扩展，这里只看任务
	services := modernNPCServices(npc, catalog)
	if len(services) != 1 {
		t.Fatalf("services = %#v", services)
	}
	service := services[0]
	if service.mark != markTask || service.label != "任务" {
		t.Fatalf("task service = %#v", service)
	}
	if service.value != "75730,75731" || service.source != "taskcatalog.storytasklist" {
		t.Fatalf("task service value/source = %q/%q", service.value, service.source)
	}
	// 未绑定任务的 NPC 不产生任务服务（大小写不敏感查询）。
	unbound := testNPCSpawn()
	unbound.resolved.ConfigID = "npc_without_tasks"
	unbound.resolved.Extensions = nil
	if got := modernNPCServices(unbound, catalog); len(got) != 0 {
		t.Fatalf("unbound npc services = %#v", got)
	}
}

func TestOpenTaskCatalogAtPrefersUsableSnapshot(t *testing.T) {
	catalog := taskdata.NewCatalog(
		map[int]taskdata.Task{75730: {Title: "虎啸疑云"}},
		[]taskdata.Binding{{TaskID: 75730, NPCConfigID: "FuncNpc01608", Source: "storytasklist"}},
		nil)
	data, err := taskdata.MarshalSnapshot(catalog.SnapshotAt(time.Unix(1700000000, 0).UTC()))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "tasks.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	loaded := openTaskCatalogAt(path)
	if loaded.Count() != 1 || loaded.BindingCount() != 1 {
		t.Fatalf("snapshot load counts tasks=%d bindings=%d", loaded.Count(), loaded.BindingCount())
	}
	if tasks := loaded.TasksForNPC("FuncNpc01608"); len(tasks) != 1 || tasks[0].Title != "虎啸疑云" {
		t.Fatalf("tasks from snapshot = %#v", tasks)
	}
}

func TestNpcServiceScriptSeesBoundTasks(t *testing.T) {
	previous := taskCatalog
	taskCatalog = taskdata.NewCatalog(
		map[int]taskdata.Task{75730: {Title: "<font face=\"Default\">虎啸疑云</font>"}},
		[]taskdata.Binding{{TaskID: 75730, NPCConfigID: "FuncNpc01608", Source: "storytasklist"}},
		nil)
	t.Cleanup(func() { taskCatalog = previous })

	var logs []string
	dir := t.TempDir()
	script := `game.on("npc_service", function(ctx)
	local tasks = ctx:tasks()
	local first = "none"
	if #tasks > 0 then first = tasks[1].id .. ":" .. tasks[1].title end
	game.log("bound tasks=%d first=%s", #tasks, first)
	return true
end)
`
	if err := os.WriteFile(filepath.Join(dir, "task.lua"), []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	host := luascript.New(dir, func(format string, args ...any) {
		logs = append(logs, strings.TrimSpace(fmt.Sprintf(format, args...)))
	})
	if _, err := host.LoadDir(); err != nil {
		t.Fatal(err)
	}
	defer host.Close()

	conn := &captureMessageConnection{}
	lifecycle := newTaskServiceLifecycle(t, conn, host)
	lifecycle.run.mu.Lock()
	err := lifecycle.openSelectedServiceLocked(mustEntity(t, lifecycle), npcService{mark: markTask, label: "任务", source: "taskcatalog.storytasklist", value: "75730"})
	lifecycle.run.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(logs, "\n")
	if !strings.Contains(joined, "bound tasks=1 first=75730:虎啸疑云") {
		t.Fatalf("script log missing bound task: %q", joined)
	}
}

func TestPlayableNPCServicesKeepsOnlyConfiguredImplementedServices(t *testing.T) {
	services := playableNPCServices([]npcService{
		{mark: markShop, value: "Shop_yaopin_00100"},
		{mark: markDepot, value: "1"},
		{mark: markTransport, value: "37"},
		{mark: markTask},
	})
	if len(services) != 2 || services[0].mark != markShop || services[1].mark != markDepot {
		t.Fatalf("playable services=%#v", services)
	}
}

func TestModernTalkMenuUsesCustomGMenuIDs(t *testing.T) {
	items := talkMenuItems([]npcService{{mark: 0x1001}, {mark: 0x1002}}, nil)
	if len(items) != 3 || items[0].funcID != 805000000 || items[1].funcID != 807000000 || items[2].funcID != 600000000 {
		t.Fatalf("talk items = %#v", items)
	}
}

func TestBagViewContainersUseCurrentCreateViewLayout(t *testing.T) {
	// 17 视图容器（8 主页 BaseCap=18 + 8 扩展页 + View1 纸娃娃）由
	// inventory_test.go TestBagViewCreateFramesLayout 逐项断言；
	// 这里保留旧用例的视图覆盖面检查（页签与扩展页齐全）。
	seen := make(map[uint16]bool)
	for _, frame := range bagViewCreateFrames() {
		if frame[0] != 0x15 {
			t.Fatalf("bag view frame opcode=%#x, want 0x15", frame[0])
		}
		seen[binary.LittleEndian.Uint16(frame[1:])] = true
	}
	for _, required := range []uint16{1, 2, 3, 121, 122, 123, 124, 125, 126,
		174, 175, 176, 177, 178, 179, 180, 181} {
		if !seen[required] {
			t.Fatalf("missing required bag view %d", required)
		}
	}
}

type captureMessageConnection struct {
	mu     sync.Mutex
	frames [][]byte
}

func (c *captureMessageConnection) WriteFrame(frame []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.frames = append(c.frames, append([]byte(nil), frame...))
	return nil
}

func (c *captureMessageConnection) Frames() [][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([][]byte(nil), c.frames...)
}

func TestNoRoleLoginSuccess(t *testing.T) {
	now := time.Date(2026, 7, 11, 12, 34, 56, 0, time.Local)
	msg := noRoleLoginSuccess(now)
	if len(msg) != 0x29 || msg[0] != 0x04 {
		t.Fatalf("header len=%d opcode=%#x", len(msg), msg[0])
	}
	if got := binary.LittleEndian.Uint32(msg[0x25:]); got != 0 {
		t.Fatalf("role count=%d, want 0", got)
	}
	if got := binary.LittleEndian.Uint32(msg[0x09:]); got != 2026 {
		t.Fatalf("year=%d, want 2026", got)
	}
}

func TestWorldInfo(t *testing.T) {
	msg := worldInfo(0, "0,book1,0,0")
	if msg[0] != 0x05 || binary.LittleEndian.Uint16(msg[1:]) != 0 {
		t.Fatalf("header = %x", msg[:3])
	}
	if msg[len(msg)-2] != 0 || msg[len(msg)-1] != 0 {
		t.Fatal("world info is not NUL-terminated UCS-2")
	}
}

func TestCreatedRoleList(t *testing.T) {
	msg := oneRoleLoginSuccess(time.Date(2026, 7, 11, 12, 0, 0, 0, time.Local), "本地角色", nil, "", "")
	if got := binary.LittleEndian.Uint32(msg[0x25:]); got != 1 {
		t.Fatalf("role count=%d, want 1", got)
	}
	if msg[0x29] != 6 || msg[0x2A] != 0 || msg[0x2B] != 2 {
		t.Fatalf("role archive prefix=%x", msg[0x29:0x2C])
	}
	var encodedFaction []byte
	for _, unit := range utf16.Encode([]rune(defaultRoleFaction)) {
		encodedFaction = binary.LittleEndian.AppendUint16(encodedFaction, unit)
	}
	if !bytes.Contains(msg, encodedFaction) {
		t.Fatalf("role archive does not contain default faction %q", defaultRoleFaction)
	}
}

func TestCreateRoleStrings(t *testing.T) {
	msg := make([]byte, 0x49)
	for _, value := range []string{"book4", "icon.png", "face-data", "1", "hair", "cloth", "pants", "shoes", "school_shaolin"} {
		msg = append(msg, 6)
		var size [4]byte
		binary.LittleEndian.PutUint32(size[:], uint32(len(value)+1))
		msg = append(msg, size[:]...)
		msg = append(msg, value...)
		msg = append(msg, 0)
	}
	got := createRoleStrings(msg)
	if len(got) != 9 || got[2] != "face-data" || got[8] != "school_shaolin" {
		t.Fatalf("strings=%q", got)
	}
}

func TestCreateRoleName(t *testing.T) {
	msg := make([]byte, 77)
	for i, unit := range utf16.Encode([]rune("测试角色")) {
		binary.LittleEndian.PutUint16(msg[5+i*2:], unit)
	}
	if got := createRoleName(msg); got != "测试角色" {
		t.Fatalf("name=%q", got)
	}
}

func TestSceneObjectByteProperty(t *testing.T) {
	msg := sceneObjectByteProperty(2, 1, 34, 82)
	if len(msg) != 15 || msg[0] != 0x10 || msg[1] != 0 {
		t.Fatalf("header=%x len=%d", msg[:2], len(msg))
	}
	if got := binary.LittleEndian.Uint32(msg[2:]); got != 2 {
		t.Fatalf("object id=%d", got)
	}
	if got := binary.LittleEndian.Uint32(msg[6:]); got != 1 {
		t.Fatalf("owner id=%d", got)
	}
	if got := binary.LittleEndian.Uint16(msg[10:]); got != 1 {
		t.Fatalf("property count=%d", got)
	}
	if got := binary.LittleEndian.Uint16(msg[12:]); got != 34 || msg[14] != 82 {
		t.Fatalf("property=%d value=%d", got, msg[14])
	}
}

func TestSceneObjectTransformProperties(t *testing.T) {
	transform := worldcore.Transform{X: 738.09, Y: 23.79, Z: 676.88, Orient: 0.16}
	msg := sceneObjectTransformProperties(1487, 1, transform)
	if len(msg) != 36 || msg[0] != 0x10 || msg[1] != 0 {
		t.Fatalf("transform update header=%x", msg)
	}
	if got := binary.LittleEndian.Uint16(msg[10:]); got != 4 {
		t.Fatalf("property count=%d, want 4", got)
	}
	offset := 12
	for i, want := range []struct {
		index uint16
		value float32
	}{{9, transform.X}, {10, transform.Y}, {11, transform.Z}, {12, transform.Orient}} {
		if got := binary.LittleEndian.Uint16(msg[offset:]); got != want.index {
			t.Fatalf("property %d index=%d, want %d", i, got, want.index)
		}
		if got := math.Float32frombits(binary.LittleEndian.Uint32(msg[offset+2:])); got != want.value {
			t.Fatalf("property %d value=%f, want %f", i, got, want.value)
		}
		offset += 6
	}
}

func TestServerLocation(t *testing.T) {
	transform := worldcore.Transform{X: 738.09, Y: 23.79, Z: 676.88, Orient: 0.16}
	msg := serverLocation(forkNPCID(1487), 1, transform)
	if len(msg) != 25 || msg[0] != 0x1F {
		t.Fatalf("ServerLocation header=%x len=%d", msg, len(msg))
	}
	if got := binary.LittleEndian.Uint32(msg[1:]); got != forkNPCID(1487) {
		t.Fatalf("object id=%d", got)
	}
	if got := binary.LittleEndian.Uint32(msg[5:]); got != 1 {
		t.Fatalf("owner id=%d", got)
	}
	for i, want := range []float32{transform.X, transform.Y, transform.Z, transform.Orient} {
		offset := 9 + i*4
		if got := math.Float32frombits(binary.LittleEndian.Uint32(msg[offset:])); got != want {
			t.Fatalf("transform[%d]=%f, want %f", i, got, want)
		}
	}
}

func TestSceneRemoveObject(t *testing.T) {
	message := sceneRemoveObject(2, 1)
	if len(message) != 9 || message[0] != 0x0E {
		t.Fatalf("remove message = %x", message)
	}
	if got := binary.LittleEndian.Uint32(message[1:]); got != 2 {
		t.Fatalf("object ID = %d", got)
	}
	if got := binary.LittleEndian.Uint32(message[5:]); got != 1 {
		t.Fatalf("owner ID = %d", got)
	}
}

// newTaskServiceLifecycle 构造一个只挂"任务"服务的可交互 NPC 视图，
// 供 Lua npc_service 钩子测试复用。
func newTaskServiceLifecycle(t *testing.T, conn *captureMessageConnection, host *luascript.Host) *sceneLifecycle {
	t.Helper()
	lifecycle := newSceneLifecycle("test", conn)
	lifecycle.scripts = host
	if err := lifecycle.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.registerNPCOnRun(3494, testNPCSpawn()); err != nil {
		t.Fatal(err)
	}
	entity, _ := lifecycle.npcMeta(worldcore.EntityID(forkNPCID(3494)))
	entity.services = []npcService{{mark: markTask, label: "任务", source: "task"}}
	lifecycle.writeNpcMeta(worldcore.EntityID(forkNPCID(3494)), entity)
	if _, err := lifecycle.objectRequest(clientObjectRequest{Sequence: 1, ObjectID: forkNPCID(3494), OwnerID: 1}); err != nil {
		t.Fatal(err)
	}
	return lifecycle
}

func TestNpcServiceScriptHandlesTaskMark(t *testing.T) {
	dir := t.TempDir()
	script := `game.on("npc_service", function(ctx)
	if ctx:service().mark ~= 0x1008 then return false end
	ctx:notify("任务暂未开放", "ui_shop")
	return true
end)
`
	if err := os.WriteFile(filepath.Join(dir, "task.lua"), []byte(script), 0644); err != nil {
		t.Fatal(err)
	}
	host := luascript.New(dir, nil)
	if _, err := host.LoadDir(); err != nil {
		t.Fatal(err)
	}
	defer host.Close()

	conn := &captureMessageConnection{}
	lifecycle := newTaskServiceLifecycle(t, conn, host)
	menuFrames := len(conn.Frames())
	// 直接驱动服务打开路径：菜单选择状态机（cinematic 只映射商店/仓库）
	// 与本钩子无关，openSelectedServiceLocked 才是接入点。
	lifecycle.run.mu.Lock()
	err := lifecycle.openSelectedServiceLocked(mustEntity(t, lifecycle), npcService{mark: markTask, label: "任务", source: "task"})
	lifecycle.run.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	frames := conn.Frames()
	if len(frames) != menuFrames+1 || frames[menuFrames][0] != 0x1E {
		t.Fatalf("task service via script must send one drama prompt frame: %x", frames[menuFrames:])
	}
}

func mustEntity(t *testing.T, lifecycle *sceneLifecycle) sceneEntity {
	t.Helper()
	entity, ok := lifecycle.npcMeta(worldcore.EntityID(forkNPCID(3494)))
	if !ok {
		t.Fatal("npc 3494 not registered")
	}
	return entity
}

func TestNpcServiceWithoutScriptStaysUnimplemented(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newTaskServiceLifecycle(t, conn, nil)
	lifecycle.run.mu.Lock()
	err := lifecycle.openSelectedServiceLocked(mustEntity(t, lifecycle), npcService{mark: markTask, label: "任务", source: "task"})
	lifecycle.run.mu.Unlock()
	if err == nil || !strings.Contains(err.Error(), "not yet implemented") {
		t.Fatalf("without script the task service must stay unimplemented, got %v", err)
	}
}
