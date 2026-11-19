package fssoloworld

// WorldState 世界状态机 13 态（SoloWorldState.obj rdata 顺序定案；§七）。
type WorldState int

const (
	WsUnknown WorldState = iota
	WsLoadRoles
	WsLoadScenes
	WsCanOpen
	WsOpened
	WsCloseEntry
	WsClearPlayers
	WsClearScenes
	WsCloseMember
	WsClosePublic
	WsCloseAuction
	WsCloseStore
	WsClosed
	WsError // 异常分支（GetState "Error"）
)

var worldStateNames = map[WorldState]string{
	WsUnknown: "Unknown", WsLoadRoles: "LoadRoles", WsLoadScenes: "LoadScenes",
	WsCanOpen: "CanOpen", WsOpened: "Opened", WsCloseEntry: "CloseEntry",
	WsClearPlayers: "ClearPlayers", WsClearScenes: "ClearScenes",
	WsCloseMember: "CloseMember", WsClosePublic: "ClosePublic",
	WsCloseAuction: "CloseAuction", WsCloseStore: "CloseStore",
	WsClosed: "Closed", WsError: "Error",
}

func (s WorldState) String() string { return worldStateNames[s] }

// AppState 外部呈现态（main.lua WAPS_*：与内部 13 态的映射）。
func AppState(s WorldState) int {
	switch s {
	case WsOpened:
		return 4 // WAPS_OPENED
	case WsCanOpen:
		return 2 // WAPS_READY
	case WsLoadRoles, WsLoadScenes:
		return 1 // WAPS_STARTING
	case WsCloseEntry, WsClearPlayers, WsClearScenes, WsCloseMember,
		WsClosePublic, WsCloseAuction, WsCloseStore:
		return 5 // WAPS_CLOSING
	case WsClosed:
		return 6
	default:
		return 0 // WAPS_UNKNOWN
	}
}

// worldTransitions 官方迁移条件集（每态处理函数语义摘要）：
//
//	Unknown     --SendToStore(load roles)--> LoadRoles
//	LoadRoles   --NameUid 同步完成-->        LoadScenes
//	LoadScenes  --CheckAllScene 全 Running--> CanOpen（"game server ready"+SendOpenConnect）
//	CanOpen     --OpenServer()/AutoOpen-->    Opened
//	Opened      --CloseServer-->              CloseEntry（逐 entry 关闭）
//	CloseEntry  --全 entry 关-->              ClearPlayers
//	ClearPlayers--玩家清空-->                 ClearScenes
//	ClearScenes --场景清空-->                 CloseMember→ClosePublic→CloseAuction→CloseStore→Closed
//	Closed      --QuitOnClosed--> 退出
type worldTransitions struct{}

// closedChain 关闭链顺序（WorldStateCloseMember 起的顺序推进）。
var closedChain = []WorldState{WsCloseMember, WsClosePublic, WsCloseAuction, WsCloseStore, WsClosed}
