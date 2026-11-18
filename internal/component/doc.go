// Package component 存放 structgen 的生成产物：组件定义、
// 属性元数据（sync/save/realtime 标志）、属性号映射表。
//
// 纪律：本目录只放生成代码，禁止手写、禁止手改——
// 任何时候可以整目录删除后重跑 tools/structgen 重建。
// 事实源是 testdata/struct/ 下的 struct XML（源自原版 Res/struct/，
// 定义了每个游戏对象的属性名/类型/公开性/存档性）。
//
// 组件粒度原则：按 struct XML 的对象文件分组为少量大组件
// （如 CombatProp / MoveProp / BagProp），绝不一属性一组件。
//
// 生成的元数据同时驱动三处：sync（客户端推送）、persist（落库字段）、
// proto（属性号映射）——单一事实源，三处共用。
//
//go:generate go run ../../tools/structgen -in ../../testdata/struct -out .
package component
