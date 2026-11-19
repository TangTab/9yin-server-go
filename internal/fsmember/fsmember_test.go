package fsmember

import (
	"encoding/binary"
	"io"
	"testing"
	"time"

	gonet "net"

	"github.com/lizhanyin/9yin-server-go/internal/fskit"
)

// ---- config ----

func TestParseGameMemberINI(t *testing.T) {
	cfg, err := ParseGameMemberINI([]byte(`
[System]
Sleep=1
MainEntity=MainConsole
[Main]
DistrictID=1
ServerID=11
MemberID=0
[Logic]
Modules=1
Module0=FsGame.dll
[Inner]
Addr=10.0.0.5
Port=5001
`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DistrictID != 1 || cfg.ServerID != 11 || cfg.MemberID != 0 {
		t.Fatalf("ids: %+v", cfg)
	}
	if cfg.InnerAddr != "10.0.0.5" || cfg.InnerPort != 5001 {
		t.Fatalf("inner: %s:%d", cfg.InnerAddr, cfg.InnerPort)
	}
	if len(cfg.LogicModules) != 1 || cfg.LogicModules[0] != "FsGame.dll" {
		t.Fatalf("logic modules: %v", cfg.LogicModules)
	}
	// 缺省公式：16300/15300+MemberID、广播 15000
	d, _ := ParseGameMemberINI([]byte("[Main]\nMemberID=3\n"))
	if d.InnerPort != 16303 || d.EchoPort != 15303 {
		t.Fatalf("缺省公式: inner %d echo %d", d.InnerPort, d.EchoPort)
	}
	if d.BroadAddr != "127.255.255.255" || d.BroadPort != 15000 {
		t.Fatalf("广播缺省: %s:%d", d.BroadAddr, d.BroadPort)
	}
}

func TestParseScenesMemberText(t *testing.T) {
	got := ParseScenesMemberText([]byte("Member=0: 1\r\nMember=3: 4, 118, 302, 303\r\n"))
	if len(got[0]) != 1 || got[0][0] != 1 {
		t.Fatalf("member0: %v", got[0])
	}
	want := []uint32{4, 118, 302, 303}
	if len(got[3]) != len(want) {
		t.Fatalf("member3: %v", got[3])
	}
	for i, v := range want {
		if got[3][i] != v {
			t.Fatalf("member3[%d]: %d != %d", i, got[3][i], v)
		}
	}
}

// ---- wire ----

func TestWireCodec(t *testing.T) {
	body := encodeWire(0x18, 0, 0x38, []byte{1, 2})
	hdr, payload, err := decodeWire(body)
	if err != nil {
		t.Fatal(err)
	}
	if hdr.Src != 0x18 || hdr.Dst != 0 || hdr.Type != 0x38 || len(payload) != 2 {
		t.Fatalf("%+v %v", hdr, payload)
	}
	if _, _, err := decodeWire([]byte{1, 2}); err == nil {
		t.Fatal("短帧应报 message size error")
	}
}

func TestValidSource(t *testing.T) {
	for _, ok := range []uint32{0, 1, 4, 5, 6, 7, 8, 9, 0x17} {
		if !validSource(ok) {
			t.Fatalf("%#x 应合法", ok)
		}
	}
	for _, bad := range []uint32{2, 3, 0x18, 0x28, 0xFFFF} {
		if validSource(bad) {
			t.Fatalf("%#x 应非法", bad)
		}
	}
}

func TestEntryIdentOfSerial(t *testing.T) {
	// serial 高 16 位 = EntryIdx → 8+EntryIdx（§7 定案）
	if got := EntryIdentOfSerial(0x00020003); got != 10 {
		t.Fatalf("serial %#x → ident %d != 10", 0x00020003, got)
	}
	if got := EntryIdentOfSerial(5); got != 8 {
		t.Fatalf("低 serial → ident %d != 8", got)
	}
}

// ---- 场景生命周期（官方断言串） ----

func TestSceneLifecycle(t *testing.T) {
	r := NewSceneRegistry()
	if _, err := r.CreateScene(1, ScenePrototype, "born01", 100); err != nil {
		t.Fatal(err)
	}
	// no scene creator：空名
	if _, err := r.CreateScene(2, ScenePrototype, "", 0); err == nil ||
		err.Error() != "(GameMember::CreateScene)no scene creator" {
		t.Fatalf("空名应报 no scene creator: %v", err)
	}
	// clone 生命周期
	if _, err := r.CloneScene(99, 100, "c1", 0); err == nil {
		t.Fatal("原型缺失应报错")
	}
	c, err := r.CloneScene(1, 100, "c1", 16)
	if err != nil {
		t.Fatal(err)
	}
	if c.Proto != 1 || c.Type != SceneClone {
		t.Fatalf("clone: %+v", c)
	}
	// DeleteClone 三门
	r.SetPlayerCount(100, 2)
	if err := r.DeleteClone(100); err == nil ||
		err.Error() != "(GameMember::DeleteClone)scene has player" {
		t.Fatalf("有人场景应拒删: %v", err)
	}
	r.SetPlayerCount(100, 0)
	if err := r.DeleteClone(1); err == nil ||
		err.Error() != "(GameMember::DeleteClone)scene type not clone" {
		t.Fatalf("原型不可删: %v", err)
	}
	if err := r.DeleteClone(100); err != nil {
		t.Fatal(err)
	}
	r.Remove(1) // 原型场景显式摘除（CloseMember 清场路径）
	if !r.Empty() {
		t.Fatal("清场后应空")
	}
}

// ---- 玩家注册表 ----

func TestPlayerRegistry(t *testing.T) {
	r := NewPlayerRegistry()
	if err := r.AddPlayer(7, 0x21, 0x00010005, "张三", "acct_1"); err != nil {
		t.Fatal(err)
	}
	if err := r.AddPlayer(7, 0x22, 0x00010006, "张三", "x"); err == nil {
		t.Fatal("重名应拒")
	}
	if p, ok := r.FindByAccount("acct_1"); !ok || p.Name != "张三" {
		t.Fatalf("按账号查: %+v %v", p, ok)
	}
	if err := r.ChangePlayerSceneId("张三", 8); err != nil {
		t.Fatal(err)
	}
	if p, _ := r.FindByName("张三"); p.SceneID != 8 {
		t.Fatalf("换场景: %+v", p)
	}
	if err := r.RemovePlayer(8, "张三"); err != nil {
		t.Fatal(err)
	}
	if err := r.RemovePlayer(8, "张三"); err == nil {
		t.Fatal("重复删应报 player not found")
	}
}

// ---- public 镜像 ----

func TestPublicSpacesMirror(t *testing.T) {
	p := NewPublicSpaces()
	if err := p.DeclareSpaces([]string{"global", "auction"}); err != nil {
		t.Fatal(err)
	}
	if err := p.DeclareSpaces([]string{"global", "global"}); err == nil {
		t.Fatal("重名应报 space name repeat")
	}
	if err := p.AddData("global", "rank"); err != nil {
		t.Fatal(err)
	}
	if err := p.AddData("global", "rank"); err == nil {
		t.Fatal("重复数据应报 data already exists")
	}
	if err := p.SetAttr("global", "rank", "level", "99"); err != nil {
		t.Fatal(err)
	}
	if err := p.AddRec("global", "rank", "rows", []string{"a", "b"}); err != nil {
		t.Fatal(err)
	}
	if err := p.RecAddRow("global", "rank", "rows", []string{"1", "2"}); err != nil {
		t.Fatal(err)
	}
	if err := p.RecAddRow("global", "rank", "rows", []string{"1"}); err == nil {
		t.Fatal("列数不符应报错")
	}
	if err := p.RecChange("global", "rank", "rows", 0, 1, "9"); err != nil {
		t.Fatal(err)
	}
	s, _ := p.FindSpace("global")
	if s.Datas["rank"].Attrs["level"] != "99" || s.Datas["rank"].Recs["rows"].Rows[0][1] != "9" {
		t.Fatalf("镜像内容: %+v", s.Datas["rank"])
	}
}

// ---- store 块面 ----

func TestChunkStore(t *testing.T) {
	c := NewChunkStore()
	if err := c.SetInfo(5, []uint32{1, 2, 3}); err != nil {
		t.Fatal(err)
	}
	if err := c.SetInfo(0, []uint32{1}); err == nil {
		t.Fatal("scene id error")
	}
	if err := c.Load(5, []uint32{1, 2}); err == nil {
		t.Fatal("数量不符应报 chunks amount error")
	}
	if err := c.Load(5, []uint32{1, 2, 4}); err == nil ||
		err.Error() != "(GameMember::PublishLoadChunk)set chunk data failed: 4" {
		t.Fatalf("未登记块: %v", err)
	}
	if err := c.Save(5, 2); err != nil {
		t.Fatal(err)
	}
	if err := c.Delete(5, 3); err != nil {
		t.Fatal(err)
	}
	if got := c.Snapshot(5); len(got) != 2 {
		t.Fatalf("删除后块数: %d", len(got))
	}
}

// ---- 状态机 ----

func TestStateMachineClosingGate(t *testing.T) {
	m := New(1, 11, 0)
	if MemberState(m.State.Load()) != StateUnknown {
		t.Fatal("初始 Unknown")
	}
	m.Open()
	if MemberState(m.State.Load()) != StateOpened {
		t.Fatal("Open 后应 Opened")
	}
	m.BeginClose()
	if _, err := m.Scenes.CreateScene(1, ScenePrototype, "s", 0); err != nil {
		t.Fatal(err)
	}
	m.DoWait()
	if MemberState(m.State.Load()) != StateClosing {
		t.Fatal("场景未清空应卡在 Closing")
	}
	m.Scenes.Remove(1)
	m.DoWait()
	if MemberState(m.State.Load()) != StateClosed {
		t.Fatal("场景清空后应落 Closed")
	}
}

// ---- 广播包 golden（§8 定案布局） ----

func TestBuildBroadcastGolden(t *testing.T) {
	m := New(1, 11, 2)
	pkt := m.BuildBroadcast("10.1.2.3", 16302)
	if len(pkt) != 0x28 {
		t.Fatalf("包长 %d != 0x28", len(pkt))
	}
	if binary.LittleEndian.Uint32(pkt[0x00:]) != 3 {
		t.Fatal("kind != 3")
	}
	if binary.LittleEndian.Uint32(pkt[0x04:]) != 0x300 {
		t.Fatal("magic != 0x300")
	}
	if string(pkt[0x08:0x10]) != "10.1.2.3" || pkt[0x08+8] != 0 {
		t.Fatalf("ip 区: %q", pkt[0x08:0x18])
	}
	if binary.LittleEndian.Uint32(pkt[0x18:]) != 16302 {
		t.Fatal("port 区")
	}
	if binary.LittleEndian.Uint32(pkt[0x20:]) != 11 {
		t.Fatal("ServerId 区")
	}
	if binary.LittleEndian.Uint32(pkt[0x24:]) != 2 {
		t.Fatal("MemberId 区")
	}
	// DistrictId 不入包（官方死存于包外）
	if binary.LittleEndian.Uint32(pkt[0x1C:]) != 0 {
		t.Fatal("+0x1C 应为零")
	}
}

// ---- world/entry 处理器 ----

func buildPlayerAdd(sceneID, ident, serial uint32, name, account string) []byte {
	p := make([]byte, 0x54+64)
	binary.LittleEndian.PutUint32(p[0:], sceneID)
	binary.LittleEndian.PutUint32(p[4:], ident)
	binary.LittleEndian.PutUint32(p[8:], serial)
	i := 0
	for _, r := range name {
		binary.LittleEndian.PutUint16(p[0x0C+i*2:], uint16(r))
		i++
	}
	copy(p[0x54:], account)
	return p
}

func TestWorldPlayerAddPayload(t *testing.T) {
	m := New(1, 11, 0)
	if _, err := m.Scenes.CreateScene(7, ScenePrototype, "s7", 0); err != nil {
		t.Fatal(err)
	}
	m.ProcessWorld(WorldIdent, WorldPlayerAdd, buildPlayerAdd(7, 0x21, 0x00010005, "李逍遥", "acc9"))
	if m.Players.Count() != 1 {
		t.Fatal("玩家未入表")
	}
	p, ok := m.Players.FindByName("李逍遥")
	if !ok || p.Account != "acc9" || p.SceneID != 7 || p.Ident != 0x21 {
		t.Fatalf("玩家内容: %+v", p)
	}
	if s, _ := m.Scenes.Get(7); s.CurPlayers != 1 {
		t.Fatalf("场景计数: %+v", s)
	}
	// 移除（{scene@0, ?@4, name@8}）
	rm := make([]byte, 8+72)
	binary.LittleEndian.PutUint32(rm[0:], 7)
	i := 0
	for _, r := range "李逍遥" {
		binary.LittleEndian.PutUint16(rm[8+i*2:], uint16(r))
		i++
	}
	m.ProcessWorld(WorldIdent, WorldPlayerRemove, rm)
	if m.Players.Count() != 0 {
		t.Fatal("玩家未移除")
	}
	if s, _ := m.Scenes.Get(7); s.CurPlayers != 0 {
		t.Fatalf("场景计数未回落: %+v", s)
	}
}

func TestEntryFacePaths(t *testing.T) {
	m := New(1, 11, 0)
	// online → 玩家缓冲池
	online := make([]byte, 8+4)
	binary.LittleEndian.PutUint32(online[0:], 0x21)
	binary.LittleEndian.PutUint32(online[4:], 0x00010005)
	copy(online[8:], []byte{9, 9, 9, 9})
	m.ProcessEntry(EntryIdentBase, ForwSetOnline, online)
	bufs := m.DrainPlayerBuffers()
	if len(bufs) != 1 || bufs[0].ident != 0x21 || len(bufs[0].payload) != 12 {
		t.Fatalf("缓冲池: %+v", bufs)
	}
	if got := binary.LittleEndian.Uint32(bufs[0].payload[0:]); got != 0x21 {
		t.Fatalf("缓冲头 ident: %#x", got)
	}
	if len(m.DrainPlayerBuffers()) != 0 {
		t.Fatal("Drain 应取空")
	}
	// 非 online → 场景 0x10
	var sceneGot []byte
	var sceneTyp uint32
	m.OnSceneCommand = func(sceneID uint32, typ uint32, payload []byte) {
		sceneGot = payload
		sceneTyp = typ
	}
	m.ProcessEntry(EntryIdentBase, 99, []byte{7, 7})
	if sceneTyp != SceneCmdClientMessage || string(sceneGot) != "\x07\x07" {
		t.Fatalf("场景投递: typ %#x payload %v", sceneTyp, sceneGot)
	}
}

func TestSendToClientSerialMapping(t *testing.T) {
	m := New(1, 11, 0)
	var gotIdent, gotTyp uint32
	var gotBody []byte
	m.SendToEntry = func(ident, typ uint32, payload []byte) error {
		gotIdent, gotTyp, gotBody = ident, typ, payload
		return nil
	}
	// serial=0x00030001 → entryIdx=3 → ident=11
	if err := m.SendToClient(0x77, 0x00030001, []byte{1}); err != nil {
		t.Fatal(err)
	}
	if gotIdent != 11 || gotTyp != ForwSendClient {
		t.Fatalf("下发路由: ident %d typ %#x", gotIdent, gotTyp)
	}
	if binary.LittleEndian.Uint32(gotBody[0:]) != 0x77 ||
		binary.LittleEndian.Uint32(gotBody[4:]) != 0x00030001 || gotBody[8] != 1 {
		t.Fatalf("下发帧体: %v", gotBody)
	}
}

// ---- 名字过滤 / 落地点 ----

func TestNameFilterAndLocations(t *testing.T) {
	f := NewNameFilter()
	f.Load(ParseFilterText([]byte("管理员\r\nGM 号\r\n\r\nbad\r\n")))
	if f.CheckName("我是管理员") {
		t.Fatal("违禁词应拒")
	}
	if f.CheckName("name/1") {
		t.Fatal("非法字符应拒")
	}
	if !f.CheckName("风清扬") {
		t.Fatal("正常名应过")
	}

	l := NewLocations()
	if err := l.Load([]byte("[BornCJ000001]\r\nLocation=990,6.66,532,4.7\r\nScene=ini\\scene\\born01_MoHeZhen\r\n")); err != nil {
		t.Fatal(err)
	}
	loc, ok := l.Find("BornCJ000001")
	if !ok || loc.X != 990 || loc.Z != 6.66 || loc.Y != 532 || loc.Orient != 4.7 {
		t.Fatalf("落地点: %+v", loc)
	}
	if loc.Scene != `ini\scene\born01_MoHeZhen` {
		t.Fatalf("场景路径: %q", loc.Scene)
	}
}

// ---- 内网受面 TCP 集成（注册 + 四路路由 + 非法源） ----

func readFrames(c gonet.Conn) chan []byte {
	ch := make(chan []byte, 16)
	go func() {
		hdr := make([]byte, 4)
		for {
			if _, err := io.ReadFull(c, hdr); err != nil {
				close(ch)
				return
			}
			n := binary.LittleEndian.Uint32(hdr)
			body := make([]byte, n)
			if _, err := io.ReadFull(c, body); err != nil {
				close(ch)
				return
			}
			ch <- body
		}
	}()
	return ch
}

func writeFrame(c gonet.Conn, body []byte) {
	buf := make([]byte, 4+len(body))
	binary.LittleEndian.PutUint32(buf, uint32(len(body)))
	copy(buf[4:], body)
	c.Write(buf)
}

func TestInnerRoutingTCP(t *testing.T) {
	var gotWorld, gotPublic, gotStore, gotEntry int
	in := NewInner("127.0.0.1", 0, MemberIdentBase, InnerHandler{
		OnWorld:  func(ident, typ uint32, p []byte) { gotWorld++ },
		OnPublic: func(ident, typ uint32, p []byte) { gotPublic++ },
		OnStore:  func(ident, typ uint32, p []byte) { gotStore++ },
		OnEntry:  func(ident, typ uint32, p []byte) { gotEntry++ },
		OnError:  func(format string, args ...any) { t.Logf("warn: "+format, args...) },
	})
	if err := in.Start(); err != nil {
		t.Fatal(err)
	}
	defer in.Stop()

	// world(0)/public(1)/store(4)/entry(8) 四连接注册
	idents := []uint32{WorldIdent, PublicIdent, 4, EntryIdentBase}
	conns := make(map[uint32]gonet.Conn)
	for _, id := range idents {
		c, err := gonet.Dial("tcp", in.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		conns[id] = c
		writeFrame(c, encodeWire(id, 0, IdentRegister, nil))
	}
	deadline := time.Now().Add(2 * time.Second)
	for in.ConnectedCount() < len(idents) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if in.ConnectedCount() != len(idents) {
		t.Fatalf("注册数 %d != %d", in.ConnectedCount(), len(idents))
	}

	writeFrame(conns[WorldIdent], encodeWire(WorldIdent, 0, WorldPlayerAll, []byte{0}))
	writeFrame(conns[PublicIdent], encodeWire(PublicIdent, 0, PublicSpaces, []byte{0}))
	writeFrame(conns[4], encodeWire(4, 0, StoreChunkInfo, []byte{0}))
	writeFrame(conns[EntryIdentBase], encodeWire(EntryIdentBase, 0, 99, []byte{0}))
	// 非法源（2）应被丢弃
	bad, err := gonet.Dial("tcp", in.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer bad.Close()
	writeFrame(bad, encodeWire(2, 0, IdentRegister, nil))
	time.Sleep(50 * time.Millisecond)
	writeFrame(bad, encodeWire(2, 0, WorldPlayerAll, []byte{0}))

	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) &&
		(gotWorld != 1 || gotPublic != 1 || gotStore != 1 || gotEntry != 1) {
		time.Sleep(10 * time.Millisecond)
	}
	if gotWorld != 1 || gotPublic != 1 || gotStore != 1 || gotEntry != 1 {
		t.Fatalf("路由计数 w%d p%d s%d e%d（非法源不得计入）", gotWorld, gotPublic, gotStore, gotEntry)
	}
	// 短帧丢弃不炸
	writeFrame(conns[WorldIdent], []byte{1, 2})
	time.Sleep(30 * time.Millisecond)
	if gotWorld != 1 {
		t.Fatalf("短帧后计数被污染: %d", gotWorld)
	}
	// 受面等价发面：发往已注册 ident
	if err := in.SendRaw(WorldIdent, 0x200, []byte{5}); err != nil {
		t.Fatal(err)
	}
	frames := readFrames(conns[WorldIdent])
	select {
	case f := <-frames:
		h, p, err := decodeWire(f)
		if err != nil {
			t.Fatal(err)
		}
		if h.Src != MemberIdentBase || h.Type != 0x200 || len(p) != 1 {
			t.Fatalf("回帧: %+v %v", h, p)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("回帧超时")
	}
}

// ---- DoWait 泵全链（world 入队→drain→处理器） ----

func TestDoWaitPumpE2E(t *testing.T) {
	m := New(1, 11, 0)
	var sceneCmds int
	m.OnSceneCommand = func(uint32, uint32, []byte) { sceneCmds++ }
	m.SendToWorld = func(typ uint32, p []byte) error { return nil }

	m.PutWorld(WorldIdent, WorldCreateScene, func() []byte {
		b := make([]byte, 8)
		binary.LittleEndian.PutUint32(b[0:], 9)
		binary.LittleEndian.PutUint32(b[4:], uint32(ScenePrototype))
		return b
	}())
	m.PutPublic(PublicIdent, PublicMessage, func() []byte {
		b := make([]byte, 5)
		binary.LittleEndian.PutUint32(b[0:], 9)
		b[4] = 1
		return b
	}())
	m.PutStore(4, StoreChunkInfo, func() []byte {
		b := make([]byte, 8)
		binary.LittleEndian.PutUint32(b[0:], 9)
		binary.LittleEndian.PutUint32(b[4:], 1)
		return b
	}())
	m.DoWait()
	if _, ok := m.Scenes.Get(9); !ok {
		t.Fatal("world CreateScene 未落场景表")
	}
	if sceneCmds != 1 {
		t.Fatalf("public 透传场景: %d", sceneCmds)
	}
	if len(m.Chunks.Snapshot(9)) != 1 {
		t.Fatal("store 块未落账")
	}
}

// ---- 角色装配器（Assemble/Assembly） ----

func TestConfigFromRawIni(t *testing.T) {
	// fskit.IniFile 同官方 IniDocument 面：ReadInteger/ReadString
	ini := fskit.NewIniFile("")
	if !ini.LoadFromString("[Main]\nDistrictID=2\nServerID=12\nMemberID=3\n[Inner]\nPort=5001\n[Logic]\nModule0=FsGame.dll\n") {
		t.Fatal("ini 装载失败")
	}
	cfg := ConfigFromRawIni(ini)
	if cfg.DistrictID != 2 || cfg.ServerID != 12 || cfg.MemberID != 3 {
		t.Fatalf("ids: %+v", cfg)
	}
	if cfg.InnerPort != 5001 {
		t.Fatalf("inner port 覆盖: %d", cfg.InnerPort)
	}
	if cfg.EchoPort != 15303 || cfg.BroadPort != 15000 {
		t.Fatalf("缺省公式: echo %d broad %d", cfg.EchoPort, cfg.BroadPort)
	}
	if len(cfg.LogicModules) != 1 || cfg.LogicModules[0] != "FsGame.dll" {
		t.Fatalf("logic: %v", cfg.LogicModules)
	}
}

func TestAssembleLifecycle(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MemberID = 5
	cfg.InnerPort = 0 // 随机口（测试面）
	cfg.EchoPort = 0
	asm, err := Assemble(nil, cfg, AssembleOptions{TPS: 50})
	if err != nil {
		t.Fatal(err)
	}
	if err := asm.Start(); err != nil {
		t.Fatal(err)
	}
	// echo 面：welcome + state
	e, err := gonet.Dial("tcp", asm.Echo.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 256)
	n, _ := e.Read(buf)
	if !containsStr(string(buf[:n]), "Welcome server 1 member 5") {
		t.Fatalf("welcome: %q", buf[:n])
	}
	e.Close()
	// 内网面：world 注册路由通
	w, err := gonet.Dial("tcp", asm.Inner.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	writeFrame(w, encodeWire(WorldIdent, 0, IdentRegister, nil))
	deadline := time.Now().Add(2 * time.Second)
	for asm.Inner.ConnectedCount() < 1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if asm.Inner.ConnectedCount() != 1 {
		t.Fatal("world 注册超时")
	}
	w.Close()
	asm.Stop() // 幂等停机
	asm.Stop()
	if MemberState(asm.Member.State.Load()) != StateOpened {
		t.Fatal("独立 cmd 路径 Stop 不动状态（装载器路径由 OnQuit 收尾）")
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
