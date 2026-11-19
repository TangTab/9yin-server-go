// Package fsmember 复刻官方 FsMember.dll + game_member.exe 的场景服务
// 进程面：内网受面（Adapter + ident 路由）、外发面（InterConnect）、
// world/public/store 三消息队列与 17+14+4 型处理器、场景/玩家注册表、
// 状态机（Unknown→Opened→Closing→Closed）、member 信息周期广播
//（kind=3/magic=0x300）、维护面（EchoSock）与进程注册（AppManager）。
//
// 权威文档：docs/official/FsMember全解与Go复刻.md（消息表/ident 命名
// 空间/payload 偏移均以 FsMember.dll 反汇编 + FsMember.pdb 符号定案）。
package fsmember
