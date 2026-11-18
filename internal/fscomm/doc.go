// Package fscomm 复刻官方 Fs 框架共享通讯组件层 FsComm.dll
// （BGserverCJ\Debug\FsComm.dll，2013-07-05 编译，带 .map/.pdb 全符号）。
//
// # 官方定位
//
// FsComm 是 Fs 系服务器的实体类通讯模块：导出标准 FxModule 接口
// （FxModule_GetEntCreator 等 8 个），向实体系统注册一批网络/运维
// 实体类，由 Lua 装配脚本或宿主 C++ 按名创建。使用方普查（Debug 全
// 目录二进制字符串）：大写 Fs 分布式体系（FsEntry/FsMember/FsPublic/
// FsStore/FsLister/FsServer/FsWatch/FsCharge/FsDumper）大量创建这些
// 实体；小写 solo 体系自带 CReactor（见 internal/reactor），仅用到
// LogicLoader/LogPool。对应本项目「分布式拆分」里程碑的 proc TCP 总线
// 与运维件。
//
// # 组件映射（官方类 → Go 类型）
//
//	Adapter（IOCP 内网侦听）	Adapter（帧模式/Raw 模式二合一）
//	MassSock（海量接入，裸字节）	Adapter{Raw:true}
//	MultiSock（自动重连客户端池）	MultiSock
//	UserSock（单客户端，同步/异步）	UserSock
//	EchoSock + EchoSockHook		EchoSock（行协议+on_input 分发内联）
//	BroadSock（UDP 广播）		BroadSock
//	AppManager + CAppBook		AppManager（进程注册表+单实例+清扫）
//	LogPool（按日日志池）		LogPool
//	LogicLoader（逻辑模块装载）	LogicLoader（Go 原生注册表）
//	CSockFIFO/SockBuf		msgFIFO（有界消息队列）
//	WinSocket（WSA 启动壳）		Go 运行时自带，不复刻
//
// # 线路格式（反汇编定案）
//
// 内网链路（Adapter/MultiSock 的 FIFO/WriteOutBuf 路径）：
//
//	帧 = [4 字节小端长度 N][N 字节帧体]，连续打包无分隔
//
//	Adapter::ProcessRead（0x10056120）：逐字节凑 4 字节头（connect_t
//	+0x74/+0x78），读出 N 后等满 N 字节再整帧投递回调；半帧留缓冲。
//	Adapter::WriteOutBuf（0x10055a60）：FIFO 逐条 Peek，写 *(u32*)dst=N
//	后 memcpy 帧体，直到 OutBufLen 装不下下一帧为止。最大帧 =
//	get_max_send_size(OutBufLen) = OutBufLen-4（0x100551a0）。
//	该格式即 internal/net.PrefixFramer 的原型，本包直接复用。
//
//	MultiSock/MassSock 的读路径向宿主回调**裸字节块**（0x10076330/
//	0x100708f0，成帧留给宿主 hook，官方客户端侧外网编码在 hook 内做）；
//	写路径同样经 WriteOutBuf 加前缀。本包以 Raw 模式覆盖该语义，
//	Go 侧配对两端同用 PrefixFramer，线上字节与官方一致。
//
//	EchoSock 行协议（0x10062790 循环）：recv 缓冲逐字节扫描，'\n'(0x0a)
//	或 '\r'(0x0d) 任一即行终止；行缓冲上限 1023 字节（超长字节丢弃），
//	空行不分发；SendLine 原样发送不补终止符（Lua 侧自带 "\r"）。
//	分发：按空格/制表符切词，首词存在则先试 "on_input_<首词>" 回调，
//	未命中回落 "on_input"(整行)（0x10064382）。
//
// # AppBook 跨进程注册表（布局留档）
//
//	共享内存名 "memory_<AppName>"，单实例互斥量 "mutex_<AppName>"
//	（.data 0x100e9b24/0x100e9b28 双前缀指针实证）。总长 0x6208 =
//	8 字节头{count,?} + 32 槽 × 0x310 book_data_t{used:bool,
//	name[0x80], path[0x100], addr[0x100], info[0x80], 尾部 state/
//	procid 等}。AppManager::DoWait（0x1005cb20）每 3000ms 持锁扫描，
//	对每槽重建 "mutex_<name>" 探活，互斥量消失（进程死亡）即摘除该槽。
//
// Go 侧以「锁文件 + JSON 注册表 + pid 探活」复刻同语义（Windows
// 命名对象不可移植；探活口径从 named-mutex 存在性换为进程存在性，
// 判定结果等价）。布局细节不作二进制兼容——Go 服只与 Go 服互通。
package fscomm
