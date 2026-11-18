// accountbackend.go MySQL 账号库后端：把官方 Game_UserLogin 等 sp 语义
// 落到本机 nx_district.dc_accounts（internal/persist 同库同表；points/
// used_points/charge_mode 列官方时代即充值域）。passwd 比对口径：请求侧
// 送大写 MD5 hex（FsServer SendRegister 家族实证），库内摘要大小写
// 不敏感比对。无 Oracle/中心库能力：礼物/点卡/商城目录恒空，行为对齐
// FreeBackend；login/validate/points 三个读路径接真库。
package charge

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// MySQLBackend 读路径接 nx_district.dc_accounts 的后端。
// 写路径（logout/renew 落账、gift/card/item）保持官方"成功空转"语义，
// 待商城/点卡表落地时扩展。
type MySQLBackend struct {
	DB         *sql.DB
	Inner      Backend // 兜底后端（空目录/默认值；nil=FreeBackend）
	FreePoints bool    // true=login 校验密码但不以点数拦截（开发默认）
}

func (m *MySQLBackend) inner() Backend {
	if m.Inner != nil {
		return m.Inner
	}
	return FreeBackend{}
}

type accountRow struct {
	password   string
	points     int64
	usedPoints int64
	chargeMode sql.NullInt64
	validTime  sql.NullTime
}

func (m *MySQLBackend) lookup(ctx context.Context, account string) (*accountRow, error) {
	// dc_accounts 主键 name（无 id 列，persist.DDL 口径）；accID 由
	// 调用方按账号稳定哈希生成。
	row := m.DB.QueryRowContext(ctx,
		"SELECT `password`, `points`, `used_points`, `charge_mode`, `valid_time` "+
			"FROM `nx_district`.`dc_accounts` WHERE `name`=?", account)
	var r accountRow
	err := row.Scan(&r.password, &r.points, &r.usedPoints, &r.chargeMode, &r.validTime)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// md5Equal 官方口令口径：两侧均为 MD5 hex，大小写不敏感。
func md5Equal(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

// deadlineOf valid_time → 官方天数浮点（空/过期由 FreePoints 决定语义）。
func deadlineOf(v sql.NullTime) float64 {
	if !v.Valid || v.Time.IsZero() {
		return 0
	}
	return daysSinceEpoch(v.Time)
}

// Login 实现 Backend：SELECT dc_accounts → 口令比对 → result/accid/
// points/charge_mode/valid_time。
func (m *MySQLBackend) Login(account, passwd string, serverID, gameID int, clientIP string) LoginResult {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	r, err := m.lookup(ctx, account)
	switch {
	case err == sql.ErrNoRows:
		return LoginResult{Result: 2} // 官方 piResult 域：非 1 即失败
	case err != nil:
		m.innerLogin(account, passwd)
		return LoginResult{Result: 1, LogonID: account}
	}
	if !md5Equal(r.password, passwd) {
		return LoginResult{Result: 3}
	}
	typ := 2
	if r.chargeMode.Valid {
		typ = int(r.chargeMode.Int64)
	}
	deadline := deadlineOf(r.validTime)
	_, _ = m.DB.ExecContext(ctx,
		"UPDATE `nx_district`.`dc_accounts` SET `lastlog_time`=CURRENT_TIMESTAMP, "+
			"`lastlog_addr`=? WHERE `name`=?", clientIP, account) //nolint:errcheck — 审计字段失败不拦登录
	return LoginResult{
		Result:     1,
		AccID:      int(hashAccount(account)),
		Points:     int(r.points),
		ChargeType: typ,
		Deadline:   deadline,
		LogonID:    account,
	}
}

// innerLogin 库不可用时的降级（对齐 FreeBackend：放行）。
func (m *MySQLBackend) innerLogin(account, _ string) LoginResult {
	res := m.inner().Login(account, "", 0, 0, "")
	return res
}

// Validate 实现 Backend：与 Login 同源（无会话校验通道）。
func (m *MySQLBackend) Validate(account, passwd, ip string, mode int) LoginResult {
	return m.Login(account, passwd, 0, 0, ip)
}

// RetPoints 实现 Backend：读 points 列。
func (m *MySQLBackend) RetPoints(account string, accID int) (int, int) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	r, err := m.lookup(ctx, account)
	if err != nil {
		return m.inner().RetPoints(account, accID)
	}
	return 1, int(r.points)
}

// Renew 实现 Backend：免费模式恒成功（类型 2）；接点数计费时在此
// 扣减（Account_Subtract 语义：points -= 单价，不足即 result!=1）。
func (m *MySQLBackend) Renew(account string, accID int) (int, int, int, float64) {
	return 1, 2, 0, 0
}

// Logout 实现 Backend：空转成功（扣费落账在 Renew 域）。
func (m *MySQLBackend) Logout(string, int, int) int { return 1 }

// RegisterServer 实现 Backend：恒成功（分区表无 server 登记需求）。
func (m *MySQLBackend) RegisterServer(svr ServerReg) (int, int, int, int) {
	return 1, svr.LogicalID, svr.Port, 0
}

// UnregisterServer 实现 Backend。
func (m *MySQLBackend) UnregisterServer(int) int { return 1 }

// QueryGift 实现 Backend：空目录。
func (m *MySQLBackend) QueryGift(string, int) (int, []GiftEntry) { return 1, nil }

// GetGift 实现 Backend。
func (m *MySQLBackend) GetGift(string, int, int, string, int) (int, string) { return 0, "" }

// SellCard 实现 Backend。
func (m *MySQLBackend) SellCard(_ string, _ int, t CardTrade) (int, string, int, string) {
	return 0, t.CardNo, t.Price, ""
}

// UnsellCard 实现 Backend。
func (m *MySQLBackend) UnsellCard(string, int, string) int { return 0 }

// BuyCard 实现 Backend。
func (m *MySQLBackend) BuyCard(_ string, _ int, t CardTrade) (int, string, string) {
	return 0, t.CardNo, t.Account
}

// BuyItem 实现 Backend。
func (m *MySQLBackend) BuyItem(_ string, _ int, t ItemTrade) (int, string, int, int) {
	return 0, "", t.Amount, 0
}

// BuyItemGive 实现 Backend。
func (m *MySQLBackend) BuyItemGive(_ string, _ int, t ItemTrade) (int, string, int, int, int, string) {
	return 0, "", t.Amount, 0, 0, t.GiveTo
}

// ItemInfo 实现 Backend：空目录。
func (m *MySQLBackend) ItemInfo(string) (int, int, int, []ItemEntry) { return 1, 0, 0, nil }

var _ = fmt.Sprintf // 保留 fmt 依赖位（错误串将来接入）
