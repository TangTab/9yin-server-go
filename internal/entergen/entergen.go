// Package entergen 进场/就绪数据原生生成器（2433 方言）。
//
// 移植自 b-line cmd/protocol-probe 生成器族（官方 7074 线抓包逐字节
// 定版的帧布局），按当前仓架构参数化：playerObjectID/playerOwnerID
// 等会话身份由调用方注入，数据值取 internal/rolestate（建号落库的
// role_state_json 同源）。帧清单与处置台账见
// docs/official/透传数据清单.md。
//
// 方言锚点（全部经 clientdata 按 2433 表名解析）：
//   - 0x0B 进场：idx178 Name / idx1468 MaxVisuals=20000 / idx1447 ConfigID /
//     idx1448 Resource（b-line scene_transition.go 定版）
//   - 0x10 属性：[10][mode@1][obj@2][own@6][u16 n@10]
//   - 0x11 记录加行：行编码按官方 schema 列型（b-line serverRecordAddCells）
//   - 0x14 记录挂载：[14][sub][obj@2][own@6][rec u16@10]（初始账号方言定版）
//   - 0x15/0x17/0x18 视图族 / 0x1F 位置 / 0x1E TVarList 自定义消息
//   - 0x28 化身：mode=0x46 明文（fork 客户端实证接受；0x47 全字面量备用）
//   - 0x29 批量：0x47 全字面量包装（零池依赖，wrapAs47Literal）
package entergen

import (
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
	"github.com/lizhanyin/9yin-server-go/internal/taskdata"
)

// Identity 会话身份（每次进场由调用方注入，见 fssoloworld/ident.go）。
// 官服抓包实证（四抓包，2026-11-24；详见 docs/fork-era/id-system-analysis.md
// 文首「官服规则」）：
//
//	ObjectID / OwnerID  玩家本次进场的 PERSISTID{ident, serial}——每进一个
//	                   场景换一套，由 0x0B 的 [5:9]/[9:13] 宣告：
//	                     ident = sceneID<<24 | 场景内序号（高 8 位=场景 ID
//	                             低字节，玩家与同场景 NPC 共用）
//	                     serial = 场景内序列号（单调递增）
//	Sequence            登录会话 serial——0x0B 的 [1:5]，等于 0x04 的
//	                   serial@0x21，整个会话不变（不是进场流水号）
type Identity struct {
	ObjectID uint32 // 玩家 wire objectID（PERSISTID.ident）
	OwnerID  uint32 // 玩家 wire ownerID（PERSISTID.serial）
	Sequence uint32 // 0x0B [1:5] 登录会话 serial（= 0x04 serial@0x21）
}

// Session 登录会话信息（m366/m623/m655/m607/m52/m586 数据源）。
type Session struct {
	ClientIP   string // 客户端公网 IP 回显（m366）
	ServerID   int32  // 服务器 ID（m655）
	ServerName string // 服务器名（m655）
	Account    string // 账号（m655/m607）
	AccountID  int64  // 平台账号 ID（m655）
}

// Context 一次进场生成的全部输入。
type Context struct {
	Identity   Identity
	Session    Session
	RoleName   string // 角色名（m607 宽串）
	State      rolestate.RoleState
	Keybind    string   // 存档键位串（0x28 档案 ShortcutKey；空=省略）
	NPCFrames  [][]byte // 场景 NPC 物化帧族（npcfeed 预生成；0x28 官方言默认）
	SceneIndex int      // 客户端 scenes.ini 全局段号（开图位图/迷雾行；-1=表缺席降级）
	// Tasks 任务目录（Task_Accepted / Task_Record 行的任务主表列来源）；
	// nil＝没有任务目录，已接行除任务号外各列为空、不下发目标行。
	Tasks *taskdata.Catalog
	// Stamp 任务目标行 stamp 列的计时戳（0＝当前时间；测试面注入定值）。
	Stamp int64
	// SP 本次进场下发的当前怒气（SP 属性）：会话值——world 传战斗会话态的
	// 当前怒气，没进过战斗 = 0。不取存档 State.SP：怒气每次登录从 0 起
	//（2026-11-26 定案；此前下发建号初值 80，而服务端战斗态从 0 起，两端
	// 对不上）。上限另走 baseMaxSP。
	SP int
}
