package fssolostore

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/lizhanyin/9yin-server-go/internal/dbconn"
	"github.com/lizhanyin/9yin-server-go/internal/dbconn/dbconntest"
)

// ---- 假库单测（无需真库）：库重启窗口内的 store 行为 ----

// fakeStore 以 dbconntest 假库装配 MySQLStore（退避调短）。
func fakeStore(t *testing.T) (*MySQLStore, *dbconntest.Fake) {
	t.Helper()
	f := dbconntest.New()
	db, err := dbconn.OpenConnector(context.Background(), f, dbconn.Options{
		Name:         "store-test",
		RetryTimeout: 3 * time.Second,
		PingTimeout:  100 * time.Millisecond,
		Backoff:      2 * time.Millisecond,
		BackoffMax:   4 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("连假库: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return newMySQLStore(db, nil), f
}

// row 单行结果集构造。
func row(vals ...driver.Value) *dbconntest.Rows {
	return &dbconntest.Rows{Cols: make([]string, len(vals)), Vals: [][]driver.Value{vals}}
}

// TestMySQLStoreSurvivesRestart 库重启窗口内到达的存档/查询挂起重发，
// 库起后全部成功——调用方看不到连接错误；Ping 是探活，窗口内快速落败。
func TestMySQLStoreSurvivesRestart(t *testing.T) {
	s, f := fakeStore(t)
	f.Down()

	start := time.Now()
	if err := s.Ping(); err == nil {
		t.Fatal("库停期间 Ping 应落败")
	} else if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("Ping 不该在库停期间阻塞等待，耗时 %v", elapsed)
	}

	type op struct {
		name string
		run  func() error
	}
	ops := []op{
		{"SaveRole", func() error { return s.SaveRole("侠客", RoleMeta{TotalSec: 9}, []byte("save")) }},
		{"QueryAccount", func() error { _, err := s.QueryAccount("acct", 11); return err }},
		{"ManageInfos", func() error { _, err := s.ManageInfos(); return err }},
		{"DeleteChunk", func() error { return s.DeleteChunk(17, 3) }},
		{"LoadRole", func() error {
			_, _, ok, err := s.LoadRole("acct", "侠客", 11)
			if ok {
				return errors.New("空库不该有档")
			}
			return err
		}},
	}
	errCh := make(chan error, len(ops))
	for _, o := range ops {
		go func() { errCh <- o.run() }()
	}
	select {
	case err := <-errCh:
		t.Fatalf("库停期间不得报错: %v", err)
	case <-time.After(40 * time.Millisecond):
	}
	f.Up()
	for range ops {
		if err := <-errCh; err != nil {
			t.Fatalf("库起后应重发成功: %v", err)
		}
	}
	if f.Hits("UPDATE nx_roles SET total_sec") == 0 {
		t.Fatal("存档应已落库")
	}
	if err := s.Ping(); err != nil {
		t.Fatalf("库起后 Ping 应转绿: %v", err)
	}
}

// TestMySQLStoreRetriesMidStatement 语句发出去才发现断线（invalid
// connection）同样重发；SQL 语义错误不重发。
func TestMySQLStoreRetriesMidStatement(t *testing.T) {
	s, f := fakeStore(t)
	f.Liveness = false
	f.On("UPDATE nx_roles SET total_sec", dbconntest.FailN(2, mysql.ErrInvalidConn))
	f.On("UPDATE nx_roles SET deleted", func(*dbconntest.Call) (*dbconntest.Rows, error) {
		return nil, dbconntest.SQLError(1064, "You have an error in your SQL syntax")
	})

	if err := s.SaveRole("侠客", RoleMeta{}, nil); err != nil {
		t.Fatalf("断连 2 次后应重发成功: %v", err)
	}
	if n := f.Hits("UPDATE nx_roles SET total_sec"); n != 3 {
		t.Fatalf("存档应执行到第 3 次，实际 %d", n)
	}
	if _, err := s.DeleteRole("acct", "侠客", 11); err == nil {
		t.Fatal("语法错应原样返回")
	}
	if n := f.Hits("UPDATE nx_roles SET deleted"); n != 1 {
		t.Fatalf("语法错不该重发，实际 %d 次", n)
	}
}

// TestQueryAccountNoDuplicatesOnRetry 角色清单交到一半库挂：整句重发后
// 名册不重复。
func TestQueryAccountNoDuplicatesOnRetry(t *testing.T) {
	s, f := fakeStore(t)
	f.Liveness = false
	names := []string{"甲", "乙", "丙"}
	f.On("SELECT name, uid, role_index", func(c *dbconntest.Call) (*dbconntest.Rows, error) {
		r := &dbconntest.Rows{Cols: []string{"name", "uid", "role_index", "sys_flags", "save_para1"}}
		for i, n := range names {
			r.Vals = append(r.Vals, []driver.Value{n, "uid-" + n, int64(i), int64(0), ""})
		}
		if c.N == 1 {
			r.FailAfter = 1 // 交出第一行后库挂
		}
		return r, nil
	})
	briefs, err := s.QueryAccount("acct", 11)
	if err != nil {
		t.Fatalf("重发后应成功: %v", err)
	}
	if len(briefs) != len(names) {
		t.Fatalf("名册应 %d 人（重跑不叠加），实际 %d: %+v", len(names), len(briefs), briefs)
	}
	if f.Hits("SELECT name, uid, role_index") != 2 {
		t.Fatal("应重发 1 次")
	}
}

// ---- 建角：INSERT 不可重复，丢回包后按 uid 认领 ----

// createRoleRules 建角脚本：gate 接「重名门」第几次查询的返回行（nil =
// 名字可用），insert 接「INSERT」第几次执行的结果。
func createRoleRules(f *dbconntest.Fake, gate func(n int) *dbconntest.Rows, insert func(n int) error) {
	f.On("SELECT uid FROM nx_roles", func(c *dbconntest.Call) (*dbconntest.Rows, error) {
		return gate(c.N), nil
	})
	f.On("INSERT INTO nx_roles", func(c *dbconntest.Call) (*dbconntest.Rows, error) {
		return nil, insert(c.N)
	})
}

// TestCreateRoleLostAckClaim INSERT 已落库但回包丢在断连里：重发前门里
// 认出行上的 uid 就是本请求的 → 判成功，不再发第二遍。
func TestCreateRoleLostAckClaim(t *testing.T) {
	s, f := fakeStore(t)
	createRoleRules(f,
		func(n int) *dbconntest.Rows {
			if n == 1 {
				return nil // 首查：名字可用
			}
			return row("uid-mine") // 自己上一轮已落库
		},
		func(int) error { return mysql.ErrInvalidConn })

	ok, err := s.CreateRole("acct", "侠客", "uid-mine", 11, RoleMeta{}, nil)
	if err != nil || !ok {
		t.Fatalf("丢回包的建角应认领成功，得 ok=%v err=%v", ok, err)
	}
	if n := f.Hits("INSERT INTO nx_roles"); n != 1 {
		t.Fatalf("INSERT 不可重复执行，实际 %d 次", n)
	}
}

// TestCreateRoleNotAppliedRetry 库没收到 INSERT：照常重发。
func TestCreateRoleNotAppliedRetry(t *testing.T) {
	s, f := fakeStore(t)
	createRoleRules(f,
		func(int) *dbconntest.Rows { return nil },
		func(n int) error {
			if n == 1 {
				return mysql.ErrInvalidConn
			}
			return nil
		})
	if ok, err := s.CreateRole("acct", "侠客", "uid-mine", 11, RoleMeta{}, nil); err != nil || !ok {
		t.Fatalf("重发后应成功，得 ok=%v err=%v", ok, err)
	}
	if n := f.Hits("INSERT INTO nx_roles"); n != 2 {
		t.Fatalf("INSERT 应重发 1 次，实际 %d 次", n)
	}
}

// TestCreateRoleDuplicate 真重名（行上 uid 不是本请求的）→ false，不发 INSERT。
func TestCreateRoleDuplicate(t *testing.T) {
	s, f := fakeStore(t)
	createRoleRules(f,
		func(int) *dbconntest.Rows { return row("uid-other") },
		func(int) error { return errors.New("重名不该走到 INSERT") })
	ok, err := s.CreateRole("acct", "侠客", "uid-mine", 11, RoleMeta{}, nil)
	if err != nil || ok {
		t.Fatalf("重名应 ok=false err=nil，得 ok=%v err=%v", ok, err)
	}
}

// ---- 登录：首登建号 INSERT 丢回包 ----

// dcAccountsRules ensureDcAccounts 的两条探表查询（表已存在、密码列已扩宽）。
func dcAccountsRules(f *dbconntest.Fake) {
	f.On("information_schema.tables", func(*dbconntest.Call) (*dbconntest.Rows, error) {
		return row(int64(1)), nil
	})
	f.On("CHARACTER_MAXIMUM_LENGTH", func(*dbconntest.Call) (*dbconntest.Rows, error) {
		return row(int64(64)), nil
	})
}

// TestAccountLogonLostAck 首登 INSERT 已落库、回包丢失：重发撞主键后回读
// 命中（密码/会话都对）→ 照常放行，而不是把首次登录拒掉。
func TestAccountLogonLostAck(t *testing.T) {
	s, f := fakeStore(t)
	dcAccountsRules(f)
	f.On("SELECT user_type, points", func(c *dbconntest.Call) (*dbconntest.Rows, error) {
		if c.N == 1 {
			return nil, nil // 首登：账号不存在
		}
		return row(int64(0), int64(0), int64(0), "pw", int64(5)), nil // 自己刚插的行
	})
	f.On("INSERT INTO `nx_district`.`dc_accounts`", func(c *dbconntest.Call) (*dbconntest.Rows, error) {
		if c.N == 1 {
			return nil, mysql.ErrInvalidConn // 已落库，回包丢
		}
		return nil, dbconntest.SQLError(1062, "Duplicate entry") // 重发撞主键
	})

	info, ok, err := s.AccountLogon("user", "pw", "127.0.0.1", 5)
	if err != nil || !ok {
		t.Fatalf("丢回包的首登应放行，得 ok=%v err=%v", ok, err)
	}
	if info != (AccountInfo{}) {
		t.Fatalf("info = %+v", info)
	}
}

// TestAccountLogonDupKeyStillRejects 重复键且回读不命中（他会被占）→ 拒签。
func TestAccountLogonDupKeyStillRejects(t *testing.T) {
	s, f := fakeStore(t)
	dcAccountsRules(f)
	f.On("SELECT user_type, points", func(*dbconntest.Call) (*dbconntest.Rows, error) {
		return nil, nil // SELECT 门始终不命中
	})
	f.On("INSERT INTO `nx_district`.`dc_accounts`", func(*dbconntest.Call) (*dbconntest.Rows, error) {
		return nil, dbconntest.SQLError(1062, "Duplicate entry")
	})
	if _, ok, err := s.AccountLogon("user", "pw", "127.0.0.1", 5); ok || err != nil {
		t.Fatalf("应拒签 ok=false err=nil，得 ok=%v err=%v", ok, err)
	}
}
