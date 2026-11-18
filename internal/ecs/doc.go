// Package ecs 实现手写的 sparse-set ECS 内核，是整套架构的对象模型基础，
// 对应原版 Fs 框架中 FsKit/FxCore 承载的实体体系。
//
// 纪律（必须守住）：
//   - 本包零依赖：不 import 本仓库任何其他包；组件通过注册接口进入，
//     本包不认识 internal/component。
//   - EntityID 携带 generation 位，防悬挂句柄——所有跨系统传递的对象引用
//     一律是 EntityID，不是指针。这同时是对原版 nx_exists 句柄语义的翻译。
//
// 计划形态：
//
//	World        世界容器：实体表 + 各组件存储 + 查询缓存
//	EntityID     uint64：index | generation
//	Storage[T]   sparse-set 组件存储
//	Query        按 (All/Any/None, ...) 组件集合迭代
//	System       每帧或事件驱动的逻辑单元
//	Scheduler    System 调度（先单线程 tick，预留按场景分片并发）
//
// 规模约束：核心 ≤ 2000 行，不引入外部 ECS 框架——本项目的查询模式固定
// （AOI/同步/定时器），自写比通用框架可控。
package ecs
