package fsgame

import (
	"os"
	"path/filepath"
	"testing"
)

// assemble2 装配第一批 + 第二批（社交底座）十六模块。
func assemble2(t *testing.T) (*Kernel, []Module) {
	t.Helper()
	td := filepath.Join("..", "..", "testdata")
	lc := mustRead(t, filepath.Join(td, "logic_class.xml"))
	px := mustRead(t, filepath.Join(td, "struct", "player", "player.xml"))
	ex := mustRead(t, filepath.Join(td, "struct", "player", "exp.xml"))

	k := NewKernel()
	k.ResourcePath = td
	if err := k.LoadLogicClassXML(lc); err != nil {
		t.Fatalf("LoadLogicClassXML: %v", err)
	}
	if err := k.LoadStruct("player", px); err != nil {
		t.Fatalf("LoadStruct player: %v", err)
	}
	if err := k.LoadStruct("player", ex); err != nil {
		t.Fatalf("LoadStruct player(exp): %v", err)
	}
	var r Registry
	if err := RegisterFirstBatch(&r); err != nil {
		t.Fatal(err)
	}
	if err := RegisterSecondBatch(&r); err != nil {
		t.Fatal(err)
	}
	mods, err := r.Assemble(k)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	return k, mods
}

func mustRead(t *testing.T, p string) []byte {
	t.Helper()
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("读 %s: %v", p, err)
	}
	return data
}

// newPlayer2 建玩家并跑登录链到 OnReady（社交面前的就绪态）。
func newPlayer2(t *testing.T, k *Kernel, name string) PersistID {
	t.Helper()
	id, err := k.CreateObject("player")
	if err != nil {
		t.Fatalf("CreateObject: %v", err)
	}
	if err := k.SetString(id, "name", name); err != nil {
		t.Fatalf("SetString name: %v", err)
	}
	_ = k.SetInt(id, "Level", 10)
	_ = k.SetInt(id, "Job", 2)
	for _, ev := range []string{"OnRecover", "OnEntry", "OnReady"} {
		k.RunEvent("player", ev, id, NullPersistID, nil)
	}
	return id
}

func module2[T Module](t *testing.T, k *Kernel, name string) T {
	t.Helper()
	m, ok := k.GetLogicModule(name).(T)
	if !ok {
		t.Fatalf("模块 %s 未装配", name)
	}
	return m
}

// TestSecondBatchAssemble 装配序（官方 Init 序相对序：Gmcc→Post→Team→Friend→
// Chat→Log）+ 配置装载。
func TestSecondBatchAssemble(t *testing.T) {
	k, mods := assemble2(t)
	want := []string{"GmccModule", "PostModule", "TeamModule", "FriendModule", "ChatModule", "LogModule"}
	var got []string
	for _, m := range mods {
		switch m.Name() {
		case "GmccModule", "PostModule", "TeamModule", "FriendModule", "ChatModule", "LogModule":
			got = append(got, m.Name())
		}
	}
	if len(got) != 6 {
		t.Fatalf("第二批模块 %d 个: %v", len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("装配序[%d]=%s 期望 %s", i, got[i], want[i])
		}
	}
	fm := module2[*FriendModule](t, k, "FriendModule")
	if fm.Friendship == nil || fm.Friendship.MaxValue != 1000 {
		t.Fatalf("friendship.xml 未装载: %+v", fm.Friendship)
	}
	if len(fm.Friendship.FriendshipLevels) != 6 {
		t.Fatalf("FriendshipLevel 阈值 %v", fm.Friendship.FriendshipLevels)
	}
	if v, ok := fm.FVEvents["FVTeamTask"]; !ok || v.IncreaseValue != 2 {
		t.Fatalf("friendvalueevent.xml FVTeamTask=%+v", fm.FVEvents["FVTeamTask"])
	}
	pm := module2[*PostModule](t, k, "PostModule")
	if pm.Rule == nil || pm.Rule.PostBoxMax != 100 || pm.Rule.UrgentPostStage != 50000 {
		t.Fatalf("post.xml 未装载: %+v", pm.Rule)
	}
	tm := module2[*TeamModule](t, k, "TeamModule")
	if tm.SummonCost == nil {
		t.Fatal("summon_cost.xml 未装载")
	}
}

// TestChatPageAndSilence ChatModule：chatpage_rec 五默认页 + 禁言闭环
//（GMCC 禁言 → CanChat 拒 → H_Silence 倒计时 → 解禁公告）+ 聊天页子命令。
func TestChatPageAndSilence(t *testing.T) {
	k, _ := assemble2(t)
	chat := module2[*ChatModule](t, k, "ChatModule")
	id := newPlayer2(t, k, "阿武")

	o := k.GetGameObj(id)
	r := o.Record("chatpage_rec")
	if r == nil || r.Rows() != len(chatDefaultPages) {
		t.Fatalf("chatpage_rec 行数 %d 期望 %d", r.Rows(), len(chatDefaultPages))
	}
	if got := r.Row(0)[1].(string); got != "ui_zonghe" {
		t.Fatalf("首页名 %q 期望 ui_zonghe", got)
	}
	// 子命令 100：新页。
	k.DispatchCommand("player", 1, id, NullPersistID, VarList{int32(100), int32(9), "ui_diy"})
	if r.Rows() != len(chatDefaultPages)+1 {
		t.Fatalf("新页未追加: %d", r.Rows())
	}
	// 子命令 99：改名。
	k.DispatchCommand("player", 1, id, NullPersistID, VarList{int32(99), int32(9), "ui_diy2"})
	if got := r.Row(r.Rows() - 1)[1].(string); got != "ui_diy2" {
		t.Fatalf("改名失败: %q", got)
	}

	// 聊天发送：正常。
	var broadcast string
	chat.ChatOut = func(ch int32, from, content string) { broadcast = content }
	k.DispatchStringCommand("player", "chat", id, NullPersistID, VarList{int32(ChannelWorld), "你好世界"})
	if broadcast != "你好世界" {
		t.Fatalf("世界频道广播未发出: %q", broadcast)
	}
	// 禁言 → CanChat 拒（sys_item_0083）。
	k.SetInt(id, "SilenceTick", 2)
	if chat.CanChat(k, id) {
		t.Fatal("禁言期 CanChat 应拒")
	}
	has83 := false
	for _, s := range chat.SysMsgs() {
		if s == "sys_item_0083" {
			has83 = true
		}
	}
	if !has83 {
		t.Fatal("禁言未下发 sys_item_0083")
	}
	// H_Silence 两拍倒完 → 解禁公告 BG_131001。
	k.Bus().TickHeartBeats(k, 1)
	k.Bus().TickHeartBeats(k, 2)
	if o.GetInt("SilenceTick") != 0 {
		t.Fatalf("SilenceTick=%d 期望 0", o.GetInt("SilenceTick"))
	}
	found := false
	for _, s := range chat.SysMsgs() {
		if s == "BG_131001" {
			found = true
		}
	}
	if !found {
		t.Fatal("解禁未公告 BG_131001")
	}
	if !chat.CanChat(k, id) {
		t.Fatal("解禁后 CanChat 应过")
	}
}

// TestFriendFlows FriendModule：加好友双向入表 → 重复拒(0007) → 拉黑绝交(0033)
// → 私聊黑名单拒(0018) → 好友度累加(864) → Level 临界联动好友行。
func TestFriendFlows(t *testing.T) {
	k, _ := assemble2(t)
	fm := module2[*FriendModule](t, k, "FriendModule")
	a := newPlayer2(t, k, "甲")
	b := newPlayer2(t, k, "乙")

	// 自加拒。
	k.DispatchCommand("player", 110, a, NullPersistID, VarList{int32(1), "甲"})
	if rowByName(k.GetGameObj(a).Record("friend_rec"), "甲") >= 0 {
		t.Fatal("自加好友未被拒（sys_friend_0003）")
	}
	// 甲加乙：双向入表 + 0024。
	k.DispatchCommand("player", 110, a, NullPersistID, VarList{int32(1), "乙"})
	ra := k.GetGameObj(a).Record("friend_rec")
	rb := k.GetGameObj(b).Record("friend_rec")
	if rowByName(ra, "乙") < 0 || rowByName(rb, "甲") < 0 {
		t.Fatal("加好友未双向入表")
	}
	if !hasTpl(fm.SysMsgs(), "sys_friend_0024") {
		t.Fatal("加好友未通知 sys_friend_0024")
	}
	// 在线/等级列回填。
	if ra.Row(rowByName(ra, "乙")).Int(frOnline) != 1 {
		t.Fatal("好友在线位未回填")
	}
	// 重复加 → 0007。
	k.DispatchCommand("player", 110, a, NullPersistID, VarList{int32(1), "乙"})
	if !hasTpl(fm.SysMsgs(), "sys_friend_0007") {
		t.Fatal("重复加好友未拒（sys_friend_0007）")
	}
	// 好友度：FVTeamTask 规则值 2。
	k.DispatchCommand("player", 864, a, NullPersistID, VarList{"乙", "FVTeamTask", int32(0)})
	if got := ra.Row(rowByName(ra, "乙")).Int(frValue); got != 2 {
		t.Fatalf("好友度=%d 期望 2（FVTeamTask 规则）", got)
	}
	if fm.FriendshipLevel(2) != 1 {
		t.Fatalf("好友度等级 %d 期望 1", fm.FriendshipLevel(2))
	}
	// Level 临界 → 乙的 friend_rec[甲] 行等级联动。
	k.SetInt(a, "Level", 30)
	if got := rb.Row(rowByName(rb, "甲")).Int(frLevel); got != 30 {
		t.Fatalf("好友行等级联动=%d 期望 30", got)
	}
	// 乙拉黑甲 → 乙侧好友行删除 + 0033。
	k.DispatchCommand("player", 110, b, NullPersistID, VarList{int32(5), "甲"})
	if rowByName(rb, "甲") >= 0 {
		t.Fatal("拉黑未移出好友表")
	}
	if !hasTpl(fm.SysMsgs(), "sys_friend_0033") {
		t.Fatal("拉黑未通知 sys_friend_0033")
	}
	// 甲跨场景私聊乙：乙黑名单含甲 → 拒（sys_friend_0018）。
	k.DispatchCommand("player", 914, a, NullPersistID, VarList{"乙", "在吗"})
	if !hasTpl(module2[*ChatModule](t, k, "ChatModule").SysMsgs(), "sys_friend_0018") {
		t.Fatal("黑名单私聊未被拒")
	}
	// 删好友（甲删乙）。
	k.DispatchCommand("player", 110, a, NullPersistID, VarList{int32(2), "乙"})
	if rowByName(ra, "乙") >= 0 {
		t.Fatal("删好友未生效")
	}
	// 被杀 → 仇人表（0x268）。
	k.DispatchCommand("player", 616, a, NullPersistID, VarList{"丙"})
	if rowByName(k.GetGameObj(a).Record("enemy_rec"), "丙") < 0 {
		t.Fatal("被杀未入仇人表")
	}
}

// TestTeamFlows TeamModule：建队(0x136) → 入队(0x1C1) → team_rec 镜像 →
// 踢人(0x137) → 移交队长(0x139) → 离队解散链 + HPRatio 临界同步。
func TestTeamFlows(t *testing.T) {
	k, _ := assemble2(t)
	tm := module2[*TeamModule](t, k, "TeamModule")
	a := newPlayer2(t, k, "队长")
	b := newPlayer2(t, k, "队员")

	// 甲建队。
	if !k.DispatchCommand("player", 0x136, a, NullPersistID, VarList{int32(0x136), int32(0)}) {
		t.Fatal("0x136 未命中")
	}
	oa := k.GetGameObj(a)
	teamID := oa.GetInt("TeamID")
	if teamID == 0 {
		t.Fatal("建队未写 TeamID")
	}
	tm1 := tm.Teams()[teamID]
	if tm1 == nil || tm1.Captain != "队长" || len(tm1.Members) != 1 {
		t.Fatalf("建队态 %+v", tm1)
	}
	if oa.GetString("TeamUniqueID") == "" {
		t.Fatal("TeamUniqueID 未生成")
	}
	// 乙入队（0x1C1）。
	k.DispatchCommand("player", 0x1C1, b, NullPersistID, VarList{int32(0x1C1), teamID})
	if len(tm1.Members) != 2 {
		t.Fatalf("入队后成员 %v", tm1.Members)
	}
	ob := k.GetGameObj(b)
	if ob.GetInt("TeamID") != teamID || ob.GetInt("CurMembers") != 2 {
		t.Fatal("入队属性链不完整")
	}
	// team_rec 镜像（双方两行）。
	if oa.Record("team_rec").Rows() != 2 || ob.Record("team_rec").Rows() != 2 {
		t.Fatal("team_rec 未镜像全员")
	}
	// HPRatio 临界 → 全员 team_rec 列同步。
	k.SetInt(b, "HPRatio", 66)
	rb := oa.Record("team_rec")
	if got := rb.Row(rowByName(rb, "队员")).Int(trHPRatio); got != 66 {
		t.Fatalf("HPRatio 同步=%d 期望 66", got)
	}
	// 非队长踢人 → 0026 拒。
	k.DispatchCommand("player", 0x137, b, NullPersistID, VarList{int32(0x137), "队长"})
	if !hasTpl(tm.SysMsgs(), "sys_team_0026") {
		t.Fatal("非队长踢人未拒（sys_team_0026）")
	}
	// 队长踢乙。
	k.DispatchCommand("player", 0x137, a, NullPersistID, VarList{int32(0x137), "队员"})
	if ob.GetInt("TeamID") != 0 || ob.Record("team_rec").Rows() != 0 {
		t.Fatal("踢出未清属性/表")
	}
	// 乙再入队 → 移交队长 → 乙离队（队长移交甲）。
	k.DispatchCommand("player", 0x1C1, b, NullPersistID, VarList{int32(0x1C1), teamID})
	k.DispatchCommand("player", 0x139, a, NullPersistID, VarList{int32(0x139), "队员"})
	if tm1.Captain != "队员" {
		t.Fatalf("移交后队长 %q", tm1.Captain)
	}
	k.DispatchCommand("player", 0x138, b, NullPersistID, VarList{int32(0x138)})
	if len(tm1.Members) != 1 || tm1.Captain != "队长" {
		t.Fatalf("队长离队未移交: %+v", tm1)
	}
	// 甲离队 → 空队解散。
	k.DispatchCommand("player", 0x138, a, NullPersistID, VarList{int32(0x138)})
	if len(tm.Teams()) != 0 {
		t.Fatal("空队未解散")
	}
	// Lua 面。
	if tm.NxIsInTeam(k, a) || tm.NxGetCaptain(k, a) != "" {
		t.Fatal("离队后 Lua 面异常")
	}
}

// TestPostLetterFlow PostModule：普通信即时达 → 金钱信走中间表 →
// H_Update_Letter 到期转正 → 加急信召邮差 → 附件提取。
func TestPostLetterFlow(t *testing.T) {
	k, _ := assemble2(t)
	pm := module2[*PostModule](t, k, "PostModule")
	a := newPlayer2(t, k, "寄件人")
	b := newPlayer2(t, k, "收件人")

	// 子命令 2 发普通信：即时入 RecvLetterRec。
	k.DispatchCommand("player", 350, a, NullPersistID,
		VarList{int32(2), "收件人", "战书", "明日决战", int32(0), int32(0), "", ""})
	recv := k.GetGameObj(b).Record("RecvLetterRec")
	if recv == nil || recv.Rows() != 1 {
		t.Fatalf("普通信未即时达: %d 行", recv.Rows())
	}
	serial := recv.Row(0).String(lrSerial)
	if recv.Row(0).String(lrTitle) != "战书" {
		t.Fatal("信件题目列错位")
	}
	// 读信（子命令 3）→ 已读。
	k.DispatchCommand("player", 350, b, NullPersistID, VarList{int32(3), serial})
	if recv.Row(0).Int(lrRead) != 1 {
		t.Fatal("已读标记未置位")
	}
	// 金钱信（子命令 11 汇款）→ 中间表倒计时。
	k.DispatchCommand("player", 350, a, NullPersistID, VarList{int32(11), "收件人", "500"})
	mid := k.GetGameObj(b).Record("RecvLetterMidRec")
	if mid == nil || mid.Rows() != 1 {
		t.Fatal("金钱信未入中间表")
	}
	// 到期：直接把剩余时间改到过去，跑 H_Update_Letter。
	row := mid.Row(0)
	row[lmRemain] = int64(unixNow() - 1)
	mid.ModifyRow(0, row)
	k.Bus().TickHeartBeats(k, 1)
	if mid.Rows() != 0 {
		t.Fatal("到期信未移出中间表")
	}
	if recv.Rows() != 2 {
		t.Fatalf("到期信未转正式表: %d", recv.Rows())
	}
	if !hasTpl(pm.SysMsgs(), "sys_mail_0009") {
		t.Fatal("到信未通知 sys_mail_0009")
	}
	// 加急信 → 邮差召唤出口。
	summoned := ""
	pm.MailNpcOut = func(target, ser string) { summoned = target }
	k.DispatchCommand("player", 350, a, NullPersistID,
		VarList{int32(2), "收件人", "急件", "速来", int32(0), int32(1), "", ""})
	k.Bus().TickHeartBeats(k, 2)
	if summoned != "收件人" {
		t.Fatal("加急信未召唤邮差")
	}
	// 提取金钱附件（子命令 4）。
	i := rowByCol(recv, lrGold, "500")
	if i < 0 {
		t.Fatal("金钱附件信缺失")
	}
	k.DispatchCommand("player", 350, b, NullPersistID, VarList{int32(4), recv.Row(i).String(lrSerial)})
	if recv.Row(i).String(lrGold) != "" {
		t.Fatal("附件未提取")
	}
	// 删信（子命令 5）。
	k.DispatchCommand("player", 350, b, NullPersistID, VarList{int32(5), serial})
	if recv.Rows() != 2 {
		t.Fatalf("删信后 %d 行", recv.Rows())
	}
	// 存档面：RecvLetterRec save=true。
	_, recs := k.GetGameObj(b).Archive()
	if _, ok := recs["RecvLetterRec"]; !ok {
		t.Fatal("信件未入存档面")
	}
}

// TestGmccMuteFlow GmccModule：禁言（经 ChatModule.SetSilence）与解禁。
func TestGmccMuteFlow(t *testing.T) {
	k, _ := assemble2(t)
	gm := module2[*GmccModule](t, k, "GmccModule")
	id := newPlayer2(t, k, "被封者")

	// cmd 296 禁言 1 分钟 → SilenceTick=60。
	k.DispatchCommand("player", 296, id, NullPersistID, VarList{"silence", int32(1)})
	if got := k.GetGameObj(id).GetInt("SilenceTick"); got != 60 {
		t.Fatalf("SilenceTick=%d 期望 60", got)
	}
	// 上行指令出口。
	var uplinked string
	gm.GmccOut = func(player, op string, args VarList) { uplinked = op }
	k.DispatchCommand("player", 171, id, NullPersistID, VarList{"complain", "卡bug了"})
	if uplinked != "complain" {
		t.Fatalf("SendToGmcc 出口未触发: %q", uplinked)
	}
	// 解禁。
	k.DispatchCommand("player", 296, id, NullPersistID, VarList{"unsilence"})
	if k.GetGameObj(id).GetInt("SilenceTick") != 0 {
		t.Fatal("解禁未清 SilenceTick")
	}
}

// TestLogLifecycle LogModule：entry_scene/leave_scene/logout 生命周期日志 +
// nx_save_playerlog/nx_append_itemlog Lua 面。
func TestLogLifecycle(t *testing.T) {
	k, _ := assemble2(t)
	lm := module2[*LogModule](t, k, "LogModule")
	id := newPlayer2(t, k, "记录员")
	k.SetInt(id, "EntrySceneTime", 111)

	k.RunEvent("player", "OnBeforeEntryScene", id, NullPersistID, nil)
	k.RunEvent("player", "OnLeaveScene", id, NullPersistID, nil)
	k.RunEvent("player", "OnEntryGame", id, NullPersistID, nil)
	k.RunEvent("player", "OnStore", id, NullPersistID, nil)

	kinds := map[string]bool{}
	for _, e := range lm.Logs() {
		kinds[e.Kind] = true
	}
	for _, want := range []string{"entry_scene", "leave_scene", "entry_game", "logout"} {
		if !kinds[want] {
			t.Fatalf("生命周期日志缺 %s: %v", want, lm.Logs())
		}
	}
	// leave_scene 字段（EntrySceneTime/LeaveSceneTime）。
	for _, e := range lm.Logs() {
		if e.Kind == "leave_scene" {
			if varToInt64(e.Fields["EntrySceneTime"]) != 111 { // schema int64
				t.Fatalf("leave_scene EntrySceneTime=%v", e.Fields["EntrySceneTime"])
			}
			if e.Fields["LeaveSceneTime"] == nil {
				t.Fatal("leave_scene 缺 LeaveSceneTime")
			}
		}
	}
	// Lua 面。
	lm.NxSavePlayerLog("记录员", 10, "完成了新手引导")
	lm.NxAppendItemLog(1, "UID-1", "还魂丹", 3)
	last := lm.Logs()[len(lm.Logs())-1]
	if last.Kind != "itemlog" || last.Fields["Amount"] != Var(int32(3)) {
		t.Fatalf("itemlog=%+v", last)
	}
	// LogOut 出口。
	var got string
	lm.LogOut = func(kind string, f map[string]Var) { got = kind }
	lm.NxSavePlayerLog("记录员", 10, "再记一条")
	if got != "playerlog" {
		t.Fatal("LogOut 出口未触发")
	}
}

func hasTpl(msgs []string, tpl string) bool {
	for _, m := range msgs {
		if m == tpl {
			return true
		}
	}
	return false
}

// rowByCol 按列值定位行（信件测试面）。
func rowByCol(r *Record, col int, val string) int {
	for i := 0; i < r.Rows(); i++ {
		if r.Row(i).String(col) == val {
			return i
		}
	}
	return -1
}
