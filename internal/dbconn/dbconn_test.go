package dbconn

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/lizhanyin/9yin-server-go/internal/dbconn/dbconntest"
)

// testOpt 缩短退避/日志间隔，让「重启窗口」类测试跑得快。
func testOpt(logf func(string, ...any)) Options {
	return Options{
		Name:         "t",
		Logf:         logf,
		RetryTimeout: 2 * time.Second,
		PingTimeout:  100 * time.Millisecond,
		Backoff:      2 * time.Millisecond,
		BackoffMax:   4 * time.Millisecond,
		LogEvery:     10 * time.Millisecond,
	}
}

// logBuf 收集日志的线程安全缓冲。
type logBuf struct {
	mu    sync.Mutex
	lines []string
}

func (l *logBuf) logf(format string, a ...any) {
	l.mu.Lock()
	l.lines = append(l.lines, fmt.Sprintf(format, a...))
	l.mu.Unlock()
}

func (l *logBuf) count(sub string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := 0
	for _, s := range l.lines {
		if strings.Contains(s, sub) {
			n++
		}
	}
	return n
}

// openFake 连假库；logf 可为 nil。
func openFake(t *testing.T, f *dbconntest.Fake, logf func(string, ...any)) *DB {
	t.Helper()
	db, err := OpenConnector(context.Background(), f, testOpt(logf))
	if err != nil {
		t.Fatalf("连假库: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// ---- 启动连库 ----

// TestOpenWaitsForDatabaseRestart 库在重启中启动服务：等到库起来再返回，
// 而不是 Ping 一次失败就退出。
func TestOpenWaitsForDatabaseRestart(t *testing.T) {
	f := dbconntest.New()
	f.Down()
	lg := &logBuf{}
	type result struct {
		db  *DB
		err error
	}
	done := make(chan result, 1)
	go func() {
		db, err := OpenConnector(context.Background(), f, testOpt(lg.logf))
		done <- result{db, err}
	}()

	select {
	case r := <-done:
		t.Fatalf("库未起就返回: err=%v", r.err)
	case <-time.After(60 * time.Millisecond):
	}
	f.Up()
	r := <-done
	if r.err != nil {
		t.Fatalf("库起后应连上: %v", r.err)
	}
	r.db.Close()
	if f.Dials() < 2 {
		t.Fatalf("库停期间应持续重拨，dials=%d", f.Dials())
	}
	if lg.count("mysql 未就绪，等待重试") != 1 {
		t.Fatalf("首拍应打一条未就绪日志: %v", lg.lines)
	}
}

// TestOpenPermanentErrorFailsFast 密码错/库不存在这类等不来的错误不重试。
func TestOpenPermanentErrorFailsFast(t *testing.T) {
	for _, code := range []uint16{1045, 1049} {
		f := dbconntest.New()
		f.DialErr = dbconntest.SQLError(code, "permanent")
		start := time.Now()
		_, err := OpenConnector(context.Background(), f, testOpt(nil))
		if err == nil || !strings.Contains(err.Error(), fmt.Sprint(code)) {
			t.Fatalf("code %d: err = %v，应带错误码", code, err)
		}
		if n := f.Dials(); n != 1 {
			t.Fatalf("code %d: 永久性错误不该重试，dials=%d", code, n)
		}
		if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
			t.Fatalf("code %d: 应立即失败，耗时 %v", code, elapsed)
		}
	}
}

// TestOpenGivesUpOnDeadline 调用方 ctx / ConnectTimeout 到期即放弃，
// 错误带重试次数与最后一次连接错误。
func TestOpenGivesUpOnDeadline(t *testing.T) {
	newDown := func() *dbconntest.Fake {
		f := dbconntest.New()
		f.Down()
		return f
	}

	f := newDown()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()
	_, err := OpenConnector(ctx, f, testOpt(nil))
	if err == nil || !strings.Contains(err.Error(), "已重试") {
		t.Fatalf("ctx 到期应带重试统计返回，得 %v", err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("ctx 到期后末次错误应是 ctx 错误，得 %v", err)
	}

	f = newDown()
	opt := testOpt(nil)
	opt.ConnectTimeout = 60 * time.Millisecond
	if _, err := OpenConnector(context.Background(), f, opt); err == nil || !strings.Contains(err.Error(), "已重试") {
		t.Fatalf("ConnectTimeout 到期应放弃，得 %v", err)
	}
}

// TestMySQLConfigTimeoutDefaults DSN 没写超时才注入兜底；写了的保持原样。
func TestMySQLConfigTimeoutDefaults(t *testing.T) {
	cfg, err := mysqlConfig("u:p@tcp(127.0.0.1:3306)/db", Options{DialTimeout: 5 * time.Second, IOTimeout: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Timeout != 5*time.Second || cfg.ReadTimeout != time.Minute || cfg.WriteTimeout != time.Minute {
		t.Fatalf("兜底超时 timeout=%v read=%v write=%v", cfg.Timeout, cfg.ReadTimeout, cfg.WriteTimeout)
	}

	cfg, err = mysqlConfig("u:p@tcp(127.0.0.1:3306)/db?timeout=9s&readTimeout=8s", Options{DialTimeout: 5 * time.Second, IOTimeout: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Timeout != 9*time.Second || cfg.ReadTimeout != 8*time.Second || cfg.WriteTimeout != time.Minute {
		t.Fatalf("显式超时应保留 timeout=%v read=%v write=%v", cfg.Timeout, cfg.ReadTimeout, cfg.WriteTimeout)
	}

	cfg, err = mysqlConfig("u:p@tcp(127.0.0.1:3306)/db", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Timeout != 0 || cfg.ReadTimeout != 0 || cfg.WriteTimeout != 0 {
		t.Fatal("Options 零值不该注入超时")
	}

	if _, err := mysqlConfig("://bad dsn", Options{}); err == nil {
		t.Fatal("非法 DSN 应报错")
	}
}

// TestOptionsEnvOverrides env 覆盖重试时限；0/off = 不限；非法值回缺省。
func TestOptionsEnvOverrides(t *testing.T) {
	t.Setenv("NINEYIN_DB_RETRY_TIMEOUT", "45s")
	t.Setenv("NINEYIN_DB_CONNECT_TIMEOUT", "1m")
	opt := NewOptions("x", nil)
	if opt.RetryTimeout != 45*time.Second || opt.ConnectTimeout != time.Minute {
		t.Fatalf("env 覆盖未生效: retry=%v connect=%v", opt.RetryTimeout, opt.ConnectTimeout)
	}

	t.Setenv("NINEYIN_DB_RETRY_TIMEOUT", "off")
	t.Setenv("NINEYIN_DB_CONNECT_TIMEOUT", "0")
	opt = NewOptions("x", nil)
	if opt.RetryTimeout >= 0 || opt.ConnectTimeout >= 0 {
		t.Fatalf("0/off 应为不限: retry=%v connect=%v", opt.RetryTimeout, opt.ConnectTimeout)
	}

	t.Setenv("NINEYIN_DB_RETRY_TIMEOUT", "不久")
	t.Setenv("NINEYIN_DB_CONNECT_TIMEOUT", "")
	opt = NewOptions("x", nil)
	if opt.RetryTimeout != 30*time.Second || opt.ConnectTimeout >= 0 {
		t.Fatalf("缺省应为 30s / 不限: retry=%v connect=%v", opt.RetryTimeout, opt.ConnectTimeout)
	}
}

// ---- 运行期重试 ----

// TestRetrySurvivesRestart 库重启窗口内到达的语句挂起重发，库起后成功
// 落库——而不是把连接错误抛给调用方。
func TestRetrySurvivesRestart(t *testing.T) {
	f := dbconntest.New()
	lg := &logBuf{}
	db := openFake(t, f, lg.logf)

	f.Down()
	errCh := make(chan error, 1)
	go func() {
		_, err := db.Exec("UPDATE nx_roles SET online = 0 WHERE name = ?", "x")
		errCh <- err
	}()
	select {
	case err := <-errCh:
		t.Fatalf("库停期间不得报错: %v", err)
	case <-time.After(60 * time.Millisecond):
	}
	f.Up()
	if err := <-errCh; err != nil {
		t.Fatalf("库起后应重发成功: %v", err)
	}
	if lg.count("mysql 连接中断，等待重试") != 1 || lg.count("mysql 连接恢复") != 1 {
		t.Fatalf("断连应打「中断」+「恢复」各一条: %v", lg.lines)
	}
}

// TestRetryMidStatementDeath 活性检查漏检、语句发出去才发现断线
// （invalid connection）同样重发。
func TestRetryMidStatementDeath(t *testing.T) {
	f := dbconntest.New()
	f.Liveness = false
	f.On("UPDATE nx_roles", dbconntest.FailN(2, mysql.ErrInvalidConn))
	db := openFake(t, f, nil)

	if _, err := db.Exec("UPDATE nx_roles SET online = 1 WHERE name = ?", "x"); err != nil {
		t.Fatalf("断连 2 次后第 3 次应成功: %v", err)
	}
	if n := f.Hits("UPDATE nx_roles"); n != 3 {
		t.Fatalf("语句应重发到第 3 次，实际 %d 次", n)
	}
}

// TestRetryNoResendOnSQLError SQL 语义错误不重发。
func TestRetryNoResendOnSQLError(t *testing.T) {
	f := dbconntest.New()
	f.On("INSERT INTO t", func(*dbconntest.Call) (*dbconntest.Rows, error) {
		return nil, dbconntest.SQLError(1064, "You have an error in your SQL syntax")
	})
	db := openFake(t, f, nil)

	_, err := db.Exec("INSERT INTO t VALUES (?)", 1)
	var myErr *mysql.MySQLError
	if !errors.As(err, &myErr) || myErr.Number != 1064 {
		t.Fatalf("应原样返回 1064，得 %v", err)
	}
	if n := f.Hits("INSERT INTO t"); n != 1 {
		t.Fatalf("语法错不该重发，实际 %d 次", n)
	}
}

// TestRetryBudgetExhausted 重试总时限到即放弃，带统计日志。
func TestRetryBudgetExhausted(t *testing.T) {
	f := dbconntest.New()
	lg := &logBuf{}
	opt := testOpt(lg.logf)
	db, err := OpenConnector(context.Background(), f, opt)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	opt.RetryTimeout = 80 * time.Millisecond
	db.opt = opt

	f.Down()
	start := time.Now()
	_, err = db.Exec("SELECT 1")
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("库一直不起应最终失败")
	}
	if elapsed < 80*time.Millisecond || elapsed > time.Second {
		t.Fatalf("应在 RetryTimeout(80ms) 附近放弃，耗时 %v", elapsed)
	}
	if lg.count("放弃") != 1 {
		t.Fatalf("放弃时应打一条统计日志: %v", lg.lines)
	}
}

// TestRetryCallerCtxBound 调用方 ctx 到期即返回，不等 RetryTimeout。
func TestRetryCallerCtxBound(t *testing.T) {
	f := dbconntest.New()
	db := openFake(t, f, nil)
	f.Down()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := db.ExecContext(ctx, "SELECT 1")
	if err == nil {
		t.Fatal("库停且 ctx 到期应失败")
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("应随 ctx(60ms) 返回，耗时 %v", elapsed)
	}
}

// TestRetryOutageLogNotSpammy 并发多个操作同时撞上重启：日志只有一份断连
// 状态，不逐操作刷「中断/恢复」。
func TestRetryOutageLogNotSpammy(t *testing.T) {
	f := dbconntest.New()
	lg := &logBuf{}
	db := openFake(t, f, lg.logf)

	f.Down()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := db.Exec("UPDATE nx_roles SET online = 0"); err != nil {
				t.Errorf("库起后应成功: %v", err)
			}
		}()
	}
	time.Sleep(40 * time.Millisecond)
	f.Up()
	wg.Wait()

	if n := lg.count("mysql 连接中断"); n != 1 {
		t.Fatalf("「连接中断」应只有 1 条，实际 %d: %v", n, lg.lines)
	}
	if n := lg.count("mysql 连接恢复"); n != 1 {
		t.Fatalf("「连接恢复」应只有 1 条，实际 %d: %v", n, lg.lines)
	}
}

// ---- Row / Query 门面 ----

// TestRowScanRetries 单行扫描断连后重发；无行仍是 sql.ErrNoRows。
func TestRowScanRetries(t *testing.T) {
	f := dbconntest.New()
	f.Liveness = false
	f.On("SELECT n", func(c *dbconntest.Call) (*dbconntest.Rows, error) {
		if c.N == 1 {
			return nil, mysql.ErrInvalidConn
		}
		return &dbconntest.Rows{Cols: []string{"n"}, Vals: [][]driver.Value{{int64(7)}}}, nil
	})
	db := openFake(t, f, nil)

	var n int
	if err := db.Row("SELECT n FROM t WHERE id = ?", 3).Scan(&n); err != nil || n != 7 {
		t.Fatalf("Row.Scan = %d, %v，应 7, nil", n, err)
	}
	if got := f.Hits("SELECT n"); got != 2 {
		t.Fatalf("应重发 1 次共 2 次，实际 %d", got)
	}

	if err := db.Row("SELECT missing FROM t").Scan(&n); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("无行应返回 sql.ErrNoRows，得 %v", err)
	}
	if got := f.Hits("SELECT missing"); got != 1 {
		t.Fatalf("ErrNoRows 不该重发，实际 %d 次", got)
	}
}

// TestQueryRetriesMidIteration 结果集交到一半库挂：整句重发并重跑消费
// 函数——消费函数须可重入（累加器先清空）。
func TestQueryRetriesMidIteration(t *testing.T) {
	f := dbconntest.New()
	f.Liveness = false
	f.On("SELECT name", func(c *dbconntest.Call) (*dbconntest.Rows, error) {
		r := &dbconntest.Rows{Cols: []string{"name"}, Vals: [][]driver.Value{{"a"}, {"b"}, {"c"}}}
		if c.N == 1 {
			r.FailAfter = 1
		}
		return r, nil
	})
	db := openFake(t, f, nil)

	var out []string
	err := db.Query(func(rows *sql.Rows) error {
		out = out[:0]
		for rows.Next() {
			var s string
			if err := rows.Scan(&s); err != nil {
				return err
			}
			out = append(out, s)
		}
		return rows.Err()
	}, "SELECT name FROM t")
	if err != nil {
		t.Fatalf("重发后应成功: %v", err)
	}
	if strings.Join(out, ",") != "a,b,c" {
		t.Fatalf("结果应为重跑后的 a,b,c（无重复），得 %v", out)
	}
	if got := f.Hits("SELECT name"); got != 2 {
		t.Fatalf("应重发 1 次共 2 次，实际 %d", got)
	}
}

// TestQueryNoResendAfterConsume 消费函数已正常返回，丢弃剩余行时断连
// 不触发重发（结果已拿到，重发只会重复消费）。
func TestQueryNoResendAfterConsume(t *testing.T) {
	f := dbconntest.New()
	f.Liveness = false
	f.On("SELECT name", func(c *dbconntest.Call) (*dbconntest.Rows, error) {
		return &dbconntest.Rows{
			Cols: []string{"name"}, Vals: [][]driver.Value{{"a"}, {"b"}, {"c"}}, FailAfter: 1,
		}, nil
	})
	db := openFake(t, f, nil)

	var got []string
	err := db.Query(func(rows *sql.Rows) error {
		for rows.Next() {
			var s string
			if err := rows.Scan(&s); err != nil {
				return err
			}
			got = append(got, s)
			return nil // 只取首行
		}
		return nil
	}, "SELECT name FROM t")
	if err != nil {
		t.Fatalf("消费成功后不应因丢弃剩余行失败而报错: %v", err)
	}
	if strings.Join(got, ",") != "a" || f.Hits("SELECT name") != 1 {
		t.Fatalf("不应重发: got=%v hits=%d", got, f.Hits("SELECT name"))
	}
}

// ---- Ping ----

// TestPingBounded 库主机失联（拨号挂死）时 Ping 按 PingTimeout 返回，
// 不拖住调用它的定时 tick；库恢复后 Ping 转绿并打「恢复」。
func TestPingBounded(t *testing.T) {
	f := dbconntest.New()
	lg := &logBuf{}
	db := openFake(t, f, lg.logf)

	// 池内连接全部判死（库重启过）+ 新拨号挂死：Ping 必须走到拨号并受
	// PingTimeout 封顶。
	f.Down()
	f.DialBlock = true
	start := time.Now()
	if err := db.Ping(); err == nil {
		t.Fatal("拨号挂死时 Ping 应失败")
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("Ping 应受 PingTimeout(100ms) 封顶，耗时 %v", elapsed)
	}
	if lg.count("mysql 连接中断") != 1 {
		t.Fatalf("Ping 落败应记一次断连: %v", lg.lines)
	}

	f.Up()
	f.DialBlock = false
	if err := db.Ping(); err != nil {
		t.Fatalf("库恢复后 Ping 应成功: %v", err)
	}
	if lg.count("mysql 连接恢复") != 1 {
		t.Fatalf("库恢复后应打「恢复」: %v", lg.lines)
	}
}

// TestPingPastUndetectedDeadConn 真库杀库重启实测到的洞：库重启后池里的
// 死连接骗过了驱动活性检查，到 COM_PING 才暴露（driver.ErrBadConn），
// database/sql 对这一步不换连——Ping 须自己换连再探，库已恢复就不该
// 误报 Break。
func TestPingPastUndetectedDeadConn(t *testing.T) {
	f := dbconntest.New()
	f.Liveness = false // 活性检查漏检
	lg := &logBuf{}
	db := openFake(t, f, lg.logf)

	// 池里那条连接在库重启中死了（纪元已换），库本身已经回来
	f.Down()
	f.Up()
	for i := 0; i < 3; i++ {
		if err := db.Ping(); err != nil {
			t.Fatalf("第 %d 次 Ping：漏检的死连接不该让库已恢复时误报: %v", i+1, err)
		}
	}
	if n := lg.count("mysql 连接中断"); n != 0 {
		t.Fatalf("库没停不该记断连（%d 条）: %v", n, lg.lines)
	}
}

// ---- IsConnErr ----

func TestIsConnErr(t *testing.T) {
	opErr := func(err error) error { return &net.OpError{Op: "dial", Net: "tcp", Err: err} }
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"driver.ErrBadConn", driver.ErrBadConn, true},
		{"mysql.ErrInvalidConn", mysql.ErrInvalidConn, true},
		{"wrapped invalid", fmt.Errorf("exec: %w", mysql.ErrInvalidConn), true},
		{"io.EOF", io.EOF, true},
		{"unexpected EOF", io.ErrUnexpectedEOF, true},
		{"refused", dbconntest.Refused(), true},
		{"reset", &net.OpError{Op: "read", Net: "tcp", Err: os.NewSyscallError("wsarecv", syscall.ECONNRESET)}, true},
		{"dial timeout", opErr(&net.DNSError{Err: "i/o timeout", IsTimeout: true}), true},
		{"driver read timeout", context.DeadlineExceeded, true},
		{"2006 gone away", dbconntest.SQLError(2006, "MySQL server has gone away"), true},
		{"2013 lost", dbconntest.SQLError(2013, "Lost connection to MySQL server during query"), true},
		{"1053 shutdown", dbconntest.SQLError(1053, "Server shutdown in progress"), true},
		{"1040 too many connections", dbconntest.SQLError(1040, "Too many connections"), true},
		{"4031 idle killed", dbconntest.SQLError(4031, "The client was disconnected by server due to inactivity."), true},
		{"1064 syntax", dbconntest.SQLError(1064, "syntax"), false},
		{"1062 dup key", dbconntest.SQLError(1062, "Duplicate entry"), false},
		{"1045 access denied", dbconntest.SQLError(1045, "Access denied"), false},
		{"1049 unknown db", dbconntest.SQLError(1049, "Unknown database"), false},
		{"1146 no such table", dbconntest.SQLError(1146, "Table doesn't exist"), false},
		{"sql.ErrNoRows", sql.ErrNoRows, false},
		{"db closed", errors.New("sql: database is closed"), false},
		{"plain error", errors.New("invalid connection"), false},
		{"dns not found", opErr(&net.DNSError{Err: "no such host", Name: "db", IsNotFound: true}), false},
		{"dns temporary", opErr(&net.DNSError{Err: "server misbehaving", Name: "db", IsTemporary: true}), true},
		{"unknown network", opErr(errors.New("unknown network udp")), false},
		{"context canceled", context.Canceled, false},
	}
	for _, c := range cases {
		if got := IsConnErr(c.err); got != c.want {
			t.Errorf("%s: IsConnErr = %v，want %v", c.name, got, c.want)
		}
	}
}
