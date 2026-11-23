package fssologlobal

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/lizhanyin/9yin-server-go/internal/dbconn"
	"github.com/lizhanyin/9yin-server-go/internal/dbconn/dbconntest"
)

// ---- SQLStore（dbconntest 假库）：库重启窗口与 INSERT 丢回包 ----

// fakeSQLStore 以假库装配 SQLStore（退避调短）。
func fakeSQLStore(t *testing.T) (*SQLStore, *dbconntest.Fake) {
	t.Helper()
	f := dbconntest.New()
	db, err := dbconn.OpenConnector(context.Background(), f, dbconn.Options{
		Name:         "global-test",
		RetryTimeout: 3 * time.Second,
		PingTimeout:  100 * time.Millisecond,
		Backoff:      2 * time.Millisecond,
		BackoffMax:   4 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("连假库: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return &SQLStore{db: db}, f
}

// globalRow 单行结果集构造。
func globalRow(vals ...driver.Value) *dbconntest.Rows {
	return &dbconntest.Rows{Cols: make([]string, len(vals)), Vals: [][]driver.Value{vals}}
}

// addNameRules 登记脚本：gate 接「查名门」第几次查询的返回行（nil = 名
// 字可用），claim 接「回包丢失后认领」的 owner/server_id 行，insert 接
// 「INSERT」第几次执行的结果。
func addNameRules(f *dbconntest.Fake, gate func(n int) *dbconntest.Rows, insert func(n int) error) {
	f.On("SELECT name FROM nx_role_names", func(c *dbconntest.Call) (*dbconntest.Rows, error) {
		return gate(c.N), nil
	})
	f.On("SELECT owner, server_id", func(*dbconntest.Call) (*dbconntest.Rows, error) {
		return globalRow("owner", int64(11)), nil
	})
	f.On("INSERT INTO nx_role_names", func(c *dbconntest.Call) (*dbconntest.Rows, error) {
		return nil, insert(c.N)
	})
}

// TestSQLStoreSurvivesRestart 库重启窗口内到达的登记/释放挂起重发，库起
// 后成功——而不是把连接错误抛给调用方（原来的循环重试两次即放弃）。
func TestSQLStoreSurvivesRestart(t *testing.T) {
	s, f := fakeSQLStore(t)
	addNameRules(f, func(int) *dbconntest.Rows { return nil }, func(int) error { return nil })
	f.Down()

	type result struct {
		ok  bool
		err error
	}
	added := make(chan result, 1)
	removed := make(chan result, 1)
	go func() { ok, err := s.AddName("nx_role_names", "侠客", "owner", 11); added <- result{ok, err} }()
	go func() { ok, err := s.RemoveName("nx_role_names", "路人"); removed <- result{ok, err} }()
	select {
	case r := <-added:
		t.Fatalf("库停期间不得返回: ok=%v err=%v", r.ok, r.err)
	case r := <-removed:
		t.Fatalf("库停期间不得返回: ok=%v err=%v", r.ok, r.err)
	case <-time.After(40 * time.Millisecond):
	}
	f.Up()
	if r := <-added; !r.ok || r.err != nil {
		t.Fatalf("库起后登记应成功: ok=%v err=%v", r.ok, r.err)
	}
	if r := <-removed; r.ok || r.err != nil {
		t.Fatalf("库起后释放应正常返回「名不存在」: ok=%v err=%v", r.ok, r.err)
	}
}

// TestAddNameLostAckClaim INSERT 已落库但回包丢在断连里：重发前查名门
// 认出行上的 owner 就是本请求的 → 判成功，不再发第二遍。
func TestAddNameLostAckClaim(t *testing.T) {
	s, f := fakeSQLStore(t)
	addNameRules(f,
		func(n int) *dbconntest.Rows {
			if n == 1 {
				return nil // 首查：名字可用
			}
			return globalRow("侠客") // 自己上一轮已落库
		},
		func(int) error { return mysql.ErrInvalidConn })

	ok, err := s.AddName("nx_role_names", "侠客", "owner", 11)
	if err != nil || !ok {
		t.Fatalf("丢回包的登记应认领成功，得 ok=%v err=%v", ok, err)
	}
	if n := f.Hits("INSERT INTO nx_role_names"); n != 1 {
		t.Fatalf("INSERT 不可重复执行，实际 %d 次", n)
	}
}

// TestAddNameRaceLost INSERT 发出去时被同名他人抢注：认领对不上 → false。
func TestAddNameRaceLost(t *testing.T) {
	s, f := fakeSQLStore(t)
	f.On("SELECT name FROM nx_role_names", func(c *dbconntest.Call) (*dbconntest.Rows, error) {
		if c.N == 1 {
			return nil, nil
		}
		return globalRow("侠客"), nil // 他人刚占
	})
	f.On("SELECT owner, server_id", func(*dbconntest.Call) (*dbconntest.Rows, error) {
		return globalRow("someone-else", int64(11)), nil
	})
	f.On("INSERT INTO nx_role_names", func(*dbconntest.Call) (*dbconntest.Rows, error) {
		return nil, mysql.ErrInvalidConn
	})
	ok, err := s.AddName("nx_role_names", "侠客", "owner", 11)
	if err != nil || ok {
		t.Fatalf("被抢注应 ok=false err=nil，得 ok=%v err=%v", ok, err)
	}
	if n := f.Hits("INSERT INTO nx_role_names"); n != 1 {
		t.Fatalf("INSERT 不该重发，实际 %d 次", n)
	}
}

// TestAddNameDuplicate 真重名 → false，不发 INSERT。
func TestAddNameDuplicate(t *testing.T) {
	s, f := fakeSQLStore(t)
	addNameRules(f,
		func(int) *dbconntest.Rows { return globalRow("侠客") },
		func(int) error { return errors.New("重名不该走到 INSERT") })
	ok, err := s.AddName("nx_role_names", "侠客", "owner", 11)
	if err != nil || ok {
		t.Fatalf("重名应 ok=false err=nil，得 ok=%v err=%v", ok, err)
	}
}

// ---- Break 自愈 ----

// pingGate 可切换 Ping 结果的 DBAccess。
type pingGate struct {
	*LocalStore
	pingErr error
}

func (p *pingGate) Ping() error { return p.pingErr }

// TestDatabaseBreakSelfHeal 库重启后 Break 自动摘除（官方只有 echo 能清）；
// 库仍不通则保持 Break；dbErr 累计不清零。
func TestDatabaseBreakSelfHeal(t *testing.T) {
	ls, err := NewLocalStore("", nil)
	if err != nil {
		t.Fatal(err)
	}
	gate := &pingGate{LocalStore: ls, pingErr: errors.New("dial tcp 127.0.0.1:3306: refused")}
	s := New(Options{Store: gate, UseAppRecord: true, ManualTick: true, Logf: func(string, ...any) {}})

	s.setDatabaseBreak()
	s.onDatabaseError()
	s.TimerCheck()
	if line := s.StatusLine(); !strings.Contains(line, `database="Break"`) {
		t.Fatalf("库仍不通应保持 Break: %q", line)
	}

	gate.pingErr = nil
	s.TimerCheck()
	line := s.StatusLine()
	if !strings.Contains(line, `database="OK"`) {
		t.Fatalf("库恢复后应自愈: %q", line)
	}
	if !strings.Contains(line, `db_err="1"`) {
		t.Fatalf("db_err 是断连期间的累计，不该被自愈清零: %q", line)
	}
}
