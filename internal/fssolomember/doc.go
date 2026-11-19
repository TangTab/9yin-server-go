// Package fssolomember 复刻 FsSoloMember.dll（小写系单机形态场景服务器）。
//
// 官方形态（docs/official/FsSoloMember全解与Go复刻.md，VA 证据见其 §十二）：
//
//	game_member.exe 裸跑 → [Module] fskit/fscomm/fscollide/fssolomember.dll，
//	MainEntity=SoloMember，脚本 ..\lua\solomember，玩法层另载 [Logic] fsgame.dll。
//	DLL 内嵌 CReactor/CInnerHook/CEchoHook + 整套游戏对象内核（Kernel/
//	GameObject/GameScene…，2.2MB .text）——对象内核按既定路线由 ECS+Lua
//	承接，本包复刻**框架层**：
//
//	 ├ SoloMember 实体面     → Server（42 属性 ctor 缺省对齐 §二）
//	 ├ Start/Stop/Execute    → Start/Stop（§三装配序）
//	 ├ TimerCheck/Beat/Scan  → Tick/定时器等价
//	 ├ UDP 15000 服务发现    → 自宣 kind=3 0x28B + on_broadcast 四分支外连（§四）
//	 ├ CInnerHook            → InnerLink（selfComm=0x18+MemberID；§五）
//	 ├ ProcessInnerMsg 路由  → processInnerMsg（world/public/auction/entry/
//	 │                        store id2 重映射，§五/§六）
//	 ├ 状态机五态            → state.go（0x29 引导→Loading 三闩→Opened→
//	 │                        Closing 空场景门→0x1F 关播→Closed，§七）
//	 └ entry 客户端下行      → SendToClient/2/Broadcast/Channel（§八）
//
// 不复刻：EchoHook 调试台方言（fscomm 等价物）、AppRecord/LogShareMem/
// LogLauncher 进程监管件、CMemPool/CClassSet 对象池与内核、Lua 宿主
// （Go 原生）、互斥单实例（单进程部署无对拍方，呈现等价日志）。
package fssolomember
