// Package fsunite 复刻 game_unite.exe（合一服）的装配编排。
//
// 官方形态：game_unite.exe 与 game_world.exe/game_entry.exe 等全部
// game_*.exe 是同一通用 solo 宿主二进制（MD5 195e1ce1…），行为差异
// 全在 ini + Lua 引导——game_unite.ini 装载 11 个 solo 模块全量，
// MainEntity=SoloWorld，脚本 ..\lua\solounite\main.lua（1380 行）在
// 单一进程内创建七个实体并编排：
//
//	logger(SoloLog) → world(SoloWorld) → store(SoloStore) →
//	entry(SoloEntry) → public(SoloPublic) → auction(SoloAuction) →
//	member(SoloMember)
//
// 全部实体 InnerPort=0（系统分配后广播自宣回填）、共享同一 UDP 广播
// 口（SO_REUSEADDR 共绑，Win11 实测 127.255.255.255 定向广播全量
// 投递含发送者自收）；world 持 echo 15001 维护面（命令集比 soloworld
// 多一个 perf）；store 注册 29 张自定义日志表（与
// fssolostore.NewLogTableSet 默认集同源）。
//
// 启动序（main.lua 408-421）：logger→world→store→entry→public→
// auction→member；停机序（force_close 927-952）：store→auction→
// public→entry→member→world→logger。
//
// 已知偏差（详见 docs/official/game_unite与game_world全解与Go复刻.md）：
//   - fsgame.dll/SnailEditor.dll/fspublogic.dll/fsauctionlogic.dll
//     逻辑层不装载（"load loigc module" 日志行呈现，官方拼写保留）；
//   - DB 五连接串仅呈现，后端 = fssolostore.NewMemoryStore（内存）；
//   - fssolo* 服务无 LogShareMem 生产者 → logger AutoExit 看门狗关闭
//     （否则合一起动 ~5s 后被误杀）；
//   - world 的 LoadNameFilter/LoadCharImage/LogLauncher、member 的
//     ScriptPath/HideEmptyProperty/location.xml 为呈现字段（包现状）。
package fsunite
