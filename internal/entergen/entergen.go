// Package entergen 进场/就绪数据原生生成器（2433 方言）。
//
// 移植自 b-line cmd/protocol-probe 生成器族（官方 7074 线抓包逐字节
// 定版的帧布局），按当前仓架构参数化：playerObjectID/playerOwnerID
// 等会话身份由调用方注入，数据值取 internal/rolestate（建号落库的
// role_state_json 同源）。帧清单与处置台账见
// docs/official/透传数据清单.md。
//
// 方言锚点（全部经 clientdata 按 2433 表名解析）：
//   - 0x0B 进场：idx178 Name / idx1468 MaxVisuals=230 / idx1447 ConfigID /
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
)

// Identity 会话身份（每次进场注入；官方每会话不同——0x0B 的 seq 实证
// 0x1521/0x2F05 非常量，obj/own 为玩家实体 PERSISTID）。
type Identity struct {
	ObjectID uint32 // 玩家 wire objectID（fork 客户端恒 1）
	OwnerID  uint32 // 玩家 wire ownerID（fork 客户端恒 1）
	Sequence uint32 // 0x0B 进场流水号（每次进场自增）
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
	Identity Identity
	Session  Session
	RoleName string // 角色名（m607 宽串）
	State    rolestate.RoleState
	Keybind  string // 存档键位串（0x28 档案 ShortcutKey；空=省略）
}

// DefaultIdentity fork 客户端方言的玩家恒等身份（1-1）：
// stage_main 开图与 0x11 记录绑定都依赖这一身份（b-line 定版）。
func DefaultIdentity() Identity { return Identity{ObjectID: 1, OwnerID: 1} }
