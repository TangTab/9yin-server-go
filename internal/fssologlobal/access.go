package fssologlobal

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DBAccess 是官方 CAccess→IDBAccess vtbl 面（access.cpp）的 Go 等价：
// 名字登记/释放 + 连接健康检查。官方由 DBSystem 实体
// （main.lua: nx_create("MysqlDBSystem")）派发，本进程内注入实现。
type DBAccess interface {
	// AddName 官方 AddName：名不存在 → INSERT（登记成功 true）；
	// 名已存在 → false；owner/serverID 原样入库。
	AddName(table, name, owner string, serverID int) (bool, error)
	// RemoveName 官方 RemoveName：名存在 → DELETE（true）；不存在 → false。
	RemoveName(table, name string) (bool, error)
	// Ping 官方 CheckConnect 面（连接自愈后应可用）。
	Ping() error
	// Close 官方 Stop。
	Close() error
}

// SQL 三句原文（rdata 0x1003b498/0x1003b504/0x1003b5a4 逐字节）。
const (
	sqlSelectName = "SELECT name FROM %s WHERE name = '%s' "
	sqlInsertName = "INSERT INTO %s ( name, owner, server_id, create_time ) VALUES ( '%s', '%s', %d, NOW() )"
	sqlDeleteName = "DELETE FROM %s WHERE name = '%s' "
)

// SQLStore 是 MySQL 落地的 DBAccess（database/sql）。
// 官方快照连接串为 SQLOLEDB 方言，Go 复刻统一走 MySQL DSN；
// SQL 文本三句零改动（NOW() MySQL 原生）。
type SQLStore struct {
	db      *sql.DB
	retries int // 官方 FailCanRetry 上限（m_FailCount < 2）
	logf    func(string, ...any)
}

// OpenSQLStore 连 MySQL 并确保名字表存在。
// 缺表时按官方 SQL 所需列建表（快照未携带该库 DDL：
// name 唯一键 + owner/server_id/create_time）。
func OpenSQLStore(ctx context.Context, dsn string, logf func(string, ...any)) (*SQLStore, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("fssologlobal: open mysql: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("fssologlobal: ping mysql: %w", err)
	}
	if logf == nil {
		logf = func(string, ...any) {}
	}
	return &SQLStore{db: db, retries: 2, logf: logf}, nil
}

// EnsureTable 按官方列序建缺表（IF NOT EXISTS，utf8mb4）。
func (s *SQLStore) EnsureTable(ctx context.Context, table string) error {
	if !validIdent(table) {
		return fmt.Errorf("fssologlobal: 表名非法：%q", table)
	}
	const ddl = "CREATE TABLE IF NOT EXISTS `%s` (" +
		" `name` varchar(64) NOT NULL," +
		" `owner` varchar(64) NOT NULL DEFAULT ''," +
		" `server_id` int NOT NULL DEFAULT 0," +
		" `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP," +
		" PRIMARY KEY (`name`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4"
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(ddl, table))
	return err
}

// AddName 官方语义 + 失败重试（连接断开重连后最多 retries 次）。
func (s *SQLStore) AddName(table, name, owner string, serverID int) (bool, error) {
	if !validIdent(table) {
		return false, fmt.Errorf("fssologlobal: 表名非法：%q", table)
	}
	var lastErr error
	for attempt := 0; attempt <= s.retries; attempt++ {
		ok, err := s.addNameOnce(table, name, owner, serverID)
		if err == nil {
			return ok, nil
		}
		lastErr = err
		s.logf("(CAccess::AddName)failed: %v（第 %d 次重试）", err, attempt+1)
		if pingErr := s.db.Ping(); pingErr != nil {
			// 等价官方 FailCanRetry 放弃路径
			return false, fmt.Errorf("fssologlobal: add_name connection error: %w", pingErr)
		}
	}
	return false, lastErr
}

func (s *SQLStore) addNameOnce(table, name, owner string, serverID int) (bool, error) {
	q := fmt.Sprintf(sqlSelectName, table, sqlEscape(name))
	rows, err := s.db.Query(q)
	if err != nil {
		return false, err
	}
	taken := rows.Next()
	rows.Close()
	if taken {
		return false, nil // 名已登记
	}
	ins := fmt.Sprintf(sqlInsertName, table, sqlEscape(name), sqlEscape(owner), serverID)
	res, err := s.db.Exec(ins)
	if err != nil {
		return false, err
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return false, nil
	}
	return true, nil
}

// RemoveName 官方语义 + 同款重试。
func (s *SQLStore) RemoveName(table, name string) (bool, error) {
	if !validIdent(table) {
		return false, fmt.Errorf("fssologlobal: 表名非法：%q", table)
	}
	var lastErr error
	for attempt := 0; attempt <= s.retries; attempt++ {
		ok, err := s.removeNameOnce(table, name)
		if err == nil {
			return ok, nil
		}
		lastErr = err
		s.logf("(CAccess::RemoveName)failed: %v（第 %d 次重试）", err, attempt+1)
		if pingErr := s.db.Ping(); pingErr != nil {
			return false, fmt.Errorf("fssologlobal: remove_name connection error: %w", pingErr)
		}
	}
	return false, lastErr
}

func (s *SQLStore) removeNameOnce(table, name string) (bool, error) {
	q := fmt.Sprintf(sqlSelectName, table, sqlEscape(name))
	rows, err := s.db.Query(q)
	if err != nil {
		return false, err
	}
	exists := rows.Next()
	rows.Close()
	if !exists {
		return false, nil // 名不存在
	}
	del := fmt.Sprintf(sqlDeleteName, table, sqlEscape(name))
	res, err := s.db.Exec(del)
	if err != nil {
		return false, err
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return false, nil
	}
	return true, nil
}

func (s *SQLStore) Ping() error { return s.db.Ping() }

func (s *SQLStore) Close() error { return s.db.Close() }

// validIdent 限制表名为官方域表可表达的字符集（字母数字下划线），
// 拼接 SQL 前的兜底门。
func validIdent(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
		default:
			return false
		}
	}
	return true
}

// sqlEscape 官方以 '%s' 直拼 SQL；此处对单引号/反斜杠做 MySQL
// 转义（其余字节原样，保持 SQL 文本形状）。
func sqlEscape(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `'`, `\'`)
	return r.Replace(s)
}

// LocalStore 是无外部数据库时的兜底 DBAccess（Go 扩展；官方持久化
// 完全依赖 DBSystem 实体直连数据库）。语义与官方 SQL 三句一致：
// AddName 名不存在才登记；RemoveName 名存在才释放；精确匹配
// （真实库的大小写折叠取决于服务端 collation，此处保守取精确）。
// 可选 path 时落 JSON 快照（每次变更即写，名字表数据量小）。
type LocalStore struct {
	mu   sync.Mutex
	path string
	logf func(string, ...any)
	reg  map[string]map[string]localEntry // table → name → entry
}

type localEntry struct {
	Owner      string    `json:"owner"`
	ServerID   int       `json:"server_id"`
	CreateTime time.Time `json:"create_time"`
}

// localSnapshot 是 JSON 落盘形态。
type localSnapshot struct {
	Tables map[string]map[string]localEntry `json:"tables"`
}

// NewLocalStore 建本地名册；path 非空时先装载既有快照。
func NewLocalStore(path string, logf func(string, ...any)) (*LocalStore, error) {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	s := &LocalStore{path: path, logf: logf, reg: map[string]map[string]localEntry{}}
	if path == "" {
		return s, nil
	}
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		var snap localSnapshot
		if jerr := json.Unmarshal(data, &snap); jerr != nil {
			return nil, fmt.Errorf("fssologlobal: 解析本地名册 %s: %w", path, jerr)
		}
		if snap.Tables != nil {
			s.reg = snap.Tables
		}
		s.logf("local store: 装载快照 %s（%d 表）", path, len(s.reg))
	case errors.Is(err, os.ErrNotExist):
		// 首跑无快照
	default:
		return nil, err
	}
	return s, nil
}

// AddName 官方语义（名不存在才登记）。
func (s *LocalStore) AddName(table, name, owner string, serverID int) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if hasName(s.reg[table], name) {
		return false, nil
	}
	if s.reg[table] == nil {
		s.reg[table] = map[string]localEntry{}
	}
	s.reg[table][name] = localEntry{Owner: owner, ServerID: serverID, CreateTime: time.Now()}
	return true, s.persistLocked()
}

// RemoveName 官方语义（名存在才释放）。
func (s *LocalStore) RemoveName(table, name string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !hasName(s.reg[table], name) {
		return false, nil
	}
	delete(s.reg[table], name)
	return true, s.persistLocked()
}

func hasName(m map[string]localEntry, name string) bool {
	_, ok := m[name]
	return ok
}

func (s *LocalStore) Ping() error { return nil }

func (s *LocalStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.persistLocked()
}

func (s *LocalStore) persistLocked() error {
	if s.path == "" {
		return nil
	}
	data, err := json.MarshalIndent(localSnapshot{Tables: s.reg}, "", "  ")
	if err != nil {
		return err
	}
	// 快照目录可能不存在（首跑 data/global/…），建父目录后落盘。
	if dir := filepath.Dir(s.path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(s.path, data, 0o644)
}

// Count 呈现/测试用：某表现行数。
func (s *LocalStore) Count(table string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.reg[table])
}
