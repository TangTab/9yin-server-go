package fssoloworld

// npcinteract_test.go NPC 交互链单测：object request 解析/服务列/菜单
// 编码/选中与菜单路由（帧出口经 inner 广播，断言面=菜单簿记与编码器）。

import (
	"encoding/binary"
	"testing"

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

func TestNPCServicesAndMenuItems(t *testing.T) {
	services := npcServicesOf(map[string]string{
		"ShopID": "Shop_yaopin_00101", "DepotID": "0", "Job": "3",
	})
	if len(services) != 2 {
		t.Fatalf("服务列 = %d, want 2（DepotID=0 不算）", len(services))
	}
	items := npcTalkMenuItems(services)
	if len(items) != 3 { // 商店+生活职业+离开
		t.Fatalf("菜单项 = %d, want 3", len(items))
	}
	if items[len(items)-1].funcID != 600000000 || items[len(items)-1].textID != "ui_form_talk_leave" {
		t.Error("尾项应为离开")
	}
	if items[0].funcID != 805000000 || items[0].textID != "ui_shop" {
		t.Errorf("商店项映射异常: %+v", items[0])
	}
	// 无服务兜底「交谈」。
	bare := npcTalkMenuItems(nil)
	if len(bare) != 2 || bare[0].funcID != 860000001 || bare[0].textID != "ui_talk" {
		t.Errorf("无服务兜底异常: %+v", bare)
	}
}

func TestNPCCinematicTalkMenu(t *testing.T) {
	frames, err := npcCinematicTalkMenu(0x11000005, 0x00660005, npcTalkMenuItems(nil))
	if err != nil {
		t.Fatalf("编码: %v", err)
	}
	if len(frames) != 4 { // begin + 2 项 + end
		t.Fatalf("帧数 = %d, want 4", len(frames))
	}
	for i, frame := range frames {
		if frame[0] != 0x1E {
			t.Errorf("帧 %d 非 0x1E: %#x", i, frame[0])
		}
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
