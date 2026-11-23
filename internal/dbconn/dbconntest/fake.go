// Package dbconntest 提供可注入故障的假 MySQL 驱动（database/sql 的
// driver.Connector），供各连库包的单测模拟「数据库重启窗口」——拨号拒连、
// 池内连接猝死、语句级脚本错误——无需真库。
//
// 用法：f := New() 建库；dbconn.OpenConnector(ctx, f, opt) 连库；
// f.Down()/f.Up() 模拟库停/库起；f.On(关键字, 脚本) 按语句注入结果。
// 错误形态对齐 go-sql-driver：取池内死连接 → ResetSession 回
// driver.ErrBadConn；语句发出去才发现断线 → mysql.ErrInvalidConn；
// 库没起来 → 拨号拒连 net.OpError。
package dbconntest

import (
	"context"
	"database/sql/driver"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"syscall"

	"github.com/go-sql-driver/mysql"
)

// Fake 假库：实现 driver.Connector，直接交给 sql.OpenDB / dbconn。
// 脚本与拨号故障须在并发使用前配置好。
type Fake struct {
	mu    sync.Mutex
	up    bool
	epoch int // 每次 Down 自增：旧纪元的连接一律判死
	dials int // 累计拨号次数
	conns int // 累计拨通次数
	rules []rule
	hits  []string

	// DialErr 非空时拨号一律失败（模拟密码错 1045 / 库不存在 1049 等
	// 永久性错误）。
	DialErr error
	// DialBlock true 时拨号挂起到 ctx 结束（模拟库主机失联）。
	DialBlock bool
	// Liveness true（缺省）= 池内死连接在取出时即被 ResetSession 识别；
	// false = 漏检，语句发出去才报断连。
	Liveness bool
}

// Call 一次语句执行：N = 命中该脚本的第几次执行，Args = 绑定参数。
type Call struct {
	N    int
	Args []driver.NamedValue
}

// Rows 脚本返回的结果集。
type Rows struct {
	Cols []string
	Vals [][]driver.Value
	// FailAfter >0 时交出前 FailAfter 行后连接猝死（模拟结果集中途库挂）；
	// 零 = 整个结果集可读完。
	FailAfter int
}

type rule struct {
	substr string
	fn     func(*Call) (*Rows, error)
	n      int
}

// New 建一个活着的库。
func New() *Fake { return &Fake{up: true, Liveness: true} }

// Down 库停：此后拨号拒连，既有连接判死。
func (f *Fake) Down() {
	f.mu.Lock()
	f.up = false
	f.epoch++
	f.mu.Unlock()
}

// Up 库起：拨号恢复；既有连接仍是死的（重启前的连接不会复活）。
func (f *Fake) Up() {
	f.mu.Lock()
	f.up = true
	f.mu.Unlock()
}

// Dials 累计拨号次数。
func (f *Fake) Dials() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.dials
}

// Conns 累计拨通次数。
func (f *Fake) Conns() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.conns
}

// Hits 含 substr 的语句累计执行次数。
func (f *Fake) Hits(substr string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, q := range f.hits {
		if strings.Contains(q, substr) {
			n++
		}
	}
	return n
}

// On 给含 substr 的语句挂脚本（先挂的先匹配）。脚本须可并发调用，
// 可在其中调 Down/Up 模拟语句执行途中库挂。
func (f *Fake) On(substr string, fn func(*Call) (*Rows, error)) {
	f.mu.Lock()
	f.rules = append(f.rules, rule{substr: substr, fn: fn})
	f.mu.Unlock()
}

// Driver implements driver.Connector.
func (f *Fake) Driver() driver.Driver { return drv{} }

// Connect implements driver.Connector.
func (f *Fake) Connect(ctx context.Context) (driver.Conn, error) {
	f.mu.Lock()
	f.dials++
	block, dialErr, up, epoch := f.DialBlock, f.DialErr, f.up, f.epoch
	f.mu.Unlock()

	if block {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	if dialErr != nil {
		return nil, dialErr
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !up {
		return nil, Refused()
	}
	f.mu.Lock()
	f.conns++
	f.mu.Unlock()
	return &conn{f: f, epoch: epoch}, nil
}

// run 执行一条语句：记账 → 找脚本（锁外调用，脚本里可调 Down/Up）。
func (f *Fake) run(c *conn, query string, args []driver.NamedValue) (driver.Result, driver.Rows, error) {
	fn := func(*Call) (*Rows, error) { return nil, nil }
	call := &Call{Args: args}
	f.mu.Lock()
	f.hits = append(f.hits, query)
	for i := range f.rules {
		if strings.Contains(query, f.rules[i].substr) {
			f.rules[i].n++
			call.N = f.rules[i].n
			fn = f.rules[i].fn
			break
		}
	}
	f.mu.Unlock()

	out, err := fn(call)
	if err != nil && (errors.Is(err, mysql.ErrInvalidConn) || errors.Is(err, driver.ErrBadConn)) {
		c.kill()
	}
	if out == nil {
		out = &Rows{Cols: []string{"v"}}
	}
	// 写语句脚本不带行集：按「影响了 1 行」回（真驱动 INSERT 的口径）。
	affected := int64(len(out.Vals))
	if affected == 0 {
		affected = 1
	}
	return driver.RowsAffected(affected), &rows{rs: out, c: c}, err
}

// ---- 连接 ----

type conn struct {
	f     *Fake
	epoch int
	dead  bool // f.mu 保护
}

// kill 判死本连接并返回断连错误。
func (c *conn) kill() error {
	c.f.mu.Lock()
	c.dead = true
	c.f.mu.Unlock()
	return mysql.ErrInvalidConn
}

// alive 连接是否仍可用；库重启过（纪元变更）即死。
func (c *conn) alive() error {
	c.f.mu.Lock()
	up, epoch, dead := c.f.up, c.f.epoch, c.dead
	c.f.mu.Unlock()
	if dead || epoch != c.epoch || !up {
		return c.kill()
	}
	return nil
}

func (c *conn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("dbconntest: 仅支持 Exec/Query 直执行")
}

func (c *conn) Close() error { return nil }

func (c *conn) Begin() (driver.Tx, error) {
	return nil, errors.New("dbconntest: 不支持事务")
}

// Ping 对齐驱动：探活时才发现连接已死 = 一个字节都没写出去 → 回
// driver.ErrBadConn（database/sql 对这一步不再换连，正是 dbconn.Ping
// 要兜的洞）。
func (c *conn) Ping(context.Context) error {
	if err := c.alive(); err != nil {
		return driver.ErrBadConn
	}
	return nil
}

// ResetSession 对齐驱动 CheckConnLiveness：取出池内连接时探活，死连接
// 回 driver.ErrBadConn 让 database/sql 丢弃换连。
func (c *conn) ResetSession(context.Context) error {
	c.f.mu.Lock()
	liveness, dead := c.f.Liveness, c.dead
	c.f.mu.Unlock()
	if dead {
		return driver.ErrBadConn // 驱动自己关过的连接：closed 标志位先于活性探测
	}
	if !liveness {
		return nil
	}
	if err := c.alive(); err != nil {
		return driver.ErrBadConn
	}
	return nil
}

func (c *conn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := c.alive(); err != nil {
		return nil, err
	}
	res, _, err := c.f.run(c, query, args)
	return res, err
}

func (c *conn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := c.alive(); err != nil {
		return nil, err
	}
	_, rows, err := c.f.run(c, query, args)
	return rows, err
}

// ---- 结果集 ----

type rows struct {
	rs *Rows
	c  *conn
	i  int
}

func (r *rows) Columns() []string { return r.rs.Cols }

// Close 对齐驱动 readUntilEOF：提前退出时把剩余行读弃，途中同样会撞上断连。
func (r *rows) Close() error {
	dest := make([]driver.Value, len(r.rs.Cols))
	for r.i < len(r.rs.Vals) {
		if err := r.Next(dest); err != nil {
			return err
		}
	}
	return nil
}

func (r *rows) Next(dest []driver.Value) error {
	if r.i >= len(r.rs.Vals) {
		return io.EOF
	}
	if r.rs.FailAfter > 0 && r.i >= r.rs.FailAfter {
		return r.c.kill() // 交行途中库挂：驱动断连
	}
	copy(dest, r.rs.Vals[r.i])
	r.i++
	return nil
}

// ---- 错误与脚本 ----

// Refused 拨号拒连（Windows 文案；net.OpError 包 Errno，与真驱动同构）。
func Refused() error {
	return &net.OpError{
		Op:   "dial",
		Net:  "tcp",
		Addr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 3306},
		Err:  os.NewSyscallError("connectex", syscall.ECONNREFUSED),
	}
}

// SQLError 服务端错误（Number 取 MySQL 错误码）。
func SQLError(number uint16, message string) error {
	return &mysql.MySQLError{Number: number, Message: message}
}

// FailN 前 n 次执行以 err 失败、之后成功。
func FailN(n int, err error) func(*Call) (*Rows, error) {
	return func(c *Call) (*Rows, error) {
		if c.N <= n {
			return nil, err
		}
		return nil, nil
	}
}

type drv struct{}

func (drv) Open(string) (driver.Conn, error) {
	return nil, errors.New("dbconntest: 请经 Connector 连接")
}
