// Package snailsharemem 复刻官方 SharingMemory.dll（BGserverCJ\Debug，2014-03-22）：
// SnailEditor 管线的跨线程/跨进程共享内存 IPC 库。
//
// 官方形态：COM 风格（IMyUnknown {dtor,AddRef,Release}——无 QueryInterface；
// GetFactory→CFactory::CreateInstance("SharingMemoryRead"/"SharingMemoryWrite")）。
// 布局为 0x814 字节单帧信箱：+0x000 vtable(仅写端进程有效)、+0x004 data[0x800]、
// +0x804 dataLen、+0x808 valid、+0x80C readableBitmap、+0x810 registeredBitmap。
// 映射名 "SNAILEDITOR_" + 用户名（swprintf 宽字符），掩码 0xF001F。
// 读端创建（CreateFileMappingW 0x814，首建者零初始化头部并注册读者 ID），
// 写端打开（OpenFileMappingW，映射不存在即失败）。
// 协议：写端 ASSERT 有读者注册；Write 载荷截 0x800，写后 readableBitmap=
// registeredBitmap 且 valid=1；读者 Read 前置三查（valid/注册位/可读位），
// ReadEnd 清自身可读位，全部读者读完（readableBitmap==0）后写端 GetWritable。
//
// 本包 Mailbox 完整复刻该协议（字节布局对齐，可跨进程与原生 DLL 互通）；
// Windows 映射经 golang.org/x/sys/windows（缺省回落内建 syscall 声明）。
//
// 权威文档：docs/official/SnailEditor四件套全解与Go复刻.md
package snailsharemem
