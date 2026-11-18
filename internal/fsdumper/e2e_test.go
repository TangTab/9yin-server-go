package fsdumper

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

// 全链路：TCP 行 → server → callee → DumpRole → fakeDB upsert。
func TestE2ELineToUpsert(t *testing.T) {
	dbinfo := `<?xml version="1.0" encoding="UTF-8"?>
<table name="nx_role_dumpinfo" primary="role_name" index="role_uid">
	<property desc="名" col="role_name" dbtype="nvarchar" type="prop" prop="Name" child="" createsql="role_name varchar(32) NOT NULL"/>
	<property desc="级" col="level" dbtype="int" type="prop" prop="Level" child="" createsql="level int NULL"/>
	<property desc="货币" col="money_rec" dbtype="varchar" type="record" prop="capital_rec" createsql="money_rec varchar(4096) NULL"/>
</table>`
	rec := `<?xml version="1.0" encoding="UTF-8"?>
<Object><record desc="货币表" rec_name="capital_rec" col_num="3" col_0="capital_name" col_1="capital_type" col_2="capital_number"/></Object>`
	role, err := NewDumpRole([]byte(dbinfo), []byte(rec), func(f string, a ...any) { t.Logf("role: "+f, a...) })
	if err != nil {
		t.Fatal(err)
	}
	fdb := &fakeDB{}
	s := NewServer("127.0.0.1", 0, NewDefaultCallee(role, nil), func(f string, a ...any) { t.Logf("srv: "+f, a...) })
	s.Kernel.AddAccess("output", fdb)
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", s.ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	br := make([]byte, 4096)
	n, _ := conn.Read(br)
	t.Logf("template: %s", br[:n])

	line := "$dump_role_msg 2 3 #\\u6D4B\\u8BD5 88 #\\u9ED1\\u91D1,0,100"
	fmt.Fprintf(conn, "%s\r\n", line)
	time.Sleep(300 * time.Millisecond)

	for i, st := range fdb.statements() {
		t.Logf("stmt[%d]: %s", i, st)
	}
	if len(fdb.statements()) != 2 { // CREATE + upsert
		t.Fatalf("语句数=%d（期望 2：建表+upsert）", len(fdb.statements()))
	}
	if !strings.Contains(fdb.statements()[1], "INSERT INTO `nx_role_dumpinfo`") {
		t.Fatalf("缺 upsert: %s", fdb.statements()[1])
	}
}

// 官方 15 列模板 + 同线路快照（对线上失灵问题的进程内复刻）。
func TestE2ELineToUpsertOfficial(t *testing.T) {
	dbinfo, ok := officialFile(t, "Res/dumpconfig/dump_role_dbinfo.xml")
	if !ok {
		return
	}
	rec, ok := officialFile(t, "Res/dumpconfig/dump_role_record.xml")
	if !ok {
		return
	}
	role, err := NewDumpRole(dbinfo, rec, func(f string, a ...any) { t.Logf("role: "+f, a...) })
	if err != nil {
		t.Fatal(err)
	}
	fdb := &fakeDB{}
	s := NewServer("127.0.0.1", 0, NewDefaultCallee(role, nil),
		func(f string, a ...any) { t.Logf("srv: "+f, a...) })
	s.Kernel.AddAccess("output", fdb)
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", s.ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	br := make([]byte, 8192)
	n, _ := conn.Read(br)
	t.Logf("template(%d): %s", n, br[:n])

	line := "$dump_role_msg 2 15 " +
		"#\u6D4B\u8BD5\u4FA0 " +
		"#uid-t1 #acct-t1 " +
		"#2026-11-19\\x2010:00:00 " +
		"1 0 43200 0 1 1 2 88 777 " +
		"#\u5E2E\u4F1A " +
		"#\u9ED1\u91D1,0,100;\u91D1\u5E01,1,50"
	fmt.Fprintf(conn, "%s\r\n", line)
	time.Sleep(300 * time.Millisecond)

	stmts := fdb.statements()
	t.Logf("statements=%d", len(stmts))
	for i, st := range stmts {
		t.Logf("stmt[%d]: %.120s", i, st)
	}
	if len(stmts) != 2 || !strings.Contains(stmts[1], "INSERT INTO `nx_role_dumpinfo`") {
		t.Fatalf("official template snapshot missing upsert (%d statements)", len(stmts))
	}
}

// 双命令同连接：cmd1(版本纠偏) 后跟 cmd2(快照)——线上静默复现。
func TestE2ETwoCommandsOneConn(t *testing.T) {
	dbinfo, ok := officialFile(t, "Res/dumpconfig/dump_role_dbinfo.xml")
	if !ok {
		return
	}
	rec, ok := officialFile(t, "Res/dumpconfig/dump_role_record.xml")
	if !ok {
		return
	}
	role, err := NewDumpRole(dbinfo, rec, func(f string, a ...any) { t.Logf("role: "+f, a...) })
	if err != nil {
		t.Fatal(err)
	}
	fdb := &fakeDB{}
	s := NewServer("127.0.0.1", 0, NewDefaultCallee(role, nil),
		func(f string, a ...any) { t.Logf("srv: "+f, a...) })
	s.Kernel.AddAccess("output", fdb)
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", s.ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	br := make([]byte, 8192)
	_, _ = conn.Read(br) // 模板

	fmt.Fprintf(conn, "$dump_role_msg 1 999\r\n")
	time.Sleep(200 * time.Millisecond)
	_, _ = conn.Read(br) // 纠偏模板
	t.Log("fixup received")

	fmt.Fprintf(conn, "$dump_role_msg 2 15 #a #b #c #d 1 0 1 0 1 1 1 1 1 #g #h\r\n")
	time.Sleep(400 * time.Millisecond)

	stmts := fdb.statements()
	t.Logf("statements=%d", len(stmts))
	if len(stmts) != 2 {
		for i, st := range stmts {
			t.Logf("stmt[%d]: %.100s", i, st)
		}
		t.Fatalf("第二命令未落 upsert（%d 条）", len(stmts))
	}
}
