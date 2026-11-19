package fsserver

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/charge"
)

// ---- wire ----

func TestInnerFrameRoundtrip(t *testing.T) {
	payload := []byte("hello")
	frame := EncodeInner(InnerHeader{Src: 0, Dst: 0x18, Type: MemberSceneAll}, payload)
	h, p, err := DecodeInner(frame)
	if err != nil || h.Src != 0 || h.Dst != 0x18 || h.Type != MemberSceneAll || !bytes.Equal(p, payload) {
		t.Fatalf("roundtrip mismatch: %+v %q %v", h, p, err)
	}
	if _, _, err := DecodeInner([]byte{1, 2, 3}); err == nil {
		t.Fatal("short frame must fail")
	}
}

func TestEntryStructs(t *testing.T) {
	reg := EntryRegister{DistrictID: 1, ServerID: 11, EntryIdx: 3}
	if got, err := DecodeEntryRegister(reg.Encode()); err != nil || got != reg {
		t.Fatalf("register roundtrip: %+v %v", got, err)
	}
	if _, err := DecodeEntryRegister(make([]byte, 11)); err == nil {
		t.Fatal("register size mismatch must fail")
	}
	acc := EntryClientAccept{Ident: 7, Serial: 9, Port: 9600}
	acc.Addr[0] = '1'
	acc.Addr[1] = '2'
	acc.Addr[2] = '7'
	if got, err := DecodeEntryClientAccept(acc.Encode()); err != nil || got.Ident != 7 || got.Serial != 9 || got.Port != 9600 || got.Addr[1] != '2' {
		t.Fatalf("accept roundtrip: %+v %v", got, err)
	}
	if len(acc.Encode()) != 28 {
		t.Fatalf("accept struct must be 28B, got %d", len(acc.Encode()))
	}
}

func TestErrFrame(t *testing.T) {
	f := ErrFrame(ErrServerFull)
	if f[0] != 0x03 || binary.LittleEndian.Uint32(f[1:]) != 0x4E92 || len(f) != 5 {
		t.Fatalf("err frame: % x", f)
	}
}

// ---- archive ----

func TestArchiveRoundtrip(t *testing.T) {
	w := NewStoreWriter(0)
	w.WriteInt(-5)
	w.WriteWord(0xBEEF)
	w.PutByte(0x7F)
	w.WriteBool(true)
	w.WriteInt64(1 << 40)
	w.WriteFloat(1.5)
	w.WriteDouble(2.25)
	w.WriteString("abc")
	w.WriteWideStr("角色")
	w.WriteObject(0x1234)

	r := NewLoadReader(w.Bytes())
	if v, _ := r.ReadInt(); v != -5 {
		t.Fatalf("int %d", v)
	}
	if v, _ := r.ReadWord(); v != 0xBEEF {
		t.Fatalf("word %x", v)
	}
	if v, _ := r.ReadByte(); v != 0x7F {
		t.Fatalf("byte %x", v)
	}
	if v, _ := r.ReadBool(); !v {
		t.Fatal("bool")
	}
	if v, _ := r.ReadInt64(); v != 1<<40 {
		t.Fatalf("int64 %d", v)
	}
	if v, _ := r.ReadFloat(); v != 1.5 {
		t.Fatalf("float %v", v)
	}
	if v, _ := r.ReadDouble(); v != 2.25 {
		t.Fatalf("double %v", v)
	}
	if v, _ := r.ReadString(); v != "abc" {
		t.Fatalf("string %q", v)
	}
	if v, _ := r.ReadWideStr(); v != "角色" {
		t.Fatalf("widestr %q", v)
	}
	if v, _ := r.ReadObject(); v != 0x1234 {
		t.Fatalf("object %x", v)
	}
}

func TestVaryMessageRoundtrip(t *testing.T) {
	vals := []VaryValue{
		{Tag: VaryTagInt, I: 42},
		{Tag: VaryTagInt64, I: 1 << 41},
		{Tag: VaryTagFloat, F: 0.5},
		{Tag: VaryTagDouble, F: 1.25},
		{Tag: VaryTagString, S: "$s"},
		{Tag: VaryTagWideStr, S: "宽"},
		{Tag: VaryTagObject, I: 7},
	}
	w := NewStoreWriter(0)
	if !PutVaryMessage(w, vals) {
		t.Fatal("put failed")
	}
	got, err := GetVaryMessage(len(vals), NewLoadReader(w.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	for i := range vals {
		if got[i].Tag != vals[i].Tag || got[i].I != vals[i].I || got[i].F != vals[i].F || got[i].S != vals[i].S {
			t.Fatalf("elem %d: %+v != %+v", i, got[i], vals[i])
		}
	}
	if _, err := GetVaryMessage(1, NewLoadReader([]byte{99})); err == nil {
		t.Fatal("bad tag must fail")
	}
}

// ---- matrixcode ----

func TestMatrixCodeDeterministic(t *testing.T) {
	a, b := NewMatrixCode(), NewMatrixCode()
	if a.GetMaxtrixCode(12345) != b.GetMaxtrixCode(12345) {
		t.Fatal("same seed must yield same matrix")
	}
	before := a.GetMaxtrixCode(0)
	a.IncMatrixPos()
	if a.GetMaxtrixCode(0) == before {
		t.Fatal("position rotation must change code")
	}
	a.SetMatrixPos(0)
	if a.GetMaxtrixCode(0) != before {
		t.Fatal("SetMatrixPos must restore")
	}
	// 挑战串形态：三组 列字母+行号 逗号分隔
	s := a.GetMatrixString(123)
	parts := strings.Split(s, ",")
	if len(parts) != 3 {
		t.Fatalf("challenge must have 3 groups: %q", s)
	}
	for _, p := range parts {
		if len(p) < 2 || p[0] < 'A' || p[0] > 'G' {
			t.Fatalf("bad group %q in %q", p, s)
		}
	}
}

// ---- players / scenes ----

func TestPlayersRegistry(t *testing.T) {
	ps := NewPlayers()
	p := ps.AddPlayer(1, 2, 5)
	if ps.FindByConnect(1, 2) != p {
		t.Fatal("find by connect")
	}
	ps.SetPlayerAccount(p, "acct1")
	if ps.FindByAccount("acct1") != p {
		t.Fatal("find by account")
	}
	ps.SetupPlayerName(p, "role1")
	if ps.FindByName("role1") != p {
		t.Fatal("find by name")
	}
	// 同连接重加：旧玩家替换
	p2 := ps.AddPlayer(1, 2, 6)
	if ps.FindByConnect(1, 2) != p2 || ps.Count() != 1 {
		t.Fatal("re-add must replace")
	}
	if ps.FindByAccount("acct1") != nil {
		t.Fatal("re-add must unbind account")
	}
	if !ps.RemovePlayer(1, 2) || ps.Count() != 0 {
		t.Fatal("remove")
	}
}

func TestScenesRegistry(t *testing.T) {
	ss := NewScenes()
	if _, err := ss.AddScene(31, 0, "scene", "cfg"); err != nil {
		t.Fatal(err)
	}
	if _, err := ss.AddPrototypeScene(77, 1, "fb", "cfg2"); err != nil {
		t.Fatal(err)
	}
	if ss.GetScene(77).Kind != 1 {
		t.Fatal("prototype kind")
	}
	c, err := ss.CreateClone(77)
	if err != nil || c.ProtoID != 77 || c.Kind != 2 || c.ID < CloneBase {
		t.Fatalf("clone: %+v %v", c, err)
	}
	if _, err := ss.CreateClone(31); err == nil {
		t.Fatal("clone of non-prototype must fail")
	}
	ss.SetState(c.ID, SSceneRunning)
	if ss.GetScene(c.ID).State != SSceneRunning {
		t.Fatal("clone state")
	}
	if !ss.DeleteClone(c.ID) || ss.GetScene(c.ID) != nil {
		t.Fatal("delete clone")
	}
	n, cl := ss.Counts()
	if n != 2 || cl != 0 {
		t.Fatalf("counts %d %d", n, cl)
	}
}

// ---- 广播包 ----

func TestBroadcastPacket(t *testing.T) {
	pkt := BuildBroadcastPacket("127.0.0.1", 16001, 1, 11, "server_11", 3, true)
	if len(pkt) != 120 {
		t.Fatalf("packet size %d", len(pkt))
	}
	if binary.LittleEndian.Uint32(pkt[0:]) != 0 || binary.LittleEndian.Uint32(pkt[4:]) != 0x300 {
		t.Fatal("kind/magic")
	}
	if binary.LittleEndian.Uint32(pkt[0x18:]) != 16001 {
		t.Fatal("port")
	}
	if binary.LittleEndian.Uint32(pkt[0x1C:]) != 1 || binary.LittleEndian.Uint32(pkt[0x20:]) != 11 {
		t.Fatal("district/server")
	}
	if binary.LittleEndian.Uint32(pkt[0x70:]) != 3 || binary.LittleEndian.Uint32(pkt[0x74:]) != 1 {
		t.Fatal("players/open")
	}
	// 名域 wchar（UTF-16LE）：'s',0,'e',0...
	if pkt[0x24] != 's' || pkt[0x25] != 0 || pkt[0x26] != 'e' || pkt[0x27] != 0 {
		t.Fatalf("name (wchar) % x", pkt[0x24:0x34])
	}
}

// ---- 测试脚手架：假 IC + 收集器 ----

type sink struct {
	t          []uint32
	entries    [][]byte
	broadcasts [][]byte
}

func (s *sink) send(ident, dst, typ uint32, payload []byte) error {
	s.t = append(s.t, typ)
	s.entries = append(s.entries, payload)
	return nil
}

func newTestWorld() (*World, *sink) {
	w := NewWorld()
	w.MaxPlayers = 100
	s := &sink{}
	w.IC = &InterConnect{Send: s.send, Connected: func(uint32) bool { return true }}
	w.Logf = func(string, ...any) {}
	return w, s
}

// 客户端帧构造（world 层方言：21B 头 + 长度前缀串）。
func clientFrame(opcode byte) []byte {
	w := NewStoreWriter(0)
	w.PutByte(opcode)
	w.WriteInt(0) // 头部填充至 21B
	w.WriteInt(0)
	w.WriteInt(0)
	w.WriteInt(0)
	w.WriteInt(0)
	return w.Bytes()[:0x15]
}

func loginFrame(acct, pass string) []byte {
	f := clientFrame(ClientOpLogin)
	w := NewStoreWriter(0)
	w.buf = f
	w.pos = len(f)
	w.WriteString(acct)
	w.WriteString(pass)
	return w.Bytes()
}

func chooseFrame(role string) []byte {
	f := clientFrame(ClientOpChooseRole)
	w := NewStoreWriter(0)
	w.buf = f
	w.pos = len(f)
	w.WriteWideStr(role)
	return w.Bytes()
}

// ---- 世界状态机 ----

func TestWorldStateMachineOpen(t *testing.T) {
	w, s := newTestWorld()
	// Unknown：向 store 发 RoleInfo 请求
	w.WorldStateMachine(0, 0, nil)
	if len(s.t) == 0 || s.t[0] != StoreRoleInfoReq {
		t.Fatalf("Unknown must send StoreRoleInfoReq, got %v", s.t)
	}
	// store RoleInfo 应答 → LoadRoles → LoadScenes → CanOpen → Opened
	w.PutStoreMessage(StoreIdent(0), StoreMsgRoleInfo, buildRoleInfo("r1", "a1"))
	w.Pump()
	if w.State != WStateOpened {
		t.Fatalf("state %s", w.State)
	}
	// Opened 态发过 init_connect（给 entry）
	found := false
	for _, e := range s.entries {
		if len(e) == 4 {
			found = true
		}
	}
	if !found {
		t.Fatal("init_connect 4B not sent")
	}
}

func buildRoleInfo(role, acct string) []byte {
	w := NewStoreWriter(0)
	w.WriteWideStr(role)
	w.WriteString(acct)
	w.WriteBool(false)
	return w.Bytes()
}

func TestWorldStateMachineClose(t *testing.T) {
	w, _ := newTestWorld()
	for st := WStateCloseEntry; st < WStateClosed; st++ {
		w.SetWorldState(st)
		w.WorldStateMachine(0, 0, nil)
	}
	if w.State != WStateClosed {
		t.Fatalf("close chain must reach Closed, got %s", w.State)
	}
}

// ---- 玩家主链：accept → login → roles → choose → online → close ----

func TestPlayerLifecycle(t *testing.T) {
	w, s := newTestWorld()
	// 打开世界 + 一张 Running 场景
	w.SetWorldState(WStateOpened)
	if _, err := w.Scenes.AddScene(31, 0, "scene", "cfg"); err != nil {
		t.Fatal(err)
	}
	w.Scenes.SetState(31, SSceneRunning)

	// entry accept（ident=8 即 entry 0）
	acc := EntryClientAccept{Ident: 0x1105, Serial: 0x03DE, Port: 9600}
	w.PutEntryMessage(EntryIdent(0), EntryMsgTypeClientAccept, acc.Encode())
	w.Pump()
	if w.Players.Count() != 1 {
		t.Fatal("player must be added")
	}
	p := w.Players.FindByConnect(0x1105, 0x03DE)
	if p == nil || p.EntryIdx != 0 {
		t.Fatalf("player entry binding: %+v", p)
	}

	// 登录帧
	w.PutEntryMessage(EntryIdent(0), EntryMsgTypeClientMessage,
		append(u32b(0x1105), append(u32b(0x03DE), loginFrame("test001", "pw123")...)...))
	w.Pump()
	p = w.Players.FindByConnect(0x1105, 0x03DE)
	if p == nil || p.Account != "test001" {
		t.Fatalf("login: %+v", p)
	}

	// store Roles 应答（含一个角色）→ 0x04 下发
	roles := NewStoreWriter(0x40)
	roles.Seek(0x10)
	roles.WriteInt(1)
	roles.WriteInt(2)
	roles.WriteWideStr("hero")
	roles.WriteWideStr("para1")
	w.PutStoreMessage(StoreIdent(0), StoreMsgRoles, append(u32b(0x1105), append(u32b(0x03DE), roles.Bytes()...)...))
	w.Pump()
	if p.State != PStateLogged {
		t.Fatalf("after roles state %s", p.State)
	}
	var rolesMsg []byte
	for i := len(s.entries) - 1; i >= 0; i-- {
		if len(s.entries[i]) > 8 && s.entries[i][8] == 0x04 {
			rolesMsg = s.entries[i][8:]
			break
		}
	}
	if rolesMsg == nil {
		t.Fatal("0x04 roles frame not sent to client")
	}
	if rolesMsg[0] != 0x04 || binary.LittleEndian.Uint32(rolesMsg[0x25:]) != 1 {
		t.Fatalf("roles frame layout: % x", rolesMsg[:0x30])
	}

	// 选角 → Choosing → LoadPlayer 请求
	w.PutEntryMessage(EntryIdent(0), EntryMsgTypeClientMessage,
		append(u32b(0x1105), append(u32b(0x03DE), chooseFrame("hero")...)...))
	w.Pump()
	if p.State != PStateChoosing {
		t.Fatalf("after choose state %s", p.State)
	}

	// store Load 应答 → 进场 Online
	w.PutStoreMessage(StoreIdent(0), StoreMsgLoad, append(u32b(0x1105), append(u32b(0x03DE), []byte("DATA")...)...))
	w.Pump()
	if p.State != PStateOnline || !p.InGame || p.SceneID != 31 {
		t.Fatalf("online: state=%s inGame=%v scene=%d", p.State, p.InGame, p.SceneID)
	}
	if w.Scenes.GetScene(31).PlayerCnt != 1 {
		t.Fatal("scene player count")
	}
	// PlayerAdd 广播 + SetOnline 均已发
	sawAdd, sawOnline := false, false
	for _, typ := range s.t {
		if typ == MemberPlayerAdd {
			sawAdd = true
		}
		if typ == ForwSetOnline {
			sawOnline = true
		}
	}
	if !sawAdd || !sawOnline {
		t.Fatalf("member add %v / set online %v", sawAdd, sawOnline)
	}

	// 断线 → Offline 善后 → 移除
	w.PutEntryMessage(EntryIdent(0), EntryMsgTypeClientClosed, append(u32b(0x1105), u32b(0x03DE)...))
	w.Pump()
	if w.Players.Count() != 0 || w.Scenes.GetScene(31).PlayerCnt != 0 {
		t.Fatalf("cleanup: players=%d sceneCnt=%d", w.Players.Count(), w.Scenes.GetScene(31).PlayerCnt)
	}
	sawRemove := false
	for _, typ := range s.t {
		if typ == MemberPlayerRemove {
			sawRemove = true
		}
	}
	if !sawRemove {
		t.Fatal("PlayerRemove not broadcast")
	}
}

func TestLoginRejections(t *testing.T) {
	w, _ := newTestWorld()
	w.SetWorldState(WStateOpened)
	acc := EntryClientAccept{Ident: 1, Serial: 1}
	w.PutEntryMessage(EntryIdent(0), EntryMsgTypeClientAccept, acc.Encode())
	w.Pump()
	// 空账号 → 0x4E90 + 移除
	bad := clientFrame(ClientOpLogin)
	w.PutEntryMessage(EntryIdent(0), EntryMsgTypeClientMessage,
		append(u32b(1), append(u32b(1), bad...)...))
	w.Pump()
	if w.Players.Count() != 0 {
		t.Fatal("bad login must drop player")
	}
	// 满员
	w.MaxPlayers = 0
	acc2 := EntryClientAccept{Ident: 2, Serial: 2}
	w.PutEntryMessage(EntryIdent(0), EntryMsgTypeClientAccept, acc2.Encode())
	w.PutEntryMessage(EntryIdent(0), EntryMsgTypeClientMessage,
		append(u32b(2), append(u32b(2), loginFrame("a", "p")...)...))
	w.Pump()
	if w.Players.Count() != 1 {
		t.Fatal("full server keeps player for reply")
	}
}

// ---- member 面/inner 面 ----

func TestMemberSceneState(t *testing.T) {
	w, s := newTestWorld()
	w.SetWorldState(WStateOpened)
	w.Scenes.AddScene(5, 2, "scene", "cfg")
	w.TickSeconds() // Opened 态巡检：CreateScene 下发（type 0）
	sentCreate := false
	for _, typ := range s.t {
		if typ == MemberCreateScene {
			sentCreate = true
		}
	}
	if !sentCreate {
		t.Fatal("CreateScene not sent for unknown scene")
	}
	// member 回报 Running → SceneAdd 广播
	b := make([]byte, 8)
	binary.LittleEndian.PutUint32(b[0:], 5)
	binary.LittleEndian.PutUint32(b[4:], uint32(SSceneRunning))
	w.PutMemberMessage(MemberIdent(2), MemberMsgTypeSceneState, b)
	w.Pump()
	if w.Scenes.GetScene(5).State != SSceneRunning {
		t.Fatal("scene state not applied")
	}
}

func TestInnerSyncAllOnMemberConnect(t *testing.T) {
	w, s := newTestWorld()
	w.SetWorldState(WStateOpened)
	w.Scenes.AddScene(9, 3, "scene", "cfg")
	w.Scenes.SetState(9, SSceneRunning)
	p := w.Players.AddPlayer(1, 1, 0)
	w.Players.SetPlayerAccount(p, "a")
	w.Players.SetupPlayerName(p, "r")
	p.InGame = true
	w.PutInnerMessage(MemberIdent(3), InnerMsgTypeMemberConnected, []byte{3})
	w.Pump()
	sawSceneAll, sawPlayerAll, sawRoleInfo := false, false, false
	for _, typ := range s.t {
		switch typ {
		case MemberSceneAll:
			sawSceneAll = true
		case MemberPlayerAll:
			sawPlayerAll = true
		case MemberRoleInfo:
			sawRoleInfo = true
		}
	}
	if !sawSceneAll || !sawPlayerAll || !sawRoleInfo {
		t.Fatalf("sync triple: scene=%v player=%v role=%v", sawSceneAll, sawPlayerAll, sawRoleInfo)
	}
}

// ---- charge 面 ----

func TestChargeClientWire(t *testing.T) {
	var lines []string
	c := NewChargeClient(1, 11)
	c.SendLine = func(line string) error { lines = append(lines, line); return nil }
	var v charge.VarList
	v.AddString("keep")
	if err := c.SendIdentMsg(v); err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || !strings.HasPrefix(lines[0], "$s_0_0_1_11 $keep") || !strings.HasSuffix(lines[0], "\r\n") {
		t.Fatalf("charge line %q", lines[0])
	}
	// 收侧：echo 标记 + 命令 + 参数
	got := make([]charge.VarList, 0, 1)
	c.OnVarList = func(v charge.VarList) { got = append(got, v) }
	c.ReceiveLine("$echo login acct001 1\r\n")
	if len(got) != 1 || got[0].StringVal(1) != "login" {
		t.Fatalf("decode %v", got)
	}
}

func TestChargeLoginDispatch(t *testing.T) {
	w, _ := newTestWorld()
	w.SetWorldState(WStateOpened)
	p := w.Players.AddPlayer(1, 1, 0)
	w.Players.SetPlayerAccount(p, "acct001")
	w.Players.SetPlayerAccount(w.Players.AddPlayer(2, 2, 0), "acct002")
	var ok charge.VarList
	ok.AddString("echo")
	ok.AddString("login")
	ok.AddString("acct001")
	ok.AddInt(1)
	w.PutChargeVarList(ok)
	w.Pump()
	if p.Account != "acct001" { // 存活即未误删
		t.Fatal("charge login ok path")
	}
	var bad charge.VarList
	bad.AddString("echo")
	bad.AddString("login")
	bad.AddString("acct002")
	bad.AddInt(0)
	w.PutChargeVarList(bad)
	w.Pump()
	if w.Players.FindByAccount("acct002") != nil {
		t.Fatal("charge login fail must drop player")
	}
}

// ---- config ----

func TestParseGameWorldINI(t *testing.T) {
	cfg, err := ParseGameWorldINI([]byte(`
[System]
Sleep=1
[Main]
DistrictID=2
ServerID=33
ServerName=server_33
MaxPlayers=4096
[Inner]
Addr=10.0.0.1
Port=16500
[Broadcast]
Port=15100
[Echo]
Port=15101
[Charge]
Inuse=true
Addr=10.0.0.2
Port=6000
[ManageServer]
Inuse=true
Addr=10.0.0.3
Port=4002
`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DistrictID != 2 || cfg.ServerID != 33 || cfg.ServerName != "server_33" || cfg.MaxPlayers != 4096 {
		t.Fatalf("main: %+v", cfg)
	}
	if cfg.InnerAddr != "10.0.0.1" || cfg.InnerPort != 16500 || cfg.BroadPort != 15100 || cfg.EchoPort != 15101 {
		t.Fatalf("addrs: %+v", cfg)
	}
	if !cfg.ChargeInuse || cfg.ChargeAddr != "10.0.0.2" || cfg.ChargePort != 6000 {
		t.Fatalf("charge: %+v", cfg)
	}
	if !cfg.ManageInuse {
		t.Fatal("manage")
	}
}

func TestParseOfficialGameWorldINI(t *testing.T) {
	// 官方快照：Inner/Broadcast 全注释 → main.lua 缺省 16001/15000
	cfg, err := ParseGameWorldINI([]byte(`
[Main]
DistrictID=1
ServerID=11
ServerName=server_11
[Inner]
//Addr=127.0.0.1
//Port=5000
[Broadcast]
//Addr=127.255.255.255
//Port=4001
[Charge]
Inuse=false
`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.InnerPort != 16001 || cfg.BroadPort != 15000 || cfg.EchoPort != 15001 {
		t.Fatalf("defaults: %+v", cfg)
	}
	if cfg.ServerID != 11 {
		t.Fatal("server id")
	}
}

// ---- 真 TCP 内网面（adapter 注册 + 源路由） ----

func TestInnerRealTCP(t *testing.T) {
	w, _ := newTestWorld()
	w.SetWorldState(WStateOpened)
	port := freePort(t)
	in := NewInner(w, "127.0.0.1", port)
	// Assemble 同款接线：IC 发送面 → 真实 Adapter
	w.IC = &InterConnect{
		Send:      func(ident, dst, typ uint32, payload []byte) error { return in.sendIdent(ident, dst, typ, payload) },
		Connected: in.Connected,
	}
	if err := in.Start(); err != nil {
		t.Fatal(err)
	}
	defer in.Stop()

	conn, err := net.Dial("tcp", net.JoinHostPort("127.0.0.1", itoa(port)))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	// 0xFFFF 注册为 member 0
	sendFrame(t, conn, EncodeInner(InnerHeader{Src: MemberIdent(0), Dst: 0, Type: IdentRegister}, nil))
	time.Sleep(100 * time.Millisecond)
	if !in.Connected(MemberIdent(0)) {
		t.Fatal("member ident must register")
	}
	// member SceneState 消息 → 路由 member 队列
	w.Scenes.AddScene(5, 0, "scene", "cfg")
	b := make([]byte, 8)
	binary.LittleEndian.PutUint32(b[0:], 5)
	binary.LittleEndian.PutUint32(b[4:], uint32(SSceneRunning))
	sendFrame(t, conn, EncodeInner(InnerHeader{Src: MemberIdent(0), Dst: 0, Type: MemberMsgTypeSceneState}, b))
	time.Sleep(100 * time.Millisecond)
	w.Pump()
	if w.Scenes.GetScene(5).State != SSceneRunning {
		t.Fatal("member msg must route")
	}
	// world→member 下发（SendToMember）；此前 member 注册触发的
	// 三段同步帧可能排队在前——循环找 PlayerAdd
	if err := w.IC.SendToMember(0, 0, MemberPlayerAdd, []byte("X")); err != nil {
		t.Fatal(err)
	}
	found := false
	for i := 0; i < 8; i++ {
		got := readFrame(t, conn, time.Second)
		if len(got) < wireHeaderSize {
			break
		}
		h, payload, _ := DecodeInner(got)
		if h.Src == WorldIdent && h.Type == MemberPlayerAdd && string(payload) == "X" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("PlayerAdd frame not received")
	}
	// 非法源丢弃
	sendFrame(t, conn, EncodeInner(InnerHeader{Src: 0x99, Dst: 0, Type: 0}, nil))
	time.Sleep(50 * time.Millisecond)
}

// ---- 助手 ----

func u32b(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}

func sendFrame(t *testing.T, conn net.Conn, frame []byte) {
	t.Helper()
	out := make([]byte, 4+len(frame))
	binary.LittleEndian.PutUint32(out, uint32(len(frame)))
	copy(out[4:], frame)
	if _, err := conn.Write(out); err != nil {
		t.Fatal(err)
	}
}

func readFrame(t *testing.T, conn net.Conn, timeout time.Duration) []byte {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	var lenBuf [4]byte
	if _, err := readFull(conn, lenBuf[:]); err != nil {
		return nil
	}
	n := binary.LittleEndian.Uint32(lenBuf[:])
	body := make([]byte, n)
	if _, err := readFull(conn, body); err != nil {
		return nil
	}
	return body
}

func readFull(conn net.Conn, b []byte) (int, error) {
	total := 0
	for total < len(b) {
		n, err := conn.Read(b[total:])
		if n > 0 {
			total += n
		}
		if err != nil {
			return total, err
		}
	}
	return total, nil
}
