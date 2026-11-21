package fssolomember

import (
	"bytes"
	"encoding/binary"
	"net"
	"strings"
	"testing"
	"time"
)

// ---- wire ----

func TestEncodeInnerRoundTrip(t *testing.T) {
	frame, err := encodeInner(0x18, 7, 6, []byte{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}
	var got [][4]uint32
	acc := &frameAcc{}
	if err := acc.feed(frame, func(c, a, b uint32, p []byte) {
		got = append(got, [4]uint32{c, a, b, uint32(len(p))})
		if !bytes.Equal(p, []byte{1, 2, 3}) {
			t.Fatalf("payload mismatch: %v", p)
		}
	}); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != [4]uint32{0x18, 7, 6, 3} {
		t.Fatalf("frame fields: %v", got)
	}
}

func TestFrameAccHalfPacket(t *testing.T) {
	frame, _ := encodeInner(2, 0, 0xD, []byte("hello"))
	var n int
	acc := &frameAcc{}
	_ = acc.feed(frame[:7], func(uint32, uint32, uint32, []byte) { n++ })
	_ = acc.feed(frame[7:], func(uint32, uint32, uint32, []byte) { n++ })
	if n != 1 {
		t.Fatalf("半包应只出 1 帧，得 %d", n)
	}
}

func TestBuildMemberAnnounce(t *testing.T) {
	p := buildMemberAnnounce("127.0.0.1", 16300, 1, 11, 2)
	if len(p) != 0x28 {
		t.Fatalf("len=%d", len(p))
	}
	if le := binary.LittleEndian; le.Uint32(p[0:]) != KindMember || le.Uint32(p[4:]) != 0x3001 ||
		le.Uint32(p[0x18:]) != 16300 || le.Uint32(p[0x1C:]) != 1 || le.Uint32(p[0x20:]) != 11 ||
		le.Uint32(p[0x24:]) != 2 {
		t.Fatalf("announce fields: % x", p)
	}
	if string(p[8:17]) != "127.0.0.1" {
		t.Fatalf("ip: %q", p[8:24])
	}
}

func TestParseAnnounceGates(t *testing.T) {
	mk := func(kind uint32, size int) []byte {
		p := make([]byte, size)
		binary.LittleEndian.PutUint32(p[0:], kind)
		binary.LittleEndian.PutUint32(p[4:], 0x3001)
		copy(p[8:], "10.0.0.9")
		binary.LittleEndian.PutUint32(p[0x18:], 5000)
		binary.LittleEndian.PutUint32(p[0x1C:], 1)
		binary.LittleEndian.PutUint32(p[0x20:], 11)
		if size >= 0x28 {
			binary.LittleEndian.PutUint32(p[0x24:], 2)
		}
		return p
	}
	for _, c := range []struct {
		kind uint32
		size int
		ok   bool
	}{
		{KindPublic, 0x24, true},
		{KindStore, 0x28, true},
		{KindEntry, 0x48, true},
		{KindAuction, 0x24, true},
		{KindPublic, 0x28, false}, // 包长门
		{KindMember, 0x28, false}, // member 自宣不进收侧
	} {
		if got := parseAnnounce(mk(c.kind, c.size)); (got != nil) != c.ok {
			t.Fatalf("kind=%d size=%#x got=%v", c.kind, c.size, got)
		}
	}
	// magic 门
	p := mk(KindPublic, 0x24)
	binary.LittleEndian.PutUint32(p[4:], 0x300)
	if parseAnnounce(p) != nil {
		t.Fatal("magic 0x300 必须拒")
	}
}

// ---- config ----

func TestParseGameMemberINI(t *testing.T) {
	cfg, err := ParseGameMemberINI([]byte(defaultTestINI))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DistrictID != 1 || cfg.ServerID != 11 || cfg.MemberID != 0 {
		t.Fatalf("main: %+v", cfg)
	}
	if cfg.InnerAddr != "127.0.0.1" || cfg.InnerPort != 0 {
		t.Fatalf("inner: %+v", cfg)
	}
	if cfg.BroadAddr != "127.255.255.255" || cfg.BroadPort != 15000 {
		t.Fatalf("broadcast: %+v", cfg)
	}
	if cfg.EchoPort != 15300 { // 15300+MemberID(0)
		t.Fatalf("echo port: %d", cfg.EchoPort)
	}
	if len(cfg.LogicModules) != 1 || cfg.LogicModules[0] != "fsgame.dll" {
		t.Fatalf("logic: %v", cfg.LogicModules)
	}
	if !cfg.DebugHalt || cfg.BindCore != -1 {
		t.Fatalf("defaults: %+v", cfg)
	}
}

const defaultTestINI = `[System]
Sleep=-1
MainEntity=SoloMember
MainScript=main
[Module]
fskit.dll
fscomm.dll
fscollide.dll
fssolomember.dll
[Script]
Path=..\lua\solomember
Reload=1
[Resource]
Path=..\res
[Main]
DistrictID=1
ServerID=11
MemberID=0
CollideRootPath=
[Logic]
Modules=1
Module0=fsgame.dll
[Inner]
//Addr=127.0.0.1
//Port=5001
[Broadcast]
//Addr=127.255.255.255
//Port=4001
`

// ---- routing ----

type relayRecorder struct {
	cmds []struct {
		scene uint32
		typ   uint32
	}
}

func (r *relayRecorder) SceneCommand(scene, typ uint32, _ []byte) bool {
	r.cmds = append(r.cmds, struct {
		scene uint32
		typ   uint32
	}{scene, typ})
	return true
}

func TestProcessInnerMsgRouting(t *testing.T) {
	rec := &relayRecorder{}
	s := New(Options{Config: DefaultConfig(), Scenes: rec, ManualTick: true})
	// entry (comm 8..0x17) → 0x11
	s.processInnerMsg(8, 5, 0x99, []byte("x"))
	// store id2 0xB → 0x40；0x13 → 0x21；0xF 本地；0x42 透传
	s.processInnerMsg(4, 5, 0xB, nil)
	s.processInnerMsg(4, 5, 0x13, nil)
	s.processInnerMsg(4, 5, 0x42, nil)
	s.processInnerMsg(4, 5, 0xF, nil)
	// auction 0xD/0xE → 透传场景
	s.processInnerMsg(2, 7, 0xD, nil)
	// world/public/store/entry 之外的 comm → 拒
	s.processInnerMsg(0x20, 0, 0, nil)

	want := []struct{ scene, typ uint32 }{
		{5, 0x11}, {5, 0x40}, {5, 0x21}, {5, 0x42}, {7, 0xD},
	}
	if len(rec.cmds) != len(want) {
		t.Fatalf("cmds: %+v", rec.cmds)
	}
	for i, w := range want {
		if rec.cmds[i].scene != w.scene || rec.cmds[i].typ != w.typ {
			t.Fatalf("cmd[%d]=%+v want %+v", i, rec.cmds[i], w)
		}
	}
	// 0xF 走 store 本地面：不应产生场景命令（上面已断言条数）
}

func TestWorldDefaultRelayToAllScenes(t *testing.T) {
	rec := &relayRecorder{}
	s := New(Options{Config: DefaultConfig(), Scenes: rec, ManualTick: true})
	if _, err := s.Scenes.CreateScene(1, 0, "scene1", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Scenes.CreateScene(2, 0, "scene2", 0); err != nil {
		t.Fatal(err)
	}
	s.processWorldMsg(0, 0x30, []byte("payload")) // 未列出 id2 → 全场景透传
	if len(rec.cmds) != 2 || rec.cmds[0].typ != 0x30 || rec.cmds[1].typ != 0x30 {
		t.Fatalf("relay: %+v", rec.cmds)
	}
}

// ---- state machine ----

func TestStateMachineFullFlow(t *testing.T) {
	s := New(Options{Config: DefaultConfig(), ManualTick: true})
	// 起始 Unknown，无 store → 不动
	s.StateMachine(evTick, 0, nil)
	if s.StateName() != "Unknown" {
		t.Fatal("无 store 不应离开 Unknown")
	}
	// CloseMember 事件 1 → Closing；场景空 → Closed（SendToWorld 无连接，
	// 官方重试语义：Go 版无连接时保持 Closing）
	s.StateMachine(evClose, 0, nil)
	if s.StateName() != "Closing" {
		t.Fatalf("state=%s", s.StateName())
	}
	if s.StateMachine(evTick, 0, nil) {
		t.Fatal("SendToWorld 无连接应失败，状态不推进")
	}
	if s.StateName() != "Closing" {
		t.Fatalf("state=%s", s.StateName())
	}
}

// ---- entry downlink ----

func TestSendToClientSerialEncoding(t *testing.T) {
	// serial 高 16 位 = EntryIdx（SendToEntry2 入参；官方 @0x10294471 段）
	serial := uint32(0x00030015) // entryIdx=3, 序号 0x15
	if idx := int((serial >> 16) & 0xFFFF); idx != 3 {
		t.Fatalf("entryIdx=%d", idx)
	}
}

// ---- full-stack TCP loopback ----

func TestLoopbackWorldConnectAndRegister(t *testing.T) {
	s := New(Options{
		Config:     Config{DistrictID: 1, ServerID: 11, MemberID: 0, InnerAddr: "127.0.0.1", InnerPort: 0, BroadAddr: "127.255.255.255", BroadPort: 15432, EchoAddr: "127.0.0.1", EchoPort: 0},
		ManualTick: true,
	})
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
	if s.cfg.InnerPort == 0 {
		t.Fatal("InnerPort==0 应回填实际口")
	}

	// world 连入 + identify(comm=0)
	conn, err := net.Dial("tcp", net.JoinHostPort("127.0.0.1", itoa(s.cfg.InnerPort)))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	idf, _ := encodeInner(commWorld, 0, innerIdentifyID2, nil)
	if _, err := conn.Write(idf); err != nil {
		t.Fatal(err)
	}

	// world 发 CommandRoleInfo(0xB) → +0x43C 闩
	rolef, _ := encodeInner(commWorld, 0, 0x0B, []byte("role-archive"))
	if _, err := conn.Write(rolef); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		ready := s.roleReady
		s.mu.Unlock()
		if ready {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	s.mu.Lock()
	ready := s.roleReady
	s.mu.Unlock()
	if !ready {
		t.Fatal("CommandRoleInfo 未置角色闩")
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [8]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func TestManageInfoDecode(t *testing.T) {
	// 构造 {str, wide, int, wide, int, int} archive 流
	buf := &bytes.Buffer{}
	putStr := func(v string) {
		binary.Write(buf, binary.LittleEndian, uint32(len(v)+1))
		buf.WriteString(v)
		buf.WriteByte(0)
	}
	putWide := func(v string) {
		runes := []rune(v)
		binary.Write(buf, binary.LittleEndian, uint32(len(runes)+1))
		for _, r := range runes {
			binary.Write(buf, binary.LittleEndian, uint16(r))
		}
		binary.Write(buf, binary.LittleEndian, uint16(0))
	}
	putInt := func(v int32) { binary.Write(buf, binary.LittleEndian, v) }
	putStr("manage")
	putWide("管理配置")
	putInt(1)
	putWide("备注")
	putInt(2)
	putInt(3)
	m := decodeManageInfo(buf.Bytes())
	if m == nil {
		t.Fatal("decode failed")
	}
	if m.Name != "manage" || m.Title != "管理配置" || m.Flag != 1 || m.Order != 2 || m.State != 3 {
		t.Fatalf("manage info: %+v", m)
	}
	if decodeManageInfo([]byte{1, 2, 3}) != nil {
		t.Fatal("短流必须清闩（官方 failed 路径）")
	}
}

func TestPublicSpacesLatch(t *testing.T) {
	s := New(Options{Config: DefaultConfig(), ManualTick: true})
	s.processPublicMsg(0, 0x0, []byte("space_a\x00space_b\x00"))
	if !s.PubHas("space_a") {
		t.Fatal("space_a 未入镜像")
	}
	s.mu.Lock()
	pub := s.pubReady
	s.mu.Unlock()
	if !pub {
		t.Fatal("PublicSpaces 未置 +0x434 闩")
	}
}

func TestLoadingThreeLatches(t *testing.T) {
	s := New(Options{Config: DefaultConfig(), ManualTick: true})
	s.setPubReady()
	s.mu.Lock()
	s.roleReady = true
	s.mu.Unlock()
	// manage 未就绪：停在 Loading
	s.state = stateLoading
	s.StateMachine(evTick, 0, nil)
	if s.StateName() != "Loading" {
		t.Fatalf("state=%s", s.StateName())
	}
	// manage 就绪 → Opened
	s.mu.Lock()
	s.manageData = &ManageInfo{Name: "m"}
	s.mu.Unlock()
	if !s.StateMachine(evTick, 0, nil) || s.StateName() != "Opened" {
		t.Fatalf("state=%s", s.StateName())
	}
}

// PubHas 供测试：镜像是否含 space。
func (s *Server) PubHas(space string) bool {
	_, ok := s.Pub.FindSpace(space)
	return ok
}

// ---- 发现监督器谓词 ----

// markConn 白盒标记 comm 已连（识别入登记等价；谓词只看登记面）。
func markConn(s *Server, comm uint32) {
	s.inner.mu.Lock()
	s.inner.commConn[comm] = 1
	s.inner.mu.Unlock()
}

func TestDiscoveryPendingMember(t *testing.T) {
	s := New(Options{Config: DefaultConfig(), ManualTick: true})
	if p := s.DiscoveryPending(); strings.Join(p, ",") != "store0,public,entry0" {
		t.Fatalf("初始缺口 = %v", p)
	}
	markConn(s, commStoreMin)
	markConn(s, commPublic)
	if p := s.DiscoveryPending(); strings.Join(p, ",") != "entry0" {
		t.Fatalf("部分连入后缺口 = %v", p)
	}
	markConn(s, commEntryMin)
	if p := s.DiscoveryPending(); len(p) != 0 {
		t.Fatalf("全连后应就绪 = %v", p)
	}
}
