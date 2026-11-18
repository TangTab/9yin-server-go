// Package engine 是引擎壳，对应原版 FsKit 的启动装配层 + FsLoader 的职责：
// 读启动配置、按配置装配 Module/System、驱动主循环。
//
// 与原版的对应关系：
//
//	FsLoader.ini [Module] 段  →  本包的模块清单
//	logic_class.xml           →  逻辑类注册表（System 装配表）
//	MainEntity/MainScript     →  进程主实体 + main 引导（Go 里是装配函数）
//
// tick 模型：单线程主循环（对应原版 Sleep=-1 不休眠循环），每帧顺序执行
// Scheduler 里的 System；并发分片留到场景层（aoi/scene 按 shard 开 goroutine），
// 本包不提前引入复杂度。
package engine
