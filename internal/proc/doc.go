// Package proc 定义进程拓扑与跨进程消息路由，对应原版
// Entry/Lister/Member/World/Public/Store/Auction 多进程体系。
//
// 核心抽象是 Bus 接口（Publish / Subscribe / Request）：
//   - solo 形态：inproc 总线实现，进程内直连（第一里程碑）
//   - 分布式形态：TCP 实现替换，业务层零改动
//
// 这是对原版"Fs* 与 FsSolo* 同逻辑不同壳"做法的翻译：原版靠两套 DLL，
// 本实现靠同一业务代码 + 可替换的总线实现。进程角色定义
// （对应原版各 game_*.exe 的职责边界）也放在本包。
package proc
