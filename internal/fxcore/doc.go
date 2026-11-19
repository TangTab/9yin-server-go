// Package fxcore 复刻官方 FxCore.dll（Fs 框架引擎内核，flexis 血统）。
//
// # 官方定位（逆向定案，快照 BGserverCJ\Debug\FxCore.dll 1.47MB +
// FxCore.pdb 5.16MB；源码树 d:\flexis\client\fxcore\{core,dllmain}.cpp；
// 全解见 docs/official/FxCore全解与Go复刻.md）
//
// FxCore = CCore 运行时：装载器（FsLoader.exe）经 FxModule_CoreInitialize
// 取得 ICore*，CoreStartup(config) 五段启动，随后泵循环逐拍调 vtable+0x4c
//（CCore::Execute）。管理器全家（PDB 符号定案）：
//
//	CEntManager/CEntFactory	实体注册表/工厂（命名实体、帧执行表）
//	CIntManager		接口注册表（引用计数 + 帧执行表）
//	CFuncManager		全局扩展函数
//	CLogicManager/CLogicInfo	逻辑类（回调链）
//	CGameTimer		帧时钟（真实 dt + 定步推进）
//	CAsyncManager/CAsyncProc	异步过程 + 事件等待（GenEvent 唤醒）
//	CScpManager		脚本注册表 + mtime 热重载（Reload=1 每帧轮询）
//	CScpDebugger/CLuaScript	Lua 面（nx_* 73 函数；Lua VM=里程碑二）
//	CHookManager		钩子
//	CIdManager		PERSISTID 发号
//	CLogTool/CLogFile	trace.log + SaveLogList
//	CMemoryPool/CAllocator	块分配 + 泄漏转储（allocator.log）
//
// # Execute() 泵语义（0x100d7b90 反汇编定案——本包核心复刻物）
//
//	if 停机: return false
//	dt = GameTimer.Update()（可经帧回调重塑）
//	totalTime/virtualTime/lastElapsed 三钟推进
//	AsyncManager 第一遍
//	if fpsCap>0:                       # 定步模式
//	    step = 1/fpsCap
//	    for i in 0..9:                 # 每拍最多补 10 帧
//	        if frameClock > virtualTime: break
//	        Idle(); Timer.Execute(step); Ent.Execute(step)
//	        frameClock += step（第 10 次强制对齐 virtualTime）
//	    offset = virtualTime + step - frameClock  # assert >= 0
//	    Ent.ExecuteOffset(offset)      # 尾帧余量
//	else:                              # 全速模式
//	    Idle(); Timer.Execute(dt); Ent.Execute(dt); Ent.ExecuteOffset(0)
//	AsyncManager 第二遍
//	ScpManager.Execute()               # 脚本 mtime 热重载
//	统计（idle/timer/frame/offset/total 五钟）+ 日志节拍 + 事件帧速器
//	return true
//
// Startup 五段序（错误串定序）：VFS→脚本系统→模块循环（上限 100）→
// 主实体→主脚本（默认 "main"）→主逻辑（可选）。
//
// # Go 映射
//
//	官方			Go
//	CCore			Core（core.go，实现 fsloader.Core）
//	CCore::Execute		(*Core).Process（定步泵 + Stats）
//	CGameTimer		GameTimer（timer.go）
//	CEntManager/Factory	Entity/EntityStore（entity.go）
//	CAsyncManager/CAsyncProc	AsyncManager（event.go，单线程泵驱动）
//	CScpManager		ScriptManager（script.go，mtime 热重载）
//	nx_* 73 函数		API 表 + Go 桥（api.go；Lua VM 里程碑二）
//	CLogTool/trace.log	LogTool（logx.go）
//	XLog 共享内存日志	暂以 LogTool 文本等价（Win32 file-mapping 专有）
package fxcore
