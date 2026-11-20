package fsgame

import (
	"fmt"
)

// GmccModule 复刻官方 commonmodule/gmccmodule（Init @0x7BF300，玩法表
// Init 序 13；符号 329）——客服（GMCC）桥。
//
// 官方 Init 注册面（p2scan 反汇编定案）：
//
//	AddIntCustomHook ("player",296=0x128,@0x7BF440)   OnCustom（客服指令）
//	AddIntCommandHook ("player",171=0xAB,@0x7BF4E0)   OnPlayerCmd
//	AddEventCallback ("scene","OnGmccCustom",@0x7BFA90) OnGmNews（gmcc 下行）
//
// OnCustom/OnPlayerCmd 行为（字符串定案）：取 ChatModule 协作（GetLogicModule）
// + sys_item_0083（禁言模板）+ sysinfo + SilenceTick —— 客服禁言/解禁言，
// 其余指令经 IKernel::SendToGmcc 上行客服中心。
type GmccModule struct {
	BaseModule

	// GmccOut 客服上行出口（官方 IKernel::SendToGmcc → FsSoloPublic/gmcc 服）。
	GmccOut func(player string, op string, args VarList)

	ops []gmccOp
}

type gmccOp struct {
	Player string
	Op     string
}

// Name 模块名。
func (m *GmccModule) Name() string { return "GmccModule" }

// Init 注册期（对齐官方 0x7BF300）。
func (m *GmccModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnCommand("player", 296, m.onCustom)
	b.OnIntCommand("player", 171, m.onPlayerCmd)
	b.OnEvent("scene", "OnGmccCustom", m.onGmNews)
	return nil
}

// onCustom 官方 @0x7BF440（命令 296）：客服指令多路。
// args: 指令号[, 参数…]；已知面：禁言（ SilenceTick 经 ChatModule.SetSilence，
// 下发 sys_item_0083 模板）与 sysinfo 类公告。
func (m *GmccModule) onCustom(k *Kernel, self, from PersistID, args VarList) {
	m.handle(k, self, args)
}

// onPlayerCmd 官方 @0x7BF4E0（命令 171）：玩家侧客服命令入口（同 handle）。
func (m *GmccModule) onPlayerCmd(k *Kernel, self, from PersistID, args VarList) {
	m.handle(k, self, args)
}

// handle 公共处理（官方两入口同构，字符串证据：ChatModule/sys_item_0083/
// sysinfo/SilenceTick）。
func (m *GmccModule) handle(k *Kernel, self PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	name, _ := o.Get("name").(string)
	op := args.String(0)
	switch op {
	case "silence", "mute": // 客服禁言：分钟数 → SilenceTick
		if chat, ok := k.GetLogicModule("ChatModule").(*ChatModule); ok && chat != nil {
			chat.SetSilence(k, self, args.Int(1)*60)
		}
		m.ops = append(m.ops, gmccOp{Player: name, Op: "silence"})
	case "unsilence", "unmute": // 解禁
		_ = k.SetInt(self, "SilenceTick", 0)
		m.ops = append(m.ops, gmccOp{Player: name, Op: "unsilence"})
	default: // 其余指令上行客服中心（SendToGmcc）
		if m.GmccOut != nil {
			m.GmccOut(name, op, args)
		}
		m.ops = append(m.ops, gmccOp{Player: name, Op: op})
	}
	k.Tracef("GmccModule: %s op=%s", name, op)
}

// onGmNews 官方 OnGmNews @0x7BFA90：gmcc 下行（系统公告/客服回复注入）。
func (m *GmccModule) onGmNews(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("GmccModule: OnGmNews")
}

// Ops 指令流水（测试面）。
func (m *GmccModule) Ops() []gmccOp { return m.ops }

// ---- LogModule ----

// LogModule 复刻官方 commonmodule/logmodule（Init @0x72F090，玩法表 Init 序 240；
// 符号 447）——玩家生命周期日志。
//
// 官方 Init 注册面（p2scan 反汇编定案）：
//
//	AddEventCallback ("player","OnRecover",@0x734D90)          EntryGameTime/first_send_moniter_msg
//	AddEventCallback ("player","OnEntryGame",@0x7348E0)        LoginTime/EntryGameTime/Account/Address/MacAddress
//	AddEventCallback ("player","OnBeforeEntryScene",@0x733E40) EntrySceneTime/entry_scene/Amount
//	AddEventCallback ("player","OnAfterEntryScene",@0x734970)
//	AddEventCallback ("player","OnReady",@0x7349C0)
//	AddEventCallback ("player","OnContinue",@0x7349F0)
//	AddEventCallback ("player","OnLeaveScene",@0x734280)       leave_scene/EntrySceneTime/LeaveSceneTime/Address/MacAddress
//	AddEventCallback+AddClassCallback ("player","OnStore",@0x734A40) logout
//	DeclareLuaExt ("nx_append_itemlog",@0x72EA50)
//	DeclareLuaExt ("nx_save_playerlog",@0x72EE60)
//
// 日志字面（串定案）：entry_scene/leave_scene/logout；字段 Name/Account/
// Address/MacAddress/EntrySceneTime/LeaveSceneTime/Amount/UniqueID。
// Lua 面：nx_save_playerlog(widestr,int,widestr)、nx_append_itemlog(int,object,widestr×3)。
type LogModule struct {
	BaseModule

	// LogOut 日志出口（官方 IKernel::CustomLog → fssololog → MySQL）。
	LogOut func(kind string, fields map[string]Var)

	// logs 日志流水（测试面）。
	logs []LogEntry
}

// LogEntry 生命周期日志条目。
type LogEntry struct {
	Kind   string // entry_scene / leave_scene / logout / itemlog / playerlog
	Fields map[string]Var
}

// Name 模块名。
func (m *LogModule) Name() string { return "LogModule" }

// Init 注册期（对齐官方 0x72F090）。
func (m *LogModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnEvent("player", "OnRecover", m.onRecover)
	b.OnEvent("player", "OnEntryGame", m.onEntryGame)
	b.OnEvent("player", "OnBeforeEntryScene", m.onBeforeEntryScene)
	b.OnEvent("player", "OnAfterEntryScene", m.onAfterEntryScene)
	b.OnEvent("player", "OnReady", m.onReady)
	b.OnEvent("player", "OnContinue", m.onContinue)
	b.OnEvent("player", "OnLeaveScene", m.onLeaveScene)
	b.OnEvent("player", "OnStore", m.onStore)
	b.OnClassEvent("player", "OnStore", func(k *Kernel, class string) {})
	return nil
}

// emit 日志出口（LogOut 挂点 + 本地流水）。
func (m *LogModule) emit(kind string, fields map[string]Var) {
	m.logs = append(m.logs, LogEntry{Kind: kind, Fields: fields})
	if m.LogOut != nil {
		m.LogOut(kind, fields)
	}
}

// identity 玩家身份字段集（Name/Account/Address/MacAddress 官方面）。
func identity(o *GameObject) map[string]Var {
	get := func(p string) Var {
		if v := o.Get(p); v != nil {
			return v
		}
		return ""
	}
	return map[string]Var{
		"Name":       get("name"),
		"Account":    get("Account"),
		"Address":    get("Address"),
		"MacAddress": get("MacAddress"),
	}
}

// onRecover 官方 @0x734D90：EntryGameTime 记账 + first_send_moniter_msg。
func (m *LogModule) onRecover(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	f := identity(o)
	f["EntryGameTime"] = nowUnix()
	m.emit("recover", f)
}

// onEntryGame 官方 @0x7348E0：进游戏（LoginTime/EntryGameTime）。
func (m *LogModule) onEntryGame(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	f := identity(o)
	f["LoginTime"] = o.Get("LoginTime")
	f["EntryGameTime"] = o.Get("EntryGameTime")
	m.emit("entry_game", f)
}

// onBeforeEntryScene 官方 @0x733E40：EntrySceneTime 落记。
func (m *LogModule) onBeforeEntryScene(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	f := identity(o)
	f["EntrySceneTime"] = nowUnix()
	m.emit("entry_scene", f)
}

// onAfterEntryScene 官方 @0x734970：入场景完成。
func (m *LogModule) onAfterEntryScene(k *Kernel, self, from PersistID, args VarList) {}

// onReady 官方 @0x7349C0。
func (m *LogModule) onReady(k *Kernel, self, from PersistID, args VarList) {}

// onContinue 官方 @0x7349F0。
func (m *LogModule) onContinue(k *Kernel, self, from PersistID, args VarList) {}

// onLeaveScene 官方 @0x734280：leave_scene（EntrySceneTime/LeaveSceneTime）。
func (m *LogModule) onLeaveScene(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	f := identity(o)
	f["EntrySceneTime"] = o.Get("EntrySceneTime")
	f["LeaveSceneTime"] = nowUnix()
	m.emit("leave_scene", f)
}

// onStore 官方 @0x734A40：logout。
func (m *LogModule) onStore(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	f := identity(o)
	f["EntryGameTime"] = o.Get("EntryGameTime")
	m.emit("logout", f)
}

// NxSavePlayerLog nx_save_playerlog(name, level, note) Lua 面的 Go 等价
//（官方参数校验串：argument 1 not wide string / 2 not int / 3 not wide string）。
func (m *LogModule) NxSavePlayerLog(name string, level int32, note string) {
	if name == "" || note == "" {
		return // 官方参数校验失败静默
	}
	m.emit("playerlog", map[string]Var{"Name": name, "Level": level, "Note": note})
}

// NxAppendItemLog nx_append_itemlog(op, obj, uid, name, amount) Lua 面
//（官方字段 UniqueID/Amount）。
func (m *LogModule) NxAppendItemLog(op int32, uid, name string, amount int32) {
	m.emit("itemlog", map[string]Var{"Op": op, "UniqueID": uid, "Name": name, "Amount": amount})
}

// Logs 日志流水（测试面）。
func (m *LogModule) Logs() []LogEntry { return m.logs }

// String 日志流水概览。
func (e LogEntry) String() string {
	return fmt.Sprintf("%s %v", e.Kind, e.Fields)
}
