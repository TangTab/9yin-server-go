package fssoloworld

// npcinteract_test.go NPC 交互链单测：object request 解析/服务列/菜单
// 编码/选中与菜单路由（帧出口经 inner 广播，断言面=菜单簿记与编码器）。

import (
	"bytes"
	"encoding/binary"
	"testing"
	"unicode/utf16"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
)

func buildObjectRequest(objectID, ownerID uint32, funcID int32) []byte {
	frame := make([]byte, 33)
	frame[0] = 0x07
	binary.LittleEndian.PutUint32(frame[17:], 1)
	binary.LittleEndian.PutUint32(frame[21:], objectID)
	binary.LittleEndian.PutUint32(frame[25:], ownerID)
	binary.LittleEndian.PutUint32(frame[29:], uint32(funcID))
	return frame
}

func TestParseNPCObjectRequest(t *testing.T) {
	req, ok := parseNPCObjectRequest(buildObjectRequest(0x11000005, 0x00660005, 0))
	if !ok || req.ObjectID != 0x11000005 || req.OwnerID != 0x00660005 || req.FuncID != 0 {
		t.Fatalf("解析异常: %+v ok=%v", req, ok)
	}
	if _, ok := parseNPCObjectRequest(buildObjectRequest(0, 1, 0)); ok {
		t.Error("零 objectID 应拒")
	}
	if _, ok := parseNPCObjectRequest([]byte{0x07, 1, 2}); ok {
		t.Error("短帧应拒")
	}
	if _, ok := parseNPCObjectRequest(append([]byte{0x08}, buildObjectRequest(1, 1, 0)[1:]...)); ok {
		t.Error("非 0x07 应拒")
	}
}

func TestNPCServerMenu(t *testing.T) {
	services := npcServicesOf(map[string]string{
		"ShopID": "Shop_yaopin_00101", "DepotID": "0", "Job": "3",
	})
	if len(services) != 2 {
		t.Fatalf("服务列 = %d, want 2（DepotID=0 不算）", len(services))
	}
	frame, err := npcServerMenu(0x11000005, 0x00660005, services)
	if err != nil {
		t.Fatalf("编码: %v", err)
	}
	if frame[0] != 0x1C {
		t.Fatalf("op = %#x, want 0x1C", frame[0])
	}
	if obj := binary.LittleEndian.Uint32(frame[1:]); obj != 0x11000005 {
		t.Errorf("obj = %x", obj)
	}
	n := int(binary.LittleEndian.Uint16(frame[9:]))
	if n != 3 { // 商店+生活职业+离开
		t.Fatalf("项数 = %d, want 3", n)
	}
	// 走查项：type u8=0 / mark u16 / UTF16LE 串（含 NUL）。
	off := 11
	marks := make([]uint16, 0, n)
	labels := make([]string, 0, n)
	for i := 0; i < n; i++ {
		if frame[off] != 0 {
			t.Errorf("项 %d type 非 0", i)
		}
		off++
		mark := binary.LittleEndian.Uint16(frame[off:])
		off += 2
		byteLen := int(binary.LittleEndian.Uint32(frame[off:]))
		off += 4
		units := make([]uint16, 0, byteLen/2)
		for j := 0; j+1 < byteLen; j += 2 {
			if u := binary.LittleEndian.Uint16(frame[off+j:]); u != 0 { // 尾 NUL 不入
				units = append(units, u)
			}
		}
		off += byteLen
		marks = append(marks, mark)
		labels = append(labels, string(utf16.Decode(units)))
	}
	if off != len(frame) {
		t.Fatalf("走查终点 %d ≠ 帧长 %d", off, len(frame))
	}
	if marks[0] != npcMarkShop || labels[0] != "商店" {
		t.Errorf("商店项异常: mark=%x label=%q", marks[0], labels[0])
	}
	if marks[n-1] != 0 || labels[n-1] != "离开" {
		t.Errorf("尾项应为离开: mark=%x label=%q", marks[n-1], labels[n-1])
	}
	// 无服务兜底「交谈」+离开。
	bare, _ := npcServerMenu(1, 1, nil)
	if got := binary.LittleEndian.Uint16(bare[9:]); got != 2 {
		t.Errorf("无服务兜底项数 = %d, want 2", got)
	}
}

func TestNPCObjectRequestFlow(t *testing.T) {
	s := newNpcTestServer(t)
	pl := &Player{Conn1: 42, State: PlOnline}
	pl.npcMu.Lock()
	pl.npcScene = "born03"
	pl.npcMu.Unlock()

	catalog := s.sceneNPCCatalog("born03")
	if len(catalog.spawns) == 0 {
		t.Skip("born03 真档缺席")
	}
	// 找一个非战斗类 spawn（白名单目录里全是非 attack 类）。
	target := catalog.spawns[0]
	if target.Script == "attacknpc" || target.Script == "bossnpc" {
		for _, spawn := range catalog.spawns {
			target = spawn
			break
		}
	}
	// 首段：弹菜单 + 簿记。
	if !s.handleNPCObjectRequest(pl, parseReq(t, target.ObjectID, target.OwnerID, 0)) {
		t.Fatal("首段应消费")
	}
	npcMenuMu.Lock()
	menu := npcMenuState[pl.Conn1]
	npcMenuMu.Unlock()
	if menu == nil || menu.objectID != target.ObjectID {
		t.Fatalf("菜单簿记异常: %+v", menu)
	}
	// 第二段：离开项 → 关闭并清簿记。
	if !s.handleNPCObjectRequest(pl, parseReq(t, target.ObjectID, target.OwnerID, 600000000)) {
		t.Fatal("离开应消费")
	}
	npcMenuMu.Lock()
	gone := npcMenuState[pl.Conn1]
	npcMenuMu.Unlock()
	if gone != nil {
		t.Error("离开后菜单簿记未清")
	}
	// 未知对象：静默消费不炸。
	if !s.handleNPCObjectRequest(pl, parseReq(t, 0x99, 0x99, 0)) {
		t.Fatal("未知对象应静默消费")
	}
}

func parseReq(t *testing.T, objectID, ownerID uint32, funcID int32) npcObjectRequest {
	t.Helper()
	req, ok := parseNPCObjectRequest(buildObjectRequest(objectID, ownerID, funcID))
	if !ok {
		t.Fatal("构造请求失败")
	}
	return req
}

func TestEntryRouteObjectRequest(t *testing.T) {
	s := newNpcTestServer(t)
	pl := s.players.AddPlayer(5, 1, "tester")
	s.players.SetPlayerState(pl, PlOnline)
	pl.npcMu.Lock()
	pl.npcScene = "born03"
	pl.npcMu.Unlock()
	catalog := s.sceneNPCCatalog("born03")
	if len(catalog.spawns) == 0 {
		t.Skip("born03 真档缺席")
	}
	target := catalog.spawns[len(catalog.spawns)/2]
	// entry 内网包：8B 头{clientID,serial} + 33B op0x07 完整帧。
	frame := buildObjectRequest(target.ObjectID, target.OwnerID, 0)
	payload := make([]byte, 8, 8+len(frame))
	binary.LittleEndian.PutUint32(payload[0:], 5)
	binary.LittleEndian.PutUint32(payload[4:], 1)
	payload = append(payload, frame...)
	if !s.ProcessEntryMsg(0x100, 3, payload) {
		t.Fatal("entry 消息未被消费")
	}
	npcMenuMu.Lock()
	menu := npcMenuState[pl.Conn1]
	npcMenuMu.Unlock()
	if menu == nil || menu.objectID != target.ObjectID {
		t.Fatalf("entry 全路径未弹菜单: %+v", menu)
	}
}

func TestNPCSelectTarget(t *testing.T) {
	s := newNpcTestServer(t)
	pl := &Player{Conn1: 7, State: PlOnline}
	message := entergen.CustomInbound{Opcode: 0x1E, Values: []entergen.CustomInboundValue{
		{Type: 2, Int32: 431},
		{Type: 8}, // object：Raw 为 packed 8B
	}}
	message.Values[1].Raw[0] = 5 // obj 低 32 位 5（其余零）
	s.handleNPCSelectTarget(pl, message)
	message.Values[1].Raw[0] = 0 // 清零目标分支
	s.handleNPCSelectTarget(pl, message)
}

func TestNPCTalkMenuM5(t *testing.T) {
	// 官方双源定版：每窗必带 m5 [5, "task_<ConfigID>"]（女号抓包 88.2s
	// "task_funnpcborn003031" / npc.pcapng "task_WorldNpc01230"）——缺它
	// 客户端不开窗。帧序 m4 → m5 → m6×N（尾 ui_exit）→ m7。
	frames, err := npcCinematicTalkMenu(0x43, 0x3500003E, "funnpcborn003031", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 4 {
		t.Fatalf("无服务菜单帧数 = %d, want 4（m4/m5/m6 离开/m7）", len(frames))
	}
	for i, want := range []byte{4, 5, 6, 7} {
		if id := binary.LittleEndian.Uint32(frames[i][4:]); id != uint32(want) {
			t.Fatalf("帧 %d 消息号 = %d, want %d", i, id, want)
		}
	}
	if !bytes.Contains(frames[1], []byte("task_funnpcborn003031")) {
		t.Fatalf("m5 缺任务链键: % x", frames[1])
	}
	if !bytes.Contains(frames[2], []byte("ui_exit")) {
		t.Fatalf("尾行缺 ui_exit: % x", frames[2])
	}
}
