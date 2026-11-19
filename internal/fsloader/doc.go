// Package fsloader 复刻官方 FsLoader.exe（Fs 框架进程装载器）。
//
// # 官方定位（逆向定案，快照 BGserverCJ\Debug\FsLoader.exe + FsLoader.pdb；
// 同源 FsSoloLoader.map 提供全符号；详见 docs/official/FsLoader全解与Go复刻.md）
//
// FsLoader.exe 是 16KB 级薄壳 main.cpp（d:\blackgoldserverengine\serverengine\
// branches\server(20120807)\fsloader\main.cpp），进程形态为：
//
//	FsLoader.exe [ini名] [附加参数...]
//
// 五步序列（_main 反汇编，0x4449b0）：
//  1. SetConsoleCtrlHandler（Ctrl+C → 置退出标志）；失败直接退出
//  2. app_load_core_dll("FxCore.dll")：GetProcAddress 五导出，
//     FxModule_GetType()=="FxCore"、FxModule_GetVersion()==1003(0x3EB)
//     双门校验，失败文案 "(LoadCoreDll)*"
//  3. ini 发现：app_get_ini_name(argv[0])→"<exe>.ini"，空回退字面量
//     "FsLoader.ini"；argc>1 时 **argv[1] 覆盖 ini 名**（GameLister.bat
//     = `start FsLoader.exe GameLister.ini`）；argv[2..] 追加进配置
//  4. app_load_core_ini(ini, CCoreConfig&)：官方 CIniFile 方言解析
//     （fskit.IniFile 逐字节同方言），脚本/资源目录存在性校验
//  5. CoreInitialize(config)→ICore* → CoreStartup(core, config) →
//     泵循环 `while(!退出 && core->Process())` + Sleep 毫秒节流 →
//     CoreShutdown
//
// 内核侧（FxCore.dll，CCore::Startup 错误串定案序列）：VFS（FileSystem/
// FileConfig）→ CDllManager 装载 [Module] 模块（空=全部）→ 创建 MainEntity
// → 绑定 MainLogic（可选）→ 跑 MainScript → 泵驱动 GameTimer/异步脚本/
// 脚本热重载（[Script] Reload=1）。
//
// # Go 映射
//
//	官方 C++		Go
//	FsLoader.exe壳		Loader.Main（loader.go）
//	app_load_core_dll	Core 工厂 + CoreType/CoreVersion 双门（core.go）
//	app_load_core_ini	LoadCoreConfig（config.go）
//	CCoreConfig		CoreConfig
//	FxCore.dll CCore	Core 接口 + EngineCore（core.go，装配 engine.App）
//	CDllManager		ModuleRegistry（module.go，类型/版本/重名四校验）
//	FxModule 8 导出		Module 接口（Type/Version/Space/Init/Creators）
//	ERRORyyyymmdd….log	WriteErrorLog（crashlog.go，panic→栈转储）
//
// 崩溃面：官方 MiniDumpWriteDump 产 DUMP*.dmp + ERROR*.log（寄存器/栈）；
// Go 以 recover + goroutine 栈文本等价（dmp 为 Win32 专有，不复刻）。
package fsloader
