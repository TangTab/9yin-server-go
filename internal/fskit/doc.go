// Package fskit 复刻官方 FsKit.dll（Fs 框架工具箱模块）。
//
// 权威依据：BGserverCJ\Debug\FsKit.dll（带 .map/.pdb 的
// Debug 符号快照）逆向结论，全解见 docs/official/FsKit.dll全解与Go复刻.md。
// 官方模块构成：
//   - CIniFile   INI 引擎（utils/inifile.cpp，非实体）
//   - IniDocument 实体（包装 CIniFile，21 方法）
//   - ArrayList  实体（子项树，大小写敏感 131 哈希）
//   - FileSearch 实体（目录/文件枚举，"%s*.*"/"%s%s"/"%s*.%s" 通配方言）
//   - MainConsole 实体（14 色 SetColor + CRLF WriteLine）
//   - 6 个全局扩展函数（split_string/split_file_path/split_file_name/
//     get_file_time/get_full_path/gen_unique_name）
//   - FxModule 8 导出装载协议（CRT 静态构造即注册）
//
// Go 侧不逐字模拟 IEntCreator 链表与 CRT debug 堆，改为声明式注册表
// （module.go）；INI 引擎则逐字节对齐官方 Build/Save 方言（inifile.go）。
package fskit
