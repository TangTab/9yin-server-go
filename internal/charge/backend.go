// backend.go 充值中心计费后端。官方由 CAccess 线程池对 Oracle
// sn_chargeapp 跑 15 个存储过程（Init@0x63fa0 存储过程名表）：
//
//	REGISTER   sn_account.SnailGame.GameServer_Register
//	UNREGISTER sn_account.SnailGame.GameServer_Logout
//	LOGIN      sn_account.SnailGame.Game_UserLogin
//	LOGOUT     sn_account.SnailGame.Game_UserLogout
//	RENEW      sn_account.SnailGame.Account_Subtract
//	QUERY_GIFT sn_account.SnailGame.Query_GamePresents
//	GET_GIFT   sn_account.SnailGame.Take_GamePresents
//	SELL_CARD  sn_account.SnailGame.Card_Vend
//	UNSELL     sn_account.SnailGame.Card_Cancel
//	BUY_CARD   sn_account.SnailGame.Card_Buy
//	BUY_ITEM   sn_account.SnailGame.Property_Buy
//	ITEM_INFO  sn_account.SnailGame.getPropertyPrice
//	BUY_GIVE   sn_account.SnailGame.Property_Buy
//	RET_POINTS sn_account.SnailGame.getAccountBalance
//	VALIDATE   sn_account.SnailGame.Validate_Account
//
// Go 复刻将 sp 语义收敛为 Backend 接口；Game_UserLogin 的输出参数集
// （piResult/piAid/piGMLevel/psIsFree/piAccPoints/pdAccDlimit/pdLogonTime/
// psLogonID/psChargeInfo，RequestLogin@0x5abb0 实证）即 LoginResult 字段。
// 官方应答为异步（DB 线程→队列→Publish*）；Go 侧同步调用，对单连接
// 保持命令序即可（官方跨连接本就无序）。
package charge

// ServerReg 是 register 命令携带的游戏服注册信息
// （ServerStateUnknown@0x700f0：serverid = vl[2] + ServerIDOffset）。
type ServerReg struct {
	LogicalID int    // 请求 vl[2]（逻辑服编号，未加偏移）
	Name      string // vl[3] 服务器名
	Ident     string // vl[4] 身份串（客户端放 MD5，charge 侧透传）
	Port      int    // vl[5] 端口
	Addr      string // 连接对端地址（accept 时记录）
}

// LoginResult 对应 Game_UserLogin 输出参数（access_back_login_t）。
type LoginResult struct {
	Result     int     // piResult（1=成功）
	AccID      int     // piAid 用户 ID
	GMLevel    int     // piGMLevel
	Points     int     // piAccPoints 剩余点数
	ChargeType int     // 计费类型（2=点数型；1/3=时限型，CheckUsers 关心）
	Deadline   float64 // pdAccDlimit 到期时间（官方天数浮点，epoch 起）
	LogonID    string  // psLogonID 会话票据
	ChargeInfo string  // psChargeInfo 计费信息串
}

// GiftEntry 是 query_gift 应答的礼物条目（N × {id, name, amount}）。
type GiftEntry struct {
	ID     int
	Name   string
	Amount int
}

// ItemEntry 是 item_info 应答的物品条目（N × {name, points}）。
type ItemEntry struct {
	Name   string
	Points int
}

// CardTrade 覆盖 sell_card/unsell_card/buy_card 的共通字段。
type CardTrade struct {
	UserIdx  int
	CardType string
	CardNo   string
	Price    int
	Password string
	Account  string // buy_card 充值目标账号
}

// ItemTrade 覆盖 buy_item/buy_item_give 的共通字段。
type ItemTrade struct {
	UserIdx int
	Account string // 大写账号（可为空=未登录直购）
	ItemID  int
	Amount  int
	GiveTo  string // buy_item_give 受赠者
}

// Backend 是计费数据源。全部方法须并发安全（多连接同时调用）。
// 返回值仅承载官方 sp 输出参数；错误（后端不可用）由实现方内部
// 记日志并折算为 Result!=1 的失败应答，避免 charge 主循环阻塞。
type Backend interface {
	// RegisterServer 对应 GameServer_Register：登记游戏服。
	// 返回 (result, dbServerID, gamePort, zone)——成功时 serverID 回填
	// server_t（access_back_register_t +0x50）。
	RegisterServer(svr ServerReg) (result int, serverID, gamePort, zone int)
	// UnregisterServer 对应 GameServer_Logout：注销。
	UnregisterServer(serverID int) (result int)
	// Login 对应 Game_UserLogin（psMemAccount/psMd5Pass/piServerID/
	// piGameID/psClientIP）。passwd 为大写化后的请求原文。
	Login(account, passwd string, serverID, gameID int, clientIP string) LoginResult
	// Validate 对应 Validate_Account：无会话校验（客服/web 通道）。
	Validate(account, passwd, clientIP string, mode int) LoginResult
	// Logout 对应 Game_UserLogout：结算落账。points 为 charge 侧缓存值。
	Logout(account string, accID, points int) (result int)
	// Renew 对应 Account_Subtract：周期扣费。返回 (result, chargeType,
	// unitType, deadline, balanceTime)。chargeType==2 时才向游戏服回发
	// （PublishRenew@0x6b010 实证）。
	Renew(account string, accID int) (result, chargeType, unitType int, deadline float64)
	// RetPoints 对应 getAccountBalance。
	RetPoints(account string, accID int) (result, points int)
	// QueryGift 对应 Query_GamePresents。
	QueryGift(account string, accID int) (result int, gifts []GiftEntry)
	// GetGift 对应 Take_GamePresents。
	GetGift(account string, accID, giftID int, serial string, amount int) (result int, name string)
	// SellCard 对应 Card_Vend。
	SellCard(account string, accID int, t CardTrade) (result int, cardNo string, price int, note string)
	// UnsellCard 对应 Card_Cancel（userIdx==0 为系统撤销）。
	UnsellCard(account string, accID int, cardNo string) (result int)
	// BuyCard 对应 Card_Buy。
	BuyCard(account string, accID int, t CardTrade) (result int, cardNo, fillAccount string)
	// BuyItem 对应 Property_Buy。
	BuyItem(account string, accID int, t ItemTrade) (result int, item string, amount, points int)
	// BuyItemGive 对应 Property_Buy（赠送）。
	BuyItemGive(account string, accID int, t ItemTrade) (result int, item string, amount, points, orderID int, givee string)
	// ItemInfo 对应 getPropertyPrice：分页目录。
	ItemInfo(query string) (result, total, start int, items []ItemEntry)
}

// FreeBackend 开发后端：全部放行、点数为 0、计费类型 2（点数型，即
// CheckUsers 不主动推 renew 的免打扰形态）、礼物/商城目录为空。
// 用于无 MySQL 环境的独立联调（官方 FlushErrorMax=0 语义下免费可玩）。
type FreeBackend struct{}

// RegisterServer 实现 Backend：登记恒成功，serverID 原样回显。
func (FreeBackend) RegisterServer(svr ServerReg) (int, int, int, int) {
	return 1, svr.LogicalID, svr.Port, 0
}

// UnregisterServer 实现 Backend。
func (FreeBackend) UnregisterServer(int) int { return 1 }

// Login 实现 Backend：恒成功，票据回显账号。
func (FreeBackend) Login(account, _ string, _, _ int, _ string) LoginResult {
	return LoginResult{Result: 1, AccID: 0, Points: 0, ChargeType: 2, LogonID: account, ChargeInfo: ""}
}

// Validate 实现 Backend：恒成功。
func (FreeBackend) Validate(account, _, _ string, _ int) LoginResult {
	return LoginResult{Result: 1, ChargeType: 2, LogonID: account}
}

// Logout 实现 Backend。
func (FreeBackend) Logout(string, int, int) int { return 1 }

// Renew 实现 Backend：恒成功、类型 2。
func (FreeBackend) Renew(string, int) (int, int, int, float64) {
	return 1, 2, 0, 0
}

// RetPoints 实现 Backend：0 点。
func (FreeBackend) RetPoints(string, int) (int, int) { return 1, 0 }

// QueryGift 实现 Backend：空目录。
func (FreeBackend) QueryGift(string, int) (int, []GiftEntry) { return 1, nil }

// GetGift 实现 Backend。
func (FreeBackend) GetGift(string, int, int, string, int) (int, string) { return 0, "" }

// SellCard 实现 Backend。
func (FreeBackend) SellCard(_ string, _ int, t CardTrade) (int, string, int, string) {
	return 0, t.CardNo, t.Price, ""
}

// UnsellCard 实现 Backend。
func (FreeBackend) UnsellCard(string, int, string) int { return 0 }

// BuyCard 实现 Backend。
func (FreeBackend) BuyCard(_ string, _ int, t CardTrade) (int, string, string) {
	return 0, t.CardNo, t.Account
}

// BuyItem 实现 Backend。
func (FreeBackend) BuyItem(_ string, _ int, t ItemTrade) (int, string, int, int) {
	return 0, "", t.Amount, 0
}

// BuyItemGive 实现 Backend。
func (FreeBackend) BuyItemGive(_ string, _ int, t ItemTrade) (int, string, int, int, int, string) {
	return 0, "", t.Amount, 0, 0, t.GiveTo
}

// ItemInfo 实现 Backend：空目录。
func (FreeBackend) ItemInfo(string) (int, int, int, []ItemEntry) { return 1, 0, 0, nil }
