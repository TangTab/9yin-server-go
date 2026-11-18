// Package entry 复刻官方接入网关进程（game_entry.exe + FsEntry.dll）。
//
// 证据源：BGserverCJ 快照 FsEntry.dll（PDB2.00 无 map，经字符串交叉引用 +
// FsSoloEntry.map 孪生符号反推）、Lua/GameEntry/{main,echo}.lua、
// GameEntry.ini 与 127.0.0.1.lic。权威文档：
// docs/official/FsEntry全解与Go复刻.md。
//
// 架构（官方四平面）：
//   - 外网面（CMassSockHook/CClients）：客户端 TCP，EE 转义 + XOR 线路，
//     纯代理——按 member 绑定状态把客户端消息转发给 world/member；
//   - 内网面（CAdapterHook/CInterConnect，FsComm Adapter 线路）：
//     u32 前缀帧 + {src,dst,type} 12 字节头，world=0、member=0x18+idx，
//     ident 注册（type=0xFFFF）；entry_forw_* 16 型控制协议；
//   - 维护面（EchoSock 15200+entry_id）：行协议反射 + quit；
//   - 广播面（BroadSock :15000 UDP）：0x3001 72 字节 entry 信息包。
//
// AoI：场景网格（自适应 cell∈{1,2,4,8}，cols≤0x200）+ 255×255 螺旋
// 偏移表（按距离升序）实现 send_to_sight 地理扇出。
//
// License：`<外网IP>.lic` 256 字节 = 3DES-EDE-ECB(md5(IP+盐16+0xCC×16))
// 加密的 {8B 魔数, 248×00}；魔数对即进程身份域（TimerCheck 周期复验）。
package entry
