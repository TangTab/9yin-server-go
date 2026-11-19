package fswatch

import (
	"fmt"
	"strings"
	"sync"
)

// Query 子命令 → DB 消息 id（CAccess::ProcessMsg 跳转表 0x10061294 与
// ReceiveQuery stricmp 链对齐）。
const (
	MsgRequestRoles     = 0
	MsgRequestRoleData  = 1
	MsgRequestPubData   = 2
	MsgRequestChunkData = 3
	MsgRequestLogData   = 4
	MsgRequestItemLog   = 5
	MsgRequestChatLog   = 6
	MsgRequestGmLog     = 7
	MsgRequestNames     = 10
	MsgRequestSqlSelect = 11
)

// queryCommands 官方 ReceiveQuery stricmp 链（rdata 0x100f90bc 起）。
var queryCommands = map[string]int{
	"request_roles":     MsgRequestRoles,
	"request_role_data": MsgRequestRoleData,
	"request_pub_data":  MsgRequestPubData,
	"request_chunk_data": MsgRequestChunkData,
	"request_log_data":  MsgRequestLogData,
	"request_item_log":  MsgRequestItemLog,
	"request_chat_log":  MsgRequestChatLog,
	"request_gm_log":    MsgRequestGmLog,
	"request_names":     MsgRequestNames,
	"request_sql_select": MsgRequestSqlSelect,
}

// QueryResult 官方 CAccess::SendResponse(msgid, cnx, data, len) 载荷：
// msgid 对应 ProcessDbMsg 分发，publish 名即响应首串。
type QueryResult struct {
	MsgID int
	// Publish 名（ProcessDbMsg case0..9 → PublishRoles..PublishSqlSelect）。
	// 实证响应名：role（PublishRoles）/log（PublishLogData）；其余按
	// 命令族命名（role_data/pub_data/chunk_data/item_log/chat_log/gm_log/
	// names/sql_select）。
	Publish string
	Rows    [][]string // 行式结果（官方结果集行 → varlist 逐行回推）
	Err     error      // 非空 → 上层回 no_database/failed
}

// publishNames DB 消息 id → 响应名。
var publishNames = map[int]string{
	MsgRequestRoles:      "role",
	MsgRequestRoleData:   "role_data",
	MsgRequestPubData:    "pub_data",
	MsgRequestChunkData:  "chunk_data",
	MsgRequestLogData:    "log",
	MsgRequestItemLog:    "item_log",
	MsgRequestChatLog:    "chat_log",
	MsgRequestGmLog:      "gm_log",
	MsgRequestNames:      "names",
	MsgRequestSqlSelect:  "sql_select",
}

// QueryBackend 官方 IDBSystem（FsSql SqlDBSystem 注入）的 Go 面：
// CAccess 线程在背后执行查询，结果经队列回到主循环 Publish*。
type QueryBackend interface {
	// Query 执行子命令；args 为去掉命令/密码后的剩余 token。
	Query(msgID int, sub string, args []string) (*QueryResult, error)
	Close() error
}

// LocalQueryBackend 无 access.ini/无 MySQL 时的官方行为兜底：
// 查询子命令一律回 no_database（ReceiveQuery 'no_database' 分支，
// rdata 0x100f77d0f）。
type LocalQueryBackend struct{}

func (LocalQueryBackend) Query(msgID int, sub string, args []string) (*QueryResult, error) {
	return nil, fmt.Errorf("no_database")
}
func (LocalQueryBackend) Close() error { return nil }

// QueryService 官方 CQueryDb+CAccess 面：后台单线程（AccessLoopProc）
// 顺序消费请求，响应回调投递回主循环。
type QueryService struct {
	mu       sync.Mutex
	backend  QueryBackend
	pending  chan queryRequest
	stopCh   chan struct{}
	wg       sync.WaitGroup
	dropped  bool
	logf     func(string, ...any)
}

type queryRequest struct {
	msgID int
	sub   string
	args  []string
	cnx   int
	done  func(*QueryResult)
}

// NewQueryService 启动 Access 线程等价（goroutine）。
func NewQueryService(backend QueryBackend, logf func(string, ...any)) *QueryService {
	if backend == nil {
		backend = LocalQueryBackend{}
	}
	if logf == nil {
		logf = func(string, ...any) {}
	}
	q := &QueryService{
		backend: backend,
		pending: make(chan queryRequest, 256),
		stopCh:  make(chan struct{}),
		logf:    logf,
	}
	q.wg.Add(1)
	go q.loop()
	return q
}

func (q *QueryService) loop() {
	defer q.wg.Done()
	for {
		select {
		case <-q.stopCh:
			return
		case req := <-q.pending:
			res, err := q.backend.Query(req.msgID, req.sub, req.args)
			if err != nil && res == nil {
				res = &QueryResult{MsgID: req.msgID, Err: err}
			}
			res.MsgID = req.msgID
			if res.Publish == "" {
				res.Publish = publishNames[req.msgID]
			}
			if req.done != nil {
				req.done(res)
			}
		}
	}
}

// Put 官方 CAccess::PutMessage（0x60d80）：请求入队（满即丢弃——官方
// 队列有界，溢出行为为阻塞主循环之外的丢弃）。
func (q *QueryService) Put(msgID int, sub string, args []string, cnx int, done func(*QueryResult)) bool {
	q.mu.Lock()
	dropped := q.dropped
	q.mu.Unlock()
	if dropped {
		return false
	}
	select {
	case q.pending <- queryRequest{msgID: msgID, sub: sub, args: args, cnx: cnx, done: done}:
		return true
	default:
		q.logf("(CAccess)put message overflow, sub=%s", sub)
		return false
	}
}

// Stop 官方 CAccess::Stop（0x607c0）。
func (q *QueryService) Stop() {
	q.mu.Lock()
	q.dropped = true
	q.mu.Unlock()
	close(q.stopCh)
	q.wg.Wait()
	if err := q.backend.Close(); err != nil {
		q.logf("(CAccess)close backend: %v", err)
	}
}

// NormalizeSubQueryName 供测试/日志：官方 stricmp 归一。
func NormalizeSubQueryName(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
