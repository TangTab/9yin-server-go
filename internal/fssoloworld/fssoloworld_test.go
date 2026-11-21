package fssoloworld

import (
	"github.com/lizhanyin/9yin-server-go/internal/auth"
	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
	"github.com/lizhanyin/9yin-server-go/internal/proto"

	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"
	"unicode/utf16"
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
// buildTestOp02 构造真机方言 op02（auth.EncryptFields 同源；golden 333 版本）。
func buildTestOp02(account, password string) []byte {
	acctCt, pwCt := auth.EncryptFields(account, auth.ComputeLoginPassword(password))
	fld := func(slot []byte) []byte {
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, uint32(len(slot)))
		return append(b, slot...)
	}
	buf := make([]byte, 25, 25+len(acctCt)+len(pwCt)+64)
	buf[0] = 0x02
	binary.LittleEndian.PutUint32(buf[1:], 333)
	// 账号长度嵌在头内 [0x15:0x19]，槽体紧随 0x19（无自身长度前缀）
	binary.LittleEndian.PutUint32(buf[0x15:], uint32(len(acctCt)))
	buf = append(buf, acctCt...)
	buf = append(buf, fld(pwCt)...) // 密码长度为独立 u32
	buf = append(buf, 0, 0, 0, 0)   // clientType（不占槽）
	buf = append(buf, fld(make([]byte, 16))...)
	buf = append(buf, 0, 0, 0, 0) // loginType，收帧尾
	return buf
}

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
	if !s.ProcessEntryMsg(0, 1, acc) {
		t.Fatal("accept")
	}
	if s.Players().Count() != 1 {
		t.Fatal("player count")
	}
	// Login：真机方言 op02（25B 头 + DES 双槽；auth 包同源加密）。
	// 无 inner 的单测形态下 0x36 发送失败静默，玩家停在 PlLogging、
	// 账号由解密明文绑定。
	op := buildTestOp02("acct", "pw")
	msg := make([]byte, 0, 8+len(op))
	var h [8]byte
	binary.LittleEndian.PutUint32(h[0:], 7)
	binary.LittleEndian.PutUint32(h[4:], 1)
	msg = append(msg, h[:]...)
	msg = append(msg, byte(clientMsgLogin+1)) // 客户端 opcode（内网号+1）
	msg = append(msg, op[1:]...)
	if !s.ProcessEntryMsg(0, 3, msg) {
		t.Fatal("login")
	}
	p := s.Players().FindByConnect(7, 1)
	if p == nil || p.Account != "acct" {
		t.Fatalf("player %+v", p)
	}
	if p.State != PlLogging {
		t.Fatalf("state %s", p.State)
	}
	// store 0x25 回执（result=1）→ PlGetting + 下行 0x04（nil inner 静默）
	rep := make([]byte, 0x18)
	binary.LittleEndian.PutUint32(rep[0:], 7)
	binary.LittleEndian.PutUint32(rep[4:], 1)
	binary.LittleEndian.PutUint32(rep[8:], 1)
	if !s.accountLogonReply(rep) {
		t.Fatal("logon reply")
	}
	if p.State != PlGetting {
		t.Fatalf("after reply state %s", p.State)
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
	msg2 = append(msg2, clientMsgCreateRole+1)
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
	// Closed（type 2 连接关闭回报）：ClientClosed 清场
	if !s.ProcessEntryMsg(0, 2, acc) {
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

// innerTestConn 测试伪实体连接（identify 登记 + 内网帧收发）。
type innerTestConn struct {
	conn net.Conn
	rd   *bufio.Reader
	t    *testing.T
}

// dialInnerTest 伪实体拨号 world inner 口并 identify。
func dialInnerTest(t *testing.T, port int, comm uint32) *innerTestConn {
	t.Helper()
	var c net.Conn
	var err error
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	for i := 0; i < 50; i++ {
		c, err = net.Dial("tcp", addr)
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("dial %s: %v", addr, err)
	}
	ic := &innerTestConn{conn: c, rd: bufio.NewReader(c), t: t}
	t.Cleanup(func() { c.Close() })
	ic.send(comm, 0, innerIdentifyID2, nil)
	return ic
}

func (m *innerTestConn) send(comm, id1, id2 uint32, payload []byte) {
	f, err := encodeInner(comm, id1, id2, payload)
	if err != nil {
		m.t.Fatal(err)
	}
	if _, err := m.conn.Write(f); err != nil {
		m.t.Fatal(err)
	}
}

func (m *innerTestConn) recv() (comm, id1, id2 uint32, payload []byte) {
	m.conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	head := make([]byte, 4)
	if _, err := io.ReadFull(m.rd, head); err != nil {
		m.t.Fatalf("read head: %v", err)
	}
	body := make([]byte, binary.LittleEndian.Uint32(head))
	if _, err := io.ReadFull(m.rd, body); err != nil {
		m.t.Fatalf("read body: %v", err)
	}
	comm = binary.LittleEndian.Uint32(body[0:4])
	id1 = binary.LittleEndian.Uint32(body[4:8])
	id2 = binary.LittleEndian.Uint32(body[8:12])
	return comm, id1, id2, body[12:]
}

// recvClientFrame 伪 entry 收 entrySendClient 下行并剥 {client,serial} 头。
func (m *innerTestConn) recvClientFrame(t *testing.T) []byte {
	t.Helper()
	comm, _, id2, p := m.recv()
	if comm != commWorld || id2 != entrySendClient {
		t.Fatalf("entry downlink comm=%d id2=%#x", comm, id2)
	}
	if len(p) < 8 {
		t.Fatalf("entry downlink len %d", len(p))
	}
	return p[8:]
}

// buildRealMachineOp05 真机方言 op05 帧（fork 实证布局）：[4B 保留]
// + 裸 UTF-16LE 名字 NUL + pad 0x48 + tag6 外观串。
func buildRealMachineOp05(name string, appearance []string) []byte {
	body := make([]byte, 0x48)
	units := utf16.Encode([]rune(name))
	for i, u := range units {
		binary.LittleEndian.PutUint16(body[4+i*2:], u)
	}
	for _, s := range appearance {
		body = append(body, 6)
		body = binary.LittleEndian.AppendUint32(body, uint32(len(s)+1))
		body = append(body, s...)
		body = append(body, 0)
	}
	frame := make([]byte, 0, 9+len(body))
	frame = append(frame, 5) // 客户端 opcode（op05；内网号折算 4）
	return append(frame, body...)
}

// TestCreateRoleEndToEnd 建角闭环（官方通路全链）：op02 登录 → 0x36/0x25
// → 无角 0x04 + 0x10 空清单 → op05（真机方言）→ store 0x0A → 0x06 回执
// → 下行有角 0x04（rolePara 36 字段 golden 断言）→ 0x0D 重登列表。
func TestCreateRoleEndToEnd(t *testing.T) {
	cfg := DefaultConfig()
	cfg.InnerAddr = "127.0.0.1"
	cfg.InnerPort = 0
	cfg.BroadPort = 0
	cfg.EchoAddr = ""
	s := New(Options{Config: cfg, ManualTick: true, Logf: func(format string, args ...any) {
		t.Logf(format, args...)
	}})
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	st := dialInnerTest(t, s.InnerPort(), commStoreMin) // 伪 store
	en := dialInnerTest(t, s.InnerPort(), commEntryMin) // 伪 entry
	// identify 登记是 reactor 异步面：等 commConn 就绪再驱动登录链
	waitComm := func(comm uint32) {
		t.Helper()
		for i := 0; i < 200; i++ {
			if s.inner.GetConnected(comm) {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatalf("comm %d not registered", comm)
	}
	waitComm(commStoreMin)
	waitComm(commEntryMin)

	// ClientAccept {7,1} + op02（真机 DES 方言）
	h := make([]byte, 8)
	binary.LittleEndian.PutUint32(h[0:], 7)
	binary.LittleEndian.PutUint32(h[4:], 1)
	if !s.ProcessEntryMsg(0, 1, h) {
		t.Fatal("accept")
	}
	op := buildTestOp02("acct", "pw")
	msg := append([]byte{}, h...)
	msg = append(msg, byte(clientMsgLogin+1))
	msg = append(msg, op[1:]...)
	if !s.ProcessEntryMsg(0, 3, msg) {
		t.Fatal("login")
	}

	// 伪 store 收 0x36（account@0xC）→ 回 0x25 result=1
	if _, _, id2, p := st.recv(); id2 != 0x36 || string(p[0xC:0xC+4]) != "acct" {
		t.Fatalf("logon to store id2=%#x acct=%q", id2, p[0xC:0xC+4])
	}
	rep := make([]byte, 0x18)
	binary.LittleEndian.PutUint32(rep[0:], 7)
	binary.LittleEndian.PutUint32(rep[4:], 1)
	binary.LittleEndian.PutUint32(rep[8:], 1)
	st.send(commStoreMin, 0, 0x25, rep)

	// 0x10 清单请求 → 0x0D 空回执 → 无角 0x04（统一 0x04 下行路径）
	if _, _, id2, p := st.recv(); id2 != 0x10 || string(p[8:8+4]) != "acct" {
		t.Fatalf("query account id2=%#x", id2)
	}
	st.send(commStoreMin, 0, 0x0D, emptyRolesReply(h))
	if f := en.recvClientFrame(t); len(f) != 0x29 || f[0] != 0x04 ||
		binary.LittleEndian.Uint32(f[0x25:]) != 0 {
		t.Fatalf("no-role 0x04 len=%d", len(f))
	}

	// op0x18 心跳 → 0x38 整帧回显（选角页面期 RTT 配对依赖）
	ping := append([]byte{}, h...)
	ping = append(ping, 0x18, 1, 2, 3, 4, 5, 6, 7, 8)
	if !s.ProcessEntryMsg(0, 3, ping) {
		t.Fatal("ping")
	}
	if echo := en.recvClientFrame(t); echo[0] != 0x38 || len(echo) < 20+18-10 { // 回显+18B 尾（真机 12B 心跳=30B；测试短 ping）
		t.Fatalf("heartbeat echo op=%#x len=%d", echo[0], len(echo))
	}

	// op05 真机方言（girl 外观 → rolePara f0=1）
	appearance := []string{"book1", `icon\players\girl01_create.png`, `obj\char\g_hair\g_hair1`}
	op5 := append([]byte{}, h...)
	op5 = append(op5, buildRealMachineOp05("王五", appearance)...)
	if !s.ProcessEntryMsg(0, 3, op5) {
		t.Fatal("createrole")
	}
	if p := s.Players().FindByConnect(7, 1); p == nil || p.State != PlCreating {
		t.Fatalf("creating state %+v", p)
	}

	// 伪 store 收 0x0A：{account@0xC, name@0x2D, para@0x71}
	_, _, id2, cp := st.recv()
	if id2 != 0x0A {
		t.Fatalf("expect 0x0A, got %#x", id2)
	}
	if string(cp[0xC:0xC+4]) != "acct" {
		t.Fatalf("create account %q", cp[0xC:0xC+4])
	}
	if got := wideNulName(cp[0x2D : 0x2D+0x44]); got != "王五" {
		t.Fatalf("create name %q", got)
	}
	if got := wideNulName(cp[0x71:]); got != joinAppearance(appearance) {
		t.Fatalf("create para %q", got)
	}

	// 0x06 回执（{client,serial,arch-str uid,arch-wstr name} 金样形态）
	sr := make([]byte, 0, 32)
	sr = append(sr, h...)
	var l [4]byte
	binary.LittleEndian.PutUint32(l[0:], 5)
	sr = append(sr, l[:]...)
	sr = append(sr, []byte("GUID9")...)
	binary.LittleEndian.PutUint32(l[0:], 2)
	sr = append(sr, l[:]...)
	for _, u := range utf16.Encode([]rune("王五")) {
		sr = append(sr, byte(u), byte(u>>8))
	}
	st.send(commStoreMin, 0, 6, sr)

	// 下行有角 0x04：[0x25]=1 + 6 值 TVarList（名字/rolePara golden）
	f := en.recvClientFrame(t)
	if f[0] != 0x04 || binary.LittleEndian.Uint32(f[0x25:]) != 1 {
		t.Fatalf("one-role 0x04 head %#x", f[0])
	}
	vl, err := proto.DecodeVarList(f[0x29:])
	if err != nil {
		t.Fatalf("varlist: %v", err)
	}
	if len(vl.Values) != 6 || vl.Values[2].Text != "王五" {
		t.Fatalf("role name slot %+v", vl.Values)
	}
	fields := strings.Split(vl.Values[3].Text, ",")
	if len(fields) != 36 || fields[0] != "1" || fields[1] != "" ||
		fields[32] != "book1" || fields[24] != "scene_name_017" ||
		fields[25] != "introduction_qdz01_title" {
		t.Fatalf("rolePara %q", vl.Values[3].Text)
	}
	if p := s.Players().FindByConnect(7, 1); p == nil || p.State != PlLogged ||
		s.Players().FindByName("王五") != p || !s.Names().FindName("王五") {
		t.Fatalf("logged state %+v", p)
	}

	// 重登清单：0x0D 带 1 角色（para 往返）→ 有角 0x04
	lr := make([]byte, 0, 64)
	lr = append(lr, h...)
	lr = binary.LittleEndian.AppendUint32(lr, 1)
	lr = binary.LittleEndian.AppendUint32(lr, uint32(len("GUID9")+1))
	lr = append(lr, "GUID9"...)
	lr = append(lr, 0)
	lr = appendArchWideTest(lr, "王五")
	lr = appendArchWideTest(lr, joinAppearance(appearance))
	st.send(commStoreMin, 0, 0x0D, lr)
	f = en.recvClientFrame(t)
	if f[0] != 0x04 || binary.LittleEndian.Uint32(f[0x25:]) != 1 {
		t.Fatalf("reload one-role 0x04")
	}
	vl, err = proto.DecodeVarList(f[0x29:])
	if err != nil || len(vl.Values) != 6 || vl.Values[2].Text != "王五" {
		t.Fatalf("reload varlist err=%v vals=%d", err, len(vl.Values))
	}

	// 删角：op06（uid arch-str）→ PlDeleting + 0x0B → 0x07 → 名册移除
	// + 0x10 刷新 → 0x0D 空 → 无角 0x04（界面回建角页）
	del := append([]byte{}, h...)
	del = append(del, 6) // 客户端 opcode（op06）
	del = binary.LittleEndian.AppendUint32(del, uint32(len("GUID9")))
	del = append(del, "GUID9"...)
	if !s.ProcessEntryMsg(0, 3, del) {
		t.Fatal("deleterole")
	}
	if p := s.Players().FindByConnect(7, 1); p == nil || p.State != PlDeleting {
		t.Fatalf("deleting state %+v", p)
	}
	_, _, id2, dp := st.recv()
	if id2 != 0x0B {
		t.Fatalf("expect 0x0B, got %#x", id2)
	}
	if got := wideNulName(dp[0x2D : 0x2D+0x44]); got != "王五" {
		t.Fatalf("delete name %q", got)
	}
	if string(dp[0x71:0x71+5]) != "GUID9" {
		t.Fatalf("delete uid %q", dp[0x71:0x71+5])
	}
	// 0x07 回执（{client,serial,arch-str uid,arch-wstr name}，0x06 同构）
	dr := make([]byte, 0, 32)
	dr = append(dr, h...)
	binary.LittleEndian.PutUint32(l[0:], 5)
	dr = append(dr, l[:]...)
	dr = append(dr, []byte("GUID9")...)
	binary.LittleEndian.PutUint32(l[0:], 2)
	dr = append(dr, l[:]...)
	for _, u := range utf16.Encode([]rune("王五")) {
		dr = append(dr, byte(u), byte(u>>8))
	}
	st.send(commStoreMin, 0, 7, dr)
	// 0x10 刷新（names.Remove 先于本请求发出，收到即名册已清）
	if _, _, id2, _ = st.recv(); id2 != 0x10 {
		t.Fatalf("refresh query id2=%#x", id2)
	}
	if s.Names().FindName("王五") {
		t.Fatal("nameuid should drop name")
	}
	st.send(commStoreMin, 0, 0x0D, emptyRolesReply(h))
	if f = en.recvClientFrame(t); len(f) != 0x29 || f[0] != 0x04 ||
		binary.LittleEndian.Uint32(f[0x25:]) != 0 {
		t.Fatalf("after delete no-role 0x04 len=%d", len(f))
	}
	// 末态 Getting：0x0D 刷新置列表态（客户端可再 op05 建角）
	if p := s.Players().FindByConnect(7, 1); p == nil || p.State != PlGetting {
		t.Fatalf("after delete state %+v", p)
	}

	// 再建角后走进场：op04（UCS2 名字，0x54 定长真机方言）→ 0x02 load
	// → 回执 → 进场帧簇（0x09/0x0A/0x0B/0x0D）→ op09 ClientReady →
	// View43/40 新手视图
	op5b := append([]byte{}, h...)
	op5b = append(op5b, buildRealMachineOp05("王五", appearance)...)
	if !s.ProcessEntryMsg(0, 3, op5b) {
		t.Fatal("re-createrole")
	}
	_, _, id2, _ = st.recv() // 0x0A
	sr2 := make([]byte, 0, 32)
	sr2 = append(sr2, h...)
	binary.LittleEndian.PutUint32(l[0:], 5)
	sr2 = append(sr2, l[:]...)
	sr2 = append(sr2, []byte("GUID9")...)
	binary.LittleEndian.PutUint32(l[0:], 2)
	sr2 = append(sr2, l[:]...)
	for _, u := range utf16.Encode([]rune("王五")) {
		sr2 = append(sr2, byte(u), byte(u>>8))
	}
	st.send(commStoreMin, 0, 6, sr2)
	en.recvClientFrame(t) // 有角 0x04（0x06 回执驱动）

	ch := make([]byte, 0x4B)
	for i, u := range utf16.Encode([]rune("王五")) {
		binary.LittleEndian.PutUint16(ch[i*2:], u)
	}
	op4 := append([]byte{}, h...)
	op4 = append(op4, 4) // op04
	op4 = append(op4, ch...)
	if !s.ProcessEntryMsg(0, 3, op4) {
		t.Fatal("chooserole")
	}
	if p := s.Players().FindByConnect(7, 1); p == nil || p.State != PlChoosing {
		t.Fatalf("choosing state %+v", p)
	}
	if _, _, id2, lp := st.recv(); id2 != 0x02 || string(lp[0xC:0xC+4]) != "acct" ||
		wideNulName(lp[0x2D:0x2D+0x44]) != "王五" {
		t.Fatalf("load to store id2=%#x", id2)
	}
	// 0x02 回执 {client,serial,ok=1,arch-wstr para}
	lr2 := make([]byte, 0, 64)
	lr2 = append(lr2, h...)
	lr2 = binary.LittleEndian.AppendUint32(lr2, 1)
	lr2 = appendArchWideTest(lr2, joinAppearance(appearance))
	st.send(commStoreMin, 0, 2, lr2)
	// 进场（真机镜像回放方言）：0x32×2 → 0x09 表 → 0x0A 表 → 外服进场段
	for i := 0; i < 2; i++ {
		if f = en.recvClientFrame(t); f[0] != 0x32 {
			t.Fatalf("inner push frame %#x", f[0])
		}
	}
	// 表方言默认 official2433（fork/7074 线客户端）：0x09 = 2433+61 尾追
	// 编码帧（与 clientdata.OfficialPropertyTableFrame 逐字节一致），
	// 0x0A = 官方 446 schema 镜像帧。golden 方言（29664/7921）由
	// NINEYIN_TABLE_DIALECT=golden 回退（init 期门控，进程内不可切）。
	wantTable := clientdata.OfficialPropertyTableFrame()
	if f = en.recvClientFrame(t); f[0] != 0x09 || !bytes.Equal(f, wantTable) {
		t.Fatalf("property table frame op=%#x len=%d want=%d", f[0], len(f), len(wantTable))
	}
	wantRecords := clientdata.HomeRecordsMirrorFrame()
	if f = en.recvClientFrame(t); f[0] != 0x0A || !bytes.Equal(f, wantRecords) {
		t.Fatalf("records mirror frame op=%#x len=%d want=%d", f[0], len(f), len(wantRecords))
	}
	// 外服进场段回放：首帧 0x0B（五属性），尾至 272 帧不等长（镜像包）
	if f = en.recvClientFrame(t); f[0] != 0x0B || binary.LittleEndian.Uint16(f[13:]) != 5 {
		t.Fatalf("golden entry head op=%#x attrs=%d", f[0], binary.LittleEndian.Uint16(f[13:]))
	}
	for i := 1; i < 272; i++ { // 其余 271 帧只验可读（0x14/0x11/0x29/0x28/0x18 混序）
		en.recvClientFrame(t)
	}
	// op09 ClientReady（单字节）→ 就绪段回放（85 帧）
	rdy := append([]byte{}, h...)
	rdy = append(rdy, 9)
	if !s.ProcessEntryMsg(0, 3, rdy) {
		t.Fatal("ready")
	}
	for i := 0; i < 85; i++ {
		en.recvClientFrame(t)
	}
	if !s.Names().FindName("王五") {
		t.Fatal("nameuid missing after full flow")
	}
}

// emptyRolesReply 0x0D 空清单回执 {client,serial,count=0}。
func emptyRolesReply(h []byte) []byte {
	b := make([]byte, 12)
	copy(b[0:8], h)
	return b
}

// wideNulName UTF-16LE 扫 NUL 解码（0x0A 帧槽断言用）。
func wideNulName(p []byte) string {
	end := 0
	for end+1 < len(p) {
		if p[end] == 0 && p[end+1] == 0 {
			break
		}
		end += 2
	}
	u := make([]uint16, end/2)
	for i := range u {
		u[i] = binary.LittleEndian.Uint16(p[i*2:])
	}
	return string(utf16.Decode(u))
}

// appendArchWideTest archive 宽串（含 NUL；world readArchiveWidestr 对偶）。
func appendArchWideTest(b []byte, s string) []byte {
	u := utf16.Encode([]rune(s))
	u = append(u, 0)
	b = binary.LittleEndian.AppendUint32(b, uint32(len(u)))
	for _, c := range u {
		b = binary.LittleEndian.AppendUint16(b, c)
	}
	return b
}

// TestParseCreateRoleBody op05 载荷双读法（archive 宽串 / 真机裸 UTF-16）
// + tag6 外观扫描。
func TestParseCreateRoleBody(t *testing.T) {
	// 读法一：archive 宽串 {u32 2, UTF-16("王五")} + 0x48 起 tag6
	ab := make([]byte, 0x48)
	binary.LittleEndian.PutUint32(ab[0:], 2)
	for i, u := range utf16.Encode([]rune("王五")) {
		binary.LittleEndian.PutUint16(ab[4+i*2:], u)
	}
	ab = append(ab, 6)
	ab = binary.LittleEndian.AppendUint32(ab, uint32(len("book1")+1))
	ab = append(ab, "book1"...)
	ab = append(ab, 0)
	name, appear := parseCreateRoleBody(ab)
	if name != "王五" || len(appear) != 1 || appear[0] != "book1" {
		t.Fatalf("archive dialect name=%q appear=%v", name, appear)
	}
	// 读法二：真机方言（4B 保留 + 裸 UTF-16 扫 NUL）
	rb := buildRealMachineOp05("李四", []string{"book2", `icon\players\boy01_create.png`})
	name, appear = parseCreateRoleBody(rb[1:]) // 剥 opcode
	if name != "李四" || len(appear) != 2 || appear[1] != `icon\players\boy01_create.png` {
		t.Fatalf("real-machine dialect name=%q appear=%v", name, appear)
	}
	// 性别推断：girl photo → f0=1；boy photo → f0=0
	if sex := inferSex(appear); sex != 0 {
		t.Fatalf("boy photo sex=%d", sex)
	}
}

// TestRolesFrameGolden 有角 0x04 golden：41B 基座 + [0x25]=count +
// 6 值 TVarList（rolePara 36 字段官方口径：f1 门派恒 0）。
func TestRolesFrameGolden(t *testing.T) {
	now := time.Date(2026, 11, 21, 10, 0, 0, 0, time.Local)
	f := rolesFrame(now, []roleEntry{{name: "赵六"}})
	if f[0] != 0x04 || binary.LittleEndian.Uint32(f[0x25:]) != 1 || len(f) <= 0x29 {
		t.Fatalf("head len=%d", len(f))
	}
	if binary.LittleEndian.Uint32(f[0x09:]) != 2026 || binary.LittleEndian.Uint32(f[0x11:]) != 21 {
		t.Fatalf("base date %#x", f[0x09:0x15])
	}
	vl, err := proto.DecodeVarList(f[0x29:])
	if err != nil || len(vl.Values) != 6 {
		t.Fatalf("varlist err=%v n=%d", err, len(vl.Values))
	}
	fields := strings.Split(vl.Values[3].Text, ",")
	if len(fields) != 36 || fields[0] != "0" || fields[1] != "" ||
		fields[9] != "cloth0002" || fields[10] != "hc_2400177" || fields[11] != "cf_composite4" ||
		fields[23] != "title001" || fields[24] != "scene_name_017" || fields[32] != "book1" ||
		fields[25] != "introduction_qdz01_title" {
		t.Fatalf("rolePara %q", vl.Values[3].Text)
	}
}

// ---- 发现监督器谓词 ----

// markConn 白盒标记 comm 已连（识别入登记等价；谓词只看登记面）。
func markConn(s *Server, comm uint32) {
	s.inner.mu.Lock()
	s.inner.commConn[comm] = 1
	s.inner.mu.Unlock()
}

func TestDiscoveryPendingWorld(t *testing.T) {
	cfg := DefaultConfig()
	cfg.InnerAddr = "127.0.0.1"
	cfg.InnerPort = 0
	cfg.BroadAddr = "127.255.255.255"
	cfg.BroadPort = 0
	cfg.EchoAddr = "127.0.0.1"
	cfg.EchoPort = 0
	s := New(Options{Config: cfg, ManualTick: true, Scenes: []SceneSpec{
		{Index: 1, Member: 0, Script: "scene", Config: "c1"},
		{Index: 2, Member: 3, Script: "scene", Config: "c2"},
	}})
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	if ms := s.Scenes().Members(); len(ms) != 2 || ms[0] != 0 || ms[1] != 3 {
		t.Fatalf("Members() = %v（scenes 映射 member 去重升序）", ms)
	}
	if p := s.DiscoveryPending(); strings.Join(p, ",") != "store,member0,member3,entry0,public" {
		t.Fatalf("初始缺口 = %v", p)
	}
	markConn(s, commStoreMin)
	markConn(s, commMemberMin+3)
	if p := s.DiscoveryPending(); strings.Join(p, ",") != "member0,entry0,public" {
		t.Fatalf("部分连入后缺口 = %v", p)
	}
	markConn(s, commMemberMin)
	markConn(s, commEntryMin)
	markConn(s, commPublic)
	if p := s.DiscoveryPending(); len(p) != 0 {
		t.Fatalf("全连后应就绪 = %v", p)
	}
	// auction 可选对端：缺席不影响就绪判定（上面即无 auction）
}
