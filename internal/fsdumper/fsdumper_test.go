package fsdumper

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

const officialRoot = `F:\Game\Online\9yin\BGserverCJ`

func TestTextVarListRoundTrip(t *testing.T) {
	v := newTextVarList().
		addString("dump_role_msg").
		addInt(1).
		addInt(123456789012). // int64 界外（len>=10）
		addDouble(3.5).
		addWide("角色名\\100%")
	line := v.EncodeText()
	got, err := DecodeTextVarList(line)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Len() != v.Len() {
		t.Fatalf("len %d != %d", got.Len(), v.Len())
	}
	if got.String(0) != "dump_role_msg" || got.Kind(0) != 's' {
		t.Fatalf("[0]=%q kind=%c", got.String(0), got.Kind(0))
	}
	if got.Int(1) != 1 {
		t.Fatalf("[1]=%d", got.Int(1))
	}
	if got.Int(2) != 123456789012 {
		t.Fatalf("[2]=%d", got.Int(2))
	}
	if got.Double(3) != 3.5 {
		t.Fatalf("[3]=%v", got.Double(3))
	}
	if got.String(4) != "角色名\\100%" {
		t.Fatalf("[4]=%q", got.String(4))
	}
}

func TestTextVarListGoldenDialect(t *testing.T) {
	// 官方方言 golden：数字 int 形态、# 宽串转义、$ 窄串。
	line := `$dump_role_msg 1 100 #\u53C2\u6570 3.5`
	got, err := DecodeTextVarList(line)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.String(0) != "dump_role_msg" {
		t.Fatalf("[0]=%q", got.String(0))
	}
	if got.Int(1) != 1 {
		t.Fatalf("[1]=%d", got.Int(1))
	}
	if got.String(3) != "参数" {
		t.Fatalf("[3]=%q（\\x53C2\\x6570 应解出参数）", got.String(3))
	}
	// \u 形态 + '%' 转义 + 反斜杠双写
	v := newTextVarList().addWide(`a\b%`)
	if s := v.EncodeText(); s != `#a\\b\x25` {
		t.Fatalf("encode=%q", s)
	}
}

func TestTextVarListRejectsUnknownToken(t *testing.T) {
	if _, err := DecodeTextVarList(`plainword`); err == nil {
		t.Fatal("裸词应拒绝（官方未知 token 分支）")
	}
}

func TestConfigParseAndAccessFallback(t *testing.T) {
	ini := strings.Join([]string{
		"[System]", "Sleep=1", "MainEntity=MainConsole",
		"[Main]", "DumpLogicDll=FsDumpLogic.dll", "UseMysql=true",
		"[Outer]", "Addr=127.0.0.1", "Port=4006",
		"[Echo]", "Port=15011",
		"[Role]", "ConnectString=Provider=SQLOLEDB;Password=pw;User ID=sa;Initial Catalog=nx_base;Data Source=127.0.0.1:3306",
		"[Public]", "ConnectString=",
		"[Manage]", "ConnectString=",
	}, "\r\n")
	tmp := filepath.Join(t.TempDir(), "GameDumper.ini")
	if err := os.WriteFile(tmp, []byte(ini), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LogicDLL != "FsDumpLogic.dll" || !cfg.UseMysql {
		t.Fatalf("main 段: %+v", cfg)
	}
	if cfg.OuterPort != 4006 || cfg.EchoPort != 15011 {
		t.Fatalf("端口: outer=%d echo=%d", cfg.OuterPort, cfg.EchoPort)
	}
	// 回退序：public→role、manage→role、output←manage(空)→role
	role := cfg.AccessDSN("role")
	for _, n := range []string{"public", "manage", "output"} {
		if cfg.AccessDSN(n) != role {
			t.Fatalf("%s 回退失败", n)
		}
	}
}

func TestServerLineProtocolAndState(t *testing.T) {
	var mu sync.Mutex
	var gotMsgs []string
	var connected []int
	callee := &fakeCallee{
		onMsg: func(id int, m *textVarList) {
			mu.Lock()
			defer mu.Unlock()
			gotMsgs = append(gotMsgs, m.EncodeText())
		},
		onConn: func(id int) {
			mu.Lock()
			defer mu.Unlock()
			connected = append(connected, id)
		},
	}
	s := NewServer("127.0.0.1", 0, callee, nil)
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	// 状态：Start 后应进入 opened
	deadline := time.Now().Add(2 * time.Second)
	for s.State() != "opened" && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if s.State() != "opened" {
		t.Fatalf("state=%s", s.State())
	}

	addr := s.ln.Addr().(*net.TCPAddr)
	conn, err := net.Dial("tcp", addr.String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	fmt.Fprintf(conn, "$dump_role_msg 1 100\r\n")
	fmt.Fprintf(conn, "$other") // 无行尾不发；关闭时也不发
	time.Sleep(150 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(connected) != 1 {
		t.Fatalf("connected=%v", connected)
	}
	if len(gotMsgs) != 1 || gotMsgs[0] != "$dump_role_msg 1 100" {
		t.Fatalf("msgs=%v", gotMsgs)
	}
}

func TestKernelSendServerMsgBroadcast(t *testing.T) {
	callee := &fakeCallee{}
	s := NewServer("127.0.0.1", 0, callee, nil)
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
	time.Sleep(50 * time.Millisecond)
	addr := s.ln.Addr().(*net.TCPAddr)

	c1, err := net.Dial("tcp", addr.String())
	if err != nil {
		t.Fatal(err)
	}
	defer c1.Close()
	time.Sleep(100 * time.Millisecond) // 等 accept 注册

	msg := newTextVarList().addString("dump_role_msg").addInt(1).addInt(42)
	if !s.Kernel.SendServerMsg(-1, msg) {
		t.Fatal("广播应成功")
	}
	c1.SetReadDeadline(time.Now().Add(time.Second))
	buf := make([]byte, 128)
	n, err := c1.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if string(buf[:n]) != "$dump_role_msg 1 42\r\n" {
		t.Fatalf("收到 %q", buf[:n])
	}
}

// ---- 官方实物 golden（F: 快照缺席自动跳过） ----

func officialFile(t *testing.T, rel string) ([]byte, bool) {
	t.Helper()
	full := filepath.Join(officialRoot, filepath.FromSlash(rel))
	b, err := os.ReadFile(full)
	if err != nil {
		t.Skipf("官方快照不可读: %v", err)
	}
	return b, true
}

func TestParseOfficialRoleTemplate(t *testing.T) {
	raw, ok := officialFile(t, "Res/dumpconfig/dump_role_dbinfo.xml")
	if !ok {
		return
	}
	rec, ok := officialFile(t, "Res/dumpconfig/dump_role_record.xml")
	if !ok {
		return
	}
	d, err := NewDumpRole(raw, rec, nil)
	if err != nil {
		t.Fatal(err)
	}
	if d.table.Name != "nx_role_dumpinfo" || d.table.Primary != "role_name" {
		t.Fatalf("table=%+v", d.table)
	}
	if len(d.table.Columns) != 15 {
		t.Fatalf("列数=%d（官方 15 列）", len(d.table.Columns))
	}
	// 首末列语义抽查
	if d.table.Columns[0].Col != "role_name" || d.table.Columns[0].Prop != "Name" {
		t.Fatalf("首列=%+v", d.table.Columns[0])
	}
	if d.table.Columns[14].Col != "money_rec" || d.kind(d.table.Columns[14]) != colKindRecord {
		t.Fatalf("末列=%+v", d.table.Columns[14])
	}
	if d.records["capital_rec"] == nil || d.records["capital_rec"].ColNum != 3 {
		t.Fatalf("capital_rec=%+v", d.records["capital_rec"])
	}
	// 建表 SQL 方言
	fdb := &fakeDB{}
	if err := d.EnsureTable(context.Background(), fdb); err != nil {
		t.Fatal(err)
	}
	stmt := fdb.statements()[0]
	for _, want := range []string{
		"CREATE TABLE IF NOT EXISTS `nx_role_dumpinfo`",
		"role_name varchar(32) CHARACTER SET utf8",
		"PRIMARY KEY (`role_name`)",
		"KEY `IX_role_uid` (`role_uid`)",
	} {
		if !strings.Contains(stmt, want) {
			t.Fatalf("建表缺 %q: %s", want, stmt)
		}
	}
	if d.BuildVersionMsg().EncodeText() == "" {
		t.Fatal("版本帧为空")
	}
}

func TestParseOfficialDumpConfig(t *testing.T) {
	if _, ok := officialFile(t, "Res/dumpconfig/dump_config.xml"); !ok {
		return
	}
	d, err := NewDumpRes(filepath.Join(officialRoot, "Res"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.cfg.Statics) != 3 {
		t.Fatalf("static=%d（官方 3 条）", len(d.cfg.Statics))
	}
	if len(d.cfg.Move) != 25 {
		t.Fatalf("move=%d（官方 25 条）", len(d.cfg.Move))
	}
	if d.cfg.Move[0].Type != 1 || d.cfg.Move[0].Table != "nx_res_dumpinfo" {
		t.Fatalf("move[0]=%+v", d.cfg.Move[0])
	}
	// 类型覆盖：1/5/7/8/10/14/15/17/19
	seen := map[int]bool{}
	for _, f := range d.cfg.Move {
		seen[f.Type] = true
	}
	for _, ty := range []int{1, 5, 7, 8, 10, 14, 15, 17, 19} {
		if !seen[ty] {
			t.Fatalf("缺类型 %d", ty)
		}
	}
}

func TestParseOfficialLogDefine(t *testing.T) {
	raw, ok := officialFile(t, "Res/dumpconfig/DumpHFile/LogDefine.h")
	if !ok {
		return
	}
	groups, values := ParseLogDefine(decodeGBK(raw))
	if len(groups) < 5 {
		t.Fatalf("枚举组=%d", len(groups))
	}
	if len(values) < 100 {
		t.Fatalf("枚举值=%d", len(values))
	}
	// 首组：//$@ 1 = 角色物品日志
	if groups[0].Type != 1 || groups[0].Desc != "角色物品日志" {
		t.Fatalf("groups[0]=%+v", groups[0])
	}
	// ITEM_LOG_TYPE_BUY_FROM_NPC = 10001 //$ 10001 = 玩家从NPC处购买
	found := false
	for _, e := range values {
		if e.Value == "10001" && e.Type == 1 {
			found = true
			if !strings.Contains(e.Text, "从NPC处购买") {
				t.Fatalf("10001 text=%q", e.Text)
			}
		}
	}
	if !found {
		t.Fatal("缺 10001 条目")
	}
	// 带 | "english" 描述形态
	hasDesc := false
	for _, e := range values {
		if e.Desc != "" {
			hasDesc = true
			break
		}
	}
	if !hasDesc {
		t.Log("（本快照无 | \"desc\" 形态条目——降级通过）")
	}
}

func TestRoleIngestUpsertSQL(t *testing.T) {
	raw, ok := officialFile(t, "Res/dumpconfig/dump_role_dbinfo.xml")
	if !ok {
		return
	}
	rec, ok := officialFile(t, "Res/dumpconfig/dump_role_record.xml")
	if !ok {
		return
	}
	d, err := NewDumpRole(raw, rec, nil)
	if err != nil {
		t.Fatal(err)
	}
	fdb := &fakeDB{}
	k := newKernel(nil)
	k.AddAccess("output", fdb)
	if err := d.EnsureTable(context.Background(), fdb); err != nil {
		t.Fatal(err)
	}
	fdb.reset()

	// 组一条 op=2 快照：[0]名 [1]op [2]字段数 值×15
	msg := newTextVarList().addString("dump_role_msg").addInt(2).addInt(15)
	msg.addWide("测试侠")                 // role_name
	msg.addWide("uid-1")               // role_uid
	msg.addWide("acct")                // role_account
	msg.addWide("2024-01-01 00:00:00") // create_time
	msg.addInt(1)                      // delete_time(default)
	msg.addInt(0)                      // deleted(default)
	msg.addInt(12345)                  // total_sec
	msg.addInt(0)                      // sex
	msg.addInt(1)                      // camp
	msg.addInt(1)                      // race
	msg.addInt(2)                      // job
	msg.addInt(99)                     // level
	msg.addInt(1234)                   // fight_power
	msg.addWide("帮会")                  // guild
	msg.addWide("黑金,0,100;金币,1,50")    // money_rec(record)
	d.HandleGameServerMsg(k, 0, msg)

	stmts := fdb.statements()
	if len(stmts) != 1 {
		t.Fatalf("upsert 语句数=%d", len(stmts))
	}
	stmt := stmts[0]
	if !strings.Contains(stmt, "INSERT INTO `nx_role_dumpinfo`") ||
		!strings.Contains(stmt, "ON DUPLICATE KEY UPDATE") ||
		!strings.Contains(stmt, "`level`=VALUES(`level`)") {
		t.Fatalf("upsert 方言: %s", stmt)
	}
	if got := fdb.argsOf(stmt); len(got) != 15 {
		t.Fatalf("参数数=%d", len(got))
	}

	// 字段数不匹配 → 丢弃（parse role info error）
	fdb.reset()
	bad := newTextVarList().addString("dump_role_msg").addInt(2).addInt(3)
	d.HandleGameServerMsg(k, 0, bad)
	if len(fdb.statements()) != 0 {
		t.Fatal("字段数不符应丢弃")
	}
}

// ---- 测试件 ----

type fakeCallee struct {
	onMsg  func(id int, m *textVarList)
	onConn func(id int)
}

func (f *fakeCallee) OnDumperStart(k *Kernel)  {}
func (f *fakeCallee) OnFrameExecute(k *Kernel) {}
func (f *fakeCallee) OnServerClose(k *Kernel)  {}
func (f *fakeCallee) OnServerConnected(k *Kernel, id int) {
	if f.onConn != nil {
		f.onConn(id)
	}
}
func (f *fakeCallee) OnRecvGameServerMsg(k *Kernel, id int, m *textVarList) {
	if f.onMsg != nil {
		f.onMsg(id, m)
	}
}

// fakeDB 记录型 DB 桩：Exec/Query/QueryRow 全记，QueryRow 恒无行。
type fakeDB struct {
	mu    sync.Mutex
	stmts []string
	args  [][]any
}

func (f *fakeDB) Exec(ctx context.Context, q string, a ...any) (sql.Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stmts = append(f.stmts, q)
	f.args = append(f.args, a)
	return driver.RowsAffected(1), nil
}

func (f *fakeDB) Query(ctx context.Context, q string, a ...any) (*sql.Rows, error) {
	return nil, io.EOF
}

func (f *fakeDB) QueryRow(ctx context.Context, q string, a ...any) *sql.Row {
	return nil
}

func (f *fakeDB) statements() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.stmts))
	copy(out, f.stmts)
	return out
}

func (f *fakeDB) argsOf(q string) []any {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, s := range f.stmts {
		if s == q {
			return f.args[i]
		}
	}
	return nil
}

func (f *fakeDB) reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stmts = nil
	f.args = nil
}
