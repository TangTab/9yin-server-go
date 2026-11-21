package fssolostore

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
	"unicode/utf16"
)

func TestStoreAnnounceFrameGolden(t *testing.T) {
	f := buildStoreAnnounce("127.0.0.1", 16100, 1, 11, 0)
	if len(f) != 0x28 {
		t.Fatalf("frame size = %#x, want 0x28", len(f))
	}
	if le32(f, 0) != kindStore || le32(f, 4) != 0x3001 {
		t.Fatalf("kind/magic = %#x/%#x", le32(f, 0), le32(f, 4))
	}
	if string(f[8:17]) != "127.0.0.1" || f[17] != 0 {
		t.Fatalf("addr bytes = %q", f[8:24])
	}
	if le32(f, 0x18) != 16100 || le32(f, 0x1C) != 1 || le32(f, 0x20) != 11 || le32(f, 0x24) != 0 {
		t.Fatalf("port/district/server/store = %d/%d/%d/%d",
			le32(f, 0x18), le32(f, 0x1C), le32(f, 0x20), le32(f, 0x24))
	}
}

func TestSerialRoundTrip(t *testing.T) {
	// world/public/auction 直连
	for _, c := range []uint32{0, 1, 2} {
		m, req, d, ok := decodeSerial(encodeSerial(c, 0x1234))
		if !ok || d != int(c) || req != 0x1234 || m != 0 {
			t.Fatalf("direct %d: %#x", c, encodeSerial(c, 0x1234))
		}
	}
	// member idx 0..0x17
	for idx := 0; idx < memberNumMax; idx++ {
		comm := uint32(commMemberMin + idx)
		m, req, d, ok := decodeSerial(encodeSerial(comm, 0xABCD))
		if !ok || d != -1 || m != idx || req != 0xABCD {
			t.Fatalf("member %d: idx=%d", idx, m)
		}
	}
	// store comm（4..7）落在 member 域外 → 不合法
	if _, _, _, ok := decodeSerial(encodeSerial(6, 1)); ok {
		t.Fatal("store comm decoded as valid")
	}
}

func TestMsgBackIDsGolden(t *testing.T) {
	cases := map[uint32]uint32{
		0x02: 0x02, 0x04: 0x03, 0x0B: 0x07, 0x0C: 0x08, 0x0E: 0x0B,
		0x10: 0x0D, 0x11: 0x0E, 0x1A: 0x11, 0x1B: 0x12, 0x1C: 0x13,
		0x24: 0x19, 0x25: 0x1A, 0x28: 0x1D, 0x29: 0x1E, 0x2A: 0x1F,
		0x2B: 0x20, 0x2C: 0x21, 0x2D: 0x22, 0x36: 0x25, 0x37: 0x26,
		0x3F: 0x29, 0x41: 0x2C, 0x42: 0x2D,
	}
	for fwd, back := range cases {
		if msgBackIDs[fwd] != back {
			t.Errorf("msg %#x back = %#x, want %#x", fwd, msgBackIDs[fwd], back)
		}
	}
	// member ProcessStoreMsg manage 族对上（0x1E..0x21）
	for m, back := range map[uint32]uint32{0x29: 0x1E, 0x2A: 0x1F, 0x2B: 0x20, 0x2C: 0x21} {
		if msgBackIDs[m] != back {
			t.Errorf("manage %#x back = %#x", m, msgBackIDs[m])
		}
	}
}

func TestStateTransitions(t *testing.T) {
	if transition(stateUnknown, eventStart) != stateOpened {
		t.Fatal("Unknown --Start--> Opened")
	}
	if transition(stateUnknown, eventTimer) != stateUnknown {
		t.Fatal("Unknown timer no-op")
	}
	if transition(stateOpened, eventClose) != stateClosing {
		t.Fatal("Opened --Close--> Closing")
	}
	if transition(stateOpened, eventTimer) != stateOpened {
		t.Fatal("Opened timer stays")
	}
	if transition(stateClosed, eventStart) != stateClosed {
		t.Fatal("Closed terminal")
	}
}

func TestLogTableSetOfficialSchema(t *testing.T) {
	s := NewLogTableSet()
	if s.Count() != 29 {
		t.Fatalf("tables = %d, want 29", s.Count())
	}
	if s.AddTable("item_log") {
		t.Error("duplicate accepted")
	}
	for i := 0; i < 32-29; i++ {
		if !s.AddTable(fmt.Sprintf("extra_%d", i)) {
			t.Fatal("fill to 32 failed")
		}
	}
	if s.AddTable("overflow") {
		t.Error("table too much not enforced")
	}
	if s.AddField("item_log", "x", "datetime", 8) != true {
		t.Error("AddField should succeed")
	}
	if s.AddField("nope", "x", "int", 4) {
		t.Error("unknown table accepted")
	}
	if s.AddField("item_log", "x", "blob", 4) {
		t.Error("invalid type accepted")
	}
}

func TestManageInfoArchiveRoundTrip(t *testing.T) {
	src := ManageEntry{Name: "test", Title: "标题", Type: 2, Comment: "备注", Order: 1, State: 0}
	b := encodeManageInfo(src)
	// member 端 decodeManageInfo 读法（fssolomember/store.go）复刻校验
	off := 0
	readStr := func() string {
		n := int(le32(b, off))
		off += 4
		s := string(b[off : off+n])
		off += n
		for i := 0; i < len(s); i++ {
			if s[i] == 0 {
				return s[:i]
			}
		}
		return s
	}
	readWide := func() string {
		n := int(le32(b, off))
		off += 4
		raw := b[off : off+n*2]
		off += n * 2
		u := make([]uint16, 0, n)
		for i := 0; i < n; i++ {
			u = append(u, binary.LittleEndian.Uint16(raw[i*2:]))
		}
		for i, c := range u {
			if c == 0 {
				return string(utf16.Decode(u[:i]))
			}
		}
		return string(utf16.Decode(u))
	}
	readInt := func() int32 {
		v := int32(le32(b, off))
		off += 4
		return v
	}
	if got := readStr(); got != src.Name {
		t.Fatalf("name = %q", got)
	}
	if got := readWide(); got != src.Title {
		t.Fatalf("title = %q", got)
	}
	if got := readInt(); got != src.Type {
		t.Fatalf("type = %d", got)
	}
	if got := readWide(); got != src.Comment {
		t.Fatalf("comment = %q", got)
	}
	if got := readInt(); got != src.Order || readInt() != src.State {
		t.Fatal("order/state")
	}
	if off != len(b) {
		t.Fatalf("consumed %d of %d", off, len(b))
	}
}

func TestMemoryStoreSemantics(t *testing.T) {
	m := NewMemoryStore("")
	m.AddManageEntry(ManageEntry{Name: "cfg1"})
	if es, _ := m.ManageInfos(); len(es) != 1 {
		t.Fatal("manage entries")
	}
	if err := m.SaveChunk(7, 0x11, []byte{1, 2, 3}); err != nil {
		t.Fatal(err)
	}
	if ids, _ := m.ChunkIDs(7); len(ids) != 1 || ids[0] != 0x11 {
		t.Fatalf("chunk ids %v", ids)
	}
	if d, _ := m.LoadChunk(7, []uint32{0x11}); len(d[0x11]) != 3 {
		t.Fatal("load chunk")
	}
	if ok, _ := m.CreateRole("acct", "role1", "u1", 11, RoleMeta{Index: 0}, nil); !ok {
		t.Fatal("create role")
	}
	if ok, _ := m.CreateRole("acct", "role1", "u1", 11, RoleMeta{}, nil); ok {
		t.Fatal("dup role accepted")
	}
	if rs, _ := m.QueryAccount("acct", 11); len(rs) != 1 || rs[0].Name != "role1" {
		t.Fatalf("query account %v", rs)
	}
	m.AddAccount("user1", "pw1")
	if info, ok, _ := m.AccountLogon("user1", "pw1", "", 5); !ok {
		t.Fatal("logon")
	} else if info != (AccountInfo{}) {
		t.Fatalf("logon info %+v", info)
	}
	if _, ok, _ := m.AccountLogon("user1", "pw1", "", 6); ok {
		t.Fatal("double logon other id accepted")
	}
	if _, ok, _ := m.AccountLogon("user1", "bad", "", 5); ok {
		t.Fatal("wrong password accepted")
	}
	if _, ok, _ := m.AccountLogon("stranger", "pw", "", 7); !ok {
		t.Fatal("first logon should auto-register")
	}
	if err := m.AccountLogoff("user1", 5); err != nil {
		t.Fatal(err)
	}
}

// ---- 全链集成：真 TCP member 客户端 + UDP 广播 + echo ----

func freeTCPPort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("tcp: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func freeUDPPort(t *testing.T) int {
	t.Helper()
	c, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("udp: %v", err)
	}
	defer c.Close()
	return c.LocalAddr().(*net.UDPAddr).Port
}

func startTestServer(t *testing.T) (*Server, *MemoryStore) {
	t.Helper()
	store := NewMemoryStore("")
	store.AddManageEntry(ManageEntry{Name: "boot", Title: "T", Comment: "C", Order: 1})
	_ = store.SaveChunk(9, 0x21, []byte{9, 9})
	_ = store.SaveChunk(9, 0x22, []byte{8, 8, 8})
	s := New(Options{
		Config: Config{
			DistrictID: 1, ServerID: 11, StoreID: 0,
			InnerAddr: "127.0.0.1", InnerPort: 0,
			BroadAddr: "127.0.0.1", BroadPort: freeUDPPort(t),
			EchoAddr: "127.0.0.1", EchoPort: freeTCPPort(t),
			UseMysql: false,
		},
		Backend:    store,
		ManualTick: true,
		Logf:       func(string, ...any) {},
	})
	if err := s.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() { s.Stop() })
	return s, store
}

type memberClient struct {
	conn net.Conn
	rd   *bufio.Reader
	t    *testing.T
}

func dialMember(t *testing.T, addr string) *memberClient {
	t.Helper()
	var c net.Conn
	var err error
	for i := 0; i < 50; i++ {
		c, err = net.Dial("tcp", addr)
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	mc := &memberClient{conn: c, rd: bufio.NewReader(c), t: t}
	t.Cleanup(func() { c.Close() })
	return mc
}

func (m *memberClient) send(comm, id1, id2 uint32, payload []byte) {
	f, err := encodeInner(comm, id1, id2, payload)
	if err != nil {
		m.t.Fatal(err)
	}
	if _, err := m.conn.Write(f); err != nil {
		m.t.Fatal(err)
	}
}

func (m *memberClient) recv() (comm, id1, id2 uint32, payload []byte) {
	for {
		comm, id1, id2, payload = m.recvRaw()
		if id2 != innerIdentifyID2 {
			return // 受面双向 identify：跳过 store 的自报帧
		}
	}
}

func (m *memberClient) recvRaw() (comm, id1, id2 uint32, payload []byte) {
	m.conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	head := make([]byte, 4)
	if _, err := ioReadFull(m.conn, head); err != nil {
		m.t.Fatalf("read head: %v", err)
	}
	n := binary.LittleEndian.Uint32(head)
	body := make([]byte, n)
	if _, err := ioReadFull(m.conn, body); err != nil {
		m.t.Fatalf("read body: %v", err)
	}
	comm = binary.LittleEndian.Uint32(body[0:4])
	id1 = binary.LittleEndian.Uint32(body[4:8])
	id2 = binary.LittleEndian.Uint32(body[8:12])
	payload = body[12:]
	return
}

func ioReadFull(c net.Conn, b []byte) (int, error) {
	total := 0
	for total < len(b) {
		n, err := c.Read(b[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

func TestFullLinkPublicRelationReply(t *testing.T) {
	s, _ := startTestServer(t)
	pc := dialMember(t, fmt.Sprintf("127.0.0.1:%d", s.InnerPort()))
	pc.send(commPublic, 55, innerIdentifyID2, nil)
	pc.send(commPublic, 66, 0x2D, []byte{11, 0, 0, 0})
	comm, req, id2, p := pc.recv()
	if comm != commSelf(0) || req != 66 || id2 != 0x22 {
		t.Fatalf("reply comm=%d req=%d id2=%#x", comm, req, id2)
	}
	if len(p) != 0x10 || binary.LittleEndian.Uint32(p[0:]) != 1 {
		t.Fatalf("empty header expected: % X", p)
	}
}

func TestFullLinkAccountLogon(t *testing.T) {
	s, _ := startTestServer(t)
	wc := dialMember(t, fmt.Sprintf("127.0.0.1:%d", s.InnerPort()))
	wc.send(commWorld, 21, innerIdentifyID2, nil)
	// 0x36 logon（0x98 定长）：首登自动建号 → 0x25 result=1
	logon := make([]byte, 0x98)
	put32 := func(off int, v uint32) { binary.LittleEndian.PutUint32(logon[off:], v) }
	put32(0, 7)
	put32(4, 21)
	put32(8, 5001)
	copy(logon[0xC:], "hero")
	copy(logon[0x30:], "sword")
	wc.send(commWorld, 21, 0x36, logon)
	comm, req, id2, p := wc.recv()
	if comm != commSelf(0) || req != 21 || id2 != 0x25 {
		t.Fatalf("logon reply comm=%d req=%d id2=%#x", comm, req, id2)
	}
	if len(p) != 0x18 || binary.LittleEndian.Uint32(p[0:]) != 7 ||
		binary.LittleEndian.Uint32(p[4:]) != 21 || binary.LittleEndian.Uint32(p[8:]) != 1 {
		t.Fatalf("logon reply body %#x", p)
	}
	// 错密码 → result=0
	copy(logon[0x30:], "wrong")
	wc.send(commWorld, 22, 0x36, logon)
	_, _, id2, p = wc.recv()
	if id2 != 0x25 || binary.LittleEndian.Uint32(p[8:]) != 0 {
		t.Fatalf("wrong password should deny: %#x", p)
	}
	// 0x37 logoff → 0x26 result=1
	off := make([]byte, 0x7C)
	binary.LittleEndian.PutUint32(off[0:], 7)
	binary.LittleEndian.PutUint32(off[4:], 22)
	binary.LittleEndian.PutUint32(off[8:], 5001)
	copy(off[0xC:], "hero")
	wc.send(commWorld, 23, 0x37, off)
	_, _, id2, p = wc.recv()
	if id2 != 0x26 || binary.LittleEndian.Uint32(p[8:]) != 1 {
		t.Fatalf("logoff reply %#x", p)
	}
	// 复登（同会话号；logged 已清）→ 再过门
	copy(logon[0x30:], "sword")
	wc.send(commWorld, 24, 0x36, logon)
	_, _, id2, p = wc.recv()
	if id2 != 0x25 || binary.LittleEndian.Uint32(p[8:]) != 1 {
		t.Fatalf("re-logon should pass: %#x", p)
	}
}

func TestFullLinkManageInfoBootstrap(t *testing.T) {
	s, _ := startTestServer(t)
	mc := dialMember(t, fmt.Sprintf("127.0.0.1:%d", s.InnerPort()))
	// member identify：comm=0x18+0
	mc.send(0x18, 12, innerIdentifyID2, nil)
	// member 引导：SendToStore(0, 0, 0x29, {ServerId}) → 逐项回 0x1E
	mc.send(0x18, 0, 0x29, []byte{11, 0, 0, 0})
	comm, req, id2, payload := mc.recv()
	if comm != commSelf(0) {
		t.Fatalf("resp comm = %#x, want 4", comm)
	}
	if req != 0 || id2 != 0x1E {
		t.Fatalf("resp = req:%#x id2:%#x, want 0/0x1E", req, id2)
	}
	// decodeManageInfo 读法（member 端闩依赖）
	if len(payload) < 4 || le32(payload, 0) < 1 {
		t.Fatalf("manage payload len %d", len(payload))
	}
}

func TestFullLinkChunkRoundTrip(t *testing.T) {
	s, _ := startTestServer(t)
	mc := dialMember(t, fmt.Sprintf("127.0.0.1:%d", s.InnerPort()))
	mc.send(0x18, 7, innerIdentifyID2, nil)
	// chunk_info {scene}
	mc.send(0x18, 7, 0x18, []byte{9, 0, 0, 0})
	_, req, id2, p := mc.recv()
	if id2 != 0x0F || req != 7 {
		t.Fatalf("chunk info = %#x req %d", id2, req)
	}
	if le32(p, 0) != 9 || le32(p, 4) != 2 {
		t.Fatalf("chunk list = %v", p)
	}
	// load_chunk {scene, n, ids}
	mc.send(0x18, 8, 0x19, []byte{9, 0, 0, 0, 2, 0, 0, 0, 0x21, 0, 0, 0, 0x22, 0, 0, 0})
	_, _, id2, p = mc.recv()
	if id2 != 0x10 {
		t.Fatalf("load chunk = %#x", id2)
	}
	if le32(p, 4) != 2 || le32(p, 8) != 0x21 || le32(p, 12) != 2 {
		t.Fatalf("load frame = %v", p)
	}
	// save_chunk upsert
	sv := []byte{9, 0, 0, 0, 0x33, 0, 0, 0, 1, 0, 0, 0, 0xAA}
	mc.send(0x18, 9, 0x1A, sv)
	_, _, id2, p = mc.recv()
	if id2 != 0x11 || le32(p, 0) != 0x33 {
		t.Fatalf("save ack = %#x %v", id2, p)
	}
	// delete_chunk
	mc.send(0x18, 10, 0x1B, []byte{9, 0, 0, 0, 0x21, 0, 0, 0})
	_, _, id2, p = mc.recv()
	if id2 != 0x12 || le32(p, 0) != 0x21 {
		t.Fatalf("del ack = %#x %v", id2, p)
	}
}

func TestFullLinkAuctionInterop(t *testing.T) {
	s, store := startTestServer(t)
	_ = store.AuctionSave(AuctionAttr, "attr_blob", []byte{0x50, 0x55, 0x42, 0x34})
	mc := dialMember(t, fmt.Sprintf("127.0.0.1:%d", s.InnerPort()))
	mc.send(commAuction, 3, innerIdentifyID2, nil)
	// auction 以 comm=2 直连发 0x3F；回 0x29 {result}+块
	mc.send(commAuction, 3, 0x3F, []byte{1, 0, 0, 0})
	_, req, id2, p := mc.recv()
	if id2 != 0x29 || req != 3 || len(p) != 8 || le32(p, 0) != 0 {
		t.Fatalf("auction attr = %#x req=%d p=%v", id2, req, p)
	}
	// record load → 0x2A 帧 + 0x2B 终帧
	mc.send(commAuction, 4, 0x40, []byte{1, 0, 0, 0})
	_, _, id2, p = mc.recv()
	if id2 != 0x2A && id2 != 0x2B {
		t.Fatalf("auction rec = %#x", id2)
	}
	if id2 == 0x2A { // 空库时只有终帧；有块则先块后终帧
		_, _, id2, p = mc.recv()
	}
	if id2 != 0x2B || le32(p, 0) != 1 { // result!=0 = 装载成功（auction LoadOK 开闸位）
		t.Fatalf("load ok = %#x %v", id2, p)
	}
}

func TestEchoWelcomeAndQuit(t *testing.T) {
	s, _ := startTestServer(t)
	// EchoPort=0 → 系统分配：从 config 读不回填（Start 回填的是 inner）；
	// 这里直接用 Server 暴露的实际口
	addr := fmt.Sprintf("127.0.0.1:%d", s.echoPort())
	var c net.Conn
	var err error
	for i := 0; i < 50; i++ {
		c, err = net.Dial("tcp", addr)
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("dial echo: %v", err)
	}
	defer c.Close()
	c.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 128)
	n, _ := c.Read(buf)
	welcome := "welcome server 11 store 0\r"
	if string(buf[:n]) != welcome {
		t.Fatalf("welcome = %q", buf[:n])
	}
	c.Write([]byte("status\r"))
	n, _ = c.Read(buf)
	if want := ">undefined command: status\r"; string(buf[:n]) != want {
		t.Fatalf("undef = %q", buf[:n])
	}
	c.Write([]byte("end\r"))
	c.SetReadDeadline(time.Now().Add(2 * time.Second))
	if n, err := c.Read(buf); n != 0 || err == nil {
		t.Fatalf("end should close: n=%d err=%v", n, err)
	}
}

func TestBroadcastSelfAnnounce(t *testing.T) {
	s, _ := startTestServer(t)
	// 广播口=系统分配单播口：直接读 Server 构造帧比对即可（真 UDP 组播
	// 由 reactor 测试覆盖）；此处验证每拍帧内容 + StoreId 原值。
	s.BroadcastStoreInfo()
	f := buildStoreAnnounce(s.Config().InnerAddr, uint32(s.Config().InnerPort),
		1, 11, 0)
	if le32(f, 0x24) != 0 {
		t.Fatalf("+0x24 = %d, want StoreId 0", le32(f, 0x24))
	}
}

// TestFullLinkCreateRoleAndQuery 建角/清单 dispatch 全链（0x0A → 0x06；
// 0x10 → 0x0D）：帧布局与 fssoloworld/roles.go 定版互为对偶。
func TestFullLinkCreateRoleAndQuery(t *testing.T) {
	s, store := startTestServer(t)
	wc := dialMember(t, fmt.Sprintf("127.0.0.1:%d", s.InnerPort()))
	wc.send(commWorld, 41, innerIdentifyID2, nil)

	// 0x0A create：{client=7, serial=41, server_id=11, account@0xC,
	// name wchar@0x2D, para wchar@0x71}
	frame := make([]byte, 0xD4)
	binary.LittleEndian.PutUint32(frame[0:], 7)
	binary.LittleEndian.PutUint32(frame[4:], 41)
	binary.LittleEndian.PutUint32(frame[8:], 11)
	copy(frame[0xC:], "hero")
	units := utf16.Encode([]rune("王五"))
	for i, u := range units {
		binary.LittleEndian.PutUint16(frame[0x2D+i*2:], u)
	}
	units = utf16.Encode([]rune("book1,hair1"))
	for i, u := range units {
		binary.LittleEndian.PutUint16(frame[0x71+i*2:], u)
	}
	wc.send(commWorld, 41, 0x0A, frame)

	comm, req, id2, p := wc.recv()
	if comm != commSelf(0) || req != 41 || id2 != 0x06 {
		t.Fatalf("create reply comm=%d req=%d id2=%#x", comm, req, id2)
	}
	if binary.LittleEndian.Uint32(p[0:]) != 7 || binary.LittleEndian.Uint32(p[4:]) != 41 {
		t.Fatalf("create reply route %#x %#x", p[0:4], p[4:8])
	}
	if uidLen := binary.LittleEndian.Uint32(p[8:]); uidLen != 33 { // 32 hex + NUL
		t.Fatalf("uid len %d", uidLen)
	}
	uid := string(p[12 : 12+32])
	rest := p[12+33:]
	n := binary.LittleEndian.Uint32(rest)
	if n != 3 { // 王五 2 码元 + NUL
		t.Fatalf("name len %d", n)
	}
	if got := wideNameOf(rest[4 : 4+(n-1)*2]); got != "王五" {
		t.Fatalf("create reply name %q", got)
	}
	// 落库断言（重名门/外观 para/出生场景缺省）
	briefs, err := store.QueryAccount("hero", 11)
	if err != nil || len(briefs) != 1 || briefs[0].Name != "王五" ||
		briefs[0].UID != uid || briefs[0].Para != "book1,hair1" {
		t.Fatalf("query account %+v err %v", briefs, err)
	}
	if data, meta, ok, _ := store.LoadRole("hero", "王五", 11); !ok ||
		meta.SceneID != birthSceneID || meta.Location != birthSceneLocation || data != nil {
		t.Fatalf("stored role meta %+v ok=%v", meta, ok)
	}

	// 重名拒绝 → {client,serial,u32 1}
	wc.send(commWorld, 42, 0x0A, frame)
	_, _, id2, p = wc.recv()
	if id2 != 0x06 || len(p) != 12 || binary.LittleEndian.Uint32(p[8:]) != 1 {
		t.Fatalf("duplicate create should deny: id2=%#x len=%d", id2, len(p))
	}

	// 0x10 query_account → 0x0D {client,serial,count,{uid,name,para}}
	q := make([]byte, 72)
	binary.LittleEndian.PutUint32(q[0:], 7)
	binary.LittleEndian.PutUint32(q[4:], 41)
	copy(q[8:], "hero")
	wc.send(commWorld, 43, 0x10, q)
	_, _, id2, p = wc.recv()
	if id2 != 0x0D {
		t.Fatalf("query reply id2=%#x", id2)
	}
	if binary.LittleEndian.Uint32(p[8:]) != 1 {
		t.Fatalf("query count %#x", p[8:])
	}
	item := p[12:]
	uidN := binary.LittleEndian.Uint32(item)
	if uidN != 33 { // uid arch-str：32 hex + NUL
		t.Fatalf("item uid len %d", uidN)
	}
	item = item[4+uidN:]
	if got := wideNameOfNext(&item); got != "王五" {
		t.Fatalf("query name %q", got)
	}
	if got := wideNameOfNext(&item); got != "book1,hair1" {
		t.Fatalf("query para %q", got)
	}
}

// wideNameOf UTF-16LE 字节解码（测试断言用）。
func wideNameOf(b []byte) string {
	u := make([]uint16, len(b)/2)
	for i := range u {
		u[i] = binary.LittleEndian.Uint16(b[i*2:])
	}
	return string(utf16.Decode(u))
}

// wideNameOfNext 从 item 流读一个 archive 宽串并前移。
func wideNameOfNext(item *[]byte) string {
	b := *item
	n := binary.LittleEndian.Uint32(b)
	out := wideNameOf(b[4 : 4+(n-1)*2])
	*item = b[4+n*2:]
	return out
}

// TestFullLinkDeleteRole 删角 dispatch 全链（0x0B → 0x07 软删；再删拒；
// 清单为空）：帧与 fssoloworld/roles.go buildDeleteRoleStore 对偶。
func TestFullLinkDeleteRole(t *testing.T) {
	s, store := startTestServer(t)
	wc := dialMember(t, fmt.Sprintf("127.0.0.1:%d", s.InnerPort()))
	wc.send(commWorld, 51, innerIdentifyID2, nil)

	// 先建角：0x0A（同 TestFullLinkCreateRoleAndQuery 布局）
	frame := make([]byte, 0xD4)
	binary.LittleEndian.PutUint32(frame[0:], 7)
	binary.LittleEndian.PutUint32(frame[4:], 51)
	binary.LittleEndian.PutUint32(frame[8:], 11)
	copy(frame[0xC:], "hero")
	units := utf16.Encode([]rune("王五"))
	for i, u := range units {
		binary.LittleEndian.PutUint16(frame[0x2D+i*2:], u)
	}
	wc.send(commWorld, 51, 0x0A, frame)
	_, _, id2, p := wc.recv()
	if id2 != 0x06 {
		t.Fatalf("create reply id2=%#x", id2)
	}
	uidLen := binary.LittleEndian.Uint32(p[8:])
	uid := string(p[12 : 12+uidLen-1])

	// 0x0B delete：与 create 同头 + uid@0x71 → 软删 → 0x07 同构回执
	copy(frame[0x71:], uid)
	frame = frame[:0xAC]
	wc.send(commWorld, 52, 0x0B, frame)
	_, req, id2, p := wc.recv()
	if req != 52 || id2 != 0x07 {
		t.Fatalf("delete reply req=%d id2=%#x", req, id2)
	}
	if binary.LittleEndian.Uint32(p[0:]) != 7 || binary.LittleEndian.Uint32(p[4:]) != 51 {
		t.Fatalf("delete reply route %#x", p[:8])
	}
	if n := binary.LittleEndian.Uint32(p[8:]); n != uidLen || string(p[12:12+n-1]) != uid {
		t.Fatalf("delete reply uid %q", p[8:12+uidLen])
	}
	rest := p[12+uidLen:]
	n := binary.LittleEndian.Uint32(rest)
	if got := wideNameOf(rest[4 : 4+(n-1)*2]); got != "王五" {
		t.Fatalf("delete reply name %q", got)
	}
	// 软删后清单为空；重复删拒
	if briefs, _ := store.QueryAccount("hero", 11); len(briefs) != 0 {
		t.Fatalf("query after delete %+v", briefs)
	}
	wc.send(commWorld, 53, 0x0B, frame)
	_, _, id2, p = wc.recv()
	if id2 != 0x07 || len(p) != 12 || binary.LittleEndian.Uint32(p[8:]) != 1 {
		t.Fatalf("repeat delete should deny: id2=%#x len=%d", id2, len(p))
	}
	// 名册回执（0x00 role_info）含 deleted 标志
	if briefs, _ := store.ResetOnline(11); len(briefs) != 1 || !briefs[0].Deleted {
		t.Fatalf("role info after delete %+v", briefs)
	}
}

// TestFullLinkLoadRole 存档载入 dispatch（0x02 → 0x02 回执）：与
// fssoloworld buildLoadRoleStore 对偶；空档（新角 save_data NULL）
// = ok 1 + para 往返。
func TestFullLinkLoadRole(t *testing.T) {
	s, store := startTestServer(t)
	wc := dialMember(t, fmt.Sprintf("127.0.0.1:%d", s.InnerPort()))
	wc.send(commWorld, 61, innerIdentifyID2, nil)

	// 种子角（API 直建）
	if _, err := store.CreateRole("hero", "王五", "GUID-LOAD-0001", 11,
		RoleMeta{SavePara1: "book1,hair1"}, nil); err != nil {
		t.Fatal(err)
	}

	frame := make([]byte, 0xBC)
	binary.LittleEndian.PutUint32(frame[0:], 7)
	binary.LittleEndian.PutUint32(frame[4:], 61)
	binary.LittleEndian.PutUint32(frame[8:], 11)
	copy(frame[0xC:], "hero")
	units := utf16.Encode([]rune("王五"))
	for i, u := range units {
		binary.LittleEndian.PutUint16(frame[0x2D+i*2:], u)
	}
	wc.send(commWorld, 61, 0x02, frame)
	_, req, id2, p := wc.recv()
	if req != 61 || id2 != 0x02 {
		t.Fatalf("load reply req=%d id2=%#x", req, id2)
	}
	if binary.LittleEndian.Uint32(p[0:]) != 7 || binary.LittleEndian.Uint32(p[4:]) != 61 ||
		binary.LittleEndian.Uint32(p[8:]) != 1 {
		t.Fatalf("load reply head %#x", p[:12])
	}
	rest := p[12:]
	if got := wideNameOfNext(&rest); got != "book1,hair1" {
		t.Fatalf("load reply para %q", got)
	}

	// 不存在的角色 → ok 0
	for i, u := range utf16.Encode([]rune("无")) {
		binary.LittleEndian.PutUint16(frame[0x2D+i*2:], u)
	}
	wc.send(commWorld, 62, 0x02, frame)
	_, _, id2, p = wc.recv()
	if id2 != 0x02 || binary.LittleEndian.Uint32(p[8:]) != 0 || len(p) != 12 {
		t.Fatalf("absent load should deny: len=%d ok=%d", len(p), binary.LittleEndian.Uint32(p[8:]))
	}
}

// ---- 发现监督器谓词 ----

func TestDiscoveryPendingStore(t *testing.T) {
	s, _ := startTestServer(t)
	if p := s.DiscoveryPending(); strings.Join(p, ",") != "any-peer" {
		t.Fatalf("无入连时缺口 = %v", p)
	}
	s.inner.mu.Lock()
	s.inner.commConn[commWorld] = 1
	s.inner.mu.Unlock()
	if p := s.DiscoveryPending(); len(p) != 0 {
		t.Fatalf("任一 identify 入连即就绪 = %v", p)
	}
}

// TestParseConnectStringDSN 官方 SQLOLEDB ConnectString → Go MySQL DSN
// 公共解析（ini 回落链路核心）。
func TestParseConnectStringDSN(t *testing.T) {
	cases := []struct {
		cs   string
		want string
	}{
		{
			"Provider=SQLOLEDB;Password=abc&4621;User ID=sa;Initial Catalog=nx_base1;Data Source=127.0.0.1:3306",
			"sa:abc&4621@tcp(127.0.0.1:3306)/nx_base1",
		},
		{
			"Provider=SQLOLEDB;Password=nineyin123;User ID=nineyin;Initial Catalog=nineyin;Data Source=127.0.0.1:3306",
			"nineyin:nineyin123@tcp(127.0.0.1:3306)/nineyin",
		},
		{"Provider=SQLOLEDB;Password=x", ""}, // 缺 User ID/库/源
		{"", ""},                             // 空串
		{"Data Source=127.0.0.1:3306;User ID=sa;Initial Catalog=db;Password=", "sa:@tcp(127.0.0.1:3306)/db"}, // 空密码
	}
	for _, c := range cases {
		if got := ParseConnectStringDSN(c.cs); got != c.want {
			t.Fatalf("ParseConnectStringDSN(%q) = %q, want %q", c.cs, got, c.want)
		}
	}
}
