// Package dbconn 给 database/sql 的 MySQL 连接补「连接级重试」。
//
// 背景：MySQL 重启/闪断期间连接池里的连接必死。database/sql 只对
// driver.ErrBadConn 做至多 3 次即时换连；go-sql-driver 在读写失败时
// 上抛的是 ErrInvalidConn("invalid connection")/EOF/拒连/超时，这些一律
// 直抛调用方——登录、存档落在重启窗口内就直接失败。
//
// 本包把这些归类为连接级错误（IsConnErr），退避后换连接重发，直到成功、
// 出现非连接级错误、调用方 ctx 到期或重试总时限到。官方 CAccess 的
// CheckConnect 重连 + FailCanRetry(<2) 重试即此语义，区别在重试期间带
// 退避等待、覆盖整个重启窗口。
//
// 用法：Open 拿 *DB（连到通为止），调用点把 *sql.DB 的 Exec/QueryRow/
// Query 换成 DB 的同名门面——*DB 有意不内嵌 *sql.DB，只暴露重试版方法，
// 绕过重试必须显式写 Raw()，不会静默漏网。
package dbconn

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-sql-driver/mysql"
)

// Options 重试策略与日志；零值字段经 filled 补缺省。
type Options struct {
	// Name 日志与错误里的库标识（如 "fssolostore"）。
	Name string
	Logf func(format string, args ...any)

	// RetryTimeout 单次操作的连接级重试总时限（自首次失败起算）；<0 = 不限。
	RetryTimeout time.Duration
	// ConnectTimeout 启动连库（Open）的等待上限；<0 = 不限，仅受调用方 ctx 约束。
	ConnectTimeout time.Duration
	// PingTimeout 单次 Ping 上限——Ping 是探活不等待，超时即判 Break。
	PingTimeout time.Duration
	// DialTimeout / IOTimeout 注入 DSN 的拨号 / 读写超时（DSN 已写明的不
	// 动，零 = 不注入）。没有超时的连接在库主机失联时会永久挂死，重试
	// 无从触发。
	DialTimeout time.Duration
	IOTimeout   time.Duration

	// Backoff / BackoffMax 退避起点与上限；LogEvery 断连期间「仍不可用」
	// 日志的最小间隔（测试可调小）。
	Backoff    time.Duration
	BackoffMax time.Duration
	LogEvery   time.Duration
}

// NewOptions 带缺省与 env 覆盖的策略：
//
//	NINEYIN_DB_RETRY_TIMEOUT   单次操作重试总时限（缺省 30s；0/off = 不限）
//	NINEYIN_DB_CONNECT_TIMEOUT 启动连库等待上限（缺省不限——库在重启就等它起来）
func NewOptions(name string, logf func(string, ...any)) Options {
	if logf == nil {
		// 生产缺省：断连/恢复走标准日志，运维看得见库重启窗口
		logf = func(format string, args ...any) { log.Printf(format, args...) }
	}
	return Options{
		Name:           name,
		Logf:           logf,
		RetryTimeout:   envDuration("NINEYIN_DB_RETRY_TIMEOUT", 30*time.Second),
		ConnectTimeout: envDuration("NINEYIN_DB_CONNECT_TIMEOUT", -1),
		PingTimeout:    2 * time.Second,
		DialTimeout:    5 * time.Second,
		IOTimeout:      60 * time.Second,
		Backoff:        200 * time.Millisecond,
		BackoffMax:     2 * time.Second,
		LogEvery:       5 * time.Second,
	}
}

// envDuration 读时长型 env；"0"/"off" 归一为 -1（=不限）。
func envDuration(key string, def time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	switch v {
	case "":
		return def
	case "0", "off", "none":
		return -1
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dbconn: %s=%q 不是合法时长（如 30s），按 %s 处理\n", key, v, def)
		return def
	}
	return d
}

func (o Options) filled() Options {
	if o.Logf == nil {
		o.Logf = func(string, ...any) {}
	}
	if o.RetryTimeout == 0 {
		o.RetryTimeout = 30 * time.Second
	}
	if o.PingTimeout == 0 {
		o.PingTimeout = 2 * time.Second
	}
	if o.Backoff <= 0 {
		o.Backoff = 200 * time.Millisecond
	}
	if o.BackoffMax < o.Backoff {
		o.BackoffMax = o.Backoff
	}
	if o.LogEvery <= 0 {
		o.LogEvery = 5 * time.Second
	}
	return o
}

// Open 解析 MySQL DSN 连库并 Ping 到通为止，返回带连接级重试的句柄。
//
// 连接级失败（拒连/超时/断连）退避重发；永久性错误（DSN 非法、账号
// 密码错、库不存在）立即返回——这类错误等不来。ctx 到期时返回最后一次
// 连接错误。
func Open(ctx context.Context, dsn string, opt Options) (*DB, error) {
	opt = opt.filled()
	cfg, err := mysqlConfig(dsn, opt)
	if err != nil {
		return nil, fmt.Errorf("%s: open mysql: %w", opt.Name, err)
	}
	connector, err := mysql.NewConnector(cfg)
	if err != nil {
		return nil, fmt.Errorf("%s: open mysql: %w", opt.Name, err)
	}
	return OpenConnector(ctx, connector, opt)
}

// mysqlConfig 解析 DSN 并补缺省超时（DSN 已写明的保持原样）。
func mysqlConfig(dsn string, opt Options) (*mysql.Config, error) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, err
	}
	if cfg.Timeout == 0 && opt.DialTimeout != 0 {
		cfg.Timeout = opt.DialTimeout
	}
	if cfg.ReadTimeout == 0 && opt.IOTimeout != 0 {
		cfg.ReadTimeout = opt.IOTimeout
	}
	if cfg.WriteTimeout == 0 && opt.IOTimeout != 0 {
		cfg.WriteTimeout = opt.IOTimeout
	}
	return cfg, nil
}

// OpenConnector 以现成 connector 连库并 Ping 到通为止（Open 的内核；
// 测试/自定义拨号器的注入点），语义同 Open。
func OpenConnector(ctx context.Context, connector driver.Connector, opt Options) (*DB, error) {
	opt = opt.filled()
	if opt.ConnectTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opt.ConnectTimeout)
		defer cancel()
	}
	d := &DB{raw: sql.OpenDB(connector), opt: opt}
	var (
		last    error
		wait    = opt.Backoff
		start   = time.Now()
		lastLog = start
	)
	for attempt := 1; ; attempt++ {
		if last = d.raw.PingContext(ctx); last == nil {
			return d, nil
		}
		if ctx.Err() != nil || !IsConnErr(last) {
			d.raw.Close()
			if attempt > 1 {
				last = fmt.Errorf("%s: ping mysql（已重试 %d 次/%.1fs）: %w",
					opt.Name, attempt-1, time.Since(start).Seconds(), last)
			} else {
				last = fmt.Errorf("%s: ping mysql: %w", opt.Name, last)
			}
			return nil, last
		}
		switch {
		case attempt == 1:
			lastLog = time.Now()
			d.logf("%s: mysql 未就绪，等待重试: %v", opt.Name, last)
		case time.Since(lastLog) >= opt.LogEvery:
			lastLog = time.Now()
			d.logf("%s: mysql 仍未就绪（已 %.1fs，第 %d 次）: %v",
				opt.Name, time.Since(start).Seconds(), attempt, last)
		}
		if !sleep(ctx, wait) {
			// ctx 到期/取消：下一次 Ping 即以 ctx 错误落败，由循环顶收口
			continue
		}
		wait = min(wait*2, opt.BackoffMax)
	}
}

// sleep 等待 d 或 ctx 结束；返回 false = ctx 已结束。
func sleep(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// DB 是带连接级重试的 *sql.DB 门面（见包文档）。
type DB struct {
	raw *sql.DB
	opt Options

	down    atomic.Bool // 处于断连期（快速路径：成功操作不必每次抢锁）
	mu      sync.Mutex
	since   time.Time // 断连起点
	lastLog time.Time // 上次「仍不可用」日志
}

// Retry 以连接级错误为信号退避重发 fn，直到成功 / 非连接级错误 / ctx
// 到期 / RetryTimeout 耗尽（后三者返回最后一次错误）。
//
// fn 须可重入：连接级失败不等于语句没生效（可能只是回包丢失），调用方
// 自行保证幂等或容忍重复。
func (d *DB) Retry(ctx context.Context, fn func() error) error {
	var (
		err   error
		wait  = d.opt.Backoff
		start time.Time
	)
	for attempt := 0; ; attempt++ {
		if err = fn(); err == nil {
			d.recovered()
			return nil
		}
		if ctx.Err() != nil || !IsConnErr(err) {
			return err
		}
		if attempt == 0 {
			start = time.Now()
		} else if elapsed := time.Since(start); d.opt.RetryTimeout > 0 && elapsed >= d.opt.RetryTimeout {
			d.logf("%s: mysql 重试 %d 次仍失败（%.1fs），放弃: %v",
				d.opt.Name, attempt, elapsed.Seconds(), err)
			return err
		}
		d.failed(err)
		if !sleep(ctx, wait) {
			return err
		}
		wait = min(wait*2, d.opt.BackoffMax)
	}
}

// failed 记一次连接级失败：断连首拍打「连接中断」，其后按 LogEvery 限频
// 打「仍不可用」（全部 goroutine 共用一份断连状态，不逐操作刷屏）。
func (d *DB) failed(err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	now := time.Now()
	if !d.down.Swap(true) {
		d.since, d.lastLog = now, now
		d.logf("%s: mysql 连接中断，等待重试: %v", d.opt.Name, err)
		return
	}
	if now.Sub(d.lastLog) >= d.opt.LogEvery {
		d.lastLog = now
		d.logf("%s: mysql 仍不可用（已 %.1fs）: %v", d.opt.Name, now.Sub(d.since).Seconds(), err)
	}
}

// recovered 断连期后首个成功操作：摘除断连态并打「恢复」。
func (d *DB) recovered() {
	if !d.down.Load() {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.down.Swap(false) {
		return
	}
	d.logf("%s: mysql 连接恢复（中断 %.1fs）", d.opt.Name, time.Since(d.since).Seconds())
}

func (d *DB) logf(format string, args ...any) { d.opt.Logf(format, args...) }

// Ping 探活（不等待重试），整体受 PingTimeout 封顶——官方 CheckConnect
// 的 AppRecord database=Break|OK 判据，不能让定时 tick 挂在挂死的拨号上。
//
// database/sql 的坏连接换连只盖到「取连接」那一步；库重启后个别死连接
// 能骗过驱动的活性检查，直到 COM_PING 写出去才暴露，驱动回
// driver.ErrBadConn 而 PingContext 对这一步不再重试——真库杀库重启实测
// 会因此在库已恢复时误报一次 Break。所以遇 ErrBadConn 就换一条连着再探
// （纯换连，不等待；连接池空闲上限缺省 2，余量足够）。
func (d *DB) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), d.opt.PingTimeout)
	defer cancel()
	var err error
	for attempt := 0; attempt < 8; attempt++ {
		if err = d.raw.PingContext(ctx); !errors.Is(err, driver.ErrBadConn) {
			break
		}
	}
	switch {
	case err == nil:
		d.recovered()
	case IsConnErr(err):
		d.failed(err)
	}
	return err
}

// Close 关连接池（此后操作返回 sql: database is closed，不重试）。
func (d *DB) Close() error { return d.raw.Close() }

// Raw 底层 *sql.DB（不经本包重试）。只给「不可重复执行」的语句用：
// 调用方在 Retry 里自己配好「是否已生效」的前置判定再发（范例
// fssolostore.MySQLStore.createRoleOnce）。database/sql 自带的换连重发
// 仅在驱动确认一字节都没写出去时才发生，对 INSERT 是安全的。
func (d *DB) Raw() *sql.DB { return d.raw }

// Exec 等价 sql.DB.Exec。
func (d *DB) Exec(query string, args ...any) (sql.Result, error) {
	return d.ExecContext(context.Background(), query, args...)
}

// ExecContext 连接级失败退避重发后的 sql.DB.ExecContext。
func (d *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	var res sql.Result
	err := d.Retry(ctx, func() (err error) {
		res, err = d.raw.ExecContext(ctx, query, args...)
		return err
	})
	return res, err
}

// Query 等价 sql.DB.QueryContext，行集由 consume 一次消费完（不必 Close）。
//
// 执行/迭代任一环节撞上连接级错误都整句重发并重跑 consume——consume 须
// 可重入（累加器先清空再写，或只写局部变量）。它正常返回后不再重发，
// 哪怕丢弃剩余行时断连：结果已经拿到，重发只会重复消费。
func (d *DB) QueryContext(ctx context.Context, consume func(*sql.Rows) error, query string, args ...any) error {
	return d.Retry(ctx, func() error {
		rows, err := d.raw.QueryContext(ctx, query, args...)
		if err != nil {
			return err
		}
		err = consume(rows)
		rows.Close()
		return err
	})
}

// Query 等价 sql.DB.Query（context.Background）。
func (d *DB) Query(consume func(*sql.Rows) error, query string, args ...any) error {
	return d.QueryContext(context.Background(), consume, query, args...)
}

// Row 是 sql.Row 的重试版：Scan 遇连接级错误整句重发。
type Row struct {
	d     *DB
	ctx   context.Context
	query string
	args  []any
}

// Row 单行查询（等价 sql.DB.QueryRow）；无行时 Scan 返回 sql.ErrNoRows。
func (d *DB) Row(query string, args ...any) *Row {
	return &Row{d: d, ctx: context.Background(), query: query, args: args}
}

// RowContext 单行查询（等价 sql.DB.QueryRowContext）。
func (d *DB) RowContext(ctx context.Context, query string, args ...any) *Row {
	return &Row{d: d, ctx: ctx, query: query, args: args}
}

// Scan 扫描首行；连接级失败退避重发后再扫（dest 复用，重试覆盖）。
func (r *Row) Scan(dest ...any) error {
	return r.d.Retry(r.ctx, func() error {
		return r.d.raw.QueryRowContext(r.ctx, r.query, r.args...).Scan(dest...)
	})
}
