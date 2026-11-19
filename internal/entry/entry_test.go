package entry

import (
	"encoding/binary"
	"net"
	"os"
	"testing"
	"time"

	gonet "net"

	fsnet "github.com/lizhanyin/9yin-server-go/internal/net"
)

// 快照官方 lic（BGserverCJ/Debug/127.0.0.1.lic）作为 golden。
const snapshotLicPath = `..\..\testdata\config-bootstrap\127.0.0.1.lic`

func TestLicenseSnapshotGolden(t *testing.T) {
	data, err := os.ReadFile(snapshotLicPath)
	if err != nil {
		t.Skipf("快照 lic 不可读: %v", err)
	}
	id, err := DecodeLicense(data, "127.0.0.1", OfficialMagic1, OfficialMagic2)
	if err != nil {
		t.Fatalf("快照 lic 解码失败: %v", err)
	}
	if id.Magic1 != OfficialMagic1 || id.Magic2 != OfficialMagic2 {
		t.Fatalf("魔数不符: %#x/%#x", id.Magic1, id.Magic2)
	}
	// 生成器自回环：同 IP 重签可再解出（快照尾部为随机填充不复刻）
	regen := GenerateLicense("127.0.0.1", OfficialMagic1, OfficialMagic2)
	if _, err := DecodeLicense(regen, "127.0.0.1", OfficialMagic1, OfficialMagic2); err != nil {
		t.Fatalf("生成器回环失败: %v", err)
	}
}

func TestLicenseRoundtripAndReject(t *testing.T) {
	lic := GenerateLicense("192.168.33.70", 0x11111111, 0x22222222)
	id, err := DecodeLicense(lic, "192.168.33.70", 0x11111111, 0x22222222)
	if err != nil {
		t.Fatalf("回环解码失败: %v", err)
	}
	if id.Magic1 != 0x11111111 || id.Magic2 != 0x22222222 {
		t.Fatalf("魔数回环不符: %#x/%#x", id.Magic1, id.Magic2)
	}
	// 换 IP 解码必须失败（密钥绑定机器）
	if _, err := DecodeLicense(lic, "10.0.0.1", 0x11111111, 0x22222222); err == nil {
		t.Fatal("异 IP 解码应当失败")
	}
	// 篡改首块检测
	tampered := append([]byte{}, lic...)
	tampered[1] ^= 0xFF
	if _, err := DecodeLicense(tampered, "192.168.33.70", 0x11111111, 0x22222222); err == nil {
		t.Fatal("篡改 lic 应当失败")
	}
	// 长度门
	if _, err := DecodeLicense(lic[:200], "192.168.33.70", 0, 0); err == nil {
		t.Fatal("短 lic 应当失败")
	}
}

func TestParseGameEntryINI(t *testing.T) {
	ini := `
[System]
Sleep=1
[Module]
FsKit.dll
[Main]
IDC=2
DistrictID=3
ServerID=11
EntryID=1
MaxPlayer=2048
;MessageEncodeKey=A5375CAE
[Outer]
Addr=192.168.33.70
Port=2001
[Inner]
Addr=10.0.0.5
[Broadcast]
Port=15001
`
	cfg, err := ParseGameEntryINI([]byte(ini))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.IDC != 2 || cfg.DistrictID != 3 || cfg.ServerID != 11 || cfg.EntryID != 1 {
		t.Fatalf("Main 域解析不符: %+v", cfg)
	}
	if cfg.MaxPlayer != 2048 {
		t.Fatalf("MaxPlayer=%d", cfg.MaxPlayer)
	}
	if cfg.OuterAddr != "192.168.33.70" || cfg.OuterPort != 2001 {
		t.Fatalf("Outer 域不符: %s:%d", cfg.OuterAddr, cfg.OuterPort)
	}
	if cfg.InnerPort != 16200+1 { // 16200+EntryID 缺省推导
		t.Fatalf("InnerPort=%d", cfg.InnerPort)
	}
	if cfg.EchoPort != 15200+1 {
		t.Fatalf("EchoPort=%d", cfg.EchoPort)
	}
	if cfg.License != "192.168.33.70.lic" {
		t.Fatalf("License=%q", cfg.License)
	}
}

func TestPickCell(t *testing.T) {
	cases := []struct {
		w, h, want int32
	}{
		{0x100, 0x100, 1},    // 256/1 ≤ 0x200 → cell 1
		{0x201, 0x100, 2},    // 513/1 > 0x200 → 2；513/2=256 不> 0x200 停
		{0x401, 0x100, 2},    // 1025/2=512 不> 0x200 停（官方严格大于）
		{0x1001, 0x100, 8},   // 4097/4=1024 > 0x200 → 8；4097/8=512 停
		{0x100000, 0x100, 8}, // 上限 8
	}
	for _, c := range cases {
		if got := pickCell(c.w, c.h); got != c.want {
			t.Fatalf("pickCell(%#x,%#x)=%d want %d", c.w, c.h, got, c.want)
		}
	}
}

func TestSightTableOrder(t *testing.T) {
	tab := buildSightTable()
	if len(tab) != 255*255 {
		t.Fatalf("表长 %d", len(tab))
	}
	if tab[0].dx != 0 || tab[0].dy != 0 || tab[0].dist != 0 {
		t.Fatalf("首项应为原点: %+v", tab[0])
	}
	for i := 1; i < len(tab); i++ {
		if tab[i].dist < tab[i-1].dist {
			t.Fatalf("第 %d 项乱序", i)
		}
	}
}

func TestClientsLifecycle(t *testing.T) {
	c := NewClients()
	idx := c.Alloc(3, "1.2.3.4")
	if idx < 0 {
		t.Fatal("分配失败")
	}
	st := c.Client(idx)
	if st.Ident != uint32(3)<<16|uint32(idx) {
		t.Fatalf("ident=%#x", st.Ident)
	}
	if st.Member != -1 || st.Online {
		t.Fatalf("初始态不符: %+v", st)
	}
	// serial 不匹配拒绝
	if c.Lookup(st.Ident, st.Serial+1) != nil {
		t.Fatal("serial 不匹配应拒绝")
	}
	// 上线绑定 member 2
	if err := c.SetOnline(st.Ident, st.Serial, 2, 0xABCD); err != nil {
		t.Fatal(err)
	}
	st = c.Client(idx)
	if !st.Online || st.Member != 2 || st.MemberIdent != 0xABCD {
		t.Fatalf("online 后不符: %+v", st)
	}
	cc, iu, on := c.Stats()
	if cc != 1 || iu != 1 || on != 1 {
		t.Fatalf("统计不符 %d/%d/%d", cc, iu, on)
	}
	// 频道
	if err := c.ChannelAdd(st.Ident, st.Serial, 7); err != nil {
		t.Fatal(err)
	}
	if len(c.ChannelMembers(7)) != 1 {
		t.Fatal("频道成员数")
	}
	c.ClearChannel(7)
	if len(c.ChannelMembers(7)) != 0 {
		t.Fatal("清频后应空")
	}
	// 释放
	c.Release(idx)
	if _, _, on = c.Stats(); on != 0 {
		t.Fatalf("释放后 online=%d", on)
	}
}

func TestSceneGridSight(t *testing.T) {
	c := NewClients()
	// 场景 5：原点 (0,0)，512×512，cell=1
	alice := c.Alloc(0, "a")
	bob := c.Alloc(0, "b")
	far := c.Alloc(0, "f")
	for _, setup := range []struct {
		idx  int
		x, y int32
	}{{alice, 100, 100}, {bob, 105, 100}, {far, 200, 200}} {
		st := c.Client(setup.idx)
		if err := c.SetPosi(st.Ident, st.Serial, 5, 0, 0, 512, 512, setup.x, setup.y); err != nil {
			t.Fatal(err)
		}
		if err := c.SetOnline(st.Ident, st.Serial, 0, 0); err != nil {
			t.Fatal(err)
		}
	}
	a := c.Client(alice)
	seen := c.SightMembers(a.Ident, 5, 100, 100, 50)
	if len(seen) != 1 || seen[0] != bob {
		t.Fatalf("视野内应为 bob: %v", seen)
	}
	// 大半径覆盖 far（螺旋表半径 ±127 格，cell=1 时上限 ±127 世界单位）
	seen = c.SightMembers(a.Ident, 5, 100, 100, 250)
	if len(seen) != 2 {
		t.Fatalf("大半径应含 far: %v", seen)
	}
	// 维度变化触发整场重建：官方语义是整场摘格、逐客户端重发 posi 再落位
	if err := c.SetPosi(a.Ident, a.Serial, 5, 0, 0, 1024, 1024, 100, 100); err != nil {
		t.Fatal(err)
	}
	if seen = c.SightMembers(a.Ident, 5, 100, 100, 50); len(seen) != 0 {
		t.Fatalf("重建后 bob 应已摘格: %v", seen)
	}
	b := c.Client(bob)
	if err := c.SetPosi(b.Ident, b.Serial, 5, 0, 0, 1024, 1024, 105, 100); err != nil {
		t.Fatal(err)
	}
	if seen = c.SightMembers(a.Ident, 5, 100, 100, 50); len(seen) != 1 || seen[0] != bob {
		t.Fatalf("重落位后视野不符: %v", seen)
	}
}

func TestMoveBatch(t *testing.T) {
	c := NewClients()
	idx := c.Alloc(0, "m")
	st := c.Client(idx)
	_ = c.SetPosi(st.Ident, st.Serial, 9, 0, 0, 256, 256, 10, 10)
	_ = c.SetOnline(st.Ident, st.Serial, 0, 0)
	moved := c.Move(9, [][4]uint32{{st.Ident, st.Serial, 20, 20}, {1, 1, 1, 1}})
	if moved != 1 {
		t.Fatalf("moved=%d", moved)
	}
	if got := c.Client(idx); got.X != 20 || got.Y != 20 {
		t.Fatalf("移动后坐标 %d/%d", got.X, got.Y)
	}
}

func TestInnerWireCodec(t *testing.T) {
	frame := encodeInner(0x18, 0x99, ForwSendClient, []byte{1, 2, 3})
	hdr, payload, err := decodeInner(frame)
	if err != nil {
		t.Fatal(err)
	}
	if hdr.Src != 0x18 || hdr.Dst != 0x99 || hdr.Type != ForwSendClient || len(payload) != 3 {
		t.Fatalf("编解码回环不符: %+v", hdr)
	}
	// 源校验
	for _, ok := range []uint32{0, 0x18, 0x27} {
		if !validSource(ok) {
			t.Fatalf("%#x 应为合法源", ok)
		}
	}
	for _, bad := range []uint32{1, 2, 0x17, 0x28, 0xFFFF} {
		if validSource(bad) {
			t.Fatalf("%#x 应为非法源", bad)
		}
	}
}

func TestUpstreamBuilders(t *testing.T) {
	acc := BuildUpstreamAccept(0x0003000A, 7, "1.2.3.4", 5678)
	if len(acc) != 28 {
		t.Fatalf("accept 通知应 28B: %d", len(acc))
	}
	if binary.LittleEndian.Uint32(acc[0:]) != 0x0003000A || binary.LittleEndian.Uint32(acc[4:]) != 7 {
		t.Fatal("ident/serial 域错位")
	}
	if binary.LittleEndian.Uint32(acc[24:]) != 5678 {
		t.Fatal("端口域错位")
	}
	data := BuildUpstreamData(1, 2, []byte{0xEE})
	if len(data) != 9 || data[8] != 0xEE {
		t.Fatal("data 帧不符")
	}
}

// fakeGateway 捕获下行发送。
type fakeGateway struct {
	sends               map[uint32][][]byte // ident → payloads
	kicked              []uint32
	shaken              []uint32
	allSend, onlineSend int
	channelSends        map[uint32][][]byte
	sightSend           int
}

func newFakeGateway() *fakeGateway {
	return &fakeGateway{sends: map[uint32][][]byte{}, channelSends: map[uint32][][]byte{}}
}

func (f *fakeGateway) ClientSend(ident, serial uint32, p []byte) error {
	f.sends[ident] = append(f.sends[ident], p)
	return nil
}
func (f *fakeGateway) ClientClose(ident, serial uint32) { f.kicked = append(f.kicked, ident) }
func (f *fakeGateway) Shake(ident, serial uint32)       { f.shaken = append(f.shaken, ident) }
func (f *fakeGateway) SendAll(p []byte)                 { f.allSend++ }
func (f *fakeGateway) SendOnline(p []byte)              { f.onlineSend++ }
func (f *fakeGateway) SendChannel(ch uint32, p []byte) {
	f.channelSends[ch] = append(f.channelSends[ch], p)
}
func (f *fakeGateway) SendSight(ident uint32, scene uint32, x, y, r int32, p []byte) { f.sightSend++ }
func (f *fakeGateway) SetClosed(closed bool)                                         {}

func TestProcessForwardAllTypes(t *testing.T) {
	clients := NewClients()
	fg := newFakeGateway()
	e := New(Options{}, clients, nil, fg)

	idx := clients.Alloc(0, "t")
	st := clients.Client(idx)

	mk := func(fields ...uint32) []byte {
		b := make([]byte, len(fields)*4)
		for i, v := range fields {
			binary.LittleEndian.PutUint32(b[i*4:], v)
		}
		return b
	}

	// 0 init_connect：开/关
	if err := e.ProcessForward(0, ForwInitConnect, mk(1)); err != nil {
		t.Fatal(err)
	}
	if e.State() != StateOpened {
		t.Fatalf("state=%v", e.State())
	}
	_ = e.ProcessForward(0, ForwInitConnect, mk(0))
	if e.State() != StatePaused {
		t.Fatalf("state=%v", e.State())
	}
	// 3 set_online
	online := mk(st.Ident, st.Serial, 1, 0x77)
	if err := e.ProcessForward(0x19, ForwSetOnline, online); err != nil {
		t.Fatal(err)
	}
	if !clients.Client(idx).Online {
		t.Fatal("应已上线")
	}
	// 4 set_posi（40B：ident,serial,_,scene,_,_,x0,y0,w,h）
	posi := mk(st.Ident, st.Serial, 0, 5, 10, 10, 0, 0, 256, 256)
	if len(posi) != 40 {
		t.Fatalf("posi 长度 %d", len(posi))
	}
	if err := e.ProcessForward(0, ForwSetPosi, posi); err != nil {
		t.Fatal(err)
	}
	// 5 set_move（hdr{scene,_,n}+n×16）
	mov := mk(5, 0, 1)
	mov = append(mov, mk(st.Ident, st.Serial, 20, 30)...)
	if err := e.ProcessForward(0, ForwSetMove, mov); err != nil {
		t.Fatal(err)
	}
	if got := clients.Client(idx); got.X != 20 || got.Y != 30 {
		t.Fatalf("move 后 %d/%d", got.X, got.Y)
	}
	// 6 send_to_sight
	sight := mk(st.Ident, st.Serial, 0, 5, 20, 30, 50)
	sight = append(sight, 0xAA)
	if err := e.ProcessForward(0, ForwSendSight, sight); err != nil {
		t.Fatal(err)
	}
	if fg.sightSend != 1 {
		t.Fatalf("sightSend=%d", fg.sightSend)
	}
	// 7 send_to_client
	if err := e.ProcessForward(0, ForwSendClient, append(mk(st.Ident, st.Serial), 0x01, 0x02)); err != nil {
		t.Fatal(err)
	}
	if len(fg.sends[st.Ident]) != 1 || fg.sends[st.Ident][0][0] != 1 {
		t.Fatalf("client 发送不符: %v", fg.sends[st.Ident])
	}
	// 8/9/15 广播三面
	_ = e.ProcessForward(0, ForwSendAll, []byte{9})
	_ = e.ProcessForward(0, ForwSendMulti, []byte{9})
	_ = e.ProcessForward(0, ForwSendOnline, []byte{9})
	if fg.allSend != 2 || fg.onlineSend != 1 {
		t.Fatalf("广播计数 all=%d multi 并入 all=%d online=%d", fg.allSend, fg.allSend, fg.onlineSend)
	}
	// 10 shake
	_ = e.ProcessForward(0, ForwShakeClient, mk(st.Ident, st.Serial))
	if len(fg.shaken) != 1 {
		t.Fatal("shake 未派发")
	}
	// 11/12/13/14 频道
	_ = e.ProcessForward(0, ForwChannelAdd, mk(st.Ident, st.Serial, 3))
	_ = e.ProcessForward(0, ForwSendChannel, append(mk(3), 0xCC))
	if len(fg.channelSends[3]) != 1 {
		t.Fatal("频道发送未派发")
	}
	_ = e.ProcessForward(0, ForwChannelRm, mk(st.Ident, st.Serial, 3))
	_ = e.ProcessForward(0, ForwClearChan, mk(3))
	// 1 break_client
	_ = e.ProcessForward(0, ForwBreakClient, mk(st.Ident, st.Serial))
	if len(fg.kicked) != 1 || fg.kicked[0] != st.Ident {
		t.Fatal("kick 未派发")
	}
	// 2 set_inuse
	_ = e.ProcessForward(0, ForwSetInuse, append(mk(st.Ident, st.Serial), 1))
	// 未知类型
	if err := e.ProcessForward(0, 0x40, nil); err == nil {
		t.Fatal("未知类型应报错")
	}
}

func TestBroadcastInfoPacket(t *testing.T) {
	clients := NewClients()
	e := New(Options{ServerID: 11, EntryID: 0}, clients, nil, nil)
	pkt := e.BroadcastInfo("game_server", "entry_11_0", [4]uint32{1, 1, 11, 0}, 0)
	if len(pkt) != 0x48 {
		t.Fatalf("广播包 %d 字节", len(pkt))
	}
	if binary.LittleEndian.Uint32(pkt[0:]) != 1 || binary.LittleEndian.Uint32(pkt[4:]) != 0x3001 {
		t.Fatal("版本/消息号域错")
	}
}

func TestBlockIP(t *testing.T) {
	b := NewBlockIPList()
	if !b.AddBlockIP("10.0.0.1", "10.0.0.9") {
		t.Fatal("登记失败")
	}
	if !b.Blocked("10.0.0.5") || b.Blocked("10.0.1.5") {
		t.Fatal("区段命中判定错")
	}
	if b.AddBlockIP("bad", "10.0.0.1") {
		t.Fatal("非法地址应拒绝")
	}
	b.Clear()
	if b.Blocked("10.0.0.5") {
		t.Fatal("清空后不应命中")
	}
}

// ---- 端到端：真实 TCP 客户端 + 网关 + 内网假 world ----

func TestGatewayEndToEnd(t *testing.T) {
	clients := NewClients()
	var e *Entry
	inner := NewInner("127.0.0.1", 0, 0x100, InnerHandler{
		OnMessage: func(src, typ uint32, p []byte) {
			if e != nil {
				_ = e.ProcessForward(src, typ, p)
			}
		},
	})
	if err := inner.Start(); err != nil {
		t.Fatal(err)
	}
	defer inner.Stop()
	innerAddr := inner.Addr().(*gonet.TCPAddr)

	// 假 world：连入内网并注册 ident 0
	world, err := gonet.Dial("tcp", innerAddr.String())
	if err != nil {
		t.Fatal(err)
	}
	defer world.Close()
	writePrefixed := func(body []byte) {
		lenBuf := make([]byte, 4)
		binary.LittleEndian.PutUint32(lenBuf, uint32(len(body)))
		world.Write(append(lenBuf, body...))
	}
	writePrefixed(encodeInner(WorldIdent, 0x100, IdentRegister, nil))
	// 等注册生效
	deadline := time.Now().Add(2 * time.Second)
	for !inner.WorldConnected() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !inner.WorldConnected() {
		t.Fatal("world 注册超时")
	}

	// 网关
	gw := NewGateway("127.0.0.1:0", 1, clients, inner, fsnet.Options{})
	addr, err := gw.Listen()
	if err != nil {
		t.Fatal(err)
	}
	defer gw.Stop()
	e = New(Options{ServerID: 11, EntryID: 1}, clients, inner, gw)
	e.SetState(true) // world 就绪：开放受理

	// 客户端接入
	cli, err := fsnet.Dial(addr.String(), fsnet.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()

	// 等 accept 通知到达 world 侧
	worldRead := make(chan []byte, 8)
	go func() {
		hdr := make([]byte, 4)
		for {
			if _, err := io_ReadFull(world, hdr); err != nil {
				return
			}
			n := binary.LittleEndian.Uint32(hdr)
			body := make([]byte, n)
			if _, err := io_ReadFull(world, body); err != nil {
				return
			}
			worldRead <- body
		}
	}()
	var acceptMsg []byte
	select {
	case m := <-worldRead:
		h, _, err := decodeInner(m)
		if err != nil {
			t.Fatal(err)
		}
		if h.Type != UpstreamAccept {
			t.Fatalf("首帧应为 accept 通知, got type=%d", h.Type)
		}
		acceptMsg = m[12:]
	case <-time.After(2 * time.Second):
		t.Fatal("accept 通知超时")
	}
	ident := binary.LittleEndian.Uint32(acceptMsg[0:])
	serial := binary.LittleEndian.Uint32(acceptMsg[4:])
	if ident == 0 || serial == 0 {
		t.Fatalf("ident/serial 无效: %#x/%d", ident, serial)
	}

	// 客户端发帧（op 0x02）→ world 应收到 type 3 转发
	if err := cli.WriteMsg([]byte{0x02, 0x11, 0x22}); err != nil {
		t.Fatal(err)
	}
	select {
	case m := <-worldRead:
		h, payload, _ := decodeInner(m)
		if h.Type != UpstreamData {
			t.Fatalf("应收到数据转发, got type=%d", h.Type)
		}
		if binary.LittleEndian.Uint32(payload[0:]) != ident || binary.LittleEndian.Uint32(payload[4:]) != serial {
			t.Fatal("转发头 ident/serial 错位")
		}
		if payload[8] != 0x02 {
			t.Fatal("帧体缺失")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("转发超时")
	}

	// world 反向：set_online + send_to_client → 客户端应收到
	mk := func(fields ...uint32) []byte {
		b := make([]byte, len(fields)*4)
		for i, v := range fields {
			binary.LittleEndian.PutUint32(b[i*4:], v)
		}
		return b
	}
	// member=1 绑定
	writePrefixed(encodeInner(0, 0x100, ForwSetOnline, mk(ident, serial, 1, 0x77)))
	payload := append(mk(ident, serial), 0x0B, 0x99)
	writePrefixed(encodeInner(0, 0x100, ForwSendClient, payload))

	got, err := cli.ReadMsg()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != 0x0B || got[1] != 0x99 {
		t.Fatalf("下行帧不符: %x", got)
	}
}

// io_ReadFull 隔离 io 依赖便于阅读。
func io_ReadFull(c net.Conn, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := c.Read(buf[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}
