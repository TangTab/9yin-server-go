package fssolostore

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

// MySQLStore 是 Backend 的 MySQL 实现（-mysql-dsn 注入；官方 SQLOLEDB
// 方言 → Go 统一 MySQL，SQL 文本零改动——docs §五 逐字）。
//
// 已建模命令走官方 SQL 原文；依赖 store_forw_* C 结构体布局的角色/
// 信件族（Payload 解码未逆向）返回 ErrUnsupported（Server 仅 trace）。
type MySQLStore struct {
	db      *sql.DB
	suffix  string
	logf    func(format string, args ...any)
	retries int
}

// OpenMySQLStore 连库并确保基础表存在（官方 DDL 不在快照内，按 SQL
// 所需列最小集补建，IF NOT EXISTS）。
func OpenMySQLStore(ctx context.Context, dsn string, logf func(string, ...any)) (*MySQLStore, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("fssolostore: open mysql: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("fssolostore: ping mysql: %w", err)
	}
	if logf == nil {
		logf = func(string, ...any) {}
	}
	s := &MySQLStore{db: db, logf: logf, retries: 2}
	if err := s.ensureTables(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *MySQLStore) ensureTables(ctx context.Context) error {
	ddls := []string{
		"CREATE TABLE IF NOT EXISTS nx_config ( id VARCHAR(64) NOT NULL PRIMARY KEY," +
			" name MEDIUMBLOB, type INT, property MEDIUMBLOB, status INT, op_name VARCHAR(255)," +
			" op_type INT, op_property MEDIUMBLOB, op_status INT, op_version INT ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		"CREATE TABLE IF NOT EXISTS nx_chunks ( serial_no VARCHAR(64) NOT NULL, name VARCHAR(255)," +
			" server_id INT, scene_id INT, create_time DATETIME, deleted INT, save_time DATETIME," +
			" save_data LONGBLOB, PRIMARY KEY (serial_no, server_id) ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		"CREATE TABLE IF NOT EXISTS nx_auction ( name VARCHAR(255) NOT NULL PRIMARY KEY, uid VARCHAR(64)," +
			" create_time DATETIME, type CHAR(1), record VARCHAR(16), version INT, amount INT," +
			" `range` INT, save_time DATETIME, save_data LONGBLOB ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
	}
	for _, q := range ddls {
		if _, err := s.db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("fssolostore: ensure table: %w", err)
		}
	}
	return nil
}

func (s *MySQLStore) Ping() error  { return s.db.Ping() }
func (s *MySQLStore) Close() error { return s.db.Close() }

func (s *MySQLStore) SplitSuffix() string { return s.suffix }

// PrepareSplitTables 官方 SHOW TABLES LIKE '%s%s' 缺则建（分周表）。
func (s *MySQLStore) PrepareSplitTables(suffix string) (string, error) {
	if suffix == "" {
		return "", nil
	}
	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx, "SHOW TABLES")
	if err != nil {
		return "", err
	}
	defer rows.Close()
	existing := map[string]bool{}
	for rows.Next() {
		var n string
		if rows.Scan(&n) == nil {
			existing[n] = true
		}
	}
	for _, t := range CustomLogTables() {
		full := t.Name + "_" + suffix
		if existing[full] {
			continue
		}
		cols := make([]string, 0, len(t.Fields)+1)
		cols = append(cols, "log_id BIGINT AUTO_INCREMENT PRIMARY KEY")
		for _, f := range t.Fields {
			sqlType := map[string]string{
				"datetime": "DATETIME", "nvarchar": fmt.Sprintf("VARCHAR(%d)", f.Size),
				"varchar": fmt.Sprintf("VARCHAR(%d)", f.Size),
				"decimal": fmt.Sprintf("DECIMAL(%d,2)", f.Size), "int": "INT",
			}[f.Type]
			cols = append(cols, "`"+f.Name+"` "+sqlType)
		}
		q := "CREATE TABLE IF NOT EXISTS `" + full + "` (" + strings.Join(cols, ", ") +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4"
		if _, err := s.db.ExecContext(ctx, q); err != nil {
			s.logf("(CSyncAccess::CreateLogTable)failed %s: %v", full, err)
			return "", err
		}
	}
	s.suffix = suffix
	return suffix, nil
}

// ---- Manage（nx_config；官方 SELECT … WHERE op_name <> ''）----

func (s *MySQLStore) ManageInfos() ([]ManageEntry, error) {
	rows, err := s.db.Query("SELECT id, op_name, op_type, op_property, op_status, op_version FROM nx_config WHERE op_name <> ''")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ManageEntry
	for rows.Next() {
		var e ManageEntry
		var opName, opProp sql.RawBytes
		if err := rows.Scan(&e.ID, &opName, &e.Type, &opProp, &e.Status, &e.Order); err != nil {
			return nil, err
		}
		e.Name = string(opName)
		e.Comment = string(opProp)
		e.State = e.Status
		out = append(out, e)
	}
	return out, rows.Err()
}

// UpsertManage 官方 MessageManageBase 的 INSERT/UPDATE 对（nx_config）。
func (s *MySQLStore) UpsertManage(e ManageEntry) error {
	ctx := context.Background()
	var n int
	if err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) as count FROM nx_config WHERE id = '%s'", e.ID).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		_, err := s.db.ExecContext(ctx, "INSERT INTO nx_config ( id, name, type, property, status,"+
			" op_name, op_type, op_property, op_status, op_version ) VALUES ( '%s', ?, %d, ?, %d, '', 0, '', 0, 0 )",
			e.ID, e.Type, e.Status)
		return err
	}
	_, err := s.db.ExecContext(ctx, "UPDATE nx_config SET name = ?, type = %d, property = ?, status = %d WHERE id = '%s'",
		e.Type, e.Status, e.ID)
	return err
}

// ---- Chunks（nx_chunks；官方四句原文）----

func (s *MySQLStore) ChunkIDs(sceneID uint32) ([]uint32, error) {
	rows, err := s.db.Query("SELECT name FROM nx_chunks WHERE server_id = %d AND scene_id = %d",
		serverIDForScene(sceneID), sceneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uint32
	for rows.Next() {
		var name string
		if rows.Scan(&name) == nil {
			var id uint32
			fmt.Sscan(name, &id)
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, rows.Err()
}

func (s *MySQLStore) LoadChunk(sceneID uint32, ids []uint32) (map[uint32][]byte, error) {
	out := map[uint32][]byte{}
	for _, id := range ids {
		var data []byte
		err := s.db.QueryRow("SELECT save_data FROM nx_chunks WHERE serial_no = '%s' AND server_id = %d ",
			chunkSerial(id), serverIDForScene(sceneID)).Scan(&data)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return nil, err
		}
		out[id] = data
	}
	return out, nil
}

func (s *MySQLStore) SaveChunk(sceneID uint32, id uint32, data []byte) error {
	ctx := context.Background()
	var name string
	err := s.db.QueryRowContext(ctx,
		"SELECT name FROM nx_chunks WHERE serial_no = '%s' AND server_id = %d ",
		chunkSerial(id), serverIDForScene(sceneID)).Scan(&name)
	switch {
	case err == sql.ErrNoRows:
		_, err = s.db.ExecContext(ctx, "INSERT INTO nx_chunks ( serial_no, name, server_id, scene_id,"+
			" create_time, deleted, save_time, save_data ) VALUES ( '%s', '%s', %d, %d, NOW(), 0, NOW(), ? )",
			chunkSerial(id), chunkSerial(id), serverIDForScene(sceneID), sceneID, data)
		return err
	case err != nil:
		return err
	default:
		_, err = s.db.ExecContext(ctx, "UPDATE nx_chunks SET name = '%s', scene_id = %d, save_time = NOW(),"+
			" save_data = ? WHERE serial_no = '%s' AND server_id = %d ", chunkSerial(id), sceneID,
			chunkSerial(id), serverIDForScene(sceneID), data)
		return err
	}
}

func (s *MySQLStore) DeleteChunk(sceneID uint32, id uint32) error {
	_, err := s.db.Exec("DELETE FROM nx_chunks WHERE serial_no = '%s' AND server_id = %d ",
		chunkSerial(id), serverIDForScene(sceneID))
	return err
}

func chunkSerial(id uint32) string { return utoa(uint64(id)) }

// serverIDForScene 场景号高 16 位派生（member scene id 布局约定）。
func serverIDForScene(sceneID uint32) int {
	if sceneID>>16 != 0 {
		return int(sceneID >> 16)
	}
	return 1
}

// ---- Pub / Auction / Account / Roles / Log ----

func (s *MySQLStore) PubInfo() ([]string, error) {
	return nil, ErrUnsupported
}

func (s *MySQLStore) PubLoad(table string) (map[string][]byte, error) {
	rows, err := s.db.Query("SELECT name, save_data FROM " + table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]byte{}
	for rows.Next() {
		var name string
		var data []byte
		if rows.Scan(&name, &data) == nil {
			out[name] = data
		}
	}
	return out, rows.Err()
}

func (s *MySQLStore) PubSave(table, name string, saveData []byte) error {
	_, err := s.db.Exec("UPDATE "+table+" SET save_time = NOW(), save_data = ? WHERE name = ?", saveData, name)
	return err
}

func (s *MySQLStore) PubAdd(table, name, uid string, saveData []byte) (bool, error) {
	return false, ErrUnsupported
}

func (s *MySQLStore) PubRemove(table, name string) (bool, error) {
	res, err := s.db.Exec("DELETE FROM %s WHERE name = '%s'", table, name)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (s *MySQLStore) AuctionLoad(kind AuctionKind) (map[string][]byte, error) {
	q := "SELECT * FROM nx_auction WHERE type = '1'"
	if kind == AuctionRec {
		q = "SELECT * FROM nx_auction WHERE type = '2'"
	}
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	out := map[string][]byte{}
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if rows.Scan(ptrs...) != nil {
			continue
		}
		name, _ := vals[0].(string)
		if d, ok := vals[len(vals)-1].([]byte); ok && name != "" {
			out[name] = d
		}
	}
	return out, rows.Err()
}

func (s *MySQLStore) AuctionSave(kind AuctionKind, name string, saveData []byte) error {
	tp := "1"
	if kind == AuctionRec {
		tp = "2"
	}
	var n int
	s.db.QueryRow("SELECT COUNT(*) FROM nx_auction WHERE name = '%s'", name).Scan(&n)
	if n == 0 {
		_, err := s.db.Exec("INSERT INTO nx_auction ( name, uid, create_time, type, record, version,"+
			" amount, `range`, save_time, save_data ) VALUES ( ?, '', NOW(), ?, 'attr', 0, 0, 0, NOW(), ? )", name, tp, saveData)
		return err
	}
	_, err := s.db.Exec("UPDATE nx_auction SET save_time = NOW(), save_data = ? WHERE name = ?", saveData, name)
	return err
}

func (s *MySQLStore) AccountLogon(name string, logged int) (bool, error) {
	return false, ErrUnsupported
}

func (s *MySQLStore) AccountLogoff(name string, logged int) (bool, error) {
	return false, ErrUnsupported
}

func (s *MySQLStore) ResetOnline(serverID int) ([]RoleBrief, error) {
	if _, err := s.db.Exec("UPDATE nx_roles SET online = 0 WHERE online = 1 AND server_id = %d", serverID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query("SELECT name, uid, deleted FROM nx_roles WHERE server_id = %d", serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RoleBrief
	for rows.Next() {
		var b RoleBrief
		var del int
		if rows.Scan(&b.Name, &b.UID, &del) == nil {
			b.Deleted = del != 0
			out = append(out, b)
		}
	}
	return out, rows.Err()
}

func (s *MySQLStore) SaveRole(name string, meta RoleMeta, saveData []byte) error {
	return ErrUnsupported
}

func (s *MySQLStore) LoadRole(account, name string, serverID int) ([]byte, RoleMeta, bool, error) {
	return nil, RoleMeta{}, false, ErrUnsupported
}

func (s *MySQLStore) CreateRole(account, name, uid string, serverID int, meta RoleMeta, saveData []byte) (bool, error) {
	return false, ErrUnsupported
}

func (s *MySQLStore) SetOnline(name string, on bool) error {
	v := 1
	if !on {
		v = 0
	}
	_, err := s.db.Exec("UPDATE nx_roles SET online = %d WHERE name = '%s' AND online <> %d", v, name, v)
	return err
}

func (s *MySQLStore) QueryAccount(account string, serverID int) ([]RoleBrief, error) {
	return nil, ErrUnsupported
}

// InsertLog 官方动态 INSERT INTO %s (%s) VALUES (%s)（CustomLog）：
// datetime 列自动 NOW()，数值/串按 LogValue。
func (s *MySQLStore) InsertLog(table string, vals []LogValue) error {
	t := (&LogTableSet{}).Table(table)
	if t == nil {
		return fmt.Errorf("(CAccess::MessageCustomLog)table not define %s", table)
	}
	cols := make([]string, 0, len(t.Fields))
	args := make([]any, 0, len(t.Fields))
	ph := make([]string, 0, len(t.Fields))
	for i, f := range t.Fields {
		if i >= len(vals) {
			break
		}
		cols = append(cols, "`"+f.Name+"`")
		if f.Type == "datetime" {
			ph = append(ph, "NOW()")
			continue
		}
		ph = append(ph, "?")
		v := vals[i]
		switch {
		case v.IsNum && !v.IsFloat:
			args = append(args, v.Int)
		case v.IsFloat:
			args = append(args, v.Float)
		default:
			args = append(args, v.Text)
		}
	}
	q := "INSERT INTO `" + table + "_" + s.suffix + "` ( " + strings.Join(cols, ", ") + " ) VALUES ( " +
		strings.Join(ph, ", ") + " )"
	_, err := s.db.Exec(q, args...)
	return err
}

func (s *MySQLStore) Generic(msg uint32, payload []byte) error { return ErrUnsupported }

var _ Backend = (*MySQLStore)(nil)
