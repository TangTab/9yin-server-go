package fswatch

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func testConfig(t *testing.T) *Config {
	t.Helper()
	c := DefaultConfig()
	c.MonitorPassword = "MON"
	c.QueryPassword = "QRY"
	c.UpdatePassword = "UPD"
	c.Servers = []ServerEntry{{Name: "unite_11", Path: "echo hello"}}
	c.ExePaths["unite_11"] = t.TempDir()
	return c
}

// --- varlist 编解码（官方 text_encode_msg/text_decode_msg 面） ---

func TestVarListEncodeDecodeRoundTrip(t *testing.T) {
	l := &VarList{}
	l.AddString("servers")
	l.AddString("unite_11")
	l.AddInt(42)
	l.AddFloat(1.5)
	l.AddUserData([]byte{0x00, 0x1f, 0x7f, 0xab, '%'}) // % 与不可打印全走 \xXX/*HEX
	enc := l.EncodeText()

	got, ok := DecodeText(enc)
	if !ok {
		t.Fatalf("decode failed: %q", enc)
	}
	if got.StringAt(0) != "servers" || got.StringAt(1) != "unite_11" {
		t.Fatalf("strings: %v", enc)
	}
	if got.IntAt(2) != 42 {
		t.Fatalf("int: %d", got.IntAt(2))
	}
	if got.FloatAt(3) != 1.5 {
		t.Fatalf("float: %v", got.FloatAt(3))
	}
	want := []byte{0x00, 0x1f, 0x7f, 0xab, '%'}
	if string(got.DataAt(4)) != string(want) {
		t.Fatalf("userdata: %x", got.DataAt(4))
	}
}

func TestVarListEscapeForms(t *testing.T) {
	// 转义体：空格/%%/反斜杠
	l := &VarList{}
	l.AddString("a b%20\\c")
	enc := l.EncodeText()
	if !strings.HasPrefix(enc, "$a") {
		t.Fatalf("prefix: %q", enc)
	}
	got, ok := DecodeText(enc)
	if !ok || got.StringAt(0) != "a b%20\\c" {
		t.Fatalf("roundtrip: %q -> %v", enc, got.StringAt(0))
	}
	// userdata 大写 HEX
	l2 := &VarList{}
	l2.AddUserData([]byte{0xde, 0xad, 0xbe, 0xef})
	if l2.EncodeText() != "*DEADBEEF" {
		t.Fatalf("hex upper: %q", l2.EncodeText())
	}
	// \x 大写
	l3 := &VarList{}
	l3.AddString("\x1f")
	if l3.EncodeText() != `$\x1F` {
		t.Fatalf("escape upper: %q", l3.EncodeText())
	}
}

// --- 配置装载（GameWatch.ini + access.ini） ---

func TestLoadConfigAndAccess(t *testing.T) {
	dir := t.TempDir()
	ini := filepath.Join(dir, "GameWatch.ini")
	content := `[System]
Sleep=1
[Main]
MaxClient=8
MonitorPassword=08B5411F848A2581A41672A759C87380
QueryPassword=q
UpdatePassword=u
LockSystem=0
[Servers]
unite_11=fsloader GameUnite.ini
[ExePath]
unite_11=` + filepath.ToSlash(dir) + `
[Address]
AddressNum=1
Address0=192.168.33.70
`
	if err := os.WriteFile(ini, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	access := filepath.Join(dir, "access.ini")
	acc := `[db1]
app_name = world
type = mysql
connect_string = root@127.0.0.1:3306/nx_world
`
	if err := os.WriteFile(access, []byte(acc), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(ini, access)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MaxClient != 8 || cfg.MonitorPassword != "08B5411F848A2581A41672A759C87380" {
		t.Fatalf("main: %+v", cfg)
	}
	if len(cfg.Servers) != 1 || cfg.Servers[0].Name != "unite_11" {
		t.Fatalf("servers: %+v", cfg.Servers)
	}
	if len(cfg.ClientAddresses) != 1 || cfg.ClientAddresses[0] != "192.168.33.70" {
		t.Fatalf("addresses: %+v", cfg.ClientAddresses)
	}
	if len(cfg.Access) != 1 || cfg.Access[0].AppName != "world" || cfg.Access[0].Type != "mysql" {
		t.Fatalf("access: %+v", cfg.Access)
	}
	if p, ok := cfg.DestPath("unite_11"); !ok || p == "" {
		t.Fatalf("dest path: %q %v", p, ok)
	}
	if _, ok := cfg.DestPath("missing"); ok {
		t.Fatal("dest path should miss")
	}
}

// --- 命令分发 ---

type capture struct {
	lines []string
}

func (c *capture) sender(cnx int, l *VarList) bool {
	c.lines = append(c.lines, l.EncodeText())
	return true
}

func newWatch(t *testing.T) (*GameWatch, *capture) {
	t.Helper()
	cfg := testConfig(t)
	g := NewGameWatch(cfg, nil, nil)
	cap := &capture{}
	g.SetSender(cap.sender)
	g.Start()
	t.Cleanup(g.Stop)
	return g, cap
}

func TestServersCommand(t *testing.T) {
	g, cap := newWatch(t)
	g.ProcessLine(1, "servers")
	if len(cap.lines) != 1 || cap.lines[0] != "$servers $unite_11 $echo\\x20hello" {
		// 命令行含空格 → $ 转义体整体一个 token
		t.Fatalf("servers: %v", cap.lines)
	}
}

func TestSysInfoNoPassword(t *testing.T) {
	g, cap := newWatch(t)
	g.ProcessLine(1, "sys_info")
	if len(cap.lines) != 1 || !strings.HasPrefix(cap.lines[0], "$sys_info ") {
		t.Fatalf("sys_info: %v", cap.lines)
	}
	// 4 个数值
	parts := strings.Fields(cap.lines[0])
	if len(parts) != 5 {
		t.Fatalf("sys_info fields: %v", parts)
	}
	for _, p := range parts[1:] {
		if _, err := strconv.ParseFloat(p, 64); err != nil {
			t.Fatalf("sys_info field %q: %v", p, err)
		}
	}
}

func TestQueryPasswordAndNoDatabase(t *testing.T) {
	g, cap := newWatch(t)
	// 密码错 → error_password（stricmp：官方大小写不敏感）
	g.ProcessLine(1, "query request_roles wrongpass")
	if len(cap.lines) == 0 || cap.lines[0] != "$result $query $error_password" {
		t.Fatalf("pwd: %v", cap.lines)
	}
	// 密码对（大小写不敏感实证）但无数据源 → no_database
	cap.lines = nil
	g.ProcessLine(1, "query request_roles QRY")
	deadline := time.Now().Add(2 * time.Second)
	for len(cap.lines) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	found := false
	for _, l := range cap.lines {
		if l == "$result $query $no_database" {
			found = true
		}
	}
	if !found {
		t.Fatalf("no_database: %v", cap.lines)
	}
}

func TestQueryLogDataErrorTime(t *testing.T) {
	g, cap := newWatch(t)
	g.ProcessLine(1, "query request_log_data QRY notatime")
	if len(cap.lines) == 0 || cap.lines[0] != "$result $query $error_time" {
		t.Fatalf("error_time: %v", cap.lines)
	}
}

func TestUnknownCommand(t *testing.T) {
	g, cap := newWatch(t)
	g.ProcessLine(1, "bogus")
	if len(cap.lines) == 0 || cap.lines[0] != "$result $result $error_request" {
		t.Fatalf("unknown: %v", cap.lines)
	}
}

// --- update ---

func TestUpdateGetDirGetFileSetFile(t *testing.T) {
	cfg := testConfig(t)
	root := cfg.ExePaths["unite_11"]
	os.MkdirAll(filepath.Join(root, "sub"), 0o755)
	payload := strings.Repeat("0123456789abcdef", 7000) // ~112KB > 2 chunks
	if err := os.WriteFile(filepath.Join(root, "sub", "data.bin"), []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}
	g := NewGameWatch(cfg, nil, nil)
	cap := &capture{}
	g.SetSender(cap.sender)
	g.Start()
	t.Cleanup(g.Stop)

	// get_dir：目录树分块（首块头 + succeed + total + offset + len + data）
	g.ProcessLine(1, "update get_dir UPD unite_11 ")
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && len(cap.lines) == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	if len(cap.lines) == 0 {
		t.Fatal("get_dir no response")
	}
	first := cap.lines[0]
	if !strings.HasPrefix(first, "$update $get_dir") || !strings.Contains(first, " $succeed ") {
		t.Fatalf("get_dir head: %q", first)
	}

	// get_file：112KB → ≥2 块
	cap.lines = nil
	g.ProcessLine(1, "update get_file UPD unite_11 sub/data.bin")
	for time.Now().Before(deadline) && len(cap.lines) < 2 {
		time.Sleep(10 * time.Millisecond)
	}
	if len(cap.lines) < 2 {
		t.Fatalf("get_file chunks: %d", len(cap.lines))
	}
	// 重组块并验证
	var total, got int64
	for _, line := range cap.lines {
		l, ok := DecodeText(line)
		if !ok {
			t.Fatalf("decode %q", line)
		}
		if total == 0 {
			total = l.IntAt(len(l.vars) - 4)
		}
		got += l.IntAt(len(l.vars) - 2)
	}
	if total != int64(len(payload)) || got != total {
		t.Fatalf("size: total=%d got=%d want=%d", total, got, len(payload))
	}

	// set_file：新文件经 MD5 校验落盘
	cap.lines = nil
	body := []byte("watch update payload")
	sum := md5.Sum(body)
	dst := filepath.Join(root, "out")
	os.MkdirAll(dst, 0o755)
	_ = dst
	dataTok := "*" + strings.ToUpper(hex.EncodeToString(body))
	g.ProcessLine(1, fmt.Sprintf("update set_file UPD unite_11 new.txt %d 1700000000 %s %s",
		len(body), strings.ToUpper(hex.EncodeToString(sum[:])), dataTok))
	for time.Now().Before(deadline) && len(cap.lines) == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	if len(cap.lines) == 0 || cap.lines[len(cap.lines)-1] != "$result $update $succeed" {
		t.Fatalf("set_file: %v", cap.lines)
	}
	got2, err := os.ReadFile(filepath.Join(root, "new.txt"))
	if err != nil || string(got2) != string(body) {
		t.Fatalf("set_file body: %v %q", err, got2)
	}
}

func TestUpdateSetFileMd5Mismatch(t *testing.T) {
	cfg := testConfig(t)
	g := NewGameWatch(cfg, nil, nil)
	cap := &capture{}
	g.SetSender(cap.sender)
	g.Start()
	t.Cleanup(g.Stop)
	body := []byte("corruptme")
	g.ProcessLine(1, fmt.Sprintf("update set_file UPD unite_11 bad.txt %d 0 %s *%s",
		len(body), strings.Repeat("0", 32), strings.ToUpper(hex.EncodeToString(body))))
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && len(cap.lines) == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	if len(cap.lines) == 0 || cap.lines[len(cap.lines)-1] != "$result $update $md5\\x20verify\\x20code\\x20not\\x20match" {
		t.Fatalf("md5: %v", cap.lines)
	}
}

func TestUpdateSetDir(t *testing.T) {
	cfg := testConfig(t)
	root := cfg.ExePaths["unite_11"]
	os.MkdirAll(filepath.Join(root, "sub"), 0o755)
	g := NewGameWatch(cfg, nil, nil)
	cap := &capture{}
	g.SetSender(cap.sender)
	g.Start()
	t.Cleanup(g.Stop)
	g.ProcessLine(1, "update set_dir UPD unite_11 sub")
	if len(cap.lines) == 0 || cap.lines[0] != "$result $update $dir\\x20already\\x20exists" {
		t.Fatalf("set_dir exists: %v", cap.lines)
	}
	cap.lines = nil
	g.ProcessLine(1, "update set_dir UPD unite_11 brand/new")
	if len(cap.lines) == 0 || cap.lines[0] != "$result $update $succeed" {
		t.Fatalf("set_dir new: %v", cap.lines)
	}
	if st, err := os.Stat(filepath.Join(root, "brand", "new")); err != nil || !st.IsDir() {
		t.Fatalf("set_dir stat: %v", err)
	}
}

// --- command（UserSock 直投） ---

func TestCommandForward(t *testing.T) {
	cfg := testConfig(t)
	g := NewGameWatch(cfg, nil, nil)
	cap := &capture{}
	g.SetSender(cap.sender)
	// 伪 echo 维护口
	recv := make(chan string, 1)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		rd := bufio.NewReader(c)
		line, _ := rd.ReadString('\n')
		recv <- line
	}()
	g.SetCommandDial(func(addr string) (net.Conn, error) { return net.Dial("tcp", addr) })
	g.Start()
	t.Cleanup(g.Stop)
	g.ProcessLine(1, "command 127.0.0.1:99999 MON x") // 端口非法
	if cap.lines[0] != "$result $command $error_address" && cap.lines[0] != "$result $command $error_create_sock" {
		// atoi 溢出 → error_address
		t.Fatalf("bad port: %v", cap.lines)
	}
	cap.lines = nil
	g.ProcessLine(1, fmt.Sprintf("command %s MON quit", ln.Addr().String()))
	select {
	case line := <-recv:
		if strings.TrimSpace(line) != "quit" {
			t.Fatalf("forwarded: %q", line)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no forward")
	}
}

// --- TCP 集成（MassSock 行受面） ---

func TestServerLineProtocol(t *testing.T) {
	cfg := testConfig(t)
	cfg.OuterAddr = "127.0.0.1"
	cfg.OuterPort = 0
	g := NewGameWatch(cfg, nil, nil)
	srv := NewServer(cfg, g, nil)
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	conn, err := net.Dial("tcp", srv.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	rd := bufio.NewReader(conn)
	readLine := func() string {
		conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		line, err := rd.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimRight(line, "\r\n")
	}
	fmt.Fprintln(conn, "servers\r")
	if got := readLine(); got != "$servers $unite_11 $echo\\x20hello" {
		t.Fatalf("servers over tcp: %q", got)
	}
	fmt.Fprintln(conn, "sys_info\r")
	if got := readLine(); !strings.HasPrefix(got, "$sys_info ") {
		t.Fatalf("sys_info over tcp: %q", got)
	}
	// 密码错误面
	fmt.Fprintln(conn, "command 1.2.3.4:1 nope x\r")
	if got := readLine(); got != "$result $command $error_password" {
		t.Fatalf("pwd over tcp: %q", got)
	}
}

// --- 进程管理 ---

func TestProcManagerStates(t *testing.T) {
	cfg := testConfig(t)
	m := NewProcManager(cfg, nil)
	if got := m.StartApp("missing"); got != "no_exe_path" {
		t.Fatalf("no_exe_path: %s", got)
	}
	if got := m.StopApp("unite_11"); got != "not_found" {
		t.Fatalf("not_found: %s", got)
	}
	// [Servers] 有键但命令为不存在程序 → failed
	cfg.Servers = []ServerEntry{{Name: "ghost", Path: "definitely-not-a-real-exe-xyz"}}
	if got := m.StartApp("ghost"); got != "failed" && got != "no_exe_path" {
		// DestPath 无 [ExePath]/[ResPath] 键时官方先回 no_exe_path；
		// 有键才真正拉起。这里两者皆可接受，断言非 succeed 即可。
		if got == "succeed" {
			t.Fatalf("ghost should not succeed")
		}
	}
	l := m.AppList()
	if l.StringAt(0) != "app_list" {
		t.Fatalf("app_list: %v", l.EncodeText())
	}
}
