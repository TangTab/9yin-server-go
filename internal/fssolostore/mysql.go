package fssolostore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	// MySQLError 并发抢注落败判定（nx_roles 主键 1062）。
	"github.com/go-sql-driver/mysql"

	"github.com/lizhanyin/9yin-server-go/internal/dbconn"
)

// splitTableTimeout 分周表预建的单次调用上限。它跑在 store 的定时 tick
// 里（Opened 首拍），不能跟着语句重试一起挂在库重启窗口上——库没好就
// 到点放弃，下一拍再来。
const splitTableTimeout = 3 * time.Second

// MySQLStore 是 Backend 的 MySQL 实现（-mysql-dsn 注入；官方 SQLOLEDB
// 方言 → Go 统一 MySQL，SQL 文本零改动——docs §五 逐字）。
//
// 连接经 internal/dbconn：库重启/闪断窗口内语句挂起退避重发（官方
// CheckConnect 重连 + FailCanRetry 重试的等价），只有重试耗尽才以 error
// 返回。可重发性：官方 SQL 全是 upsert/幂等 UPDATE；INSERT 族里
// AccountLogon 首登建号有 SELECT 门兜住，CreateRole 建角自带前置门
// （createRoleOnce），InsertLog 流水容忍重复行。
//
// 已建模命令走官方 SQL 原文；依赖 store_forw_* C 结构体布局的角色/
// 信件族（Payload 解码未逆向）返回 ErrUnsupported（Server 仅 trace）。
type MySQLStore struct {
	db     *dbconn.DB
	suffix string
	logf   func(format string, args ...any)
}

// OpenMySQLStore 连库并确保基础表存在（官方 DDL 不在快照内，按 SQL
// 所需列最小集补建，IF NOT EXISTS）。
//
// 库正在重启/暂不可达时等它起来（连接级失败退避重连，缺省不限时；
// 密码错、库不存在这类等不来的错误立即返回）。要限时就传带 deadline
// 的 ctx，或设 NINEYIN_DB_CONNECT_TIMEOUT。
func OpenMySQLStore(ctx context.Context, dsn string, logf func(format string, args ...any)) (*MySQLStore, error) {
	if logf == nil {
		logf = func(format string, args ...any) {}
	}
	db, err := dbconn.Open(ctx, dsn, dbconn.NewOptions("fssolostore", logf))
	if err != nil {
		return nil, err
	}
	s := newMySQLStore(db, logf)
	if err := s.ensureTables(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// newMySQLStore 以现成连接装配（OpenMySQLStore 的内核；测试经
// dbconntest 假库注入重启窗口）。
func newMySQLStore(db *dbconn.DB, logf func(format string, args ...any)) *MySQLStore {
	if logf == nil {
		logf = func(format string, args ...any) {}
	}
	return &MySQLStore{db: db, logf: logf}
}

func (s *MySQLStore) ensureTables(ctx context.Context) error {
	ddls := []string{
		"CREATE TABLE IF NOT EXISTS nx_config ( id VARCHAR(64) NOT NULL PRIMARY KEY," +
			" name MEDIUMBLOB, type INT, property MEDIUMBLOB, status INT, op_name VARCHAR(255)," +
			" op_type INT, op_property MEDIUMBLOB, op_status INT, op_version INT ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		"CREATE TABLE IF NOT EXISTS nx_chunks ( serial_no VARCHAR(64) NOT NULL, name VARCHAR(255)," +
			" server_id INT, scene_id INT, create_time DATETIME, deleted INT, save_time DATETIME," +
			" save_data LONGBLOB, PRIMARY KEY (serial_no, server_id) ) ENGINE=InnoDB DEFAULT CHARSET=utf8",
		"CREATE TABLE IF NOT EXISTS nx_auction ( name VARCHAR(255) NOT NULL PRIMARY KEY, uid VARCHAR(64)," +
			" create_time DATETIME, type CHAR(1), record VARCHAR(16), version INT, amount INT," +
			" `range` INT, save_time DATETIME, save_data LONGBLOB ) ENGINE=InnoDB DEFAULT CHARSET=utf8",
		// nx_roles 官方 22 列（docs §五 INSERT 原文；uid char[33]/account
		// char[33] 槽宽，name 唯一 = 重名 SELECT 语义）
		"CREATE TABLE IF NOT EXISTS nx_roles ( name VARCHAR(64) NOT NULL PRIMARY KEY," +
			" uid VARCHAR(33) NOT NULL, account VARCHAR(33) NOT NULL," +
			" create_time DATETIME, delete_time DATETIME, open_time DATETIME," +
			" server_id INT NOT NULL DEFAULT 0, total_sec INT NOT NULL DEFAULT 0," +
			" deleted INT NOT NULL DEFAULT 0, letter_max INT NOT NULL DEFAULT 0," +
			" sys_flags INT NOT NULL DEFAULT 0, illegals INT NOT NULL DEFAULT 0," +
			" online INT NOT NULL DEFAULT 0, role_index INT NOT NULL DEFAULT 0," +
			" scene_id INT NOT NULL DEFAULT 0, location VARCHAR(255)," +
			" resume_time DATETIME, resume_info VARCHAR(255), charge_info VARCHAR(255)," +
			" save_para1 TEXT, save_time DATETIME, save_data LONGBLOB," +
			" KEY idx_account (account, server_id) ) ENGINE=InnoDB DEFAULT CHARSET=utf8",
	}
	for _, q := range ddls {
		if _, err := s.db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("fssolostore: ensure table: %w", err)
		}
	}
	// 建号状态列（2026-11-21 建号数据落库决策）：JSON 存五维/内功/技能/
	// 轻功/任务门（rolestate.go）。幂等——已存在列报 1060 忽略。
	if _, err := s.db.ExecContext(ctx,
		"ALTER TABLE nx_roles ADD COLUMN role_state_json LONGTEXT NULL"); err != nil &&
		!strings.Contains(err.Error(), "1060") && !strings.Contains(err.Error(), "Duplicate column") {
		return fmt.Errorf("fssolostore: ensure role_state_json: %w", err)
	}
	return nil
}

// Ping 官方 CheckConnect 面：只探活不等待（dbconn 封顶 PingTimeout），
// TimerCheck 的 AppRecord database=Break|OK 判据。
func (s *MySQLStore) Ping() error  { return s.db.Ping() }
func (s *MySQLStore) Close() error { return s.db.Close() }

func (s *MySQLStore) SplitSuffix() string { return s.suffix }

// PrepareSplitTables 官方 SHOW TABLES LIKE '%s%s' 缺则建（分周表）。
func (s *MySQLStore) PrepareSplitTables(suffix string) (string, error) {
	if suffix == "" {
		return "", nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), splitTableTimeout)
	defer cancel()
	existing := map[string]bool{}
	err := s.db.QueryContext(ctx, func(rows *sql.Rows) error {
		existing = map[string]bool{} // 重发时整表重建
		for rows.Next() {
			var n string
			if err := rows.Scan(&n); err != nil {
				return err
			}
			existing[n] = true
		}
		return rows.Err()
	}, "SHOW TABLES")
	if err != nil {
		return "", err
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
	var out []ManageEntry
	err := s.db.Query(func(rows *sql.Rows) error {
		out = out[:0] // 重发时整表重建
		for rows.Next() {
			var e ManageEntry
			var opName, opProp sql.RawBytes
			if err := rows.Scan(&e.ID, &opName, &e.Type, &opProp, &e.Status, &e.Order); err != nil {
				return err
			}
			e.Name = string(opName)
			e.Comment = string(opProp)
			e.State = e.Status
			out = append(out, e)
		}
		return rows.Err()
	}, "SELECT id, op_name, op_type, op_property, op_status, op_version FROM nx_config WHERE op_name <> ''")
	if err != nil {
		return nil, err
	}
	return out, nil
}

// UpsertManage 官方 MessageManageBase 的 INSERT/UPDATE 对（nx_config）。
// 官方 SQL 原文是 sprintf 方言——database/sql 侧统一转 `?` 占位（值位）。
func (s *MySQLStore) UpsertManage(e ManageEntry) error {
	ctx := context.Background()
	var n int
	if err := s.db.RowContext(ctx,
		"SELECT COUNT(*) as count FROM nx_config WHERE id = ?", e.ID).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		_, err := s.db.ExecContext(ctx, "INSERT INTO nx_config ( id, name, type, property, status,"+
			" op_name, op_type, op_property, op_status, op_version ) VALUES ( ?, ?, ?, ?, ?, '', 0, '', 0, 0 )",
			e.ID, e.Name, e.Type, e.Comment, e.Status)
		return err
	}
	_, err := s.db.ExecContext(ctx, "UPDATE nx_config SET name = ?, type = ?, property = ?, status = ? WHERE id = ?",
		e.Name, e.Type, e.Comment, e.Status, e.ID)
	return err
}

// ---- Chunks（nx_chunks；官方四句原文）----

func (s *MySQLStore) ChunkIDs(sceneID uint32) ([]uint32, error) {
	var ids []uint32
	err := s.db.Query(func(rows *sql.Rows) error {
		ids = ids[:0] // 重发时整表重建
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				return err
			}
			var id uint32
			fmt.Sscan(name, &id)
			ids = append(ids, id)
		}
		return rows.Err()
	}, "SELECT name FROM nx_chunks WHERE server_id = ? AND scene_id = ?",
		serverIDForScene(sceneID), sceneID)
	if err != nil {
		return nil, err
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}

func (s *MySQLStore) LoadChunk(sceneID uint32, ids []uint32) (map[uint32][]byte, error) {
	out := map[uint32][]byte{}
	for _, id := range ids {
		var data []byte
		err := s.db.Row("SELECT save_data FROM nx_chunks WHERE serial_no = ? AND server_id = ?",
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
	err := s.db.RowContext(ctx,
		"SELECT name FROM nx_chunks WHERE serial_no = ? AND server_id = ?",
		chunkSerial(id), serverIDForScene(sceneID)).Scan(&name)
	switch {
	case err == sql.ErrNoRows:
		_, err = s.db.ExecContext(ctx, "INSERT INTO nx_chunks ( serial_no, name, server_id, scene_id,"+
			" create_time, deleted, save_time, save_data ) VALUES ( ?, ?, ?, ?, NOW(), 0, NOW(), ? )",
			chunkSerial(id), chunkSerial(id), serverIDForScene(sceneID), sceneID, data)
		return err
	case err != nil:
		return err
	default:
		_, err = s.db.ExecContext(ctx, "UPDATE nx_chunks SET name = ?, scene_id = ?, save_time = NOW(),"+
			" save_data = ? WHERE serial_no = ? AND server_id = ?",
			chunkSerial(id), sceneID, data, chunkSerial(id), serverIDForScene(sceneID))
		return err
	}
}

func (s *MySQLStore) DeleteChunk(sceneID uint32, id uint32) error {
	_, err := s.db.Exec("DELETE FROM nx_chunks WHERE serial_no = ? AND server_id = ?",
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
	out := map[string][]byte{}
	err := s.db.Query(func(rows *sql.Rows) error {
		out = map[string][]byte{} // 重发时整表重建
		for rows.Next() {
			var name string
			var data []byte
			if err := rows.Scan(&name, &data); err != nil {
				return err
			}
			out[name] = data
		}
		return rows.Err()
	}, "SELECT name, save_data FROM "+table)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *MySQLStore) PubSave(table, name string, saveData []byte) error {
	_, err := s.db.Exec("UPDATE "+table+" SET save_time = NOW(), save_data = ? WHERE name = ?", saveData, name)
	return err
}

func (s *MySQLStore) PubAdd(table, name, uid string, saveData []byte) (bool, error) {
	return false, ErrUnsupported
}

func (s *MySQLStore) PubRemove(table, name string) (bool, error) {
	// 表名是标识符不可参数化（table 来自内部配置白名单，Sprintf 安全）
	res, err := s.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE name = ?", table), name)
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
	var out map[string][]byte
	err := s.db.Query(func(rows *sql.Rows) error {
		cols, _ := rows.Columns()
		out = map[string][]byte{} // 重发时整表重建
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
		return rows.Err()
	}, q)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *MySQLStore) AuctionSave(kind AuctionKind, name string, saveData []byte) error {
	tp := "1"
	if kind == AuctionRec {
		tp = "2"
	}
	var n int
	_ = s.db.Row("SELECT COUNT(*) FROM nx_auction WHERE name = ?", name).Scan(&n)
	if n == 0 {
		_, err := s.db.Exec("INSERT INTO nx_auction ( name, uid, create_time, type, record, version,"+
			" amount, `range`, save_time, save_data ) VALUES ( ?, '', NOW(), ?, 'attr', 0, 0, 0, NOW(), ? )", name, tp, saveData)
		return err
	}
	_, err := s.db.Exec("UPDATE nx_auction SET save_time = NOW(), save_data = ? WHERE name = ?", saveData, name)
	return err
}

// dcAccountsTable 官方 Account 库全限定名（nx_district.dc_accounts）。
const dcAccountsTable = "`nx_district`.`dc_accounts`"

// ensureDcAccounts 缺表才建（官方 create_account.sql 同款；已存在零动作）。
// ParseConnectStringDSN 官方 ini ConnectString（SQLOLEDB 方言）→ Go
// MySQL DSN 公共解析：Provider=SQLOLEDB;Password=x;User ID=sa;
// Initial Catalog=nx_base1;Data Source=127.0.0.1:3306 →
// sa:x@tcp(127.0.0.1:3306)/nx_base1。缺必需键（User ID/Initial
// Catalog/Data Source）返回 ""。密码按原样进 DSN（方言值不含
// url 保留字符；特殊字符需自行转义）。
func ParseConnectStringDSN(cs string) string {
	get := func(key string) string {
		for _, part := range strings.Split(cs, ";") {
			part = strings.TrimSpace(part)
			if k, v, ok := strings.Cut(part, "="); ok && strings.TrimSpace(k) == key {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}
	user, db, source := get("User ID"), get("Initial Catalog"), get("Data Source")
	if user == "" || db == "" || source == "" {
		return ""
	}
	return user + ":" + get("Password") + "@tcp(" + source + ")/" + db
}

func (s *MySQLStore) ensureDcAccounts() error {
	var exists int
	if err := s.db.Row("SELECT COUNT(*) FROM information_schema.tables " +
		"WHERE table_schema='nx_district' AND table_name='dc_accounts'").Scan(&exists); err != nil {
		return err
	}
	if exists > 0 {
		// 存量表迁移：官方方言 password varchar(32) 装不下 fork 派生登录密码
		// （AES→base64→旋转，恒 36 字符，首登 INSERT 即 1406）→ 扩到 64。
		var plen int
		if err := s.db.Row("SELECT CHARACTER_MAXIMUM_LENGTH FROM information_schema.columns " +
			"WHERE table_schema='nx_district' AND table_name='dc_accounts' AND column_name='password'").
			Scan(&plen); err == nil && plen < 64 {
			if _, err := s.db.Exec("ALTER TABLE " + dcAccountsTable +
				" MODIFY `password` varchar(64) DEFAULT NULL"); err != nil {
				return err
			}
		}
		return nil
	}
	// 跨库表：nx_district 库缺失时先建库（DSN 用户需 CREATE 权限）
	if _, err := s.db.Exec("CREATE DATABASE IF NOT EXISTS nx_district"); err != nil {
		return err
	}
	_, err := s.db.Exec("CREATE TABLE IF NOT EXISTS " + dcAccountsTable + " (" +
		"`name` varchar(128) NOT NULL,`password` varchar(64) DEFAULT NULL," +
		"`create_time` datetime DEFAULT NULL,`lastlog_time` datetime DEFAULT NULL," +
		"`lastlog_addr` varchar(32) DEFAULT NULL,`valid_time` datetime DEFAULT NULL," +
		"`user_type` int(11) DEFAULT NULL,`status` int(11) DEFAULT NULL," +
		"`logged` int(11) DEFAULT NULL,`charge_mode` int(11) DEFAULT NULL," +
		"`points` int(11) DEFAULT NULL,`used_points` int(11) DEFAULT NULL," +
		"`comment` varchar(256) DEFAULT NULL,PRIMARY KEY (`name`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8")
	return err
}

func (s *MySQLStore) AccountLogon(name, password, addr string, session uint32) (AccountInfo, bool, error) {
	if err := s.ensureDcAccounts(); err != nil {
		return AccountInfo{}, false, err
	}
	var info AccountInfo
	var stored string
	var logged int32
	err := s.db.Row("SELECT user_type, points, used_points, password, logged FROM "+
		dcAccountsTable+" WHERE name = ? AND status = 1 AND (logged = 0 OR logged = ?)",
		name, session).
		Scan(&info.UserType, &info.Points, &info.UsedPoints, &stored, &logged)
	switch {
	case err == nil:
		if stored != password {
			return AccountInfo{}, false, nil // 密码槽等值门
		}
		_, err = s.db.Exec("UPDATE "+dcAccountsTable+
			" SET logged = ?, lastlog_time = NOW(), lastlog_addr = ? WHERE name = ?",
			session, addr, name)
		return info, err == nil, err
	case errors.Is(err, sql.ErrNoRows): // 官方 add 直名变体：首登自动建号
		_, err = s.db.Exec("INSERT INTO "+dcAccountsTable+
			" (name, password, create_time, lastlog_time, lastlog_addr, valid_time,"+
			" user_type, status, logged, charge_mode, points, used_points, comment)"+
			" VALUES (?, ?, NOW(), NOW(), ?, NOW(), 0, 1, ?, 0, 0, 0, '')",
			name, password, addr, session)
		if err == nil {
			return AccountInfo{}, true, nil
		}
		if !isDupKey(err) {
			return AccountInfo{}, false, err
		}
		// 重复键 = 行存在但 SELECT 门未命中。可能是本会话自己刚插的行——
		// INSERT 已落库、回包丢在断连里，重发撞主键：再过一遍 SELECT 门
		//（命中即密码/会话都对）照常放行。其余（旧 session 的 logged 残留
		// 等竞态）降级为拒签回包（result=0），不得让客户端无应答。
		info, ok, err := s.accountLogonExisting(name, password, addr, session)
		if errors.Is(err, sql.ErrNoRows) {
			return AccountInfo{}, false, nil
		}
		return info, ok, err
	default:
		return AccountInfo{}, false, err
	}
}

// accountLogonExisting 存量账号登录门：SELECT（status=1 且 logged 未被
// 他会话占用）→ 密码槽等值 → UPDATE logged/lastlog。未命中返回
// sql.ErrNoRows。
func (s *MySQLStore) accountLogonExisting(name, password, addr string, session uint32) (AccountInfo, bool, error) {
	var info AccountInfo
	var stored string
	var logged int32
	err := s.db.Row("SELECT user_type, points, used_points, password, logged FROM "+
		dcAccountsTable+" WHERE name = ? AND status = 1 AND (logged = 0 OR logged = ?)",
		name, session).
		Scan(&info.UserType, &info.Points, &info.UsedPoints, &stored, &logged)
	if err != nil {
		return AccountInfo{}, false, err
	}
	if stored != password {
		return AccountInfo{}, false, nil // 密码槽等值门
	}
	_, err = s.db.Exec("UPDATE "+dcAccountsTable+
		" SET logged = ?, lastlog_time = NOW(), lastlog_addr = ? WHERE name = ?",
		session, addr, name)
	return info, err == nil, err
}

// isDupKey MySQL 重复键（1062）。
func isDupKey(err error) bool {
	var myErr *mysql.MySQLError
	return errors.As(err, &myErr) && myErr.Number == 1062
}

func (s *MySQLStore) AccountLogoff(name string, session uint32) error {
	_, err := s.db.Exec("UPDATE "+dcAccountsTable+
		" SET logged = 0 WHERE name = ? AND logged = ?", name, session)
	return err
}

func (s *MySQLStore) AccountReset(session uint32) error {
	// 官方原文按 session 等值清（多服共享账号库语义）；合一/单机部署
	// 一服一库，硬杀残留的旧 session 行永不命中 → 账号永久 logged 锁死。
	// 开机全清（logged<>0 → 0），session 形参仅保留接口签名。
	_, err := s.db.Exec("UPDATE " + dcAccountsTable +
		" SET logged = 0 WHERE status = 1 AND logged <> 0")
	return err
}

func (s *MySQLStore) ResetOnline(serverID int) ([]RoleBrief, error) {
	if _, err := s.db.Exec("UPDATE nx_roles SET online = 0 WHERE online = 1 AND server_id = ?", serverID); err != nil {
		return nil, err
	}
	var out []RoleBrief
	err := s.db.Query(func(rows *sql.Rows) error {
		out = out[:0] // 重发时整表重建
		for rows.Next() {
			var b RoleBrief
			var del int
			if err := rows.Scan(&b.Name, &b.UID, &del); err != nil {
				return err
			}
			b.Deleted = del != 0
			out = append(out, b)
		}
		return rows.Err()
	}, "SELECT name, uid, deleted FROM nx_roles WHERE server_id = ?", serverID)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *MySQLStore) SaveRole(name string, meta RoleMeta, saveData []byte) error {
	// 官方 MessageSave（0x04，UPDATE nx_roles SET total_sec=… WHERE name）；
	// role_state_json 随周期存档覆盖（Phase 4 entergen 运行态回写入口）。
	_, err := s.db.Exec("UPDATE nx_roles SET total_sec = ?, letter_max = ?,"+
		" sys_flags = ?, role_index = ?, scene_id = ?, location = ?, save_para1 = ?,"+
		" save_time = NOW(), save_data = ?, role_state_json = ? WHERE name = ?",
		meta.TotalSec, meta.LetterMax, meta.SysFlags, meta.Index, meta.SceneID,
		meta.Location, meta.SavePara1, saveData,
		nullIfEmpty(meta.RoleStateJSON), name)
	return err
}

// nullIfEmpty 空串列写 NULL（role_state_json 未构建的旧档保持空）。
func nullIfEmpty(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func (s *MySQLStore) LoadRole(account, name string, serverID int) ([]byte, RoleMeta, bool, error) {
	// 官方 SELECT * FROM nx_roles WHERE account='%s' AND name='%s' AND
	// server_id=%d AND deleted=0（参数化——sprintf 方言 1064 坑）。
	// 空档（新角 save_data NULL）= ok true + nil blob。
	// account 空串 = name-only 匹配（GM 离线邮件 world 合成 clientID 查档，
	// 只知角色名；非空时行为与官方语义不变）。
	var meta RoleMeta
	var saveData []byte
	var totalSec, letterMax, sceneID, sysFlags, roleIndex int
	var location, savePara1, roleState sql.NullString
	query := "SELECT uid, account, total_sec, letter_max, scene_id, location, " +
		"sys_flags, role_index, save_para1, save_data, role_state_json FROM nx_roles " +
		"WHERE (? = '' OR account = ?) AND name = ? AND deleted = 0"
	err := s.db.Row(query, account, account, name).
		Scan(&meta.UID, &meta.Account, &totalSec, &letterMax, &sceneID, &location,
			&sysFlags, &roleIndex, &savePara1, &saveData, &roleState)
	if err == sql.ErrNoRows {
		return nil, RoleMeta{}, false, nil
	}
	if err != nil {
		return nil, RoleMeta{}, false, err
	}
	meta.ServerID = serverID
	meta.TotalSec, meta.LetterMax = int32(totalSec), int32(letterMax)
	meta.SceneID, meta.SysFlags, meta.Index = int32(sceneID), int32(sysFlags), int32(roleIndex)
	meta.Location, meta.SavePara1 = location.String, savePara1.String
	meta.RoleStateJSON = roleState.String
	return saveData, meta, true, nil
}

// CreateRole 官方 MessageCreate：重名门（SELECT name FROM nx_roles WHERE
// name='%s'）+ 22 列 INSERT 原文（初建 save_data 参数）；重名/超数 false。
//
// INSERT 不可重复执行，所以不逐语句重发，而是整段走 Retry、重发前先过
// 重名门：上一轮 INSERT 已落库只是回包丢在断连里（行上的 uid 就是本
// 请求生成的那个）→ 认领为成功，不发第二遍；uid 不同才是真重名。
func (s *MySQLStore) CreateRole(account, name, uid string, serverID int, meta RoleMeta, saveData []byte) (bool, error) {
	meta.Account, meta.ServerID, meta.UID = account, serverID, uid
	var (
		created bool
		sent    bool // 本调用的 INSERT 已发出过
	)
	err := s.db.Retry(context.Background(), func() (err error) {
		created, err = s.createRoleOnce(name, uid, serverID, meta, saveData, &sent)
		return err
	})
	return created, err
}

func (s *MySQLStore) createRoleOnce(name, uid string, serverID int, meta RoleMeta, saveData []byte, sent *bool) (bool, error) {
	var have string
	switch err := s.db.Row("SELECT uid FROM nx_roles WHERE name = ?", name).Scan(&have); {
	case err == nil:
		return *sent && have == uid, nil // 重名；同 uid = 自己上一轮已落库
	case !errors.Is(err, sql.ErrNoRows):
		return false, err
	}
	*sent = true
	// 经 Raw 发：本包的连接级重发会无脑再发一遍 INSERT，而它可能已经生效。
	_, err := s.db.Raw().Exec("INSERT INTO nx_roles ( name, uid, account, create_time, delete_time, open_time,"+
		" server_id, total_sec, deleted, letter_max, sys_flags, illegals, online,"+
		" role_index, scene_id, location, resume_time, resume_info, charge_info,"+
		" save_para1, save_time, save_data, role_state_json ) VALUES (?, ?, ?, NOW(), NULL, NOW(),"+
		" ?, ?, 0, ?, ?, 0, 0, ?, ?, ?, NULL, NULL, NULL, ?, NOW(), ?, ?)",
		name, uid, meta.Account, serverID, meta.TotalSec, meta.LetterMax, meta.SysFlags,
		meta.Index, meta.SceneID, meta.Location, meta.SavePara1, saveData,
		nullIfEmpty(meta.RoleStateJSON))
	if err != nil {
		var myErr *mysql.MySQLError
		if errors.As(err, &myErr) && myErr.Number == 1062 {
			return false, nil // 库侧唯一键兜底：并发抢注落败
		}
		return false, err
	}
	return true, nil
}

func (s *MySQLStore) SetOnline(name string, on bool) error {
	v := 1
	if !on {
		v = 0
	}
	_, err := s.db.Exec("UPDATE nx_roles SET online = ? WHERE name = ? AND online <> ?", v, name, v)
	return err
}

func (s *MySQLStore) DeleteRole(account, name string, serverID int) (bool, error) {
	// 官方软删原文（超期物理 DELETE FROM nx_roles 留回收任务）
	res, err := s.db.Exec("UPDATE nx_roles SET deleted = 1, delete_time = NOW()"+
		" WHERE name = ? AND account = ? AND deleted = 0", name, account)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (s *MySQLStore) QueryAccount(account string, serverID int) ([]RoleBrief, error) {
	// 官方列角 SELECT（docs §五；+uid 供 0x0D 回执路由）
	var out []RoleBrief
	err := s.db.Query(func(rows *sql.Rows) error {
		out = out[:0] // 重发时整表重建
		for rows.Next() {
			var b RoleBrief
			var sysFlags int
			if err := rows.Scan(&b.Name, &b.UID, &b.Index, &sysFlags, &b.Para); err != nil {
				return err
			}
			out = append(out, b)
		}
		return rows.Err()
	}, "SELECT name, uid, role_index, sys_flags, save_para1"+
		" FROM nx_roles WHERE account = ? AND server_id = ? AND deleted = 0 ORDER BY role_index",
		account, serverID)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ---- gmpanel 面板支撑（SearchRoles/RecentAlerts/SetAccountStatus；
// 面板进程直连 MySQL 的查询/管理口，不进 Backend 官方消息面） ----

// RoleRow 面板角色查询行（nx_roles 全维度浅投影，含软删/跨区服）。
type RoleRow struct {
	Account  string `json:"account"`
	Name     string `json:"name"`
	UID      string `json:"uid"`
	ServerID int    `json:"serverId"`
	Online   bool   `json:"online"`
	Deleted  bool   `json:"deleted"`
}

// SearchRoles 按账号列角色（不限 server_id、含已删——面板口径，
// 官方 QueryAccount 是登录链视角只看单区服未删）。
func (s *MySQLStore) SearchRoles(account string) ([]RoleRow, error) {
	var out []RoleRow
	err := s.db.Query(func(rows *sql.Rows) error {
		out = out[:0] // 重发时整表重建
		for rows.Next() {
			var r RoleRow
			var online, deleted int
			if err := rows.Scan(&r.Account, &r.Name, &r.UID, &r.ServerID, &online, &deleted); err != nil {
				return err
			}
			r.Online, r.Deleted = online != 0, deleted != 0
			out = append(out, r)
		}
		return rows.Err()
	}, "SELECT account, name, uid, server_id, online, deleted"+
		" FROM nx_roles WHERE account = ? ORDER BY server_id, role_index", account)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AlertRow nx_monitor 告警流水行（fssolomonitor.AlertSink 落库侧）。
type AlertRow struct {
	Name   string `json:"name"`
	Type   int    `json:"type"`
	Reason string `json:"reason"`
	Time   string `json:"time"` // DATETIME 驱动字符串形态（DSN 未开 parseTime）
}

// RecentAlerts 告警流水倒序（面板告警页）。
func (s *MySQLStore) RecentAlerts(limit int) ([]AlertRow, error) {
	var out []AlertRow
	err := s.db.Query(func(rows *sql.Rows) error {
		out = out[:0] // 重发时整表重建
		for rows.Next() {
			var r AlertRow
			if err := rows.Scan(&r.Name, &r.Type, &r.Reason, &r.Time); err != nil {
				return err
			}
			out = append(out, r)
		}
		return rows.Err()
	}, "SELECT name, type, reason, time FROM nx_monitor"+
		" ORDER BY time DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// SetAccountStatus 封号/解封（面板 GM 动作）：status=1 正常（AccountLogon
// 登录门 `status = 1`），0 = 封禁。附带 logged=0 清会话，防封禁账号
// 残留 logged 门锁死后续登录。
func (s *MySQLStore) SetAccountStatus(name string, status int) (bool, error) {
	res, err := s.db.Exec("UPDATE "+dcAccountsTable+" SET status = ?, logged = 0 WHERE name = ?",
		status, name)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// InsertLog 官方 MessageLogData/ItemLog/ChatLog/CustomLog 合一：datetime
// 列自动 NOW()，数值/串按 LogValue。
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
