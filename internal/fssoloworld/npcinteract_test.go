package fssoloworld

// npcinteract_test.go NPC 交互链单测：object request 解析/服务列/菜单
// 编码/选中与菜单路由（帧出口经 inner 广播，断言面=菜单簿记与编码器）。

import (
	"bytes"
	"encoding/binary"
	"testing"
	"unicode/utf16"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/taskdata"
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
	frames, err := npcCinematicTalkMenu(0x43, 0x3500003E, "funnpcborn003031", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	// m4/m5/[服务行×0]/赠送/侠恶/离开/m7。
	if len(frames) != 6 {
		t.Fatalf("无服务菜单帧数 = %d, want 6", len(frames))
	}
	for i, want := range []byte{4, 5, 6, 6, 6, 7} {
		if id := binary.LittleEndian.Uint32(frames[i][4:]); id != uint32(want) {
			t.Fatalf("帧 %d 消息号 = %d, want %d", i, id, want)
		}
	}
	if !bytes.Contains(frames[1], []byte("task_funnpcborn003031")) {
		t.Fatalf("m5 缺任务链键: % x", frames[1])
	}
	if !bytes.Contains(frames[2], []byte("ui_present_to_npc_item")) {
		t.Fatalf("缺赠送通用行: % x", frames[2])
	}
	if !bytes.Contains(frames[3], []byte("ui_karma_query")) {
		t.Fatalf("缺侠恶通用行: % x", frames[3])
	}
	if !bytes.Contains(frames[4], []byte("ui_exit")) {
		t.Fatalf("尾行缺 ui_exit: % x", frames[4])
	}
}

func TestNPCTaskMenuLines(t *testing.T) {
	// 任务行域定版（form_talk_data.lua g_menu_id）：接取 100*base+TID、
	// 提交 200*base+TID；行文本键 title_<TID>。已接任务翻提交行。
	catalog := taskdata.NewCatalog(
		map[int]taskdata.Task{101: {ID: 101, Title: "寻亲少年"}, 102: {ID: 102, Title: "初入江湖"}},
		[]taskdata.Binding{
			{TaskID: 101, NPCConfigID: "WorldNpc00009"},
			{TaskID: 102, NPCConfigID: "worldnpc00009"}, // 大小写不敏感去重
			{TaskID: 103, NPCConfigID: "WorldNpc00009"},
			{TaskID: 104, NPCConfigID: "WorldNpc00009"}, // 无文本不计
		}, nil)
	lines := npcTaskLines(catalog, "WorldNpc00009", map[uint32]bool{101: true}, 8)
	if len(lines) != 2 {
		t.Fatalf("任务行 = %d, want 2（103/104 无文本不计）: %+v", len(lines), lines)
	}
	if lines[0].TaskID != 101 || !lines[0].Submit {
		t.Fatalf("行 0 = %+v, want {101 Submit}", lines[0])
	}
	if lines[1].TaskID != 102 || lines[1].Submit {
		t.Fatalf("行 1 = %+v, want {102 接取}", lines[1])
	}
	frames, err := npcTaskMenuFrames(lines)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 2 {
		t.Fatalf("帧数 = %d, want 2", len(frames))
	}
	if id := binary.LittleEndian.Uint32(frames[0][4:]); id != 6 {
		t.Fatalf("消息号 = %d, want 6", id)
	}
	if funcid := binary.LittleEndian.Uint32(frames[0][9:]); funcid != 200000101 {
		t.Fatalf("提交行 funcid = %d, want 200000101", funcid)
	}
	if !bytes.Contains(frames[0], []byte("CompleteMenu_101")) || !bytes.Contains(frames[0], []byte("title_101")) {
		t.Fatalf("提交行缺双键: % x", frames[0])
	}
	if !bytes.Contains(frames[1], []byte("CanAcceptMenu_102")) || !bytes.Contains(frames[1], []byte("title_102")) {
		t.Fatalf("接取行缺双键: % x", frames[1])
	}
	if funcid := binary.LittleEndian.Uint32(frames[1][9:]); funcid != 100000102 {
		t.Fatalf("接取行 funcid = %d, want 100000102", funcid)
	}
	// 上限截断与关停。
	if got := len(npcTaskLines(catalog, "WorldNpc00009", nil, 1)); got != 1 {
		t.Fatalf("截断后 = %d, want 1", got)
	}
	if got := npcTaskLines(catalog, "WorldNpc00009", nil, -1); got != nil {
		t.Fatalf("负上限应关停: %+v", got)
	}
}

func TestNPCTaskMarkSplit(t *testing.T) {
	for _, tc := range []struct {
		mark   int32
		flag   int32
		taskID int
		ok     bool
	}{
		{100000101, 100, 101, true},
		{200000101, 200, 101, true},
		{600000000, 600, 0, false},  // 离开项 TID=0 非任务行
		{898000024, 898, 24, false}, // 通用行域外
		{805000000, 805, 0, false},  // 商店域外
		{99999999, 0, 0, false},     // 低位
	} {
		flag, taskID, ok := npcTaskMarkSplit(tc.mark)
		if ok != tc.ok || (ok && (flag != tc.flag || taskID != tc.taskID)) {
			t.Fatalf("mark %d 拆解 = (%d,%d,%v), want (%d,%d,%v)", tc.mark, flag, taskID, ok, tc.flag, tc.taskID, tc.ok)
		}
	}
}

func TestNPCTaskMarkAcceptSubmit(t *testing.T) {
	// 接取→已接列表增长+菜单刷新；提交→移除。0x29 重放走 haveRoleState
	// 门（无角色态静默跳过，不 panic）。
	s := newNpcTestServer(t)
	pl := &Player{Conn1: 9, State: PlOnline}
	menu := &npcActiveMenu{objectID: 5, ownerID: 6}
	s.handleNPCTaskMark(pl, menu, 100000301)
	if !pl.taskAcceptedContains(301) || len(pl.roleState.TaskAccepted) != 1 {
		t.Fatalf("接取后已接 = %+v", pl.roleState.TaskAccepted)
	}
	// 幂等：重复接取不重复追加。
	s.handleNPCTaskMark(pl, menu, 100000301)
	if len(pl.roleState.TaskAccepted) != 1 {
		t.Fatalf("重复接取后已接 = %d, want 1", len(pl.roleState.TaskAccepted))
	}
	s.handleNPCTaskMark(pl, menu, 200000301)
	if pl.taskAcceptedContains(301) || len(pl.roleState.TaskAccepted) != 0 {
		t.Fatalf("提交后已接 = %+v", pl.roleState.TaskAccepted)
	}
	// 201 域（奖励单选二次确认）同样按提交受理。
	pl.roleState.TaskAccepted = append(pl.roleState.TaskAccepted, 302)
	s.handleNPCTaskMark(pl, menu, 201000302)
	if pl.taskAcceptedContains(302) {
		t.Fatalf("201 提交后已接 = %+v", pl.roleState.TaskAccepted)
	}
	// 未受理任务域（300 放弃）不改状态。
	s.handleNPCTaskMark(pl, menu, 300000301)
	if len(pl.roleState.TaskAccepted) != 0 {
		t.Fatalf("放弃域不应改状态: %+v", pl.roleState.TaskAccepted)
	}
}
