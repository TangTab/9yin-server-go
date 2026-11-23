package persist

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-sql-driver/mysql"

	"github.com/lizhanyin/9yin-server-go/internal/dbconn"
	"github.com/lizhanyin/9yin-server-go/internal/login"
)

// MySQLAccountStore 是 login.AccountStore 的 MySQL 实现，落在
// nx_district.dc_accounts（表结构快照 = testdata/mysql/nx_district.sql，
// 官方 create_account.sql 同款默认值）。
//
// 方言说明：op02 双槽是客户端加扰密文（算法未破），但确定性无 nonce——
// 官方明文账号列 name 在此存 acct:v1:<sha256hex> 身份键（varchar(128)
// 容得下），password 列存密码槽 md5hex-32 摘要（对齐 varchar(32)）。
// 效果等价明文域：首登（任意账号密码）自动注册，此后同账号须同密码。
//
// 连接经 internal/dbconn：库重启/闪断窗口内登录校验挂起退避重发。
type MySQLAccountStore struct {
	db *dbconn.DB
}

// dcAccountsDDL 与 testdata/mysql/nx_district.sql 逐字节同源
// （BUILD 配置下的库名固定 nx_district）。
const dcAccountsDDL = "CREATE TABLE IF NOT EXISTS `nx_district`.`dc_accounts` (" +
	"`name` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL," +
	"`password` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL," +
	"`create_time` datetime DEFAULT NULL," +
	"`lastlog_time` datetime DEFAULT NULL," +
	"`lastlog_addr` varchar(32) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL," +
	"`valid_time` datetime DEFAULT NULL," +
	"`user_type` int(11) DEFAULT NULL," +
	"`status` int(11) DEFAULT NULL," +
	"`logged` int(11) DEFAULT NULL," +
	"`charge_mode` int(11) DEFAULT NULL," +
	"`points` int(11) DEFAULT NULL," +
	"`used_points` int(11) DEFAULT NULL," +
	"`comment` varchar(256) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL," +
	"PRIMARY KEY (`name`)" +
	") ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPACT"

// OpenAccountStore 打开账号仓：连接（DSN 默认库无关，访问全限定
// nx_district.*；库在重启就等到 ctx 到期，语义见 dbconn.Open）+ 建库
// 建表兜底（已存在则零动作，本机还原库不惊动）。
func OpenAccountStore(ctx context.Context, dsn string) (*MySQLAccountStore, error) {
	db, err := dbconn.Open(ctx, dsn, dbconn.NewOptions("persist", nil))
	if err != nil {
		return nil, err
	}
	store := &MySQLAccountStore{db: db}
	if err := store.ensureSchema(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

// Close 关闭底层连接池（幂等）。
func (s *MySQLAccountStore) Close() error { return s.db.Close() }

// ensureSchema 缺表才建（information_schema 先探，避免对无 CREATE
// 权限的账号在已还原库上空跑 DDL）。
func (s *MySQLAccountStore) ensureSchema(ctx context.Context) error {
	var exists int
	err := s.db.RowContext(ctx,
		"SELECT COUNT(*) FROM information_schema.tables "+
			"WHERE table_schema='nx_district' AND table_name='dc_accounts'").Scan(&exists)
	if err != nil {
		return fmt.Errorf("persist: probe dc_accounts: %w", err)
	}
	if exists > 0 {
		return nil
	}
	if _, err := s.db.ExecContext(ctx,
		"CREATE DATABASE IF NOT EXISTS `nx_district` DEFAULT CHARSET=utf8"); err != nil {
		return fmt.Errorf("persist: create nx_district: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, dcAccountsDDL); err != nil {
		return fmt.Errorf("persist: create dc_accounts: %w", err)
	}
	return nil
}

// Verify 实现 login.AccountStore：账号不存在即注册（官方 create_account.sql
// 同款默认值），存在则比对密码摘要，通过后刷新 lastlog_time。
func (s *MySQLAccountStore) Verify(identity login.AccountIdentity, secret string) (bool, error) {
	name := identity.String()
	var stored string
	err := s.db.Row("SELECT `password` FROM `nx_district`.`dc_accounts` WHERE `name`=?", name).Scan(&stored)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		_, insertErr := s.db.Exec(
			"INSERT INTO `nx_district`.`dc_accounts` "+
				"(`name`,`password`,`create_time`,`lastlog_time`,`lastlog_addr`,`valid_time`,"+
				"`user_type`,`status`,`logged`,`charge_mode`,`points`,`used_points`,`comment`) "+
				"VALUES (?,?,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'',CURRENT_TIMESTAMP,51,1,11,0,0,0,'')",
			name, secret)
		if insertErr == nil {
			return true, nil
		}
		var mysqlErr *mysql.MySQLError
		if !errors.As(insertErr, &mysqlErr) || mysqlErr.Number != 1062 {
			return false, fmt.Errorf("persist: register account: %w", insertErr)
		}
		// 重复键：并发首登被他连接抢注，或本连接 INSERT 已落库只是回包
		// 丢在断连里后重发——回读比对密码摘要即见分晓。
		if err := s.db.Row(
			"SELECT `password` FROM `nx_district`.`dc_accounts` WHERE `name`=?", name).Scan(&stored); err != nil {
			return false, fmt.Errorf("persist: reread account: %w", err)
		}
	case err != nil:
		return false, fmt.Errorf("persist: query account: %w", err)
	}
	if stored != secret {
		return false, fmt.Errorf("persist: account %s password mismatch", identity.Prefix())
	}
	if _, err := s.db.Exec(
		"UPDATE `nx_district`.`dc_accounts` SET `lastlog_time`=CURRENT_TIMESTAMP WHERE `name`=?",
		name); err != nil {
		return false, fmt.Errorf("persist: touch account: %w", err)
	}
	return false, nil
}
