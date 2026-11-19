package fspublic

import (
	"fmt"
	"bytes"
	"encoding/binary"
	"net"
	"strings"
	"testing"
	"time"
)

// 官方 GamePublic.ini 全文（Debug 快照字节级方言回归）。
const officialINI = `[System]
Sleep=1
MainEntity=MainConsole
MainScript=main

[Module]
FsKit.dll
FsComm.dll
FsPublic.dll

[Script]
Path=..\Lua\GamePublic
Reload=1

[Main]
DistrictID=1
ServerID=11
PublicLogicDll=FsPubLogic.dll

[Inner]
//Addr=127.0.0.1

[Broadcast]
//Addr=127.255.255.255
//Port=4001
`

func TestParseGamePublicINI(t *testing.T) {
	cfg, err := ParseGamePublicINI([]byte(officialINI))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DistrictID != 1 || cfg.ServerID != 11 || cfg.PublicLogicDll != "FsPubLogic.dll" {
		t.Fatalf("main 节解析错误: %+v", cfg)
	}
	// main.lua 固定缺省（无 +S 公式——区别于 member）
	if cfg.InnerAddr != "127.0.0.1" || cfg.InnerPort != 16003 {
		t.Fatalf("inner 缺省错误: %s:%d", cfg.InnerAddr, cfg.InnerPort)
	}
	if cfg.EchoAddr != "127.0.0.1" || cfg.EchoPort != 15003 {
		t.Fatalf("echo 缺省错误: %s:%d", cfg.EchoAddr, cfg.EchoPort)
	}
	if cfg.BroadAddr != "127.255.255.255" || cfg.BroadPort != 15000 {
		t.Fatalf("broadcast 缺省错误: %s:%d", cfg.BroadAddr, cfg.BroadPort)
	}
	// 覆盖路径
	cfg2, _ := ParseGamePublicINI([]byte("[Inner]\nAddr=10.0.0.1\nPort=17000\n[Echo]\nPort=15999\n"))
	if cfg2.InnerAddr != "10.0.0.1" || cfg2.InnerPort != 17000 || cfg2.EchoPort != 15999 {
		t.Fatalf("覆盖解析错误: %+v", cfg2)
	}
}

func TestArchiveCodec(t *testing.T) {
	a := NewStoreArchive(0x40)
	if err := a.WriteInt(-2); err != nil {
		t.Fatal(err)
	}
	if err := a.WriteInt64(1 << 40); err != nil {
		t.Fatal(err)
	}
	if err := a.WriteFloat(1.5); err != nil {
		t.Fatal(err)
	}
	if err := a.WriteDouble(-2.25); err != nil {
		t.Fatal(err)
	}
	if err := a.WriteString("空间"); err != nil {
		t.Fatal(err)
	}
	if err := a.WriteWideStr("公会"); err != nil {
		t.Fatal(err)
	}
	if err := a.WriteObject(7, 8); err != nil {
		t.Fatal(err)
	}
	l := NewLoadArchive(a.Bytes())
	if v, _ := l.ReadInt(); v != -2 {
		t.Fatalf("int: %d", v)
	}
	if v, _ := l.ReadInt64(); v != 1<<40 {
		t.Fatalf("int64: %d", v)
	}
	if v, _ := l.ReadFloat(); v != 1.5 {
		t.Fatalf("float: %v", v)
	}
	if v, _ := l.ReadDouble(); v != -2.25 {
		t.Fatalf("double: %v", v)
	}
	if s, _ := l.ReadString(); s != "空间" {
		t.Fatalf("string: %q", s)
	}
	if s, _ := l.ReadWideStr(); s != "公会" {
		t.Fatalf("widestr: %q", s)
	}
	if lo, hi, _ := l.ReadObject(); lo != 7 || hi != 8 {
		t.Fatalf("object: %d/%d", lo, hi)
	}
	// WriteString 线格式：len 含 NUL
	a2 := NewStoreArchive(8)
	_ = a2.WriteString("abc")
	raw := a2.Bytes()
	if binary.LittleEndian.Uint32(raw[0:]) != 4 || raw[4] != 'a' || raw[7] != 0 {
		t.Fatalf("string 编码错误: % x", raw)
	}
	// 无终结符读侧报官方串
	l2 := NewLoadArchive([]byte{1, 0, 0, 0, 'x'})
	if _, err := l2.ReadString(); err == nil || !strings.Contains(err.Error(), "String no end symbol") {
		t.Fatalf("err: %v", err)
	}
}

func TestVType(t *testing.T) {
	for _, v := range []VType{VTypeInt, VTypeInt64, VTypeFloat, VTypeDouble, VTypeString, VTypeWideStr, VTypeObject} {
		if !IsValidVType(v) {
			t.Fatalf("%d 应合法", v)
		}
	}
	if IsValidVType(1) || IsValidVType(9) {
		t.Fatal("合法域应为 [2,8]")
	}
}

func buildSampleData(t *testing.T) *PubData {
	t.Helper()
	d := NewPubData("team_main", "key1")
	d.UID = "0123456789ABCDEF0123456789ABCDEF"
	d.Save = true
	if err := d.SetAttr("level", VTypeInt, IntValue(42), true); err != nil {
		t.Fatal(err)
	}
	if err := d.SetAttr("name", VTypeWideStr, WideValue("队伍一"), false); err != nil {
		t.Fatal(err)
	}
	rc, err := d.AddRec("members", 0, 3, true)
	if err != nil {
		t.Fatal(err)
	}
	rc.ColType[0] = VTypeWideStr
	rc.ColType[1] = VTypeInt
	rc.ColType[2] = VTypeString
	if err := d.SetRecKey("members", []int{0}); err != nil {
		t.Fatal(err)
	}
	if err := rc.AddRow([]Value{WideValue("张三"), IntValue(1), StringValue("leader")}); err != nil {
		t.Fatal(err)
	}
	if err := rc.AddRow([]Value{WideValue("李四"), IntValue(2), StringValue("")}); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestPubDataBlobRoundTrip(t *testing.T) {
	d := buildSampleData(t)
	blob, err := d.StoreArchiveBlob(false)
	if err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint32(blob[0:]) != 0x50554234 {
		t.Fatalf("magic: %#x", binary.LittleEndian.Uint32(blob[0:]))
	}
	d2 := NewPubData("team_main", "")
	if err := d2.LoadArchiveBlob(blob); err != nil {
		t.Fatal(err)
	}
	if at, _ := d2.GetAttr("level"); at.Val.I64 != 42 {
		t.Fatalf("attr level: %+v", at)
	}
	if at, _ := d2.GetAttr("level"); at.Save {
		t.Fatal("blob 不含 save 位——新对象载入应 false（save 由域建属性定义）")
	}
	if at, _ := d2.GetAttr("name"); at.Val.Text != "队伍一" {
		t.Fatalf("attr name: %+v", at)
	}
	ri, ok := d2.GetRec("members")
	if !ok || len(ri.Rec.Rows) != 2 || ri.Rec.ColType[1] != VTypeInt {
		t.Fatalf("rec: %+v", ri)
	}
	if ri.Rec.Rows[1][0].Text != "李四" || ri.Rec.Rows[1][1].I64 != 2 {
		t.Fatalf("row1: %+v", ri.Rec.Rows[1])
	}
	if len(ri.Rec.Keys) != 1 || ri.Rec.Keys[0].Col != 0 {
		t.Fatalf("keys: %+v", ri.Rec.Keys)
	}
	// 预建属性（域 OnCreate 语义）后重载：save 标志合并不丢
	d4 := NewPubData("team_main", "")
	_ = d4.SetAttr("level", VTypeInt, IntValue(0), true)
	_ = d4.LoadArchiveBlob(blob)
	if at, _ := d4.GetAttr("level"); !at.Save || at.Val.I64 != 42 {
		t.Fatalf("save 合并: %+v", at)
	}
	// saveOnly 快照：只留 save 标记项
	sblob, err := d.StoreArchiveBlob(true)
	if err != nil {
		t.Fatal(err)
	}
	d3 := NewPubData("team_main", "")
	if err := d3.LoadArchiveBlob(sblob); err != nil {
		t.Fatal(err)
	}
	if _, ok := d3.GetAttr("name"); ok {
		t.Fatal("saveOnly 应滤掉非 save 属性")
	}
	if _, ok := d3.GetRec("members"); !ok {
		t.Fatal("save 标记记录应保留")
	}
}

func TestPubDataBlobValidation(t *testing.T) {
	d := buildSampleData(t)
	blob, _ := d.StoreArchiveBlob(false)
	// 坏 magic → "version not match"
	bad := append([]byte(nil), blob...)
	binary.LittleEndian.PutUint32(bad[0:], 0xdeadbeef)
	if err := NewPubData("x", "").LoadArchiveBlob(bad); err == nil ||
		!strings.Contains(err.Error(), "version not match") {
		t.Fatalf("err: %v", err)
	}
}

func TestRecordGates(t *testing.T) {
	d := NewPubData("x", "")
	if _, err := d.AddRec("r", 0, 0, false); err == nil {
		t.Fatal("cols=0 应拒绝")
	}
	if _, err := d.AddRec("r", 0, 0x101, false); err == nil {
		t.Fatal("cols>0x100 应拒绝")
	}
	rc, _ := d.AddRec("r", 1, 2, false)
	rc.ColType[0] = VTypeInt
	rc.ColType[1] = VTypeString
	_ = rc.AddRow([]Value{IntValue(1), StringValue("a")})
	if err := rc.AddRow([]Value{IntValue(2)}); err == nil {
		t.Fatal("行值数不匹配应拒绝")
	}
	if err := rc.AddRow([]Value{StringValue("x"), StringValue("a")}); err == nil {
		t.Fatal("列类型不匹配应拒绝")
	}
	if err := rc.AddRow([]Value{IntValue(2), StringValue("b")}); err == nil {
		t.Fatal("rowMax=1 应溢出拒绝")
	}
	if err := d.SetRecKey("r", []int{0}); err == nil ||
		!strings.Contains(err.Error(), "rec data exists") {
		t.Fatalf("有数据禁改键: %v", err)
	}
}

func TestSpaces(t *testing.T) {
	ss := NewSpaces()
	if err := ss.Add("pubdata", "nx_pubdata"); err != nil {
		t.Fatal(err)
	}
	if err := ss.Add("guild", "nx_guilds"); err != nil {
		t.Fatal(err)
	}
	if err := ss.Add("pubdata", "x"); err == nil ||
		!strings.Contains(err.Error(), "space exists") {
		t.Fatalf("重名: %v", err)
	}
	s, _ := ss.Find("guild")
	if s.TableName != "nx_guilds" || !s.GetCanModify() {
		t.Fatalf("space: %+v", s)
	}
	if _, err := s.Add("guild_common_pubdata", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddPubData("10086", "", ""); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Get("10086")
	if len(got.UID) != 32 {
		t.Fatalf("AddPubData 应分配 32 位 guid: %q", got.UID)
	}
	if _, err := s.Add("guild_common_pubdata", ""); err == nil ||
		!strings.Contains(err.Error(), "already exists") {
		t.Fatalf("重名数据: %v", err)
	}
	if _, ok := s.Remove("nope"); ok {
		t.Fatal("删除不存在应失败")
	}
	if _, ok := s.Remove("10086"); !ok {
		t.Fatal("删除应成功")
	}
}

func TestWire(t *testing.T) {
	body := encodeWire(PublicIdent, 0, PushAddAttr, []byte{1})
	hdr, payload, err := decodeWire(body)
	if err != nil || hdr.Src != PublicIdent || hdr.Type != PushAddAttr || len(payload) != 1 {
		t.Fatalf("wire: %+v %v", hdr, err)
	}
	if _, _, err := decodeWire(body[:8]); err == nil ||
		!strings.Contains(err.Error(), "message size error") {
		t.Fatalf("短帧: %v", err)
	}
	// 源校验
	for _, ok := range []bool{
		validSource(WorldIdent), validSource(StoreIdentBase), validSource(StoreIdentBase + 3),
		validSource(MemberIdentBase), validSource(MemberIdentMax),
	} {
		if !ok {
			t.Fatal("合法源被拒")
		}
	}
	for _, bad := range []uint32{2, 3, 8, 0x17, 0x28, 0xFFFF} {
		if validSource(bad) {
			t.Fatalf("非法源 %#x 被接受", bad)
		}
	}
	// 定长区
	h := BuildPushHeader("pubdata", "guild", "k1")
	if len(h) != 0x84 || getFixedString(h[0:]) != "pubdata" ||
		getFixedWideStr(h[nameLen:]) != "guild" || getFixedString(h[0x63:]) != "k1" {
		t.Fatalf("推送头: % x", h)
	}
}

func TestVaryMessage(t *testing.T) {
	vals := []Value{StringValue("guild"), WideValue("GuildLogic"), IntValue(100),
		Int64Value(1 << 33), FloatValue(0.5), DoubleValue(2.5), ObjectValue(1, 2)}
	a := NewStoreArchive(0x40)
	if err := PutVaryMessage(a, vals); err != nil {
		t.Fatal(err)
	}
	l := NewLoadArchive(a.Bytes())
	got, err := GetVaryMessage(l, len(vals))
	if err != nil {
		t.Fatal(err)
	}
	for i := range vals {
		if got[i].Type != vals[i].Type || got[i].I64 != vals[i].I64 ||
			got[i].F64 != vals[i].F64 || got[i].Text != vals[i].Text ||
			got[i].Obj != vals[i].Obj {
			t.Fatalf("值 %d: %+v != %+v", i, got[i], vals[i])
		}
	}
}

func newTestPublic(t *testing.T) (*Public, *capture) {
	t.Helper()
	p := New(1, 11)
	p.Spaces.Add("pubdata", "nx_pubdata")
	p.Spaces.Add("guild", "nx_guilds")
	c := &capture{}
	p.SendToMember = func(ident, typ uint32, payload []byte) error {
		c.member = append(c.member, sentMsg{ident, typ, payload})
		return nil
	}
	p.SendToAllMem = func(typ uint32, payload []byte) error {
		c.allMember = append(c.allMember, sentMsg{0, typ, payload})
		return nil
	}
	p.SendToStorage = func(typ uint32, payload []byte) error {
		c.storage = append(c.storage, sentMsg{0, typ, payload})
		return nil
	}
	p.SendToWorld = func(typ uint32, payload []byte) error {
		c.world = append(c.world, sentMsg{0, typ, payload})
		return nil
	}
	return p, c
}

type sentMsg struct {
	ident   uint32
	typ     uint32
	payload []byte
}

type capture struct {
	member    []sentMsg
	allMember []sentMsg
	storage   []sentMsg
	world     []sentMsg
}

func (c *capture) reset() {
	c.member, c.allMember, c.storage, c.world = nil, nil, nil, nil
}

func TestMemberRegister(t *testing.T) {
	p, c := newTestPublic(t)
	// State != Opened：静默（无应答）
	p.ProcessMemberMsg(MemberIdentBase, MemberForwRegister, make([]byte, 8))
	if len(c.member) != 0 {
		t.Fatal("未开放不应应答")
	}
	// 长度门
	p.setState(StateOpened)
	p.ProcessMemberMsg(MemberIdentBase, MemberForwRegister, make([]byte, 7))
	if len(c.member) != 0 {
		t.Fatal("长度不符不应应答")
	}
	p.ProcessMemberMsg(MemberIdentBase, MemberForwRegister, make([]byte, 8))
	if len(c.member) != 1 || c.member[0].typ != PushSpaces || c.member[0].ident != MemberIdentBase {
		t.Fatalf("应答: %+v", c.member)
	}
	l := NewLoadArchive(c.member[0].payload)
	n, _ := l.ReadInt()
	if n != 2 {
		t.Fatalf("空间数: %d", n)
	}
	s1, _ := l.ReadString()
	tb1, _ := l.ReadString()
	cnt1, _ := l.ReadInt()
	if s1 != "pubdata" || tb1 != "nx_pubdata" || cnt1 != 0 {
		t.Fatalf("清单: %s %s %d", s1, tb1, cnt1)
	}
	s2, _ := l.ReadString()
	if s2 != "guild" {
		t.Fatalf("清单2: %s", s2)
	}
}

func TestMemberData(t *testing.T) {
	p, c := newTestPublic(t)
	s, _ := p.Spaces.Find("guild")
	d, _ := s.AddPubData("GuildLogic", "gk", "UID0")
	_ = d.SetAttr("capital", VTypeInt64, Int64Value(999), true)
	p.setState(StateOpened)

	req := make([]byte, 0x29)
	binary.LittleEndian.PutUint32(req[4:], 0) // index
	putFixedString(req[8:], "guild")
	p.ProcessMemberMsg(MemberIdentBase, MemberForwData, req)
	if len(c.member) != 1 || c.member[0].typ != PushData {
		t.Fatalf("应答: %+v", c.member)
	}
	pl := c.member[0].payload
	if len(pl) < 0x90 {
		t.Fatalf("应答长度: %d", len(pl))
	}
	if binary.LittleEndian.Uint32(pl[0:]) != 1 { // count
		t.Fatal("count 错")
	}
	if getFixedString(pl[8:]) != "guild" || getFixedWideStr(pl[0x29:]) != "GuildLogic" ||
		getFixedString(pl[0x6B:]) != "gk" {
		t.Fatal("头字段错")
	}
	if binary.LittleEndian.Uint32(pl[0x8C:]) != 1 {
		t.Fatal("常量 1 错")
	}
	d2 := NewPubData("x", "")
	if err := d2.LoadArchiveBlob(pl[0x90:]); err != nil {
		t.Fatalf("快照: %v", err)
	}
	if at, _ := d2.GetAttr("capital"); at.Val.I64 != 999 {
		t.Fatal("快照值错")
	}
	// 空间不存在
	putFixedString(req[8:], "nope")
	p.ProcessMemberMsg(MemberIdentBase, MemberForwData, req)
	if len(c.member) != 1 {
		t.Fatal("坏空间不应应答")
	}
}

// fakeCallee 记录回调。
type fakeCallee struct {
	starts, closes, spaceLoads, dataLoads, frames, states, offlines int
	lastMsgID  uint32
	lastArgs   []Value
	lastServer ServerState
}

func (f *fakeCallee) OnPublicStart(k *Kernel)          { f.starts++ }
func (f *fakeCallee) OnServerClose(k *Kernel)          { f.closes++ }
func (f *fakeCallee) OnPubSpaceLoaded(k *Kernel, s string) { f.spaceLoads++ }
func (f *fakeCallee) OnPubDataLoaded(k *Kernel, s, d string) { f.dataLoads++ }
func (f *fakeCallee) OnFrameExecute(k *Kernel, el int) { f.frames++ }
func (f *fakeCallee) OnServerState(k *Kernel, st ServerState) {
	f.states++
	f.lastServer = st
}
func (f *fakeCallee) OnRoleOffline(k *Kernel, rows [][]Value) { f.offlines++ }
func (f *fakeCallee) OnPublicMessage(k *Kernel, ident, id uint32, args []Value) bool {
	f.lastMsgID = id
	f.lastArgs = args
	return true
}

func TestStateMachineLoadChain(t *testing.T) {
	p, c := newTestPublic(t)
	fc := &fakeCallee{}
	p.Callee = fc
	s, _ := p.Spaces.Find("guild")
	d, _ := s.Add("GuildLogic", "")
	d.Save = true

	// store 连接 → Unknown 发首空间 0x1E
	p.PublicStateMachine(smEventStoreConn, 0)
	if len(c.storage) != 1 || c.storage[0].typ != StorePubInfoReq {
		t.Fatalf("0x1E: %+v", c.storage)
	}
	if getFixedString(c.storage[0].payload) != "pubdata" ||
		getFixedString(c.storage[0].payload[nameLen:]) != "nx_pubdata" {
		t.Fatal("0x1E 载荷错误")
	}
	if PublicState(p.State.Load()) != StateLoadSpaces {
		t.Fatal("应进入 LoadSpaces")
	}
	// 0x13 应答（pubdata，0 条数据）→ 立即空间完 → 下一空间 0x1E
	c.reset()
	info := make([]byte, 0x4A)
	binary.LittleEndian.PutUint32(info[0:], 1)
	putFixedString(info[4:], "pubdata")
	putFixedString(info[4+nameLen:], "nx_pubdata")
	binary.LittleEndian.PutUint32(info[4+nameLen*2:], 0)
	p.ProcessStoreMsg(StoreIdentBase, StoreBackPubInfo, info)
	if len(c.storage) != 1 || c.storage[0].typ != StorePubInfoReq ||
		getFixedString(c.storage[0].payload) != "guild" {
		t.Fatalf("第二空间 0x1E: %+v", c.storage)
	}
	if fc.spaceLoads != 1 {
		t.Fatal("pubdata 空间装载完回调未触发")
	}
	// 0x13（guild，1 条）→ 0x1F 首条
	c.reset()
	info2 := make([]byte, 0x4A+wnameLen)
	binary.LittleEndian.PutUint32(info2[0:], 1)
	putFixedString(info2[4:], "guild")
	putFixedString(info2[4+nameLen:], "nx_guilds")
	binary.LittleEndian.PutUint32(info2[4+nameLen*2:], 1)
	putFixedWideStr(info2[0x4A:], "GuildLogic")
	p.ProcessStoreMsg(StoreIdentBase, StoreBackPubInfo, info2)
	if len(c.storage) != 1 || c.storage[0].typ != StorePubLoadReq {
		t.Fatalf("0x1F: %+v", c.storage)
	}
	req := c.storage[0].payload
	if binary.LittleEndian.Uint32(req[0:]) != 1 || binary.LittleEndian.Uint32(req[4:]) != 0 ||
		getFixedWideStr(req[8+nameLen*2:]) != "GuildLogic" {
		t.Fatal("0x1F 载荷错误")
	}
	// 0x14 应答（装载 blob）→ 全齐 → event2 → Opened + 广播清单
	c.reset()
	blob, _ := d.StoreArchiveBlob(false)
	back := make([]byte, 0xB1+len(blob))
	binary.LittleEndian.PutUint32(back[0:], 1)
	binary.LittleEndian.PutUint32(back[8:], 0)
	putFixedString(back[0xC:], "guild")
	putFixedWideStr(back[0x4E:], "GuildLogic")
	putFixedString(back[0x90:], "GUID1")
	copy(back[0xB1:], blob)
	p.ProcessStoreMsg(StoreIdentBase, StoreBackPubLoad, back)
	if PublicState(p.State.Load()) != StateOpened {
		t.Fatalf("应 Opened: %s", PublicState(p.State.Load()))
	}
	if len(c.allMember) != 1 || c.allMember[0].typ != PushSpaces {
		t.Fatalf("空间清单广播: %+v", c.allMember)
	}
	if fc.starts != 1 || fc.dataLoads != 1 || fc.spaceLoads != 2 {
		t.Fatalf("callee: %+v", fc)
	}
	got, _ := s.Get("GuildLogic")
	if got.UID != "GUID1" {
		t.Fatalf("SetUid: %q", got.UID)
	}
}

func TestStateMachineClosing(t *testing.T) {
	p, c := newTestPublic(t)
	fc := &fakeCallee{}
	p.Callee = fc
	s, _ := p.Spaces.Find("guild")
	d1, _ := s.Add("d1", "")
	d1.Save, d1.Modify = true, true
	d2, _ := s.Add("d2", "")
	d2.Save, d2.Modify = true, false // Modify=false 不回存
	p.setState(StateOpened)

	// world 关闭令 → Callee OnServerClose → Closing
	p.ProcessWorldMsg(WorldIdent, WorldClosePublic, nil)
	if fc.closes != 1 || PublicState(p.State.Load()) != StateClosing {
		t.Fatalf("closing: %s", PublicState(p.State.Load()))
	}
	// tick → 回存 d1（0x20）→ 全完 → world(2,{1}) → Closed
	p.PublicStateMachine(smEventTick, 1000)
	if len(c.storage) != 1 || c.storage[0].typ != StorePubSave {
		t.Fatalf("保存: %+v", c.storage)
	}
	pl := c.storage[0].payload
	if getFixedString(pl[0:]) != "guild" || getFixedString(pl[nameLen:]) != "nx_guilds" ||
		getFixedWideStr(pl[nameLen*2:]) != "d1" {
		t.Fatal("0x20 载荷错误")
	}
	if len(c.world) != 1 || c.world[0].typ != WorldCloseDone ||
		binary.LittleEndian.Uint32(c.world[0].payload) != 1 {
		t.Fatalf("CloseDone: %+v", c.world)
	}
	if PublicState(p.State.Load()) != StateClosed {
		t.Fatal("应 Closed")
	}
	// Closed 后 tick 空转
	p.PublicStateMachine(smEventTick, 1000)
	if len(c.storage) != 1 {
		t.Fatal("Closed 不应再保存")
	}
}

func TestWorldServerState(t *testing.T) {
	p, _ := newTestPublic(t)
	fc := &fakeCallee{}
	p.Callee = fc
	pl := make([]byte, 0x24)
	copy(pl[0:8], "srv-a\x00\x00\x00")
	for i := 0; i < 5; i++ {
		binary.LittleEndian.PutUint32(pl[0x10+i*4:], uint32(i+1))
	}
	p.ProcessWorldMsg(WorldIdent, WorldServerState, pl)
	if fc.states != 1 || fc.lastServer.Name != "srv-a" || fc.lastServer.Props[4] != 5 {
		t.Fatalf("state: %+v", fc.lastServer)
	}
	p.ProcessWorldMsg(WorldIdent, WorldServerState, pl[:10]) // 长度门
	if fc.states != 1 {
		t.Fatal("长度不符不应回调")
	}
}

func TestBroadcastPacket(t *testing.T) {
	p, _ := newTestPublic(t)
	pkt := p.BuildBroadcast("127.0.0.1", 16003)
	if len(pkt) != 0x24 {
		t.Fatalf("长度: %d", len(pkt))
	}
	if binary.LittleEndian.Uint32(pkt[0:]) != 4 || binary.LittleEndian.Uint32(pkt[4:]) != 0x300 {
		t.Fatal("kind/magic 错")
	}
	if getFixedString(pkt[8:]) != "127.0.0.1" {
		t.Fatalf("ip（ASCII 点分串）: % x", pkt[8:16])
	}
	if binary.LittleEndian.Uint32(pkt[0x18:]) != 16003 {
		t.Fatal("port 错")
	}
	if binary.LittleEndian.Uint32(pkt[0x1C:]) != 1 || binary.LittleEndian.Uint32(pkt[0x20:]) != 11 {
		t.Fatal("district/server 错")
	}
}

func TestDefaultCalleeCommonMessages(t *testing.T) {
	p, c := newTestPublic(t)
	dc := NewDefaultCallee()
	p.Callee = dc
	s, _ := p.Spaces.Find("guild")
	d, _ := s.AddPubData("GuildLogic", "gk", "U1")
	rc, _ := d.AddRec("members", 0, 2, true)
	rc.ColType[0] = VTypeWideStr
	rc.ColType[1] = VTypeInt
	p.setState(StateOpened)

	send := func(id uint32, operands ...Value) bool {
		// 官方 varlist：{space, dataName, msgID@2, operands@3+}
		args := append([]Value{StringValue("guild"), WideValue("GuildLogic"), IntValue(int32(id))}, operands...)
		return dc.OnPublicMessage(p.kernel(), MemberIdentBase, id, args)
	}
	// 1000 SetPubDataProp
	if !send(MsgSetPubDataProp, WideValue("capital"), Int64Value(500)) {
		t.Fatal("1000 失败")
	}
	if at, _ := d.GetAttr("capital"); at.Val.I64 != 500 || !d.Modify {
		t.Fatal("属性未写/Modify 未标")
	}
	if len(c.allMember) != 1 || c.allMember[0].typ != PushAddAttr {
		t.Fatalf("op4: %+v", c.allMember)
	}
	// 1102 AddPubDataRecRowValue
	c.reset()
	if !send(MsgAddPubDataRecRowValue, WideValue("members"), WideValue("张三"), IntValue(7)) {
		t.Fatal("1102 失败")
	}
	if len(rc.Rows) != 1 || rc.Rows[0][1].I64 != 7 {
		t.Fatal("行未写")
	}
	if len(c.allMember) != 1 || c.allMember[0].typ != PushRecAddRow {
		t.Fatalf("op8: %+v", c.allMember)
	}
	// 1103 SetPubDataRecValue
	c.reset()
	if !send(MsgSetPubDataRecValue, WideValue("members"), IntValue(0), IntValue(1), IntValue(9)) {
		t.Fatal("1103 失败")
	}
	if rc.Rows[0][1].I64 != 9 {
		t.Fatal("格未写")
	}
	if len(c.allMember) != 1 || c.allMember[0].typ != PushRecChange {
		t.Fatalf("opB: %+v", c.allMember)
	}
	// 1101 RemovePubDataRecRow
	c.reset()
	if !send(MsgRemovePubDataRecRow, WideValue("members"), IntValue(0)) {
		t.Fatal("1101 失败")
	}
	if len(rc.Rows) != 0 || len(c.allMember) != 1 || c.allMember[0].typ != PushRecRemoveRow {
		t.Fatalf("op9: %+v", c.allMember)
	}
	// 1100 ClearPubDataRec
	if !send(MsgClearPubDataRec, WideValue("members")) {
		t.Fatal("1100 失败")
	}
	// 1001 IncPubDataProp
	if !send(MsgIncPubDataProp, WideValue("capital"), IntValue(50)) {
		t.Fatal("1001 失败")
	}
	if at, _ := d.GetAttr("capital"); at.Val.I64 != 550 {
		t.Fatal("自增错")
	}
	// 非公共区间且无域 → false
	if send(0x64) {
		t.Fatal("无域应 false")
	}
}

func TestTimer(t *testing.T) {
	tm := NewPubTimer()
	fired := 0
	if err := tm.Add("domain/HB", 60_000, func(n string, el int, d []Value) { fired++ }, nil); err != nil {
		t.Fatal(err)
	}
	if err := tm.Add("domain/HB", 1000, nil, nil); err == nil ||
		!strings.Contains(err.Error(), "already exists") {
		t.Fatalf("重名: %v", err)
	}
	if err := tm.Add("bad", 0, nil, nil); err == nil {
		t.Fatal("间隔 0 应拒绝")
	}
	for i := 0; i < 59; i++ {
		tm.Run(1000)
	}
	if fired != 0 {
		t.Fatal("59s 不应触发")
	}
	tm.Run(1000)
	if fired != 1 {
		t.Fatal("60s 应触发")
	}
	tm.Remove("domain/HB")
	tm.Run(60_000)
	if fired != 1 {
		t.Fatal("删除后不应触发")
	}
}

func TestKernelAPIs(t *testing.T) {
	p, c := newTestPublic(t)
	k := p.kernel()
	// RequestServerState → world type 1（1B）
	if err := k.RequestServerState(); err != nil {
		t.Fatal(err)
	}
	if len(c.world) != 1 || c.world[0].typ != WorldRequestState || len(c.world[0].payload) != 1 {
		t.Fatalf("world: %+v", c.world)
	}
	// RequestLoadPlayer → 0x67B
	if err := k.RequestLoadPlayer("acc001", "角色一", MemberIdentBase); err != nil {
		t.Fatal(err)
	}
	pl := c.world[1].payload
	if len(pl) != 0x67 || getFixedString(pl) != "acc001" ||
		getFixedWideStr(pl[nameLen:]) != "角色一" ||
		binary.LittleEndian.Uint32(pl[nameLen+wnameLen:]) != MemberIdentBase {
		t.Fatal("LoadPlayer 载荷错误")
	}
	// SaveLog 引号门 + 0x44A
	if err := k.SaveLog("t", "n", "it's"); err == nil ||
		!strings.Contains(err.Error(), "log content error") {
		t.Fatalf("引号门: %v", err)
	}
	if err := k.SaveLog("title", "name", "text"); err != nil {
		t.Fatal(err)
	}
	if len(c.storage) != 1 || c.storage[0].typ != StoreSaveLog || len(c.storage[0].payload) != 0x44A {
		t.Fatalf("SaveLog: %+v", c.storage)
	}
	// SendPublicMessage（Opened 门）
	if err := k.SendPublicMessage("guild", "GuildLogic", 0x64, nil); err == nil {
		t.Fatal("未开放应拒绝")
	}
	p.setState(StateOpened)
	if err := k.SendPublicMessage("guild", "GuildLogic", 0x64,
		[]Value{WideValue("会一")}); err != nil {
		t.Fatal(err)
	}
	if len(c.allMember) != 1 || c.allMember[0].typ != PushMessage {
		t.Fatalf("0xD: %+v", c.allMember)
	}
	mp := c.allMember[0].payload
	if binary.LittleEndian.Uint32(mp[0:]) != 0x64 || binary.LittleEndian.Uint32(mp[4:]) != 1 {
		t.Fatal("0xD 头错误")
	}
	// MemberMessage 透传 → Callee
	fc := &fakeCallee{}
	p.Callee = fc
	arc := NewStoreArchive(0x40)
	_ = arc.WriteInt(0x65)
	_ = arc.WriteInt(3) // argc
	_ = PutVaryMessage(arc, []Value{StringValue("guild"), WideValue("GuildLogic"), IntValue(0x65)})
	p.ProcessMemberMsg(MemberIdentBase, MemberForwMessage, arc.Bytes())
	if fc.lastMsgID != 0x65 || len(fc.lastArgs) != 3 {
		t.Fatalf("透传: %d %+v", fc.lastMsgID, fc.lastArgs)
	}
}

// TestInnerTCPRealLoop 真 TCP：member 注册 → Register/Data 应答；
// store 注册触发装载链。
func TestInnerTCPRealLoop(t *testing.T) {
	p := New(1, 11)
	p.Spaces.Add("pubdata", "nx_pubdata")
	p.Spaces.Add("guild", "nx_guilds")
	fc := &fakeCallee{}
	p.Callee = fc

	disp := func(src, ident, typ uint32, payload []byte) {
		switch {
		case src == WorldIdent:
			p.PutWorldMessage(src, typ, payload)
		case src >= StoreIdentBase && src < StoreIdentBase+4:
			p.PutStoreMessage(src, typ, payload)
		default:
			p.PutMemberMessage(src, typ, payload)
		}
	}
	in := NewInner("127.0.0.1", 0, InnerHandler{}, disp)
	if err := in.Start(); err != nil {
		t.Fatal(err)
	}
	defer in.Stop()
	port := in.Addr().(*net.TCPAddr).Port

	p.SendToMember = func(ident, typ uint32, payload []byte) error {
		return in.SendRaw(ident, typ, payload)
	}
	p.SendToAllMem = func(typ uint32, payload []byte) error { return nil }
	p.SendToStorage = func(typ uint32, payload []byte) error { return nil }
	p.SendToWorld = func(typ uint32, payload []byte) error { return nil }

	// member 连接 + 注册 + Register 请求
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	memberWrite := func(typ uint32, payload []byte) {
		frame := encodeWire(MemberIdentBase, 0, typ, payload)
		var buf bytes.Buffer
		var lenb [4]byte
		binary.LittleEndian.PutUint32(lenb[:], uint32(len(frame)))
		buf.Write(lenb[:])
		buf.Write(frame)
		_, _ = conn.Write(buf.Bytes())
	}
	memberWrite(IdentRegister, nil)
	p.setState(StateOpened)
	memberWrite(MemberForwRegister, make([]byte, 8))

	deadline := time.Now().Add(2 * time.Second)
	var got []byte
	buf := make([]byte, 0x1000)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		n, err := conn.Read(buf)
		if n > 0 {
			got = append(got, buf[:n]...)
			if len(got) >= 16 {
				break
			}
		}
		if err != nil {
			break
		}
	}
	p.DoWait()
	// 再读一轮应答
	_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			got = append(got, buf[:n]...)
		}
		if err != nil || n == 0 {
			break
		}
		if len(got) > 64 {
			break
		}
	}
	if len(got) < 16 {
		t.Fatalf("应答不足: % x", got)
	}
	// 帧解析：u32 长度 + {src=1,dst=0,type=0} + 空间清单
	flen := binary.LittleEndian.Uint32(got[0:])
	if flen < 12 || int(flen) > len(got)-4 {
		t.Fatalf("帧长: %d/%d", flen, len(got))
	}
	if binary.LittleEndian.Uint32(got[4:]) != PublicIdent ||
		binary.LittleEndian.Uint32(got[12:]) != PushSpaces {
		t.Fatalf("应答帧: % x", got[:20])
	}
	l := NewLoadArchive(got[16 : 4+flen])
	n, _ := l.ReadInt()
	if n != 2 {
		t.Fatalf("空间数: %d", n)
	}
}
