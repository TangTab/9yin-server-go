package login

// AccountStore 是登录账号仓：op02 通过结构校验后的凭据比对接入点。
// persist 提供 MySQL 实现（nx_district.dc_accounts，表结构快照 =
// testdata/mysql/nx_district.sql）；nil（Flow.Accounts 未设）= 不校验，
// 测试与无 DB 模式沿用「结构合法即放行」口径。
type AccountStore interface {
	// Verify 校验账号凭据；账号不存在时自动注册（首登即建号），
	// 返回 registered=true。err 非空 = 密码不符或存储故障，
	// 调用方应拒绝整条连接（与结构非法同口径）。
	Verify(identity AccountIdentity, secret string) (registered bool, err error)
}
