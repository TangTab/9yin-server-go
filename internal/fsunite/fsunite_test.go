package fsunite

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf16"

	"github.com/lizhanyin/9yin-server-go/internal/auth"
	"github.com/lizhanyin/9yin-server-go/internal/fssoloentry"
	"github.com/lizhanyin/9yin-server-go/internal/fssolostore"
	"github.com/lizhanyin/9yin-server-go/internal/fssoloworld"
	fsnet "github.com/lizhanyin/9yin-server-go/internal/net"
)

// testINI 覆盖 game_unite.ini 全方言（含五库节/[Logic]/[Outer]）。
const testINI = `[System]
Sleep=-1
MainEntity=SoloWorld
MainScript=main

[Module]
fskit.dll
fssql.dll
fscomm.dll
fscollide.dll
fssololog.dll
fssoloentry.dll
fssolostore.dll
fssolomember.dll
fssoloworld.dll
fssolopublic.dll
fssoloauction.dll

[Script]
Path=..\lua\solounite
Reload=1

[Resource]
Path=..\res

[Main]
DistrictID=1
ServerID=11
ServerName=server_11
RoleNameSizeLimit=18
PublicLogicDll=fspublogic.dll
AuctionLogicDll=fsauctionlogic.dll
CollideRootPath=..\Res\map
UseMysql=true
;LogDBPerformance=true
;MessageEncodeKey=A5375CAE
;MessageVerifyDll=fsverify.dll

[Inner]
;Addr=127.0.0.1
;Port=5000

[Broadcast]
;Addr=127.255.255.255
;Port=4001

[Outer]
Addr=127.0.0.1
Port=2001

[Logic]
Path=
Modules=2
Module0=fsgame.dll
Module1=SnailEditor.dll

[Charge]
Inuse=false
Addr=127.0.0.1
Port=6000

[ManageServer]
Inuse=true
Addr=127.0.0.1
Port=4002

[Monitor]
Inuse=false
Addr=127.0.0.1
Port=4008

[Account]
ConnectString=Provider=SQLOLEDB;Password=pw;User ID=sa;Initial Catalog=nx_district1;Data Source=127.0.0.1:3306
ConnectPool=3

[Role]
ConnectString=Provider=SQLOLEDB;Initial Catalog=nx_base1
ConnectPool=4

[Log]
ConnectString=Provider=SQLOLEDB;Initial Catalog=nx_base1
ConnectPool=5

[Public]
ConnectString=Provider=SQLOLEDB;Initial Catalog=nx_base1
ConnectPool=6

[Manage]
ConnectString=Provider=SQLOLEDB;Initial Catalog=nx_base1
ConnectPool=7
`

func TestParseGameUniteINI(t *testing.T) {
	cfg, err := ParseGameUniteINI([]byte(testINI))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Modules) != 11 {
		t.Fatalf("modules = %d (%v)", len(cfg.Modules), cfg.Modules)
	}
	if cfg.DistrictID != 1 || cfg.ServerID != 11 || cfg.ServerName != "server_11" {
		t.Fatalf("main: %+v", cfg)
	}
	if cfg.PublicLogicDll != "fspublogic.dll" || cfg.AuctionLogicDll != "fsauctionlogic.dll" {
		t.Fatalf("logic dll: %q %q", cfg.PublicLogicDll, cfg.AuctionLogicDll)
	}
	if !cfg.UseMysql || cfg.LogDBPerformance {
		t.Fatalf("mysql flags: %+v", cfg)
	}
	if cfg.CollideRootPath != `..\Res\map` {
		t.Fatalf("collide root: %q", cfg.CollideRootPath)
	}
	if cfg.RoleNameSizeLimit != 18 {
		t.Fatalf("rolenamelimit: %d", cfg.RoleNameSizeLimit)
	}
	// 注释键不生效（官方 ini 注释态）
	if cfg.MessageEncodeKey != "" || cfg.MessageVerifyDll != "" {
		t.Fatalf("commented keys leaked: %q %q", cfg.MessageEncodeKey, cfg.MessageVerifyDll)
	}
	// 缺省（注释节回退 main.lua 字面值）
	if cfg.InnerAddr != "127.0.0.1" || cfg.InnerPort != 0 {
		t.Fatalf("inner: %s:%d", cfg.InnerAddr, cfg.InnerPort)
	}
	if cfg.BroadAddr != "127.255.255.255" || cfg.BroadPort != 15000 {
		t.Fatalf("broadcast: %s:%d", cfg.BroadAddr, cfg.BroadPort)
	}
	if cfg.EchoAddr != "127.0.0.1" || cfg.EchoPort != 15001 {
		t.Fatalf("echo: %s:%d", cfg.EchoAddr, cfg.EchoPort)
	}
	if cfg.OuterAddr != "127.0.0.1" || cfg.OuterPort != 2001 {
		t.Fatalf("outer: %s:%d", cfg.OuterAddr, cfg.OuterPort)
	}
	if !cfg.ManageInuse || cfg.ManagePort != 4002 || cfg.ChargeInuse || cfg.MonitorInuse {
		t.Fatalf("switches: %+v", cfg)
	}
	if cfg.LogicPath != "" || len(cfg.LogicModules) != 2 ||
		cfg.LogicModules[0] != "fsgame.dll" || cfg.LogicModules[1] != "SnailEditor.dll" {
		t.Fatalf("logic: %q %v", cfg.LogicPath, cfg.LogicModules)
	}
	if cfg.AccountPools != 3 || cfg.RolePools != 4 || cfg.LogPools != 5 ||
		cfg.PublicPools != 6 || cfg.ManagePools != 7 {
		t.Fatalf("pools: %+v", cfg)
	}
	if !strings.Contains(cfg.AccountBase, "nx_district1") || !strings.Contains(cfg.RoleBase, "nx_base1") {
		t.Fatalf("bases: %q %q", cfg.AccountBase, cfg.RoleBase)
	}

	// 投影
	if got := cfg.AppName(); got != "unite_11" {
		t.Fatalf("appname: %s", got)
	}
	if got := cfg.LicensePath(); got != "127.0.0.1.lic" {
		t.Fatalf("license: %s", got)
	}
	wc := cfg.WorldConfig()
	if wc.ServerID != 11 || wc.EchoPort != 15001 || wc.BroadPort != 15000 ||
		wc.MaxPlayers != 8192 || wc.AccountRoles != 5 || wc.RoleNameLimit != 18 ||
		wc.InnerPort != 0 || wc.ManageInuse != true {
		t.Fatalf("world: %+v", wc)
	}
	sc := cfg.StoreConfig()
	if sc.AppName != "store_11_0" || sc.StoreID != 0 || !sc.UseMysql ||
		sc.AccountPools != 3 || sc.WakePort != 0 || sc.InnerPort != 0 {
		t.Fatalf("store: %+v", sc)
	}
	mc := cfg.MemberConfig()
	if mc.LogicModules == nil || len(mc.LogicModules) != 2 || mc.MemberID != 0 ||
		mc.DebugHalt != true || mc.CollideRootPath != `..\Res\map` {
		t.Fatalf("member: %+v", mc)
	}
	ao := cfg.AuctionOptions()
	if ao.AppName != "auction_11" || ao.EchoPort != 16001 || ao.InnerPort != 0 {
		t.Fatalf("auction: %+v", ao)
	}
	lo := cfg.LoggerOptions()
	if lo.EveryDayAlone || lo.AutoExit || lo.LogFileDir != `log\` {
		t.Fatalf("logger: %+v", lo)
	}
}

// logBuffer 线程安全日志收集。
type logBuffer struct {
	mu  sync.Mutex
	buf strings.Builder
}

func (l *logBuffer) logf(format string, args ...any) {
	l.mu.Lock()
	fmt.Fprintf(&l.buf, format+"\n", args...)
	l.mu.Unlock()
}

func (l *logBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.String()
}

func freeTCPPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func freeUDPPort(t *testing.T) int {
	t.Helper()
	pc, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero})
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close()
	return pc.LocalAddr().(*net.UDPAddr).Port
}

// TestUniteAssembly 七服务同进程装配冒烟：共享广播口互发现 + echo
// 命令集（status/perf）+ 官方停机序。License 注入内部形态（entry 才
// 会广播 0x3001）。
func TestUniteAssembly(t *testing.T) {
	cfg, err := ParseGameUniteINI([]byte(testINI))
	if err != nil {
		t.Fatal(err)
	}
	// 测试口位：广播/echo/outer 及各实体 ctor 缺省 echo 口全部换随机
	// 空闲口（官方 15000/15001/2001/15003/15100/15200/16001）——避免与
	// 并行包测试进程（fssolopublic 等）撞固定口。
	cfg.BroadPort = freeUDPPort(t)
	cfg.EchoPort = freeTCPPort(t)
	cfg.OuterPort = freeTCPPort(t)
	cfg.testPublicEcho = freeTCPPort(t)
	cfg.testStoreEcho = freeTCPPort(t)
	cfg.testEntryEcho = freeTCPPort(t)
	cfg.testAuctionEcho = freeTCPPort(t)
	cfg.testAuctionInner = freeTCPPort(t)

	logs := &logBuffer{}
	quitCalled := make(chan struct{})
	cfg.testLogMutex = fmt.Sprintf("mutex_fs_solo_log_test_%d", time.Now().UnixNano())
	u := Assemble(cfg, Options{
		License: &fssoloentry.License{
			Internal: true,
			Fields:   [8]uint32{fssoloentry.OfficialMagic1, fssoloentry.OfficialMagic2},
		},
		WorkDir: t.TempDir(),
		Logf:    logs.logf,
		Quit:    func() { close(quitCalled) },
	})
	if err := u.Start(); err != nil {
		t.Fatalf("start: %v\nlogs:\n%s", err, logs)
	}
	defer u.Stop()

	// 装载日志行（官方 "loigc" 拼写保留）
	for _, want := range []string{
		"load loigc module fsgame.dll",
		"load loigc module SnailEditor.dll",
		"load loigc module fspublogic.dll",
		"load loigc module fsauctionlogic.dll",
	} {
		if !strings.Contains(logs.String(), want) {
			t.Fatalf("缺少装载日志 %q\nlogs:\n%s", want, logs)
		}
	}

	// 横幅序（main.lua 423-455）
	banner := strings.Join(u.Banner(), "\n")
	for _, want := range []string{
		"app name unite_11",
		"echo socket 127.0.0.1:",
		"broadcast socket 127.255.255.255:",
		"outer socket 127.0.0.1:",
		"manage socket 127.0.0.1:4002", // ManageInuse=true 才出现
		"game server 11 starting...",
	} {
		if !strings.Contains(banner, want) {
			t.Fatalf("横幅缺 %q: %s", want, banner)
		}
	}
	if strings.Contains(banner, "charge socket") {
		t.Fatalf("ChargeInuse=false 不应有 charge 行: %s", banner)
	}

	// 广播互发现端到端：member 收 entry 0x3001 广播后外连（fssolomember
	// inner 外连日志）+ entry 收到 member（comm 0x18）identify 注册——
	// UDP 共绑广播 → TCP 回环内连 → identify 握手全链成立
	deadline := time.Now().Add(15 * time.Second)
	for !strings.Contains(logs.String(), "（外连，已发 identify）") ||
		!strings.Contains(logs.String(), "unite entry: ident 0x18 注册") {
		if time.Now().After(deadline) {
			t.Fatalf("15s 内未见跨服务外连/注册（广播发现未成立）\nlogs:\n%s", logs)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// 发现就绪：world 预期对端（store+member+entry+public）全部 identify
	// 应答后监督器打点（此后快播退出、回落 1s 心跳自宣）
	deadline = time.Now().Add(10 * time.Second)
	for !strings.Contains(logs.String(), "(SoloWorld::DiscoveryWatch)发现就绪") {
		if time.Now().After(deadline) {
			t.Fatalf("world 未在窗口内发现就绪（pending=%v）\nlogs:\n%s",
				u.World().DiscoveryPending(), logs)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// world echo 命令集（unite 方言：status/perf/players）
	testEchoCommands(t, cfg.EchoAddr, cfg.EchoPort)

	// 官方停机序断言（force_close 927-952）
	u.Stop()
	out := logs.String()
	order := []string{
		"store server shutdown...",
		"auction server shutdown...",
		"public server shutdown...",
		"entry server shutdown...",
		"member server shutdown...",
		"world server shutdown...",
	}
	pos := 0
	for _, want := range order {
		i := strings.Index(out[pos:], want)
		if i < 0 {
			t.Fatalf("停机序缺 %q\nlogs:\n%s", want, out)
		}
		pos += i + len(want)
	}
}

// testEchoCommands 对 world echo 口逐命令对拍。
func testEchoCommands(t *testing.T, addr string, port int) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", addr, port), 2*time.Second)
	if err != nil {
		t.Fatalf("echo dial: %v", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))

	rd := bufio.NewReader(conn)
	welcome, err := rd.ReadString('\r')
	if err != nil {
		t.Fatalf("welcome: %v", err)
	}
	if !strings.Contains(welcome, "welcome server 11") {
		t.Fatalf("welcome = %q", welcome)
	}

	for _, tc := range []struct{ cmd, want string }{
		{"status", ">game server state is "},
		{"players", ">player: 0, online: 0, offline: 0, active: 0, queuing: 0"},
		{"perf", ">dump logic module performance..."},
		{"end", ""},
	} {
		if _, err := conn.Write([]byte(tc.cmd + "\r\n")); err != nil {
			t.Fatalf("send %s: %v", tc.cmd, err)
		}
		if tc.want == "" {
			break // end 关连接（应答行均以 \r 结尾、无 \n）
		}
		line, err := rd.ReadString('\r')
		if err != nil {
			t.Fatalf("read %s: %v", tc.cmd, err)
		}
		if !strings.Contains(line, tc.want) {
			t.Fatalf("%s 应答 %q 不含 %q", tc.cmd, line, tc.want)
		}
	}
}

// TestUniteClientLoginRoleFlow 真装配「登录后选角页面数据下发」全链：
// outer 口走真传输面（EE+XOR(InitialKey)，fsnet.Dial 客户端形态）——
// op02 → 0x04 无角 → op03 → 0x05 五书 → op05 建角 → 0x04 有角 →
// op04 选角 → 进场帧簇（0x09 表/0x0A/m366 族/0x0B/0x0D/0x1F/物理流）→
// op09 ClientReady → View43/40 + 就绪簇。world↔store 同进程真 inner 互连。
func TestUniteClientLoginRoleFlow(t *testing.T) {
	cfg, err := ParseGameUniteINI([]byte(testINI))
	if err != nil {
		t.Fatal(err)
	}
	cfg.BroadPort = freeUDPPort(t)
	cfg.EchoPort = freeTCPPort(t)
	cfg.OuterPort = freeTCPPort(t)
	cfg.testPublicEcho = freeTCPPort(t)
	cfg.testStoreEcho = freeTCPPort(t)
	cfg.testEntryEcho = freeTCPPort(t)
	cfg.testAuctionEcho = freeTCPPort(t)
	cfg.testAuctionInner = freeTCPPort(t)
	logs := &logBuffer{}
	cfg.testLogMutex = fmt.Sprintf("mutex_fs_solo_log_test_%d", time.Now().UnixNano())
	u := Assemble(cfg, Options{
		License: &fssoloentry.License{
			Internal: true,
			Fields:   [8]uint32{fssoloentry.OfficialMagic1, fssoloentry.OfficialMagic2},
		},
		WorkDir: t.TempDir(),
		Scenes: []fssoloworld.SceneSpec{ // 场景装载链驱动 world Opened → entry 开闸
			{Index: 17, Member: 0, Script: "scene", Config: `ini\scene\born03_YanYuZhuang`},
		},
		Logf: logs.logf,
		Quit: func() {},
	})
	if err := u.Start(); err != nil {
		t.Fatalf("start: %v\nlogs:\n%s", err, logs)
	}
	defer u.Stop()

	// world↔store/entry inner 互连 + 场景装载 → Opened（entry 的 Closed
	// 闸随开服通知解除——未 Opened 前客户端连接被静默关闭）
	w := u.World()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) && !(w.Inner().GetConnected(4) &&
		w.Inner().GetConnected(8) && w.State() == "Opened") {
		time.Sleep(50 * time.Millisecond)
	}
	if w.State() != "Opened" {
		t.Fatalf("world not opened (%s) logs: %s", w.State(), logs)
	}
	time.Sleep(500 * time.Millisecond) // 开服通知 → entry 开闸（异步）

	conn, err := fsnet.Dial(fmt.Sprintf("127.0.0.1:%d", cfg.OuterPort), fsnet.Options{})
	if err != nil {
		t.Fatalf("dial outer: %v", err)
	}
	defer conn.Close()
	readFrame := func(want byte) []byte {
		t.Helper()
		type res struct {
			b   []byte
			err error
		}
		ch := make(chan res, 1)
		go func() {
			b, err := conn.ReadMsg()
			ch <- res{b, err}
		}()
		select {
		case r := <-ch:
			if r.err != nil {
				t.Fatalf("read op %#x: %v logs: %s", want, r.err, logs.String())
			}
			if r.b[0] != want {
				t.Fatalf("read op %#x, got %#x (%d bytes)", want, r.b[0], len(r.b))
			}
			return r.b
		case <-time.After(8 * time.Second):
			t.Fatalf("read op %#x timeout\nlogs:\n%s", want, logs)
		}
		return nil
	}

	// op02（真机 DES 方言，auth 同源加密）
	acctCt, pwCt := auth.EncryptFields("acct", auth.ComputeLoginPassword("pw"))
	fld := func(slot []byte) []byte {
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, uint32(len(slot)))
		return append(b, slot...)
	}
	op2 := make([]byte, 25, 25+len(acctCt)+len(pwCt)+64)
	op2[0] = 0x02
	binary.LittleEndian.PutUint32(op2[1:], 333)
	binary.LittleEndian.PutUint32(op2[0x15:], uint32(len(acctCt)))
	op2 = append(op2, acctCt...)
	op2 = append(op2, fld(pwCt)...)
	op2 = append(op2, 0, 0, 0, 0)
	op2 = append(op2, fld(make([]byte, 16))...)
	op2 = append(op2, 0, 0, 0, 0)
	if err := conn.WriteMsg(op2); err != nil {
		t.Fatalf("write op02: %v", err)
	}
	// 登录后选角页面数据：0x04 无角基座（41B golden）
	if f := readFrame(0x04); len(f) != 0x29 || binary.LittleEndian.Uint32(f[0x25:]) != 0 {
		t.Fatalf("no-role 0x04 len=%d", len(f))
	}
	// op03 → 0x05 五书（UTF-16LE 文本）
	if err := conn.WriteMsg([]byte{0x03}); err != nil {
		t.Fatalf("write op03: %v", err)
	}
	if f := readFrame(0x05); len(f) < 10 {
		t.Fatalf("world info len=%d", len(f))
	}

	// op05 建角（真机方言：4B 保留 + 裸 UTF-16 名 + 0x48 起 tag6 外观）
	body := make([]byte, 0x48)
	for i, u16 := range utf16.Encode([]rune("王五")) {
		binary.LittleEndian.PutUint16(body[4+i*2:], u16)
	}
	for _, s := range []string{"book1", `icon\players\girl01_create.png`} {
		body = append(body, 6)
		body = binary.LittleEndian.AppendUint32(body, uint32(len(s)+1))
		body = append(body, s...)
		body = append(body, 0)
	}
	op5 := append([]byte{0x05}, body...)
	if err := conn.WriteMsg(op5); err != nil {
		t.Fatalf("write op05: %v", err)
	}
	if f := readFrame(0x04); binary.LittleEndian.Uint32(f[0x25:]) != 1 {
		t.Fatalf("one-role 0x04 count=%d", binary.LittleEndian.Uint32(f[0x25:]))
	}

	// op04 选角（UCS2 名字）→ 进场帧簇
	ch4 := make([]byte, 0x4B)
	for i, u16 := range utf16.Encode([]rune("王五")) {
		binary.LittleEndian.PutUint16(ch4[i*2:], u16)
	}
	if err := conn.WriteMsg(append([]byte{0x04}, ch4...)); err != nil {
		t.Fatalf("write op04: %v", err)
	}
	readAny := func() []byte {
		t.Helper()
		type res struct {
			b   []byte
			err error
		}
		ch := make(chan res, 1)
		go func() {
			b, err := conn.ReadMsg()
			ch <- res{b, err}
		}()
		select {
		case r := <-ch:
			if r.err != nil {
				t.Fatalf("read any: %v logs: %s", r.err, logs)
			}
			return r.b
		case <-time.After(8 * time.Second):
			t.Fatalf("read any timeout logs: %s", logs)
		}
		return nil
	}
	for i := 0; i < 2; i++ { // 0x32 域配置块（表前）
		readFrame(0x32)
	}
	readFrame(0x09) // 属性表（大帧过真传输面）
	readFrame(0x0A) // 记录表镜像
	if f := readFrame(0x0B); binary.LittleEndian.Uint16(f[13:]) != 5 {
		t.Fatalf("golden entry attrs=%d", binary.LittleEndian.Uint16(f[13:]))
	}
	for i := 1; i < 272; i++ { // 外服进场段其余帧
		readAny()
	}
	if err := conn.WriteMsg([]byte{0x09}); err != nil { // ClientReady
		t.Fatalf("write op09: %v", err)
	}
	for i := 0; i < 85; i++ { // 就绪段回放
		readAny()
	}
	if !w.Names().FindName("王五") {
		t.Fatal("nameuid missing after full flow")
	}
}

// TestDiscoveryWatchStaggered 错峰发现监督器 e2e（集群模式启动场景）：
// world 先起、store 晚 1.5s——world 未就绪期间持续"等待发现"汇报
// （没收到预期结果就一直广播），store 起后 200ms 快播自宣被 world
// 收到 → world 外连 identify（应答）→ store 侧"发现就绪"打点。
func TestDiscoveryWatchStaggered(t *testing.T) {
	cfg, err := ParseGameUniteINI([]byte(testINI))
	if err != nil {
		t.Fatal(err)
	}
	broad := freeUDPPort(t)
	logs := &logBuffer{}

	wc := cfg.WorldConfig()
	wc.BroadAddr, wc.BroadPort = "127.255.255.255", broad
	wc.InnerAddr, wc.InnerPort = "127.0.0.1", 0
	wc.EchoAddr, wc.EchoPort = "127.0.0.1", 0
	w := fssoloworld.New(fssoloworld.Options{
		Config: wc, AppName: "unite_11", Logf: logs.logf,
		Scenes: []fssoloworld.SceneSpec{{Index: 1, Member: 0, Script: "scene", Config: "scene_1"}},
	})
	if err := w.Start(); err != nil {
		t.Fatalf("world start: %v", err)
	}
	defer w.Stop()

	// 未就绪窗口：world 持续汇报缺口（快播进行中）
	deadline := time.Now().Add(3 * time.Second)
	for !strings.Contains(logs.String(), "(SoloWorld::DiscoveryWatch)等待发现") {
		if time.Now().After(deadline) {
			t.Fatalf("未就绪期间未见缺口汇报\nlogs:\n%s", logs)
		}
		time.Sleep(50 * time.Millisecond)
	}

	// store 晚起：world 快拍收 store 自宣 → 外连 identify → store 就绪
	time.Sleep(1500 * time.Millisecond)
	sc := cfg.StoreConfig()
	sc.BroadAddr, sc.BroadPort = "127.255.255.255", broad
	sc.InnerAddr, sc.InnerPort = "127.0.0.1", 0
	// EchoPort 0 会被 FillDefaults 回填缺省 15100（撞并行包），换随机口
	sc.EchoAddr, sc.EchoPort = "127.0.0.1", freeTCPPort(t)
	st := fssolostore.New(fssolostore.Options{
		Config: sc, Logf: logs.logf, Backend: fssolostore.NewMemoryStore(""),
	})
	if err := st.Start(); err != nil {
		t.Fatalf("store start: %v", err)
	}
	defer st.Stop()

	storeStart := time.Now()
	deadline = storeStart.Add(3 * time.Second)
	for !strings.Contains(logs.String(), "(SoloStore::DiscoveryWatch)发现就绪") {
		if time.Now().After(deadline) {
			t.Fatalf("store 起后 3s 内未发现就绪\nlogs:\n%s", logs)
		}
		time.Sleep(50 * time.Millisecond)
	}
	if el := time.Since(storeStart); el > 2*time.Second {
		t.Fatalf("快播收敛超 2s：%v", el)
	}
	// world 侧缺口不再含 store（identify 应答已入登记）
	for _, p := range w.DiscoveryPending() {
		if p == "store" {
			t.Fatalf("world 仍在等 store：pending=%v", w.DiscoveryPending())
		}
	}
}

// TestConfigMySQLDSN DSN 解析优先级（公共方法）：env NINEYIN_MYSQL_DSN
// > ini [Role] ConnectString（SQLOLEDB 方言转换）> ""（内存库）。
func TestConfigMySQLDSN(t *testing.T) {
	cfg, err := ParseGameUniteINI([]byte(testINI))
	if err != nil {
		t.Fatal(err)
	}
	// 快照 [Role] 为注释态（官方方言）→ ini 回落空
	if got := cfg.MySQLDSN(); got != "" {
		t.Fatalf("snapshot fallback should be empty, got %q", got)
	}
	// ini 回落（无 env 时，SQLOLEDB 方言转换）
	ini2 := "[Role]\r\nConnectString=Provider=SQLOLEDB;Password=pw2;User ID=u2;Initial Catalog=db2;Data Source=127.0.0.2:3307\r\n"
	cfg2, err := ParseGameUniteINI([]byte(ini2))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg2.MySQLDSN(); got != "u2:pw2@tcp(127.0.0.2:3307)/db2" {
		t.Fatalf("ini fallback got %q", got)
	}
	// env 优先（高于 ini）
	t.Setenv("NINEYIN_MYSQL_DSN", "env:pw@tcp(1.2.3.4:3306)/envdb")
	if got := cfg2.MySQLDSN(); got != "env:pw@tcp(1.2.3.4:3306)/envdb" {
		t.Fatalf("env priority got %q", got)
	}
}
