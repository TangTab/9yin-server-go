// Package snaileditornet 复刻官方 SnailEditorNet.dll（BGserverCJ\Debug，
// 2014-03-22）：SnailEditor 编辑器管线网络库。
//
// 官方形态：单例 g_SnailEditorNet（GetSnailEditorNet 导出）全量转发
// g_Con:FileCon。双角色：编辑器 GUI 侧 ServerInit/ServerListen（CIOCP +
// ServerSession 多连接管理/SendAll/IsSendFile/0x3F1 广播）；游戏服侧
// ClientInit/ClientConnect（ClientSession：ProcessMsgThread 命令队列 +
// ReadSharingMemoryThread 100ms 轮询共享内存 SNAILEDITOR_SvnUpdateFinish +
// 双回调 FuncDelaySvnUpdate/PushMsg 注入）。文件传输 0x200/0x1000 分块，
// SVN 走 TortoiseProc /command:update /path:… /notempfile /closeonend:1
// （TortoiseProc 路径取 HKLM\SOFTWARE\TortoiseSVN\ProcPath）。
//
// 线协议：帧 = {u32 totalLen(含 8B 头); u32 msgId; payload}，msgId 基址
// 0x3E8（FMSG 1000..1009；0x3F1=FMSG_SERVER_SVN_UPDATE 路径列表）。
//
// 本包以 goroutine 等价实现双端（IOCP→net 包），保留帧/命令/共享内存桥
// 语义；命令常量按处理体语义命名（精确名需编辑器客户端交叉验证）。
//
// 权威文档：docs/official/SnailEditor四件套全解与Go复刻.md
package snaileditornet
