// Package fslister 复刻官方 FsLister.dll（大写黑金 Fs 体系区服列表
// 服务器，GameLister.bat → FsLoader.exe GameLister.ini 三件套之一）。
//
// 权威规格 = docs/official/FsLister全解与Go复刻.md（2026-11-19
// PDB 全符号反汇编终局，tmp_research/fslister/fl_dis.py）。三面：
//
//   - :4000 客户端面（ClientServer）：明文行协议——逐字节状态机、
//     控制字符（含 \r）丢弃、0x1000 行缓冲截断、\n 分发、请求必须
//     精确等于 "svrlist"、每连接只应答一次、应答后 5s 温和关闭、
//     60s 无请求超时；应答 = login.EncodeServerList（$svrlist 方言，
//     与 FsSoloLister golden 同源）
//   - :15000 注册面（RegistrationServer）：UDP 双心跳——world 包(120B)
//     建服/刷新服级心跳闩锁，entry 包(72B) 建线/刷新线级闩锁；报文头
//     [kind][magic=0x300] + 分区门（DistrictId 必须相等）；5s 扫描，
//     闩锁丢失一次即除名/摘线
//   - 表与选线（GameLister）：256 服 × 16 线；应答时每服选 cur/max
//     负载率最小的线（IDC>0 时仅 line_id==IDC 的线），busy =
//     服级 cur*5/max（login.BusyLevel 同公式）
//
// 与 internal/login.Lister（小写系 FsSoloLister 复刻，静态条目 +
// 版本门）互补：本包是集群聚合形态。EchoSock/AppManager/LogPool
// 运维件不在此复刻（fscomm 已有等价物，单服部署不接线）。
package fslister
