package fssoloworld

// npcinteract_test.go NPC 交互链单测：object request 解析/服务列/菜单
// 编码/选中与菜单路由（帧出口经 inner 广播，断言面=菜单簿记与编码器）。

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/lizhanyin/9yin-server-go/internal/config"
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
	// 无业报模板的 NPC（引导 NPC）不发「赠送物品/侠恶查询」通用行：官服
	// funnpcborn003031 主窗只有 ui_exit 一条（引导抓包 98.16s）。
	frames, err := npcCinematicTalkMenu(0x43, 0x3500003E, "funnpcborn003031", nil, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 4 {
		t.Fatalf("无服务无业报菜单帧数 = %d, want 4（m4/m5/离开/m7）", len(frames))
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
	// 业报 NPC：赠送/侠恶通用行两行在服务行之后、ui_exit 之前（npc.pcapng
	// 30.275s 官方字节模板）。
	frames, err = npcCinematicTalkMenu(0x0E00147A, 0x14AB, "WorldNpc01230", nil, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	// m4/m5/[服务行×0]/赠送/侠恶/离开/m7。
	if len(frames) != 6 {
		t.Fatalf("业报菜单帧数 = %d, want 6", len(frames))
	}
	for i, want := range []byte{4, 5, 6, 6, 6, 7} {
		if id := binary.LittleEndian.Uint32(frames[i][4:]); id != uint32(want) {
			t.Fatalf("帧 %d 消息号 = %d, want %d", i, id, want)
		}
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

// officialGuideMenu 官服引导抓包（初始账号登陆-完整引导-女.pcapng 98.16s）
// funnpcborn003031 首次点击的应答五帧明文（EE 去转义 + XOR 后）：m4 /
// m5 "task_funnpcborn003031" / m6 [500001340, story_menu_482, title_1340] /
// m6 [600000000, ui_exit] / m7(obj=3500003e own=00000043)。
var officialGuideMenu = []string{
	"1e01000204000000",
	"1e0200020500000006160000007461736b5f66756e6e7063626f726e30303330333100",
	"1e04000206000000023c6acd1d060f00000073746f72795f6d656e755f34383200060b0000007469746c655f3133343000",
	"1e03000206000000020046c323060800000075695f6578697400",
	"1e02000207000000083e00003543000000",
}

func TestNPCGuideDialogMatchesOfficial(t *testing.T) {
	// 引导 NPC 首次对话：1340 是链根（无前置/无接取文案/非链式目标）——
	// 服务端直接接取，主窗即进行中行；既无接取行也无通用行。整组帧与
	// 官服抓包逐字节一致（任务目录走 testdata/ini/Quest 真档）。
	s := newNpcTestServer(t)
	catalog := s.npcTaskCatalogOf()
	if catalog == nil {
		t.Fatal("任务目录缺席（ini/Quest XML）")
	}
	pl := &Player{Conn1: 11, State: PlOnline}
	if added := s.autoAcceptChainStarts(catalog, pl, "funnpcborn003031"); len(added) != 1 || added[0] != 1340 {
		t.Fatalf("链根首谈即接 = %v, want [1340]", added)
	}
	if !pl.taskAcceptedContains(1340) {
		t.Fatalf("1340 未入已接: %+v", pl.roleState.TaskAccepted)
	}
	if again := s.autoAcceptChainStarts(catalog, pl, "funnpcborn003031"); len(again) != 0 {
		t.Fatalf("链根接取应幂等: %v", again)
	}
	lines := s.npcTaskLines(catalog, pl, "funnpcborn003031", 8)
	if len(lines) != 1 || lines[0].TaskID != 1340 || lines[0].StoryID != "482" || lines[0].Submit {
		t.Fatalf("1340 任务行 = %+v, want {1340 StoryID=482 进行中}", lines)
	}
	frames, err := npcCinematicTalkMenu(0x3500003E, 0x43, "funnpcborn003031", nil, lines, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != len(officialGuideMenu) {
		t.Fatalf("帧数 = %d, want %d", len(frames), len(officialGuideMenu))
	}
	for i, want := range officialGuideMenu {
		raw, _ := hex.DecodeString(want)
		if !bytes.Equal(frames[i], raw) {
			t.Fatalf("帧 %d 与官服不一致:\n got %x\nwant %x", i, frames[i], raw)
		}
	}
}

// npcTemplateRowsOf 真档 NpcConfig 行（按 ID 小写索引）——供业报门控对拍。
func npcTemplateRowsOf(t *testing.T, rel ...string) map[string]map[string]string {
	t.Helper()
	out := map[string]map[string]string{}
	for _, r := range rel {
		doc, err := config.LoadObjectFile(filepath.Join("..", "..", "testdata", "ini", "Npc", "NpcConfig", filepath.FromSlash(r)))
		if err != nil {
			t.Fatalf("装载 %s: %v", r, err)
		}
		for _, row := range doc.Rows {
			out[strings.ToLower(row["ID"])] = row
		}
	}
	return out
}

func TestNPCKarmaDialogRule(t *testing.T) {
	// 模板业报列门控：Character 有值且 KarmaLevel≥1；捕快类 235/236 走专属行。
	for _, tc := range []struct {
		tpl  map[string]string
		want bool
	}{
		{map[string]string{"Character": "1", "KarmaLevel": "3"}, true},
		{map[string]string{"Character": "4", "KarmaLevel": "1", "NpcType": "96"}, true},
		{map[string]string{"Character": "1", "KarmaLevel": "0"}, false},
		{map[string]string{"Character": "1"}, false},
		{map[string]string{"KarmaLevel": "5"}, false},
		{map[string]string{"Character": "1", "KarmaLevel": "6", "NpcType": "235"}, false},
		{map[string]string{"Character": "1", "KarmaLevel": "6", "NpcType": "236"}, false},
		{map[string]string{"Character": "abc", "KarmaLevel": "x"}, false},
		{nil, false},
	} {
		if got := npcKarmaDialog(tc.tpl); got != tc.want {
			t.Fatalf("npcKarmaDialog(%v) = %v, want %v", tc.tpl, got, tc.want)
		}
	}
	// 官服抓包对拍（npc-点击对话.pcapng 81 窗 + npc.pcapng）：期望值＝
	// 官服是否发了「赠送物品/侠恶查询」两行。FuncNpc00644（114，业报
	// 1/2）在车夫抓包（另一角色）里未发两行，属角色侧差异，未列入。
	rows := npcTemplateRowsOf(t, "WorldNpc/city_CommonNpc.xml", "WorldNpc/born_CommonNpc.xml", "CloneNpc/CommonNpc.xml")
	for id, want := range map[string]bool{
		"worldnpc01230": true, "worldnpc01082": true, "worldnpc01108": true,
		"worldnpc01635": true, "funcnpc00107": true, "funcnpc00102": true, "funcnpc00111": true,
		"transcity05d": false, "jinybnpc001": false, "hd_dw_001": false,
		"worldnpc01244": false, "worldnpc01245": false,
		"yqd_npc_001": false, "funnpcborn003031": false,
	} {
		row, ok := rows[id]
		if !ok {
			t.Fatalf("真档缺模板 %s", id)
		}
		if got := npcKarmaDialog(row); got != want {
			t.Fatalf("%s 通用行门控 = %v, want %v（Character=%q KarmaLevel=%q NpcType=%q）",
				id, got, want, row["Character"], row["KarmaLevel"], row["NpcType"])
		}
	}
}

func TestNPCTaskChainProgression(t *testing.T) {
	// 提交受理 = 移出已接 + 记已交 + 按AutoAcceptID 链式接取；链根任务
	// 交过之后不再「首谈即接」；链式目标只随前序任务接入。
	s := newNpcTestServer(t)
	catalog := s.npcTaskCatalogOf()
	if catalog == nil {
		t.Fatal("任务目录缺席（ini/Quest XML）")
	}
	pl := &Player{Conn1: 12, State: PlOnline}
	// 引导链：1340（剧情走完自动提交）→ 1347 → 1341。
	s.completeNPCTask(pl, 1340, "剧情走完自动提交")
	if pl.taskAcceptedContains(1340) || !npcCompletedSet(pl)[1340] || !pl.taskAcceptedContains(1347) {
		t.Fatalf("1340 提交后状态异常: accepted=%+v completed=%+v", pl.roleState.TaskAccepted, pl.roleState.TaskCompleted)
	}
	s.completeNPCTask(pl, 1347, "提交")
	if !pl.taskAcceptedContains(1341) || len(pl.roleState.TaskAccepted) != 1 {
		t.Fatalf("1347 提交后链推进异常: %+v", pl.roleState.TaskAccepted)
	}
	// 已交的链根不再自动接取；重复提交不重复记已交。
	if added := s.autoAcceptChainStarts(catalog, pl, "funnpcborn003031"); len(added) != 0 {
		t.Fatalf("已交链根不应再接: %v", added)
	}
	s.completeNPCTask(pl, 1340, "重复提交")
	if n := len(pl.roleState.TaskCompleted); n != 2 {
		t.Fatalf("已交记录应去重（1340/1347 各一条）: %+v", pl.roleState.TaskCompleted)
	}
	// 车夫：1648 是链根（首谈即接）；27509/27510 无接取文案，只随前序
	// 任务提交链式接入（提交前不出行）——官服车夫窗进行中行只有 1648。
	// 50 有接取文案但前置 27510 未交：前置门未建模（已交历史不可追溯），
	// 仍出接取行，属已知偏差。
	driver := &Player{Conn1: 13, State: PlOnline}
	if added := s.autoAcceptChainStarts(catalog, driver, "Transcity05D"); len(added) != 1 || added[0] != 1648 {
		t.Fatalf("车夫链根接取 = %v, want [1648]", added)
	}
	if lines := s.npcTaskLines(catalog, driver, "Transcity05D", 8); taskLineOf(lines, 1648) == nil || taskLineOf(lines, 1648).StoryID != "244" ||
		taskLineOf(lines, 27509) != nil || taskLineOf(lines, 27510) != nil {
		t.Fatalf("车夫任务行 = %+v, want 1648 进行中且无 27509/27510 行", lines)
	}
	s.completeNPCTask(driver, 1648, "提交")
	if !driver.taskAcceptedContains(27509) {
		t.Fatalf("1648 提交后应链式接取 27509: %+v", driver.roleState.TaskAccepted)
	}
	if lines := s.npcTaskLines(catalog, driver, "Transcity05D", 8); taskLineOf(lines, 1648) != nil ||
		taskLineOf(lines, 27509) == nil || taskLineOf(lines, 27509).StoryID != "604" {
		t.Fatalf("1648 交后 27509 应发进行中行（剧情组 604）: %+v", lines)
	}
	s.completeNPCTask(driver, 27509, "提交")
	if !driver.taskAcceptedContains(27510) {
		t.Fatalf("27509 提交后应链式接取 27510: %+v", driver.roleState.TaskAccepted)
	}
}

// taskLineOf 任务行集里指定任务的行（无则 nil）。
func taskLineOf(lines []npcTaskLine, taskID int) *npcTaskLine {
	for i := range lines {
		if lines[i].TaskID == taskID {
			return &lines[i]
		}
	}
	return nil
}

func TestNPCTaskMenuLines(t *testing.T) {
	// 任务行域定版（form_talk_data.lua g_menu_id + 引导抓包）：
	// 接取 100*base+TID / 提交 200*base+TID / 进行中 500*base+TID（有
	// 剧情段且未走完）；行文本双键。已接+剧情走完/无剧情段翻提交行；
	// 未接+无接取文案不出行（不可手动接取）；已交不出行。
	catalog := taskdata.NewCatalog(
		map[int]taskdata.Task{
			101: {ID: 101, Title: "寻亲少年"},
			102: {ID: 102, Title: "初入江湖", CanAcceptMenu: "<font>初入江湖（可承接）</font>"}, // 快照链：有接取文案→折算接取键
			105: {ID: 105, Title: "无接取文案"},                                         // 无接取文案：不可手动接取
		},
		[]taskdata.Binding{
			{TaskID: 101, NPCConfigID: "WorldNpc00009"},
			{TaskID: 102, NPCConfigID: "worldnpc00009"}, // 大小写不敏感去重
			{TaskID: 103, NPCConfigID: "WorldNpc00009"},
			{TaskID: 104, NPCConfigID: "WorldNpc00009"}, // 无文本不计
			{TaskID: 105, NPCConfigID: "WorldNpc00009"},
		}, nil)
	s := newNpcTestServer(t)
	pl := &Player{Conn1: 3, State: PlOnline}
	pl.roleState.TaskAccepted = []uint32{101}
	lines := s.npcTaskLines(catalog, pl, "WorldNpc00009", 8)
	if len(lines) != 2 {
		t.Fatalf("任务行 = %d, want 2（103/104 无文本、105 无接取文案不计）: %+v", len(lines), lines)
	}
	if lines[0].TaskID != 101 || !lines[0].Submit {
		t.Fatalf("行 0 = %+v, want {101 Submit 无剧情段直发提交行}", lines[0])
	}
	if lines[1].TaskID != 102 || lines[1].Submit || lines[1].StoryID != "" {
		t.Fatalf("行 1 = %+v, want {102 接取}", lines[1])
	}
	// 有剧情段：已接未走完=500 行；走完=200 行。
	storyTasks := map[int]taskdata.Task{205: {ID: 205, Title: "剧情任务"}}
	story2 := taskdata.NewCatalog(storyTasks, []taskdata.Binding{
		{TaskID: 205, NPCConfigID: "F", StoryID: "482", ContextID: "story_482_1", TitleID: "storymenu_482_1"},
		{TaskID: 205, NPCConfigID: "F", StoryID: "482", ContextID: "story_482_2", TitleID: "storymenu_482_2"},
	}, nil)
	pl2 := &Player{Conn1: 4, State: PlOnline}
	pl2.roleState.TaskAccepted = []uint32{205}
	got := s.npcTaskLines(story2, pl2, "F", 8)
	if len(got) != 1 || got[0].StoryID != "482" || got[0].Submit {
		t.Fatalf("剧情未走完 = %+v, want {205 StoryID=482}", got)
	}
	pl2.taskStoryDone = map[uint32]bool{205: true}
	got = s.npcTaskLines(story2, pl2, "F", 8)
	if len(got) != 1 || !got[0].Submit {
		t.Fatalf("剧情走完 = %+v, want {205 Submit}", got)
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
	// 500 进行中行双键：story_menu_<sid> + title_<TID>。
	doing, err := npcTaskMenuFrames([]npcTaskLine{{TaskID: 205, StoryID: "482"}})
	if err != nil || len(doing) != 1 {
		t.Fatalf("500 行编码: %v %+v", err, doing)
	}
	if !bytes.Contains(doing[0], []byte("story_menu_482")) || !bytes.Contains(doing[0], []byte("title_205")) {
		t.Fatalf("500 行缺双键: % x", doing[0])
	}
	if funcid := binary.LittleEndian.Uint32(doing[0][9:]); funcid != 500000205 {
		t.Fatalf("500 行 funcid = %d, want 500000205", funcid)
	}
	if !bytes.Contains(frames[1], []byte("CanAcceptMenu_102")) || !bytes.Contains(frames[1], []byte("title_102")) {
		t.Fatalf("接取行缺双键: % x", frames[1])
	}
	if funcid := binary.LittleEndian.Uint32(frames[1][9:]); funcid != 100000102 {
		t.Fatalf("接取行 funcid = %d, want 100000102", funcid)
	}
	// 上限截断与关停；已交任务不再出行。
	fresh := &Player{Conn1: 5, State: PlOnline}
	if got := len(s.npcTaskLines(catalog, fresh, "WorldNpc00009", 1)); got != 1 {
		t.Fatalf("截断后 = %d, want 1", got)
	}
	if got := s.npcTaskLines(catalog, fresh, "WorldNpc00009", -1); got != nil {
		t.Fatalf("负上限应关停: %+v", got)
	}
	pl.roleState.TaskCompleted = []uint32{102}
	if got := s.npcTaskLines(catalog, pl, "WorldNpc00009", 8); len(got) != 1 || got[0].TaskID != 101 {
		t.Fatalf("已交的 102 不应再出行: %+v", got)
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
	// 接取→已接列表增长+菜单刷新；提交→移出已接+记已交。0x29 重放走
	// haveRoleState 门（无角色态静默跳过，不 panic）。任务目录走服务端
	// XML 配置（testdata/ini/Quest）：取有接取文案且无链式后继的真实
	// 绑定任务，首点 100 走接取窗路径，不落「无文案直接接取」回退。
	s := newNpcTestServer(t)
	catalog := s.npcTaskCatalogOf()
	if catalog == nil {
		t.Fatal("任务目录缺席（ini/Quest XML）")
	}
	tid := 0
	for _, task := range catalog.TasksForNPC("WorldNpc00009") {
		if task.AcceptMenuKey != "" && task.AutoAcceptID == 0 {
			tid = task.ID
			break
		}
	}
	if tid == 0 {
		t.Fatal("WorldNpc00009 无可手动接取的绑定任务（XML 数据异常）")
	}
	pl := &Player{Conn1: 9, State: PlOnline}
	menu := &npcActiveMenu{objectID: 5, ownerID: 6}
	// 接取状态机：首次 100=推接取窗（不写已接），同 TID 二次=真接取。
	s.handleNPCTaskMark(pl, menu, npcTaskFlagAccept*npcTaskFuncBase+int32(tid))
	if pl.taskAcceptedContains(uint32(tid)) {
		t.Fatalf("首次 100 不应直接接取: %+v", pl.roleState.TaskAccepted)
	}
	s.handleNPCTaskMark(pl, menu, npcTaskFlagAccept*npcTaskFuncBase+int32(tid))
	if !pl.taskAcceptedContains(uint32(tid)) || len(pl.roleState.TaskAccepted) != 1 {
		t.Fatalf("二次确认后已接 = %+v", pl.roleState.TaskAccepted)
	}
	// 幂等：重复接取不重复追加。
	s.handleNPCTaskMark(pl, menu, npcTaskFlagAccept*npcTaskFuncBase+int32(tid))
	if len(pl.roleState.TaskAccepted) != 1 {
		t.Fatalf("重复接取后已接 = %d, want 1", len(pl.roleState.TaskAccepted))
	}
	s.handleNPCTaskMark(pl, menu, npcTaskFlagSubmit*npcTaskFuncBase+int32(tid))
	if pl.taskAcceptedContains(uint32(tid)) || len(pl.roleState.TaskAccepted) != 0 || !npcCompletedSet(pl)[uint32(tid)] {
		t.Fatalf("提交后 accepted=%+v completed=%+v（应移出已接并记已交）", pl.roleState.TaskAccepted, pl.roleState.TaskCompleted)
	}
	// 201 域（奖励单选二次确认）同样按提交受理。
	pl.roleState.TaskAccepted = append(pl.roleState.TaskAccepted, uint32(tid))
	s.handleNPCTaskMark(pl, menu, npcTaskFlagSubmitChoice*npcTaskFuncBase+int32(tid))
	if pl.taskAcceptedContains(uint32(tid)) {
		t.Fatalf("201 提交后已接 = %+v", pl.roleState.TaskAccepted)
	}
	// 未受理任务域（300 放弃）不改状态。
	s.handleNPCTaskMark(pl, menu, 300*npcTaskFuncBase+int32(tid))
	if len(pl.roleState.TaskAccepted) != 0 {
		t.Fatalf("放弃域不应改状态: %+v", pl.roleState.TaskAccepted)
	}
}

func TestNPCServicesNoDataNoLine(t *testing.T) {
	// 无数据即无功能行（2026-11-23 真机定案：NpcType 结构域回落已删——
	// 车夫等无业务列 NPC 不弹虚构入口；npc_type_define.lua 的 62 铁匠铺/
	// 61 武馆/122 生活职业/45 马车船只类型号本身不再派生服务行）。
	for _, typeVal := range []string{"62", "66", "61", "122", "45", "84", "1", "abc", ""} {
		if services := npcServicesOf(map[string]string{"NpcType": typeVal}); len(services) != 0 {
			t.Fatalf("NpcType=%s 无业务列不应出服务行: %+v", typeVal, services)
		}
	}
	// 业务列权威（"0" 值不算配置）。
	services := npcServicesOf(map[string]string{"ShopID": "Shop_x", "NpcType": "62"})
	if len(services) != 1 || services[0].value != "Shop_x" || services[0].mark != npcMarkShop {
		t.Fatalf("商店列异常: %+v", services)
	}
	services = npcServicesOf(map[string]string{"DepotID": "1", "NpcType": "67"})
	if len(services) != 1 || services[0].mark != npcMarkDepot {
		t.Fatalf("仓库列异常: %+v", services)
	}
	if services := npcServicesOf(map[string]string{"ShopID": "0"}); len(services) != 0 {
		t.Fatalf("ShopID=0 不算配置: %+v", services)
	}
}

func TestNPCTalkMenuServiceLines(t *testing.T) {
	// 武馆 NPC（业务列 ShopID）：服务行置于赠送/侠恶通用行之前、
	// ui_exit 之前。帧序 m4/m5/m6×4/m7（Trainer 映射保留——
	// 860000047/ui_skill_trainer 的 mark 域供后续训练师数据接入）。
	services := npcServicesOf(map[string]string{"ShopID": "Shop_jn_jh_00100"})
	if len(services) != 1 || services[0].mark != npcMarkShop {
		t.Fatalf("商店列服务异常: %+v", services)
	}
	if npcTalkFuncID(npcMarkTrainer) != 860000047 || npcTalkTextID(npcMarkTrainer) != "ui_skill_trainer" {
		t.Fatal("Trainer mark 映射漂移")
	}
	frames, err := npcCinematicTalkMenu(0x0E000104, 0x1010, "FuncNpc00058", services, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 7 {
		t.Fatalf("服务菜单帧数 = %d, want 7", len(frames))
	}
	for i, want := range []byte{4, 5, 6, 6, 6, 6, 7} {
		if id := binary.LittleEndian.Uint32(frames[i][4:]); id != uint32(want) {
			t.Fatalf("帧 %d 消息号 = %d, want %d", i, id, want)
		}
	}
	if !bytes.Contains(frames[2], []byte("ui_shop")) {
		t.Fatalf("m6 服务行缺 ui_shop: % x", frames[2])
	}
	if !bytes.Contains(frames[1], []byte("task_FuncNpc00058")) {
		t.Fatalf("m5 链键异常: % x", frames[1])
	}
	// 回执命名（trace 观测面）。
	if name := npcServiceNameByFuncID(860000047); name != "技能训练" {
		t.Fatalf("funcid 860000047 名 = %q", name)
	}
	if name := npcServiceNameByFuncID(805000000); name != "商店" {
		t.Fatalf("funcid 805000000 名 = %q", name)
	}
}
