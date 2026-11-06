// Package entity 定义角色逻辑状态的权威枚举，以及把 internal/sim
// 组件值投影到协议编码层的快照 DTO。
//
// 曾经存在于本包的 Actor 运行时结构（自带互斥锁的胖对象）已被
// internal/sim 的组件存储取代：NPC 经 ECS 世界持有 Vitals/Brain，
// 玩家以 playerActor 的值字段持有 Vitals/QingGong/Motion。全部
// 原语逻辑（伤害/治疗/复活/打坐/瘀血/派生上限/轻功扣点）已逐行
// 移植至 internal/sim，其行为 oracle 见 internal/sim/sim_test.go。
package entity

const (
	LogicStateNormal   uint8 = 0
	LogicStateFighting uint8 = 1
	// LogicStateSitcross 是新版原生的 LS_SITCROSS 状态。动画由客户端
	// 负责，而何时允许进入该状态由服务器决定。
	LogicStateSitcross uint8 = 102
	LogicStateDied     uint8 = 120
)

// ActorState 是供协议层（0x10 ServerObjectProperty、0x0D AddObject）
// 消费的快照 DTO。它由 sim.Vitals.ActorState 及玩家侧的
// playerActor.snapshotLocked 组装；本类型不持有任何行为。
type ActorState struct {
	ObjectID, OwnerID    uint32
	Name                 string
	HP, BaseMaxHP, MaxHP int32
	HitHP                int32
	MP, BaseMaxMP, MaxMP int32
	QingGongPoint        int32
	MaxQingGongPoint     int32
	MaxQingGongPointAdd  int32
	QingGongConsumeMul   float32
	SpeedRatio           int32
	LogicState           uint8
}
