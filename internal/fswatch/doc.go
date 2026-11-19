// Package fswatch 复刻官方 FsWatch.dll（BGserverCJ\Debug，2014-03-22）：
// 运维看门狗服务器 GameWatch。
//
// 官方形态：FsLoader.exe GameWatch.ini 宿主装载 FsKit/FsSql/FsComm/FsWatch
// 四模块（Lua\GameWatch\main.lua 装配），FsWatch 只注册 GameWatch 一个实体，
// 承担：①Outer 文本行受面（8 命令 + Monitor/Query/Update 三密码 stricmp）；
// ②EchoSock 心跳 <info state= connect= />；③start_app/stop_app 进程守护；
// ④远程热更新（目录树 + 64KB 分块 + 大写 HEX + MD5）；⑤query 数据库代理
// （10 子命令 → CAccess 线程 → Publish* 回推）。
//
// 本包对齐外显协议行为（帧格式 payload+\r\n、控制字符分段、text varlist
// 编解码 $/#/*、\xXX 大写转义），MassSock/EchoSock/AppManager 以内嵌等价
// 实现，不依赖 FsKit/FsComm。
//
// 权威文档：docs/official/FsWatch全解与Go复刻.md
package fswatch
