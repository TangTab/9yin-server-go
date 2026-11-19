// Package fsserver 复刻官方世界服逻辑模块 FsServer.dll（BGserverCJ\Debug
// 全符号快照，flexis 源码树 d:\flexis\client\fsserver\）。权威规格 =
// docs/official/FsServer全解与Go复刻.md（PDB 断言/跳表逐型实证）。
//
// 形态：GameWorld.bat → FsLoader.exe GameWorld.ini 加载
// FsKit+FsComm+FsServer（MainEntity=MainConsole，组合根 ..\Lua\GameWorld\main.lua）；
// ident 命名空间 world=0 的对端实体。GameUnite.ini 单进程形态亦包含本模块。
//
// 核心：六消息队列（inner/store/entry/member/public/charge）+ DoWait 主泵 +
// 世界 13 态/玩家 14 态/场景 4 态状态机 + CInterConnect 发送族 + 矩阵卡密保
// （CMatrixCode）+ 120B 区服信息 UDP 广播（与 FsLister world 建服包同源）。
package fsserver
