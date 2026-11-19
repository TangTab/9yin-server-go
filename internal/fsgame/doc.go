// Package fsgame 复刻官方 FsGame.dll 的模块框架与首批基础模块。
//
// 官方形态（docs/official/FsGame全解与Go复刻.md）：
//   - FsGame.dll（"FxGameLogic"，版本 0x3043）由 FsMember/FsSoloMember 宿主经
//     LogicLoader 装载；241 个 XxxModule 经 IModuleCreator 静态构造链接入，
//     Init 顺序 = 构造逆序（LogicClass → StaticDataQuery → … → GM 收尾）。
//   - ILogicModule 生命周期五槽：Init / BeforeLaunch / LoadResource / Shut / dtor；
//     Init 期完成六类注册：类回调（AddClassCallback）、事件（AddEventCallback）、
//     客户端整型命令（AddIntCustomHook）、心跳（DeclareHeartBeat）、
//     临界变量（DeclareCritical）、记录钩子（DeclareRecHook）。
//   - IKernel（宿主 CKernel，507 虚槽）提供类/对象/属性/记录/场景/存档外观；
//     IGameObj（CGameObject，120 方法）= 属性表 + Record + 子对象树 + Archive。
//   - player 七事件生命周期：
//     OnCreateClass → OnRecover → OnEntry → OnReady → OnEntryGame → OnStore → OnDisconnect。
//
// 本包范围（第一批·登录可玩最小闭环）：
//
//	框架：VarList / 类 schema（logic_class.xml + struct/*.xml，等价
//	createclass_from_xml）/ GameObject / 六类回调总线 / Kernel / 模块注册链。
//	模块：PlayerBase / SceneBase / Location / Motion / ContainerBase /
//	ItemBase / Capital / CreateRole / Level / Request（注册面与关键行为
//	均按官方反汇编定案；注册表见各模块 Init 注释）。
package fsgame
