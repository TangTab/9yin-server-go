package fssolostore

// Backend 是官方 CAccess 线程面（IDBSystem 之下、96 个 Message* 处理器
// 之上的语义收敛）——五通道 SQL 语义（docs §五）的 Go 投影。
//
// 官方为 5 通道 × N 池 CAccess 线程；Go 复刻单 Backend 并发安全实现，
// 由 Server 以 goroutine 池等价驱动。方法返回的回执 payload 采用对端
// （fssolomember/fssoloauction）已实证的解码格式。
type Backend interface {
	// ---- Role 通道（nx_roles/nx_letter/nx_chunks）----

	// ManageInfos 官方 MessageManageInfo（0x29 引导，全量 nx_config）：
	// 逐项返回（每项一帧回执 0x1E，member 端 decodeManageInfo 装载序）。
	ManageInfos() ([]ManageEntry, error)
	// ChunkIDs 官方 MessageChunkInfo（0x18）：场景块 id 清单（回执 0x0F
	// {scene,n,id[n]}，member publishChunkInfo 读法）。
	ChunkIDs(sceneID uint32) ([]uint32, error)
	// LoadChunk 官方 MessageLoadChunk（0x19）：块内容（回执 0x10
	// {scene,n,{id,len,bytes}…}，member publishLoadChunk 读法）。
	LoadChunk(sceneID uint32, ids []uint32) (map[uint32][]byte, error)
	// SaveChunk 官方 MessageSaveChunk（0x1A）：upsert 块（回执 0x11
	// {chunk_id}）。
	SaveChunk(sceneID uint32, id uint32, data []byte) error
	// DeleteChunk 官方 MessageDeleteChunk（0x1B）：删块（回执 0x12）。
	DeleteChunk(sceneID uint32, id uint32) error
	// ResetOnline 官方 MessageRoleInfo 开机复位（UPDATE online=0 WHERE
	// server_id）+ 角色名册（SELECT name,uid,deleted）。
	ResetOnline(serverID int) ([]RoleBrief, error)
	// SaveRole 官方 MessageSave（0x04，UPDATE … save_data=?；回执 0x03）。
	SaveRole(name string, meta RoleMeta, saveData []byte) error
	// LoadRole 官方 MessageLoad（0x02；回执 0x02 含 save_data）。
	LoadRole(account, name string, serverID int) (saveData []byte, meta RoleMeta, ok bool, err error)
	// CreateRole 官方 MessageCreate（INSERT；重名/超数 false；回执 0x06）。
	CreateRole(account, name, uid string, serverID int, meta RoleMeta, saveData []byte) (bool, error)
	// SetOnline/SetOffline 官方 0x05/0x06（UPDATE online；无回执）。
	SetOnline(name string, on bool) error
	// QueryAccount 官方 MessageQueryAccount（角色清单；回执 0x0D）。
	QueryAccount(account string, serverID int) ([]RoleBrief, error)

	// ---- Public 通道（nx_config / 公共空间表 %s / SNS / 交易 / 拍卖）----

	// PubInfo 官方 MessagePubInfo（公共空间表清单；回执 0x19）。
	PubInfo() ([]string, error)
	// PubLoad 官方 MessagePubLoad（单表全量 {name,save_data}；回执 0x1A）。
	PubLoad(table string) (map[string][]byte, error)
	// PubSave/PubAdd/PubRemove 官方 0x26/0x27/0x28（回执 0x1B/0x1C/0x1D）。
	PubSave(table, name string, saveData []byte) error
	PubAdd(table, name, uid string, saveData []byte) (bool, error)
	PubRemove(table, name string) (bool, error)
	// AuctionLoad 官方 MessageAuctionAttrLoad/RecordLoad（0x3F/0x40）：
	// 拍卖行 PUB4 块回放（回执 0x29 / 0x2A、终帧 0x2B）。
	AuctionLoad(kind AuctionKind) (map[string][]byte, error)
	// AuctionSave 官方 MessageAuctionAttrSave/RecordSave（0x41/0x42；
	// 回执 0x2C/0x2D）。
	AuctionSave(kind AuctionKind, name string, saveData []byte) error

	// ---- Account 通道（nx_district.dc_accounts）----

	// AccountLogon 官方 MessageAccountLogon（0x36，UPDATE logged=%d …
	// WHERE (logged=0 OR logged=%d)；回执 0x25）。
	AccountLogon(name string, logged int) (ok bool, err error)
	// AccountLogoff 官方 MessageAccountLogoff（0x37；回执 0x26）。
	AccountLogoff(name string, logged int) (ok bool, err error)

	// ---- Log 通道（29 张自定义表 + 分周后缀）----

	// InsertLog 官方 MessageLogData/ItemLog/ChatLog/CustomLog 合一：
	// table 为逻辑表名（后端自加 suffix），vals 按表 schema 列序。
	InsertLog(table string, vals []LogValue) error
	// PrepareSplitTables 官方 CSyncAccess::CreateLogTable 语义：
	// 保证分周表存在，返回生效 suffix（不支持 DDL 的后端返回 ""）。
	PrepareSplitTables(suffix string) (applied string, err error)
	// SplitSuffix 当前生效后缀（SendSplitSuffix 广播值）。
	SplitSuffix() string

	// Generic 未建模命令兜底（Server 按 back-id 表回 {u32 result}）；
	// 返回 ErrUnsupported 则仅 trace。
	Generic(msg uint32, payload []byte) error

	// Ping 官方 CheckConnect 面（AppRecord database=Break|OK 判据）。
	Ping() error
	// Close 官方 StopAccess。
	Close() error
}

// AuctionKind 拍卖块通道（nx_auction type 列）。
type AuctionKind int

const (
	AuctionAttr AuctionKind = 1 // type='1'（属性块）
	AuctionRec  AuctionKind = 2 // type='2'（记录块）
)

// ManageEntry 官方 nx_config 行（MessageManageInfo 全量回放项；
// member 端 CManageInfo {String,WideStr,Int,WideStr,Int,Int} 装载序）。
type ManageEntry struct {
	ID      string // id
	Name    string // name（串）
	Title   string // title（宽串）
	Type    int32
	Comment string // comment（宽串）
	Status  int32
	Order   int32
	State   int32
}

// RoleBrief 官方角色名册行（SELECT name, uid, deleted / 角色清单回执）。
type RoleBrief struct {
	Name    string
	UID     string
	Deleted bool
	Index   int32
}

// RoleMeta 官方 save 元数据列（UPDATE nx_roles SET total_sec=…）。
type RoleMeta struct {
	Account   string
	UID       string
	ServerID  int
	TotalSec  int32
	LetterMax int32
	SceneID   int32
	Location  string
	SavePara1 string
	Index     int32
	SysFlags  int32
}

// LogValue 日志动态值（get_vary_message 的 TVarList 投影）。
type LogValue struct {
	IsWide  bool
	Text    string // ANSI/GBK 或 UTF-16LE 解码后
	Int     int64
	IsFloat bool
	Float   float64
	IsNum   bool
}
