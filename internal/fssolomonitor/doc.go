// Package fssolomonitor 复刻官方 FsSoloMonitor.dll（反作弊监控服务器，
// game_monitor.exe 承载，BGserverCJ\Debug 小写系 solo 体系）。
//
// 官方形态：solo 集群各 game_world（FsGame.dll）与 game_store 经文本
// 行协议（Outer :4008，charge 同源 text_encode/decode 方言）把玩家
// 资金/物品/登录流水上报到本进程；实体在内存维护 7 张监控表，按
// Res\MonitorLimit.ini 限额判定异常，告警三路出口：CORE_TRACE 日志、
// alert_monitor echo 广播（15012 多客户端版）、MySQL nx_monitor 落库。
//
// 权威文档：docs/official/FsSoloMonitor全解与Go复刻.md（§十一 Go 设计）。
//
// 复用与差异：
//   - internal/charge 的 VarList 文本方言（与官方 TextProtocol 同源）；
//   - internal/reactor（CreateListener/CreateTimer 等价物）；
//   - 状态机/echo 骨架同 internal/fssologlobal；echo 口为官方多客户端
//     版（上限 64 连接、SendLine 广播、end 批量断开）；
//   - CTaskQueue 32MB 双缓冲 → buffered chan + 单 goroutine；
//   - 不复刻：AppRecord 共享内存（状态行走日志）、LogShareMem 生产者
//     （日志接 logf）、命名互斥体 exit(0)（返回错误）、
//     MonitorCreateItem Debug 基准（按官方语义降级为 no-op+日志——官方
//     实现会摧毁 CreateItemDetail 表，复刻保留表、仅记录）。
//
// 官方失配保持：MonitorLimit.ini 的 [system]（小写）节五键
// Livepoint/DiffCapital/ExchangeAmount/RoleNum/OnlineRoleNum 在官方
// ini 中并不存在（官方 ini 写在 [System] 大写节）且 CReadIni 哈希
// 大小写敏感 → 五键恒取缺省值；本包 INI 解析同样按节名/键名大小写
// 敏感处理，行为与官方逐点一致。
package fssolomonitor
