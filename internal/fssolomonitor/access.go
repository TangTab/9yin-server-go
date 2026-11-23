package fssolomonitor

import (
	"context"
	"fmt"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/dbconn"
)

// Alert 官方 store_forw_insert_table_t 的语义投影：
// +0x00 int type；+0x04 wchar name[33]；+0x46 char reason[]。
type Alert struct {
	Type   int
	Name   string // 官方宽串名（告警对象；系统级告警为 "S"）
	Reason string // 官方本地码页 reason（Go 侧 UTF-8）
}

// AlertSink 官方 CAccess 写线程面（IDBSystem 之下）：EnsureTable →
// INSERT nx_monitor。实现须并发安全；InsertAlert 由单写 goroutine
// 串行调用（Server 内 Access 队列消费侧）。
type AlertSink interface {
	// Ping 官方 InitConnect 的 Connect+IsConnected 等价（失败 →
	// DatabaseBreak）。
	Ping() error
	// InsertAlert 官方 MessageInsertTable：
	// INSERT INTO nx_monitor VALUES('%s',%d,'%s',NOW())。
	// reason 不做 SQL 转义（官方原样，注入面=运维可控输入）。
	InsertAlert(ctx context.Context, a Alert) error
	// Close 官方 Stop。
	Close() error
}

// ---- 内存实现（测试/无库兜底）----

// MemoryAlertSink 无库兜底：告警留内存，可 JSON 落盘。
type MemoryAlertSink struct {
	mu     sync.Mutex
	alerts []Alert
	drops  int
}

// NewMemoryAlertSink 建内存 sink。
func NewMemoryAlertSink() *MemoryAlertSink { return &MemoryAlertSink{} }

// Ping 恒成功（内存无断连）。
func (s *MemoryAlertSink) Ping() error { return nil }

// InsertAlert 记录告警。
func (s *MemoryAlertSink) InsertAlert(_ context.Context, a Alert) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alerts = append(s.alerts, a)
	return nil
}

// Close no-op。
func (s *MemoryAlertSink) Close() error { return nil }

// Alerts 快照（测试用）。
func (s *MemoryAlertSink) Alerts() []Alert {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Alert, len(s.alerts))
	copy(out, s.alerts)
	return out
}

// ---- MySQL 实现（-mysql-dsn 注入）----

// nxMonitorDDL 官方建表语句原文（InitConnect 0x1011d840）。
const nxMonitorDDL = "CREATE TABLE IF NOT EXISTS nx_monitor ( name VARCHAR(128)" +
	" CHARACTER SET utf8 COLLATE utf8_bin, type INT, reason TEXT, time DATETIME" +
	" ) ENGINE=InnoDB DEFAULT CHARSET=utf8"

// MySQLAlertSink 官方 CAccess→IDBSystem 的 database/sql 投影。连接经
// internal/dbconn：库重启/闪断窗口内 INSERT 挂起退避重发（官方 retry<2
// 的等价）；流水行容忍重复，逐语句重发即可。
type MySQLAlertSink struct {
	db *dbconn.DB
}

// OpenMySQLAlertSink 连库并确保 nx_monitor 存在（库在重启就等它起来，
// 语义见 dbconn.Open；建表失败不致命，仅日志）。
func OpenMySQLAlertSink(ctx context.Context, dsn string, logf func(string, ...any)) (*MySQLAlertSink, error) {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	db, err := dbconn.Open(ctx, dsn, dbconn.NewOptions("fssolomonitor", logf))
	if err != nil {
		return nil, err
	}
	if _, err := db.ExecContext(ctx, nxMonitorDDL); err != nil {
		logf("(CAccess::InitConnect)create nx_monitor failed: %v", err)
	}
	return &MySQLAlertSink{db: db}, nil
}

// Ping 连接检查（探活不等待）。
func (s *MySQLAlertSink) Ping() error { return s.db.Ping() }

// InsertAlert 官方 INSERT 原文（NOW() 原生；reason 不转义）。受 ctx
// 约束：Access 写线程停机即取消，不卡在重试里。
func (s *MySQLAlertSink) InsertAlert(ctx context.Context, a Alert) error {
	stmt := fmt.Sprintf("INSERT INTO nx_monitor VALUES('%s',%d,'%s',NOW())",
		a.Name, a.Type, a.Reason)
	_, err := s.db.ExecContext(ctx, stmt)
	return err
}

// Close 关连接。
func (s *MySQLAlertSink) Close() error { return s.db.Close() }

// ---- Access：官方 CAccess 写线程（CTaskQueue 32MB 双缓冲 → chan）----

// accessQueueSize 官方 32MB 双缓冲的 Go 等价容量（条数近似）。
const accessQueueSize = 8192

// Access 复刻 CAccess（access.cpp）：单写 goroutine 消费告警队列，
// 落库失败 DatabaseError++（Server 侧），断连 DatabaseBreak。
// 官方队列满自旋 Sleep(0)；Go 侧满则丢弃并 trace（差异见包文档）。
type Access struct {
	sink    AlertSink
	queue   chan Alert
	logf    func(format string, args ...any)
	onErr   func() // DatabaseError++
	onBreak func()

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewAccess 建写线程（Start 内拉起 goroutine）。
func NewAccess(sink AlertSink, logf func(string, ...any), onErr, onBreak func()) *Access {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	ctx, cancel := context.WithCancel(context.Background())
	a := &Access{
		sink: sink, queue: make(chan Alert, accessQueueSize),
		logf: logf, onErr: onErr, onBreak: onBreak,
		ctx: ctx, cancel: cancel,
	}
	a.wg.Add(1)
	go a.loop()
	return a
}

// Start 官方 CAccess::Start：InitConnect（失败仅 trace 不死）。
// 写线程随构造常在（官方线程在 Start 拉起；Go 侧队列消费面提前，
// 保证未启动网络的进程内告警也能落库）。
func (a *Access) Start() error {
	if err := a.sink.Ping(); err != nil {
		if a.onBreak != nil {
			a.onBreak()
		}
		a.logf("(CAccess::Start)init connect failed: %v", err)
	}
	return nil
}

// Ping 探活透传（Server 的 Break 自愈用；受 dbconn PingTimeout 封顶）。
func (a *Access) Ping() error { return a.sink.Ping() }

// PutMessage 官方 PutMessage(0,0,buf,len) 的非阻塞形态：队满丢弃+trace。
func (a *Access) PutMessage(alert Alert) {
	select {
	case a.queue <- alert:
	default:
		a.logf("(CAccess::PutMessage)queue full, alert dropped: type=%d role=%s", alert.Type, alert.Name)
	}
}

// loop 官方 DoWait 线程体：SwapMsgVector → Peek → ProcessMsg。
func (a *Access) loop() {
	defer a.wg.Done()
	for {
		select {
		case <-a.ctx.Done():
			for {
				select {
				case alert := <-a.queue:
					a.process(alert)
				default:
					return
				}
			}
		case alert := <-a.queue:
			a.process(alert)
		}
	}
}

// process 官方 ProcessMsg（msgid=0 → MessageInsertTable）。
func (a *Access) process(alert Alert) {
	if err := a.sink.InsertAlert(a.ctx, alert); err != nil {
		if a.onErr != nil {
			a.onErr()
		}
		if err := a.sink.Ping(); err != nil && a.onBreak != nil {
			a.onBreak()
		}
	}
}

// Stop 官方 Stop：停线程 + Close（失败仅 trace）。
func (a *Access) Stop() {
	a.cancel()
	a.wg.Wait()
	if err := a.sink.Close(); err != nil {
		a.logf("(CAccess::Stop)stop failed: %v", err)
	}
}
