package dbconn

import (
	"context"
	"database/sql/driver"
	"errors"
	"io"
	"net"
	"syscall"

	"github.com/go-sql-driver/mysql"
)

// connErrorNumbers 服务端连接类错误码——换连接重发即可能成功。
// 其余 MySQLError（语法 1064/重复键 1062/无权限 1142/表不存在 1146/
// 密码错 1045/库不存在 1049…）是确定性错误，重发只会再错一次。
var connErrorNumbers = map[uint16]bool{
	1040: true, // ER_CON_COUNT_ERROR：Too many connections
	1053: true, // ER_SERVER_SHUTDOWN：Server shutdown in progress（关库窗口）
	1152: true, // ER_ABORT_CONNECTION：Aborted connection
	1154: true, // ER_NET_READ_ERROR
	1155: true, // ER_NET_READ_INTERRUPTED
	1156: true, // ER_NET_ERROR_ON_WRITE
	1157: true, // ER_NET_WRITE_INTERRUPTED
	1203: true, // ER_USER_LIMIT_REACHED：max_user_connections
	1927: true, // ER_CONNECTION_KILLED
	2006: true, // CR_SERVER_GONE_ERROR：MySQL server has gone away
	2013: true, // CR_SERVER_LOST：Lost connection during query
	2014: true, // CR_SERVER_LOST：Commands out of sync
	2055: true, // CR_SERVER_LOST_EXTENDED：Lost connection at '%s'
	4031: true, // ER_CLIENT_INTERACTION_TIMEOUT：空闲被服务端掐断
}

// IsConnErr 报告 err 是否为连接级错误——换一条连接重发即可能成功的那些：
// 断连（invalid connection / bad connection / EOF / 连接被重置）、拒连、
// 拨号与读写超时、上面那张服务端连接类错误码表。
//
// 不算的：SQL 语义错误、业务哨兵（sql.ErrNoRows）、database/sql 自身
// 状态错误（库已 Close）、DNS 名不存在/地址非法这类配置硬错误。
// context.DeadlineExceeded 计入——驱动读写超时就是以它上抛的；调用方
// 自己的 ctx 到期由调用方先查 ctx.Err() 区分。
func IsConnErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, driver.ErrBadConn) || errors.Is(err, mysql.ErrInvalidConn) ||
		errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var myErr *mysql.MySQLError
	if errors.As(err, &myErr) {
		return connErrorNumbers[myErr.Number]
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return opErr.Timeout() || isConnSyscall(opErr.Err)
	}
	return false
}

// isConnSyscall 网络层内层错误是否为「线路」问题：系统调用错误码
// （拒连/重置/网络不可达）与 DNS 临时故障是；主机名不存在（配置错）
// 和 "unknown network"/"missing address" 这类裸 errors.New 不是。
func isConnSyscall(err error) bool {
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return true
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return dnsErr.IsTemporary || dnsErr.Timeout()
	}
	return false
}
