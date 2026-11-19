package fssoloworld

import (
	"bufio"
	"encoding/binary"
	"net"
	"strings"
	"testing"
	"time"
)

func TestParseGameWorldINI(t *testing.T) {
	ini := []byte(`
[System]
Sleep=-1
MainEntity=SoloWorld

[Main]
DistrictID=3
ServerID=11
ServerName=server_11
RoleNameSizeLimit=18

[Inner]
Addr=127.0.0.1

[Broadcast]
Addr=127.255.255.255
Port=4001

[Echo]
Addr=127.0.0.1
Port=15001

[ManageServer]
Inuse=true
Addr=127.0.0.1
Port=4002

[Monitor]
Inuse=true
Addr=127.0.0.1
Port=4008
`)
	cfg, err := ParseGameWorldINI(ini)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DistrictID != 3 || cfg.ServerID != 11 || cfg.ServerName != "server_11" {
		t.Fatalf("main: %+v", cfg)
	}
	if cfg.RoleNameLimit != 18 {
		t.Fatalf("rolenamelimit %d", cfg.RoleNameLimit)
	}
	if cfg.BroadAddr != "127.255.255.255" || cfg.BroadPort != 4001 {
		t.Fatalf("broadcast: %+v", cfg)
	}
	if !cfg.ManageInuse || cfg.ManageAddr != "127.0.0.1" || cfg.ManagePort != 4002 {
		t.Fatalf("manage: %+v", cfg)
	}
	if !cfg.MonitorInuse || cfg.MonitorPort != 4008 {
		t.Fatalf("monitor: %+v", cfg)
	}
	if cfg.AppName() != "world_11" {
		t.Fatalf("appname %s", cfg.AppName())
	}
}

func TestWorldAnnounceBytes(t *testing.T) {
	pkt := buildWorldAnnounce("127.0.0.1", 16001, 3, 11, "服一一", 8192, 7, true, 42)
	if len(pkt) != 0x84 {
		t.Fatalf("size %d", len(pkt))
	}
	if binary.LittleEndian.Uint32(pkt[0:]) != KindWorld {
		t.Fatal("kind")
	}
	if binary.LittleEndian.Uint32(pkt[4:]) != 0x3001 {
		t.Fatal("ident 0x3001")
	}
	if string(pkt[8:17]) != "127.0.0.1" {
		t.Fatalf("addr %q", pkt[8:24])
	}
	if binary.LittleEndian.Uint32(pkt[0x18:]) != 16001 {
		t.Fatal("port")
	}
	if binary.LittleEndian.Uint32(pkt[0x1C:]) != 3 || binary.LittleEndian.Uint32(pkt[0x20:]) != 11 {
		t.Fatal("district/server")
	}
	if binary.LittleEndian.Uint32(pkt[0x74:]) != 1 {
		t.Fatal("opened flag")
	}
	if binary.LittleEndian.Uint32(pkt[0x78:]) != 42 {
		t.Fatal("online")
	}
	// 名字 utf16 区终止零
	u16 := encodeUTF16([]rune("服一一"))
	for i, v := range u16 {
		if binary.LittleEndian.Uint16(pkt[0x24+i*2:]) != v {
			t.Fatalf("name u16 @%d", i)
		}
	}
	if binary.LittleEndian.Uint16(pkt[0x24+len(u16)*2:]) != 0 {
		t.Fatal("name NUL")
	}
}

func TestParseAnnounceHeadKindBranches(t *testing.T) {
	mk := func(kind uint32, size int, idx uint32) []byte {
		b := make([]byte, size)
		binary.LittleEndian.PutUint32(b[0:], kind)
		binary.LittleEndian.PutUint32(b[4:], 0x3001)
		copy(b[8:], "10.1.1.5")
		binary.LittleEndian.PutUint32(b[0x18:], 5000)
		binary.LittleEndian.PutUint32(b[0x1C:], 3)
		binary.LittleEndian.PutUint32(b[0x20:], 11)
		if size >= 0x28 {
			binary.LittleEndian.PutUint32(b[0x24:], idx)
		}
		return b
	}
	// member 0x28B idx=5
	h, ok := parseAnnounceHead(mk(KindMember, 0x28, 5))
	if !ok || h.kind != KindMember || h.idx != 5 || h.addr != "10.1.1.5" || h.port != 5000 {
		t.Fatalf("member: %+v", h)
	}
	// entry 0x48B idx=2
	h, ok = parseAnnounceHead(mk(KindEntry, 0x48, 2))
	if !ok || h.idx != 2 {
		t.Fatalf("entry: %+v", h)
	}
	// ident 不符 → 忽略
	bad := mk(KindPublic, 0x24, 0)
	binary.LittleEndian.PutUint32(bad[4:], 0x1234)
	if _, ok := parseAnnounceHead(bad); ok {
		t.Fatal("bad ident accepted")
	}
}

func TestFrameAccAndIdentify(t *testing.T) {
	frame, err := encodeInner(commWorld, 0, innerIdentifyID2, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(frame) != 16 {
		t.Fatalf("identify frame %d", len(frame))
	}
	if binary.LittleEndian.Uint32(frame[0:]) != 0xC {
		t.Fatal("identify N")
	}
	var got []uint32
	acc := &frameAcc{}
	if err := acc.feed(frame, func(c, a, b uint32, _ []byte) { got = []uint32{c, a, b} }); err != nil {
		t.Fatal(err)
	}
	if got[0] != commWorld || got[2] != innerIdentifyID2 {
		t.Fatalf("got %v", got)
	}
	// 半包重组：两段投喂
	big, _ := encodeInner(commWorld, 7, 9, []byte("hello"))
	acc2 := &frameAcc{}
	if err := acc2.feed(big[:5], func(c, a, b uint32, p []byte) {}); err != nil {
		t.Fatal(err)
	}
	fired := 0
	if err := acc2.feed(big[5:], func(c, a, b uint32, p []byte) {
		fired++
		if string(p) != "hello" {
			t.Fatalf("payload %q", p)
		}
	}); err != nil {
		t.Fatal(err)
	}
	if fired != 1 {
		t.Fatalf("fired %d", fired)
	}
}

func TestNameUid(t *testing.T) {
	n := NewNameUid()
	if !n.Add("张三", "GUID-1", false) {
		t.Fatal("add")
	}
	if n.Add("张三", "GUID-2", false) {
		t.Fatal("dup name accepted")
	}
	if !n.FindName("张三") || !n.FindUid("GUID-1") {
		t.Fatal("find")
	}
	if uid, _ := n.GetUid("张三"); uid != "GUID-1" {
		t.Fatal("getuid")
	}
	if !n.Remove("张三", "GUID-1") {
		t.Fatal("remove")
	}
	if n.GetCount() != 0 {
		t.Fatal("count")
	}
}

func TestWorldPlayers(t *testing.T) {
	w := NewWorldPlayers(100)
	p := w.AddPlayer(1, 2, "acct")
	if p == nil || p.State != PlUnknown {
		t.Fatal("add")
	}
	if w.Count() != 1 {
		t.Fatal("count")
	}
	if w.FindByConnect(1, 2) != p {
		t.Fatal("findbyconnect")
	}
	w.SetupPlayerName(1, 2, "李四")
	if w.FindByName("李四") != p {
		t.Fatal("findbyname")
	}
	w.SetPlayerState(p, PlOnline)
	if w.CountOnline() != 1 || w.CountActive() != 1 {
		t.Fatal("counts")
	}
	if !w.RemovePlayerByName("李四") || w.Count() != 0 {
		t.Fatal("removebyname")
	}
}

func TestWorldScenes(t *testing.T) {
	w := NewWorldScenes()
	if !w.AddScene(101, 3, 0, "scene", "chengdu") {
		t.Fatal("add")
	}
	if !w.AddScene(201, 4, 1, "fb", "wulin") {
		t.Fatal("add proto")
	}
	if w.CountNormal() != 1 || w.CountClone() != 1 {
		t.Fatal("counts")
	}
	if w.AllRunning() {
		t.Fatal("not all running yet")
	}
	w.SetState(101, ScRunning)
	w.SetState(201, ScRunning)
	if !w.AllRunning() {
		t.Fatal("all running")
	}
	w.IncPlayers(101)
	w.DecPlayers(101)
	if w.GetScene(101).Players != 0 {
		t.Fatal("players")
	}
	if !w.RemoveScene(101) || w.GetScene(101) != nil {
		t.Fatal("remove")
	}
}

// 服务器全链：广播自宣可收 + 状态机推进 + 广播发现外连 + echo 命令。
func TestServerFullChain(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DistrictID = 3
	cfg.ServerID = 11
	cfg.ServerName = "server_11"
	cfg.InnerAddr = "127.0.0.1"
	cfg.InnerPort = 0
	cfg.BroadAddr = "127.255.255.255"
	cfg.BroadPort = 0 // 随机口，避免环境冲突
	cfg.EchoAddr = "127.0.0.1"
	cfg.EchoPort = 0

	s := New(Options{Config: cfg, ManualTick: true})
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	if s.InnerPort() == 0 {
		t.Fatal("inner port not assigned")
	}

	// Unknown → 发 load roles（无 store 连接时 SendToAllStore false → 停在 Unknown）
	s.Tick()
	if s.State() != "Unknown" {
		t.Fatalf("state %s", s.State())
	}

	// 广播自宣帧可被家族方言解析（kind/ident 校验）
	s.broadcastWorldInfo()

	// 广播发现：member 自宣（kind3 idx=5 端口=world 自身测试回环）
	mb := make([]byte, 0x28)
	binary.LittleEndian.PutUint32(mb[0:], KindMember)
	binary.LittleEndian.PutUint32(mb[4:], 0x3001)
	copy(mb[8:], "127.0.0.1")
	binary.LittleEndian.PutUint32(mb[0x18:], uint32(s.InnerPort()))
	binary.LittleEndian.PutUint32(mb[0x1C:], 3)
	binary.LittleEndian.PutUint32(mb[0x20:], 11)
	binary.LittleEndian.PutUint32(mb[0x24:], 5)
	s.onBroadcast(mb, "127.0.0.1", 15000)

	// 外连成功后 comm 表登记（回环连自身 listener；world 不发 identify 阻塞）
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if s.inner.GetConnected(uint32(commMemberMin + 5)) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !s.inner.GetConnected(uint32(commMemberMin + 5)) {
		t.Fatal("member comm not connected via broadcast discovery")
	}

	// echo 控制台
	if s.echo == nil {
		t.Fatal("echo not started")
	}
	port := 0
	if s.echo.ln != nil {
		_, pstr, _ := net.SplitHostPort(s.echo.ln.Addr().String())
		for _, c := range pstr {
			port = port*10 + int(c-'0')
		}
	}
	if port == 0 {
		t.Fatal("echo port")
	}
	conn, err := net.DialTimeout("tcp", "127.0.0.1:"+pstr2(port), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	rd := bufio.NewReader(conn)
	line, err := rd.ReadString('\r')
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(line, "welcome server 11") {
		t.Fatalf("welcome %q", line)
	}
	conn.Write([]byte("status\r"))
	line, _ = rd.ReadString('\r')
	if !strings.Contains(line, ">game server state is Unknown") {
		t.Fatalf("status %q", line)
	}
	conn.Write([]byte("players\r"))
	line, _ = rd.ReadString('\r')
	if !strings.Contains(line, ">player: 0") {
		t.Fatalf("players %q", line)
	}
	conn.Write([]byte("open\r"))
	line, _ = rd.ReadString('\r')
	if !strings.Contains(line, "can't open") {
		t.Fatalf("open %q", line)
	}

	// AppRecord 心跳格式
	info := s.AppRecordInfo()
	if !strings.Contains(info, `state="Unknown"`) || !strings.Contains(info, `player="0"`) {
		t.Fatalf("apprecord %q", info)
	}
}

func pstr2(p int) string {
	if p == 0 {
		return "0"
	}
	var b [8]byte
	i := len(b)
	for p > 0 {
		i--
		b[i] = byte('0' + p%10)
		p /= 10
	}
	return string(b[i:])
}

// 状态机关闭链：Opened→CloseEntry→...→Closed。
func TestServerCloseChain(t *testing.T) {
	cfg := DefaultConfig()
	cfg.InnerAddr = "127.0.0.1"
	cfg.InnerPort = 0
	cfg.BroadPort = 0
	cfg.EchoAddr = "" // 不启 echo
	s := New(Options{Config: cfg, ManualTick: true})
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	// 直接内部驱动到 Opened（模拟 CanOpen 前置）
	s.setState(WsCanOpen)
	if !s.OpenServer() || s.State() != "Opened" {
		t.Fatalf("open state %s", s.State())
	}
	// 重复 open 拒绝
	if s.OpenServer() {
		t.Fatal("double open")
	}
	if !s.CloseServer() {
		t.Fatal("close")
	}
	// 关闭链逐 tick 推进
	want := []string{"ClearPlayers", "ClearScenes",
		"CloseMember", "ClosePublic", "CloseAuction", "CloseStore", "Closed"}
	for _, w := range want {
		s.Tick()
		if s.State() != w {
			t.Fatalf("chain: want %s got %s", w, s.State())
		}
	}
}

// echo close 命令触发关闭链。
func TestEchoCloseCommand(t *testing.T) {
	cfg := DefaultConfig()
	cfg.InnerAddr = "127.0.0.1"
	cfg.InnerPort = 0
	cfg.BroadPort = 0
	cfg.EchoAddr = "127.0.0.1"
	cfg.EchoPort = 0
	s := New(Options{Config: cfg, ManualTick: true})
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
	s.setState(WsCanOpen)
	s.OpenServer()

	conn, err := net.DialTimeout("tcp", s.echo.ln.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	rd := bufio.NewReader(conn)
	rd.ReadString('\r')
	conn.Write([]byte("close\r"))
	line, _ := rd.ReadString('\r')
	if !strings.Contains(line, "send close command") {
		t.Fatalf("close resp %q", line)
	}
	if s.State() != "CloseEntry" {
		t.Fatalf("state %s", s.State())
	}
}

// entry 消息面：Accept/Login/CreateRole 全链。
func TestEntryMessageFlow(t *testing.T) {
	cfg := DefaultConfig()
	cfg.InnerAddr = "127.0.0.1"
	cfg.InnerPort = 0
	cfg.BroadPort = 0
	cfg.EchoAddr = ""
	s := New(Options{Config: cfg, ManualTick: true})
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	// ClientAccept {client_id=7, serial=1}
	acc := make([]byte, 8)
	binary.LittleEndian.PutUint32(acc[0:], 7)
	binary.LittleEndian.PutUint32(acc[4:], 1)
	if !s.ProcessEntryMsg(0, 0, acc) {
		t.Fatal("accept")
	}
	if s.Players().Count() != 1 {
		t.Fatal("player count")
	}
	// Login {u32 len account "acct" | u32 len password "pw"}
	lg := make([]byte, 0, 16)
	lg = append(lg, byte(4), 0, 0, 0)
	lg = append(lg, []byte("acct")...)
	lg = append(lg, byte(2), 0, 0, 0)
	lg = append(lg, []byte("pw")...)
	msg := make([]byte, 0, 32)
	var h [8]byte
	binary.LittleEndian.PutUint32(h[0:], 7)
	binary.LittleEndian.PutUint32(h[4:], 1)
	msg = append(msg, h[:]...)
	msg = append(msg, clientMsgLogin)
	msg = append(msg, lg...)
	if !s.ProcessEntryMsg(0, 3, msg) {
		t.Fatal("login")
	}
	p := s.Players().FindByConnect(7, 1)
	if p == nil || p.Account != "acct" {
		t.Fatalf("player %+v", p)
	}
	if p.State != PlGetting && p.State != PlLogging {
		t.Fatalf("state %s", p.State)
	}
	// CreateRole（名 "王五"）
	cr := make([]byte, 0, 16)
	var l [4]byte
	binary.LittleEndian.PutUint32(l[0:], 2)
	cr = append(cr, l[:]...)
	for _, u := range encodeUTF16([]rune("王五")) {
		cr = append(cr, byte(u), byte(u>>8))
	}
	msg2 := make([]byte, 0, 32)
	msg2 = append(msg2, h[:]...)
	msg2 = append(msg2, clientMsgCreateRole)
	msg2 = append(msg2, cr...)
	if !s.ProcessEntryMsg(0, 3, msg2) {
		t.Fatal("createrole")
	}
	if p.State != PlCreating {
		t.Fatalf("creating %s", p.State)
	}
	// store 建角应答：{client, serial, uid, name}
	sr := make([]byte, 0, 32)
	sr = append(sr, h[:]...)
	binary.LittleEndian.PutUint32(l[0:], 5)
	sr = append(sr, l[:]...)
	sr = append(sr, []byte("GUID9")...)
	binary.LittleEndian.PutUint32(l[0:], 2)
	sr = append(sr, l[:]...)
	for _, u := range encodeUTF16([]rune("王五")) {
		sr = append(sr, byte(u), byte(u>>8))
	}
	if !s.ProcessStoreMsg(0, 6, sr) {
		t.Fatal("storecreate")
	}
	if !s.Names().FindName("王五") {
		t.Fatal("nameuid")
	}
	if s.Players().FindByName("王五") != p {
		t.Fatal("name index")
	}
	if p.State != PlLogged {
		t.Fatalf("logged %s", p.State)
	}
	// Closed：ClientClosed 清场
	if !s.ProcessEntryMsg(0, 1, acc) {
		t.Fatal("closed")
	}
	if s.Players().Count() != 0 {
		t.Fatal("cleared")
	}
}

// StoreMsgRoleInfo 全量同步 → LoadRoles→LoadScenes。
func TestStoreRoleInfoSync(t *testing.T) {
	cfg := DefaultConfig()
	cfg.InnerAddr = "127.0.0.1"
	cfg.InnerPort = 0
	cfg.BroadPort = 0
	cfg.EchoAddr = ""
	s := New(Options{Config: cfg, ManualTick: true})
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
	s.setState(WsLoadRoles)

	b := make([]byte, 0, 64)
	var u32 [4]byte
	binary.LittleEndian.PutUint32(u32[0:], 2)
	b = append(b, u32[:]...)
	appendRole := func(name, uid string, del bool) {
		runes := encodeUTF16([]rune(name))
		binary.LittleEndian.PutUint32(u32[0:], uint32(len(runes)))
		b = append(b, u32[:]...)
		for _, x := range runes {
			b = append(b, byte(x), byte(x>>8))
		}
		binary.LittleEndian.PutUint32(u32[0:], uint32(len(uid)))
		b = append(b, u32[:]...)
		b = append(b, uid...)
		if del {
			b = append(b, 1)
		} else {
			b = append(b, 0)
		}
	}
	appendRole("赵六", "GUID-A", false)
	appendRole("钱七", "GUID-B", true)

	if !s.ProcessStoreMsg(0, 0, b) {
		t.Fatal("roleinfo")
	}
	if s.Names().GetCount() != 2 {
		t.Fatal("names")
	}
	if !s.Names().GetDeleted("钱七") || s.Names().GetDeleted("赵六") {
		t.Fatal("deleted flag")
	}
	if s.State() != "LoadScenes" {
		t.Fatalf("state %s", s.State())
	}
}

// member SceneState 推进 → CanOpen → Opened。
func TestSceneStateToOpened(t *testing.T) {
	cfg := DefaultConfig()
	cfg.InnerAddr = "127.0.0.1"
	cfg.InnerPort = 0
	cfg.BroadPort = 0
	cfg.EchoAddr = ""
	s := New(Options{Config: cfg, ManualTick: true, Scenes: []SceneSpec{
		{Index: 101, Member: 3, Script: "scene", Config: "chengdu"},
	}})
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
	s.setState(WsLoadScenes)

	sm := make([]byte, 8)
	binary.LittleEndian.PutUint32(sm[0:], 101)
	binary.LittleEndian.PutUint32(sm[4:], 2) // Running
	if !s.ProcessMemberMsg(0, 5, sm) {
		t.Fatal("scenestate")
	}
	if s.State() != "CanOpen" {
		t.Fatalf("state %s", s.State())
	}
	if !s.OpenServer() || s.State() != "Opened" {
		t.Fatal("open")
	}
}
