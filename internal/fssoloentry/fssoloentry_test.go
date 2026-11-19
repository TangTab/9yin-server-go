package fssoloentry

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	gonet "net"

	"github.com/lizhanyin/9yin-server-go/internal/entry"
	fsnet "github.com/lizhanyin/9yin-server-go/internal/net"
)

// ---- license ----

func TestLicenseInternalRoundTrip(t *testing.T) {
	lic := GenerateLicense("127.0.0.1")
	if len(lic) != LicenseInternalSize {
		t.Fatalf("内部形态应 256B, got %d", len(lic))
	}
	got, err := DecodeLicense(lic, "127.0.0.1", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !got.Licensed() {
		t.Fatalf("魔数对应命中 %#X/%#X", got.Fields[0], got.Fields[1])
	}
	if !got.Internal {
		t.Fatal("应为内部形态")
	}
	if got.T0 != 0 || got.T1 != 0 {
		t.Fatal("内部形态无时间窗")
	}
}

func TestLicenseInternalAddressGate(t *testing.T) {
	lic := GenerateLicense("127.0.0.1")
	if _, err := DecodeLicense(lic, "10.1.2.3", time.Now()); err == nil ||
		!strings.Contains(err.Error(), "not internal address") {
		t.Fatalf("非内网地址应被拒: %v", err)
	}
	// 192.168.* 放行（lic 与 IP 绑定——用对应 IP 生成后解码）
	lic2 := GenerateLicense("192.168.33.70")
	if _, err := DecodeLicense(lic2, "192.168.33.70", time.Now()); err != nil {
		t.Fatalf("192.168.* 应放行: %v", err)
	}
	// 绑定错位：127.0.0.1 的 lic 用 192.168 密钥解 → 校验和失配
	if _, err := DecodeLicense(lic, "192.168.33.70", time.Now()); err == nil ||
		!strings.Contains(err.Error(), "checksum") {
		t.Fatalf("IP 绑定应导致校验和失配: %v", err)
	}
}

func TestLicenseExternalWindow(t *testing.T) {
	now := uint32(time.Now().Unix())
	lic := GenerateExternalLicense("203.0.113.7", now-100, now+100)
	if len(lic) != LicenseExternalSize {
		t.Fatalf("外部形态应 272B, got %d", len(lic))
	}
	got, err := DecodeLicense(lic, "203.0.113.7", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !got.Licensed() {
		t.Fatal("魔数对应命中")
	}
	if got.T0 != now-100 || got.T1 != now+100 {
		t.Fatalf("时间窗 %d..%d", got.T0, got.T1)
	}
	// too young / too old
	if _, err := DecodeLicense(lic, "203.0.113.7", time.Unix(int64(now)+1000, 0)); err == nil ||
		!strings.Contains(err.Error(), "too old") {
		t.Fatalf("过期应拒: %v", err)
	}
	if _, err := DecodeLicense(lic, "203.0.113.7", time.Unix(int64(now)-1000, 0)); err == nil ||
		!strings.Contains(err.Error(), "too young") {
		t.Fatalf("未生效应拒: %v", err)
	}
}

func TestLicenseWrongIPFailsChecksum(t *testing.T) {
	lic := GenerateLicense("127.0.0.1")
	if _, err := DecodeLicense(lic, "192.168.0.1", time.Now()); err == nil {
		t.Fatal("错 IP 密钥派生不同，应校验和失配")
	}
}

// ---- config ----

const officialINI = `[System]
Sleep=-1
MainEntity=SoloEntry
MainScript=main

[Module]
fskit.dll
fscomm.dll
fssoloentry.dll

[Script]
Path=..\lua\soloentry
Reload=1

[Main]
IDC=1
DistrictID=1
ServerID=11
EntryID=0
MaxConnects=1024

[Outer]
Addr=192.168.33.70
Port=2001

[Inner]
;Addr=127.0.0.1
;Port=0

[Broadcast]
;Addr=127.255.255.255
;Port=4001
`

func TestParseSoloINI(t *testing.T) {
	cfg, err := ParseSoloINI([]byte(officialINI))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ServerID != 11 || cfg.EntryID != 0 || cfg.MaxConnects != 1024 {
		t.Fatalf("Main 段: %+v", cfg)
	}
	if cfg.OuterAddr != "192.168.33.70" || cfg.OuterPort != 2001 {
		t.Fatalf("Outer 段: %+v", cfg)
	}
	if cfg.InnerPort != 0 {
		t.Fatalf("solo InnerPort 缺省应 0（动态分配），got %d", cfg.InnerPort)
	}
	if cfg.InnerAddr != "127.0.0.1" {
		t.Fatalf("InnerAddr 缺省 127.0.0.1, got %s", cfg.InnerAddr)
	}
	if cfg.BroadAddr != "127.255.255.255" || cfg.BroadPort != 15000 {
		t.Fatalf("Broadcast 缺省: %+v", cfg)
	}
	if cfg.EchoPort != 15200 {
		t.Fatalf("EchoPort 缺省 15200+EntryID, got %d", cfg.EchoPort)
	}
	if cfg.License != "192.168.33.70.lic" {
		t.Fatalf("License 缺省 <OuterAddr>.lic, got %s", cfg.License)
	}
}

func TestParseSoloINIEchoOffset(t *testing.T) {
	cfg, err := ParseSoloINI([]byte("[Main]\nEntryID=3\nServerID=9\n"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.EchoPort != 15203 {
		t.Fatalf("EchoPort 应 15200+3, got %d", cfg.EchoPort)
	}
}

// ---- 状态机 + ProcessInnerMsg ----

type downStub struct {
	sends   [][3]any // ident, serial, payload
	closed  [][2]uint32
	shakes  [][2]uint32
	all, online, channel int
	channelPayloads []byte
	setClosed []bool
}

func (d *downStub) ClientSend(ident, serial uint32, payload []byte) error {
	d.sends = append(d.sends, [3]any{ident, serial, payload})
	return nil
}
func (d *downStub) ClientClose(ident, serial uint32) { d.closed = append(d.closed, [2]uint32{ident, serial}) }
func (d *downStub) Shake(ident, serial uint32)       { d.shakes = append(d.shakes, [2]uint32{ident, serial}) }
func (d *downStub) SendAll(p []byte)                 { d.all++ }
func (d *downStub) SendOnline(p []byte)              { d.online++ }
func (d *downStub) SendChannel(ch uint32, p []byte)  { d.channel++; d.channelPayloads = append(d.channelPayloads, p...) }
func (d *downStub) SendSight(ident uint32, sceneID uint32, x, y, radius int32, p []byte) {}
func (d *downStub) SetClosed(c bool)                  { d.setClosed = append(d.setClosed, c) }

func newTestServer(t *testing.T) (*Server, *entry.Clients, *downStub) {
	t.Helper()
	clients := entry.NewClients()
	down := &downStub{}
	lic := &License{Internal: true, Fields: [8]uint32{OfficialMagic1, OfficialMagic2}}
	s := New(Options{
		ServerID: 11, EntryID: 7, MaxConnects: 1024,
		InnerAddr: "127.0.0.1", InnerPort: 16500,
		OuterAddr: "127.0.0.1", License: lic,
		ManualTick: true,
	}, clients, nil, down)
	s.Start() // ManualTick：不开循环
	return s, clients, down
}

func mkident(slot uint32) uint32  { return 7<<16 | slot }
func mkpayload(n int) []byte      { return bytes.Repeat([]byte{0xAB}, n) }

func TestSoloSourceCheck(t *testing.T) {
	for _, ok := range []uint32{0, 1, 2, 0x18, 0x27, 0x2F} {
		if !SoloSourceCheck(ok) {
			t.Fatalf("%#x 应放行", ok)
		}
	}
	for _, bad := range []uint32{3, 0x17, 0x30, 0xFFFF} {
		if SoloSourceCheck(bad) {
			t.Fatalf("%#x 应拒绝", bad)
		}
	}
}

func TestProcessInnerMsgSourceError(t *testing.T) {
	s, _, _ := newTestServer(t)
	err := s.ProcessInnerMsg(3, SoloForwInitConnect, mkpayload(8))
	if err == nil || !strings.Contains(err.Error(), "message source error") {
		t.Fatalf("非法源应拒绝: %v", err)
	}
}

func TestProcessInnerMsgAll15(t *testing.T) {
	s, clients, down := newTestServer(t)

	// 0 init_connect：开 + 窗内不动身份域
	if err := s.ProcessInnerMsg(0, SoloForwInitConnect, []byte{1, 0, 0, 0, 0, 0, 0, 0}); err != nil {
		t.Fatal(err)
	}
	if s.State() != entry.StateOpened {
		t.Fatalf("flag=1 应 Opened, got %v", s.State())
	}
	if s.Fields()[0] != OfficialMagic1 {
		t.Fatal("窗内不应清身份域")
	}
	// =0 → Paused
	if err := s.ProcessInnerMsg(0, SoloForwInitConnect, []byte{0, 0, 0, 0, 0, 0, 0, 0}); err != nil {
		t.Fatal(err)
	}
	if s.State() != entry.StatePaused {
		t.Fatalf("flag=0 应 Paused, got %v", s.State())
	}
	// 断言串：长度失配
	if err := s.ProcessInnerMsg(0, SoloForwInitConnect, mkpayload(4)); err == nil {
		t.Fatal("init_connect ==8 断言应触发")
	}

	// 预置一个客户端（槽 0，serial 1）
	idx := clients.Alloc(7, "1.2.3.4:5")
	_ = idx
	st := clients.Client(0)
	if st.Ident != mkident(0) {
		t.Fatalf("ident 应 7<<16|0, got %#x", st.Ident)
	}

	// 1 break_client
	if err := s.ProcessInnerMsg(0, SoloForwBreakClient, []byte{0, 0, 0, 0, 0, 0, 0, 0}); err != nil {
		// 传 (0,0) 找不到客户端，仅断言不炸
		_ = err
	}
	// 2 reset_client_inuse ==8
	if err := s.ProcessInnerMsg(0, SoloForwResetInuse, []byte{0, 0, 0, 0, 0, 0, 0, 0}); err == nil {
		_ = err
	}
	// 3 set_client_online ==0x10 {ident,serial,member,scene}
	online := make([]byte, 0x10)
	binary.LittleEndian.PutUint32(online[0:], mkident(0))
	binary.LittleEndian.PutUint32(online[4:], st.Serial)
	binary.LittleEndian.PutUint32(online[8:], 2) // member 2
	binary.LittleEndian.PutUint32(online[12:], 202) // scene 202
	if err := s.ProcessInnerMsg(0, SoloForwSetOnline, online); err != nil {
		t.Fatal(err)
	}
	// 5 send_to_scene {scene}+payload → 只发 scene 202 的客户端
	if err := s.ProcessInnerMsg(0, SoloForwSendScene, append([]byte{202, 0, 0, 0}, "hello"...)); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, v := range down.sends {
		if v[2].([]byte) != nil && string(v[2].([]byte)) == "hello" {
			found = true
		}
	}
	if !found {
		t.Fatal("SceneSend 应命中 scene=202 客户端")
	}
	// 非本场景不命中
	before := len(down.sends)
	_ = s.ProcessInnerMsg(0, SoloForwSendScene, append([]byte{203, 0, 0, 0}, "x"...))
	if len(down.sends) != before {
		t.Fatal("scene 203 不应命中")
	}

	// 4 send_to_multi {n}+n×{ident,serial}+payload（前缀过滤）
	multi := make([]byte, 0, 4+16+3)
	multi = binary.LittleEndian.AppendUint32(multi, 2)
	multi = binary.LittleEndian.AppendUint32(multi, mkident(0))
	multi = binary.LittleEndian.AppendUint32(multi, st.Serial)
	multi = binary.LittleEndian.AppendUint32(multi, 9<<16|1) // 别组前缀，应被滤掉
	multi = binary.LittleEndian.AppendUint32(multi, 1)
	multi = append(multi, 'm', 'u', 'l')
	before = len(down.sends)
	if err := s.ProcessInnerMsg(0, SoloForwSendMulti, multi); err != nil {
		t.Fatal(err)
	}
	if len(down.sends) != before+1 {
		t.Fatalf("MultiSend 应只命中本组 1 个, %d", len(down.sends)-before)
	}
	// n 越界断言
	bad := make([]byte, 4)
	binary.LittleEndian.PutUint32(bad[0:], 0)
	if err := s.ProcessInnerMsg(0, SoloForwSendMulti, bad); err == nil {
		t.Fatal("count=0 应触发断言")
	}

	// 6 send_to_client
	before = len(down.sends)
	sc := append(make([]byte, 8), 'o', 'k')
	binary.LittleEndian.PutUint32(sc[0:], mkident(0))
	binary.LittleEndian.PutUint32(sc[4:], st.Serial)
	_ = s.ProcessInnerMsg(0, SoloForwSendClient, sc)
	if len(down.sends) != before+1 {
		t.Fatal("send_to_client 应命中")
	}

	// 7 online_send / 8 broadcast
	_ = s.ProcessInnerMsg(0, SoloForwOnlineSend, []byte("on"))
	_ = s.ProcessInnerMsg(0, SoloForwBroadcast, []byte("all"))
	if down.online != 1 || down.all != 1 {
		t.Fatalf("online=%d all=%d", down.online, down.all)
	}

	// 9 shake_client
	_ = s.ProcessInnerMsg(0, SoloForwShakeClient, []byte{0, 0, 0, 0, 0, 0, 0, 0})
	if len(down.shakes) != 1 {
		t.Fatal("shake 应转发")
	}

	// 10..13 频道四件套
	ca := make([]byte, 0xC)
	binary.LittleEndian.PutUint32(ca[0:], mkident(0))
	binary.LittleEndian.PutUint32(ca[4:], st.Serial)
	binary.LittleEndian.PutUint32(ca[8:], 5)
	if err := s.ProcessInnerMsg(0, SoloForwChannelAdd, ca); err != nil {
		t.Fatal(err)
	}
	_ = s.ProcessInnerMsg(0, SoloForwSendChannel, append([]byte{5, 0, 0, 0}, 'c'))
	_ = s.ProcessInnerMsg(0, SoloForwChannelRm, ca)
	_ = s.ProcessInnerMsg(0, SoloForwClearChan, []byte{5, 0, 0, 0})
	if down.channel != 1 {
		t.Fatalf("channel send 次数 %d", down.channel)
	}

	// 14 close → Closed
	if err := s.ProcessInnerMsg(0, SoloForwClose, nil); err != nil {
		t.Fatal(err)
	}
	if s.State() != entry.StateClosed {
		t.Fatalf("close 应 Closed, got %v", s.State())
	}

	// 未知型
	if err := s.ProcessInnerMsg(0, 15, nil); err == nil {
		t.Fatal("未知型应报错")
	}
}

func TestWorldWindowWipesLicense(t *testing.T) {
	// 外部形态 license 带时间窗 [100, 200]
	clients := entry.NewClients()
	down := &downStub{}
	lic := &License{Fields: [8]uint32{OfficialMagic1, OfficialMagic2}, T0: 100, T1: 200}
	s := New(Options{ServerID: 1, EntryID: 1, License: lic, ManualTick: true}, clients, nil, down)
	s.Start()

	// 世界窗值在窗内 → 保留
	inner := make([]byte, 8)
	binary.LittleEndian.PutUint32(inner[0:], 1)
	binary.LittleEndian.PutUint32(inner[4:], 150)
	_ = s.ProcessInnerMsg(0, SoloForwInitConnect, inner)
	if s.Fields()[0] != OfficialMagic1 {
		t.Fatal("窗内不应清零")
	}
	// 越上界 → 清零
	binary.LittleEndian.PutUint32(inner[4:], 300)
	_ = s.ProcessInnerMsg(0, SoloForwInitConnect, inner)
	if s.Fields()[0] != 0 || s.Fields()[1] != 0 {
		t.Fatal("世界窗越界应清零身份域（撤销授权）")
	}
}

// ---- TimerCheck / 广播 ----

func TestTimerCheckLicenseGatesAndCadence(t *testing.T) {
	clients := entry.NewClients()
	var statuses []string
	var broadcasts int
	lic := &License{Internal: true, Fields: [8]uint32{OfficialMagic1, OfficialMagic2}}
	s := New(Options{
		ServerID: 11, EntryID: 0, License: lic, ManualTick: true,
		StatusSink:    func(line string) { statuses = append(statuses, line) },
		BroadcastSink: func([]byte) { broadcasts++ },
	}, clients, nil, nil)
	s.Start()
	for i := 0; i < 6; i++ {
		s.TimerCheck()
	}
	if len(statuses) != 1 {
		t.Fatalf("6 拍应产 1 条状态行, got %d", len(statuses))
	}
	if !strings.Contains(statuses[0], `state="unknown"`) || !strings.Contains(statuses[0], `tick="6"`) {
		t.Fatalf("状态行 %q", statuses[0])
	}
	if broadcasts != 6 {
		t.Fatalf("授权命中应每拍广播, got %d", broadcasts)
	}

	// 未授权：不广播 + 告警一次
	clients2 := entry.NewClients()
	var warns []string
	s2 := New(Options{ServerID: 11, ManualTick: true,
		BroadcastSink: func([]byte) { broadcasts++ },
		Logf: func(f string, a ...any) { warns = append(warns, fmt.Sprintf(f, a...)) },
	}, clients2, nil, nil)
	s2.Start()
	b0 := broadcasts
	s2.TimerCheck()
	s2.TimerCheck()
	if broadcasts != b0 {
		t.Fatal("未授权不应广播")
	}
	if len(warns) != 1 || !strings.Contains(warns[0], "not licensed server program") {
		t.Fatalf("告警一次, got %v", warns)
	}
}

func TestBroadcastInfoGolden(t *testing.T) {
	clients := entry.NewClients()
	lic := &License{Internal: true, Fields: [8]uint32{OfficialMagic1, OfficialMagic2}}
	s := New(Options{
		IDC: 1, DistrictID: 2, ServerID: 11, EntryID: 3, MaxConnects: 1024,
		InnerAddr: "127.0.0.1", InnerPort: 16507, OuterAddr: "192.168.33.70",
		License: lic, ManualTick: true,
	}, clients, nil, nil)
	pkt := s.BroadcastInfo()
	if len(pkt) != 0x48 {
		t.Fatalf("广播包 0x48B, got %d", len(pkt))
	}
	if binary.LittleEndian.Uint32(pkt[0:]) != 1 || binary.LittleEndian.Uint32(pkt[4:]) != 0x3001 {
		t.Fatal("头两域 ver=1/msg=0x3001")
	}
	if string(pkt[8:17]) != "127.0.0.1" {
		t.Fatalf("inner_addr 域 %q", pkt[8:24])
	}
	if binary.LittleEndian.Uint32(pkt[0x18:]) != 16507 {
		t.Fatal("inner_port 域")
	}
	if binary.LittleEndian.Uint32(pkt[0x1c:]) != 2 || binary.LittleEndian.Uint32(pkt[0x20:]) != 11 ||
		binary.LittleEndian.Uint32(pkt[0x24:]) != 3 || binary.LittleEndian.Uint32(pkt[0x28:]) != 1 {
		t.Fatal("district/server/entry/idc 域")
	}
	if string(pkt[0x2c:0x2c+13]) != "192.168.33.70" {
		t.Fatalf("outer_addr 域 %q", pkt[0x2c:0x3c])
	}
	if binary.LittleEndian.Uint32(pkt[0x3c:]) != 1024 {
		t.Fatal("max_connects 域")
	}
	if binary.LittleEndian.Uint32(pkt[0x40:]) != 0 {
		t.Fatal("connect_count 域")
	}
}

// ---- echo ----

func TestEchoCommands(t *testing.T) {
	blockIP := entry.NewBlockIPList()
	shutdownCalled := false
	welcome, onInput := EchoHandler(11, 0, blockIP,
		func() string { return `state="opened"` },
		func() { shutdownCalled = true })
	if welcome != "welcome server 11 entry 0\r" {
		t.Fatalf("welcome %q", welcome)
	}
	if out := onInput("loadip\r"); len(out) < 2 || !strings.Contains(out[0], "reload block ip list...") {
		t.Fatalf("loadip %v", out)
	}
	if out := onInput("whatever"); !strings.Contains(out[0], "undefined command: whatever") {
		t.Fatalf("undefined %v", out)
	}
	if out := onInput("state"); !strings.Contains(out[0], `state="opened"`) {
		t.Fatalf("state %v", out)
	}
	out := onInput("quit")
	if !strings.Contains(out[0], "server will shutdown...") {
		t.Fatalf("quit %v", out)
	}
	time.Sleep(50 * time.Millisecond) // go shutdown()
	if !shutdownCalled {
		t.Fatal("quit 应触发 shutdown")
	}
}

func TestLoadIPFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/ip.txt"
	if err := writeFile(path, "10.0.0.1,10.0.0.5\n192.168.1.1\t192.168.1.9\nbadline\n8.8.8.8 8.8.8.8\n"); err != nil {
		t.Fatal(err)
	}
	blockIP := entry.NewBlockIPList()
	logs := LoadIPFile(path, blockIP)
	if len(logs) != 3 {
		t.Fatalf("应装载 3 条, got %v", logs)
	}
	if !blockIP.Blocked("10.0.0.3") || !blockIP.Blocked("192.168.1.5") || !blockIP.Blocked("8.8.8.8") {
		t.Fatal("三种分隔符都应生效")
	}
	if blockIP.Blocked("10.0.0.9") {
		t.Fatal("区间外不应拦截")
	}
	if LoadIPFile("no-such.txt", blockIP) != nil {
		t.Fatal("缺文件应静默")
	}
}

// ---- 真 TCP e2e：solo 内网语义 + 网关链 ----

func TestSoloEndToEnd(t *testing.T) {
	clients := entry.NewClients()
	var svc *Server
	inner := entry.NewInner("127.0.0.1", 0, SoloOwnIdent(1), entry.InnerHandler{
		OnMessage: func(src, typ uint32, p []byte) {
			if svc != nil {
				_ = svc.ProcessInnerMsg(src, typ, p)
			}
		},
	})
	inner.SourceCheck = SoloSourceCheck
	inner.KickDuplicate = true
	inner.MemberBase = SoloMemberBase
	inner.MemberCount = SoloMemberCount
	if err := inner.Start(); err != nil {
		t.Fatal(err)
	}
	defer inner.Stop()
	innerAddr := inner.Addr().(*gonet.TCPAddr)
	if innerAddr.Port == 0 {
		t.Fatal("InnerPort=0 应动态分配非零端口")
	}

	// 假 world 连入并注册 ident 0
	world, err := gonet.Dial("tcp", innerAddr.String())
	if err != nil {
		t.Fatal(err)
	}
	defer world.Close()
	writePrefixed := func(body []byte) {
		buf := make([]byte, 4, 4+len(body))
		binary.LittleEndian.PutUint32(buf, uint32(len(body)))
		world.Write(append(buf, body...))
	}
	innerWrite := func(typ uint32, payload []byte) {
		hdr := make([]byte, 12, 12+len(payload))
		binary.LittleEndian.PutUint32(hdr[0:], 0)
		binary.LittleEndian.PutUint32(hdr[8:], typ)
		writePrefixed(append(hdr, payload...))
	}
	innerWrite(0xFFFF, nil) // 注册 ident 0
	deadline := time.Now().Add(2 * time.Second)
	for !inner.WorldConnected() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !inner.WorldConnected() {
		t.Fatal("world 注册超时")
	}

	// 网关 + 服务
	gw := entry.NewGateway("127.0.0.1:0", 1, clients, inner, fsnet.Options{})
	addr, err := gw.Listen()
	if err != nil {
		t.Fatal(err)
	}
	defer gw.Stop()
	lic := &License{Internal: true, Fields: [8]uint32{OfficialMagic1, OfficialMagic2}}
	svc = New(Options{ServerID: 11, EntryID: 1, License: lic, ManualTick: true}, clients, inner, gw)
	svc.Start()

	// init_connect 开放
	open := make([]byte, 8)
	binary.LittleEndian.PutUint32(open[0:], 1)
	innerWrite(0, open)
	time.Sleep(50 * time.Millisecond)
	if svc.State() != entry.StateOpened {
		t.Fatalf("init_connect 后应 Opened, got %v", svc.State())
	}

	// 客户端接入 → world 收 accept 通知
	worldRead := make(chan []byte, 8)
	go func() {
		hdr := make([]byte, 4)
		for {
			if _, err := io.ReadFull(world, hdr); err != nil {
				return
			}
			n := binary.LittleEndian.Uint32(hdr)
			body := make([]byte, n)
			if _, err := io.ReadFull(world, body); err != nil {
				return
			}
			worldRead <- body
		}
	}()
	cli, err := fsnet.Dial(addr.String(), fsnet.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()
	select {
	case m := <-worldRead:
		if binary.LittleEndian.Uint32(m[8:]) != entry.UpstreamAccept {
			t.Fatalf("应为 accept 通知 type=%d", binary.LittleEndian.Uint32(m[8:]))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("accept 通知超时")
	}

	// world：set_client_online + send_to_client → 客户端收帧
	st := clients.Client(0)
	online := make([]byte, 0x10)
	binary.LittleEndian.PutUint32(online[0:], st.Ident)
	binary.LittleEndian.PutUint32(online[4:], st.Serial)
	binary.LittleEndian.PutUint32(online[8:], 0) // member 0（发往 member 路径前置）
	binary.LittleEndian.PutUint32(online[12:], 0)
	innerWrite(3, online)
	sc := append(make([]byte, 8), 0x0B, 0xF6)
	binary.LittleEndian.PutUint32(sc[0:], st.Ident)
	binary.LittleEndian.PutUint32(sc[4:], st.Serial)
	innerWrite(6, sc)
	frameCh := make(chan []byte, 1)
	go func() {
		msg, err := cli.ReadMsg()
		if err == nil {
			frameCh <- msg
		}
	}()
	var frame []byte
	select {
	case frame = <-frameCh:
	case <-time.After(2 * time.Second):
		t.Fatal("下行帧超时")
	}
	if len(frame) < 2 || frame[0] != 0x0B || frame[1] != 0xF6 {
		t.Fatalf("客户端应收到 0B F6, got % X", frame)
	}

	// member ident 0x28（solo 24 位空间）合法直发
	innerWrite2(t, world, 0x28, 7, []byte{}) // src=0x28 type=7 → online_send
	svc.Stop()
}

// innerWrite2 直发指定 src 的内网帧
func innerWrite2(t *testing.T, c gonet.Conn, src, typ uint32, payload []byte) {
	t.Helper()
	hdr := make([]byte, 12, 12+len(payload))
	binary.LittleEndian.PutUint32(hdr[0:], src)
	binary.LittleEndian.PutUint32(hdr[8:], typ)
	buf := make([]byte, 4, 4+12+len(payload))
	binary.LittleEndian.PutUint32(buf, uint32(12+len(payload)))
	buf = append(append(buf, hdr...), payload...)
	if _, err := c.Write(buf); err != nil {
		t.Fatal(err)
	}
}

func TestInnerKickDuplicate(t *testing.T) {
	inner := entry.NewInner("127.0.0.1", 0, SoloOwnIdent(0), entry.InnerHandler{})
	inner.KickDuplicate = true
	inner.SourceCheck = SoloSourceCheck
	if err := inner.Start(); err != nil {
		t.Fatal(err)
	}
	defer inner.Stop()
	innerAddr := inner.Addr().(*gonet.TCPAddr)

	dial := func() gonet.Conn {
		c, err := gonet.Dial("tcp", innerAddr.String())
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	readOne := func(c gonet.Conn) []byte {
		hdr := make([]byte, 4)
		if _, err := io.ReadFull(c, hdr); err != nil {
			return nil
		}
		body := make([]byte, binary.LittleEndian.Uint32(hdr))
		if _, err := io.ReadFull(c, body); err != nil {
			return nil
		}
		return body
	}

	c1 := dial()
	defer c1.Close()
	reg := func(c gonet.Conn) {
		hdr := make([]byte, 12)
		binary.LittleEndian.PutUint32(hdr[0:], 0x18)
		binary.LittleEndian.PutUint32(hdr[8:], 0xFFFF)
		buf := make([]byte, 4, 16)
		binary.LittleEndian.PutUint32(buf, 12)
		c.Write(append(buf, hdr...))
	}
	reg(c1)
	time.Sleep(100 * time.Millisecond)

	c2 := dial()
	defer c2.Close()
	reg(c2)
	got := readOne(c2)
	if got == nil || binary.LittleEndian.Uint32(got[8:]) != entry.QuitFrame {
		t.Fatalf("重复 ident 应回 0xFFFE 帧, got % X", got)
	}
	if !inner.WorldConnected() && inner.ConnectedCount() < 1 {
		t.Fatal("旧连接应保留")
	}
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
