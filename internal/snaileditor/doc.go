// Package snaileditor 复刻官方 SnailEditor.dll / SnailEditorModule.dll
// （BGserverCJ\Debug，2014-03-22）：游戏服宿主内的编辑器逻辑模块
// （FxGameLogic 五导出面）。
//
// 官方装配（SnailEditorServerModule）：
//   - Init(kernel)：注册 scene.OnCreate / player(0x13) 事件 +
//     "SnailEditorServerModule::HB_ProcessMsg" 心跳；
//   - StartSnailEditor：HKLM\SOFTWARE\TortoiseSVN\ProcPath 取
//     TortoiseProc 路径 → 读 ini\SnailEditorServerModule.xml
//     （config:SvnLocalRootPath/SvnServerIP/SvnServerPort + watch 段）→
//     LoadLibraryA("SnailEditorNet.dll")+GetSnailEditorNet →
//     SetStrSvnLocalRootPath/SetStrSvnTortoiseProc/ClientInit/ClientConnect/
//     SetDelaySvnUpdate(FuncDelaySvnUpdateCallback)/
//     SetSvnUpdateCompleteCallback(PushMsg)/SetSvnWatchPathes；
//   - PushMsg(len,msg)（Net type=1 回调）：0x800 缓冲 → MsgManager 入队；
//   - FuncDelaySvnUpdateCallback（Net type=0）：并入延迟 SVN 更新路径表；
//   - HB_ProcessMsg（每帧）：PopMsg → 按 '*' 分割路径 →
//     IsFileInReloadPath 过滤 → EditorUpdate::Reload。
//
// EditorUpdate 命令面（CustomEditor/OnCustomEditor*）以可注册分发表提供
// （IniWrite/Del/Add/End、WriteObject{,Base,Ai,Task,Event,CustomEvent}、
// ReloadRes、Start/End/InfoUpdate）；varprop/extend/custom/geometry
// （SnailEditorModule 增量）仅保留 API 面常量，玩法层后置。
//
// 权威文档：docs/official/SnailEditor四件套全解与Go复刻.md
package snaileditor
