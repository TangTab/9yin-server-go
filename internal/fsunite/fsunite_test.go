package fsunite

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/fssoloentry"
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
ConnectString=Provider=SQLOLEDB;Password=pw;User ID=sa;Initial Catalog=nx_district1;Data Source=192.168.10.148:3306
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

	logs := &logBuffer{}
	quitCalled := make(chan struct{})
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
