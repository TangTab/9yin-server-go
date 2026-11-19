// Package fssoloworld 复刻官方 FsSoloWorld.dll（solo 集群世界服；
// 宿主 game_world.exe，MainEntity=SoloWorld）。
//
// 权威逆向文档：docs/official/FsSoloWorld全解与Go复刻.md。
// 框架层语义：内网受面 + 广播发现五分支主动外连 + 13 态世界机 +
// 14 态玩家机 + 场景表 + NameUid 名字登记 + entry/store/member 三消息面 +
// charge 文本代理 + echo 维护控制台。
// 不复刻（既定边界）：CMatrixCode 验证码位图 / Dump* 工具族 / 对象内核。
package fssoloworld
