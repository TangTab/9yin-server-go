package fssolopublic

// State 是官方 8 态状态机（solopublicstate.cpp，GetStateName 跳表定案；
// 存于 SoloPublic+0x34）。
type State int32

const (
	StateUnknown       State = 0 // 启动中
	StateLoadRelations State = 1 // 向 store 拉关系表
	StateLoadFeeds     State = 2 // 拉 feed 流
	StateLoadTrades    State = 3 // 拉寄售行（双门：trades+nameuid）
	StateLoadSpaces    State = 4 // 逐空间拉公共数据
	StateOpened        State = 5 // 开放服务（MemberRegister/SyncPublic 门）
	StateClosing       State = 6 // 批量回存
	StateClosed        State = 7 // 终态
)

// String 官方 GetStateName（跳表索引即值）。
func (s State) String() string {
	switch s {
	case StateUnknown:
		return "Unknown"
	case StateLoadRelations:
		return "LoadRelations"
	case StateLoadFeeds:
		return "LoadFeeds"
	case StateLoadTrades:
		return "LoadTrades"
	case StateLoadSpaces:
		return "LoadSpaces"
	case StateOpened:
		return "Opened"
	case StateClosing:
		return "Closing"
	case StateClosed:
		return "Closed"
	default:
		return ""
	}
}

// 状态机事件（PublicStateMachine arg1）。
const (
	evTick        = 0 // 每秒（TimerCheck）
	evStoreConn   = 1 // store identify 完成
	evSpacesDone  = 2 // 全空间装载完
	evWorldClose  = 3 // world 关闭令
	stateEventMax = 4 // TimerCheck 第三参（官方恒 4）
)
