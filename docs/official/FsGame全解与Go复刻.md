# FsGame 全解与 Go 复刻（框架层）＋玩法模块索引表

> 对象：`BGserverCJ\Debug\FsGame.dll`（15,822,848 B，2014-03-22，32 位）+ `FsGame.pdb`（96 MB，含全量类型）+ `FsGame.map`（37 MB，146,013 个公共符号）
> 官方源码路径（__FILE__ 断言串泄露）：`d:\bg_source\server\fsgame\`（16 个子目录，169 个 .cpp/.h 泄露，见 §6）
> 版本：`FxGameLogic_GetVersion() = 0x3043`；模块类型串 `"FxGameLogic"`
> 本文范围：**框架层全解 + 全部 241 个模块的玩法索引表**；玩法逻辑本体按约定**暂不复刻**，仅登记位置。

FsGame.dll 是整个 9yin 服务端的**游戏逻辑总库**：241 个逻辑模块（`ILogicModule` 实现类）
经由 241 个 `IModuleCreator` 静态构造链注册，寄居在场景服宿主（FsMember/FsSoloMember，由
game_member/game_unite/game_world 等进程承载）提供的 `IKernel` 内核外观之上，覆盖
玩家生命周期、场景/副本、AI、战斗、物品、任务、社交、经济、活动等一切玩法。统计面：

| 面 | 数量 |
|---|---|
| 公共符号（.map） | 146,013 |
| obj（编译单元） | 766 |
| `*Module.obj` | 183（另有 58 个非 Module 后缀模块 obj） |
| 模块创建器（IModuleCreator 链） | **241** |
| IKernel 内核 API（虚槽） | **507** |
| IGameObj 游戏对象 API | 120 方法（87 槽抽象面） |
| 客户端整型命令号 | **359 个不同号**（→ cmd_map.json） |
| 服务端心跳（DeclareHeartBeat） | 230 个 |
| 临界变量回调（DeclareCritical） | 167 个 |
| Lua 扩展函数（`nx_*`） | **473 个** |
| 逻辑类 | player/scene/role + BattleCarrier/ConfigID/ConvoyNpc/energy_well/Camp/Job… |

---

## 1. 定位与装载链

### 1.1 宿主拓扑

```
game_unite.exe / game_member.exe / game_world.exe（FsLoader 壳）
  [Module] fskit.dll fssql.dll fscomm.dll fscollide.dll fssololog.dll
           fssoloentry.dll fssolostore.dll fssolomember.dll fssoloworld.dll fssolopublic.dll …
                              └─ FsSoloMember 实体（场景服框架；Go 复刻 internal/fssolomember）
                                   └─ SoloMember::Start → LogicLoader::Load
                                        LoadLibrary("fsgame.dll")             ← 本文对象
                                        FxModule_GetType     → "FxGameLogic"（类型门，fscomm::LogicLoader 已复刻）
                                        FxModule_Init(g_pCore)                ← 宿主注入 ICore*
                                        FxGameLogic_GetVersion → 0x3043
                                        FxGameLogic_GetModuleCreator → 链头    ← 遍历创建全部模块
                                        （失败串："(SoloMember::Start)logic module interface error" /
                                          "Logic module no interface !" / "(SoloMember::Start)no logic loader"）
```

网络形态（GameMember.bat/GameWorld.bat → FsServer.dll 宿主）走同一 FxGameLogic 协议（FsMember.dll 内含同样装载串）。

### 1.2 五导出与全局

| 导出 | RVA | 行为 |
|---|---|---|
| `FxModule_GetType` | 0xB53CC0 | 返回 `"FxGameLogic"`（.rdata 0xDABB98） |
| `FxModule_Init` | 0xB53CF0 | `g_pCore = arg`（.bss 0xEB1900，`ICore*`） |
| `FxGameLogic_GetVersion` | 0xB53D20 | 返回 `0x3043` |
| `FxGameLogic_GetCreator` | 0xB53D50 | 返回 `g_pCreator`（0xEB1904，`ICreator*`，运行期值） |
| `FxGameLogic_GetModuleCreator` | 0xB53D80 | 返回 `g_pModuleCreator`（0xEB1908，**IModuleCreator 链头**） |

FsGame 不导入任何网络 DLL（无 ws2_32）：全部网络/IPC 经宿主（FsComm Reactor）与 IKernel 完成。
导入面仅 KERNEL32/USER32/ADVAPI32/OLE32(CoCreateGuid)/WINMM(timeGetTime) + 共享内存（FileMapping 族）。

### 1.3 模块创建器链（IModuleCreator）

- 全局：241 个 `XxxModuleCreator` 对象（.data 0xEB190C 起，每 8 字节一个）。
- 静态初始化：241 个 `??__EXxx_ModuleCreator@@YAXXZ`（.text 0xD5B170 起）按地址升序执行，
  每个构造器做 `m_pNext = g_pModuleCreator; g_pModuleCreator = this`（头插）。
- **构造序 = ctor RVA 升序**（GMModule 0xB54330 最先 … LogicClassModule 0xC0E290 最后）；
  链头 = 最后构造者 = LogicClassModule；宿主自链头遍历 →
  **Init 顺序 = 构造逆序：LogicClass → StaticDataQuery → StaticInstance → SystemConfig → …玩法… → GMModule 收尾**。
- `IModuleCreator` 虚面（4 槽）：`~dtor / Name() -> "XxxModule" / Create() -> ILogicModule* / Destroy(m)`。
  例：PlayerBaseModuleModuleCreator vtable @ 0xDACF18（Name=0xB8E850、Create=0xB8E880）。

### 1.4 ILogicModule 生命周期（5 虚槽）

| 槽 | 方法 | 语义 |
|---|---|---|
| 0 | `~ILogicModule` | 标量删除析构 |
| 1 | `Init(IKernel*) -> bool` | 注册期：类回调/事件/命令/心跳/临界/记录钩子/Lua 扩展全在此 |
| 2 | `Shut(IKernel*) -> bool` | 关停 |
| 3 | `BeforeLaunch(IKernel*) -> bool` | 默认实现 0x3A3590（可覆盖） |
| 4 | `LoadResource(IKernel*) -> bool` | 读静态配置（ini/*.xml，经 IKernel::PreloadConfig） |

例：PlayerBaseModule vtable @ 0xDACF40：Init=0x160EC0、Shut=0xB8EAA0、BeforeLaunch=默认、LoadResource=0x161140。

---

## 2. IKernel 内核外观（507 虚槽；宿主侧 CKernel 实现）

实现体：FsSoloMember.dll `CKernel`（`Kernel.obj`，`??0CKernel@@QAE@PAVSoloMember@@`）；FsMember.dll 同构。
完整有序槽表已导出 `tools/fsgame/ckernel_slots.txt`（解增量链接 thunk 后按 vtable 顺序）。按前缀分组：

| 组 | 槽数 | 代表方法 | 复刻要点 |
|---|---|---|---|
| 注册面 Add/Declare/Remove/Lookup/Clear | ~45 | AddLogicClass / AddClassCallback / AddEventCallback×2 / AddCommandHook / AddIntCommandHook / AddCustomHook / AddIntCustomHook / DeclareLuaExt / DeclareHeartBeat / DeclareCritical / DeclareRecHook + Remove/Clear/Lookup 族 | 模块系统核心；六类回调注册（§3） |
| 场景/副本 | ~30 | GetScene* / RequestCloneScene / SetCloneSceneDownTime / GetCloneScene* / GetPrototypeSceneId / IsPrototypeScene / GetScenePlayerList | 场景管理 + 副本克隆 |
| 地图/碰撞 | ~25 | GetMapBound / GetMapHeight / GetMapRegion / CanWalk / LineCanWalk / TraceLineWalk / GetCollideEnable / LoadCollideData | 对接 FsCollide（已复刻 internal/fscollide） |
| 对象属性 Var | ~60 | Query/Set{Int,Int64,Float,Double,String,WideStr,Object,Binary} + IncInt/IncFloat + TestAttrFlag / SetAttrHide / SetVisible / SetRealtime / SetSaving | 属性表（CustomVar）读写面 |
| Record 记录 | ~75 | AddRecord / AddRecordRow / RemoveRecordRow / ClearRecord / Set/Query/FindRecord*（按列值查行）/ SetRecordFlag / GetRecordCols/Rows/Max/ColType / FindRecHook / AddRecHook | 行列存储 + 变更钩子（持久化主载体） |
| 对象树/容器 | ~30 | CreateObject / CreateObjectArgs / CreateFromConfig / AddVisible / AddWeakChild / GetChild* / FindChild* / CreateContainer | 对象创建/父子/容器 |
| 移动/寻路 | ~25 | SwitchLocate / SwitchBorn / GetSwitchLocation / CheckMotion / CheckJump/Fly/Swim/Drift/Climb/Slide/Sink/Stop / Distance2D/3D / Angle / DotAngle | 位置与移动合法性 |
| 存档 | 8 | SaveObject / ResetObject / DeleteObject / SavePlayerData / SaveChunk / LoadChunk / DeleteChunk / FindChunk | ↔ store 服 save_data（已复刻链路） |
| 配置 | 12 | PreloadConfig / PreloadConfigTxt / LoadConfig / Find/Get/SetConfigProperty / GetConfigRecordList | ini/*.xml 静态表 |
| Lua | 30 | LuaLoadScript / LuaFindScript / LuaFindFunc / LuaRunFunc / LuaErrorHandler / LuaGetArgCount / LuaIs* / LuaTo* / LuaPush* | 宿主 Lua VM 桥（VM 承载沿 FxCore 里程碑二决策） |
| 消息外发 | 10 | SendPublicMessage / SendChargeMessage / SendChargeSafeMessage / SendManageMessage / SendExtraMessage / SendToGmcc / SendGmccCustom / SendAuctionMessage / SendEvent / RunCommand | ↔ public/charge/manage/gmcc/auction 各服（均已复刻） |
| 商城/计费 | 20 | QueryGift / GetGift / SellCard / UnsellCard / BuyCard / BuyItem / BuyItemGive / BuyItem2 / RequestPoints / RequestLimitTime / RequestAccountInfo / RequestPlayTime / RequestAllItemInfo* / RequestItemInfo / FindItemNo / GetItemNoList / GetItemInfo / GetItemPrice | 充值中心对接（FsCharge 已全解） |
| 信件/日志 | 12 | SendLetter / SystemLetter / SystemLetterByAccount / CleanLetterBySerial / BackLetterBySerial / SaveLog / ItemLog / ChatLog / GmLog / CustomLog / CustomLogWithRole | 邮件 + 日志落库（FsSoloLog 已复刻） |
| 身份/杂项 | ~20 | GetResourcePath / GetDistrictId / GetServerId / GetMemberId / SeekRoleUid / SeekRoleName / GetRoleDeleted / CheckName / Trace / Echo / GetLogicModule / GetGameObj / GetSceneObj | 环境与查询 |

（其余几何/Request*/Object* 辅助族见槽表原文；精确签名以 ckernel_slots.txt 为准。）

## 3. 六类回调注册面（模块 → 内核）

以 PlayerBaseModule::Init（0x160EC0）为范式，全部注册在 Init 期完成：

| 注册 API | 槽 | 实参 | PlayerBaseModule 实例 |
|---|---|---|---|
| `AddClassCallback(class, event, fn)` | +0x10 | ("player","OnCreateClass",fn) | 类创建回调 |
| `AddEventCallback(class, event, fn)` | +0x18 | ("player", 事件名, fn) | OnRecover/OnEntry/OnReady/OnStore/OnEntryGame/OnDisconnect |
| `AddIntCustomHook(class, cmd, fn)` | +0x28 | ("player", 0xC/0x33E/0x35C/0x3F7, fn) | **客户端整型命令入口** |
| `DeclareCritical(class, var, fn)` | +0x38 | ("player", 属性名, C_OnPlayer*Change ×4) | 属性变更回调 |
| `DeclareHeartBeat(name, fn)` | +0x34 | ("XxxModule::HB_*", fn) | 定时心跳（全库 230 个） |
| `DeclareRecHook(name, fn)` | +0x3C | 记录变更钩子（全库 32 个） | Record 行级变更 |

- `player` 类七事件：`OnCreateClass → OnRecover → OnEntry → OnReady → OnEntryGame → OnStore → OnDisconnect`（装载→入场→就绪→进游戏→存档→掉线）。
- `scene` 类（SceneBaseModule）：`OnCreate/OnDestroy/OnEntry` + 心跳 `HB_SyncCurrentTime`；`role` 类：`OnCreateRole`（CreateRoleModule，命令 0x375）。
- 事件可由模块经 `RunEventCallback/RunEvent/SendEvent` 主动触发（模块间解耦通信）。

## 4. IGameObj 游戏对象模型（宿主 CGameObject，120 方法）

属性容器型对象（与 nx_base save_data OBJ3 容器同构，见 `docs/official/nx_base-save_data容器格式.md`）：

| 面 | 方法组 |
|---|---|
| 属性表 | Query/Set{Int,Int64,Float,Double,String,WideStr,Object,Binary}(±ByIndex) ×2、GetAttr{Count,Index,List,Type}、FindAttr |
| Data 键值 | Add/Find/Remove/Get/Set/QueryData*（对象自定义数据） |
| Record | Get/Create/FindRecord*、SendRec{Append,Clear,Delete,Modify,SetRow}（记录变更下发客户端） |
| 对象树 | GetParent / GetChild* / FindChild* / GetFirstChild / GetNextChild / AddWeakChild / ClearChild / GetGroupId |
| 空间 | GetPosi{X,Y,Z} / GetOrient / GetLocation / SetLocation / MoveTo / GetMoveMode |
| 可见性 | AddVisInfo / SendVisInfo / NotifyViewer{Add,Change,Remove} / FindViewport / GetWeakBoxList / GetWeakRefs |
| 存档 | LoadArchive / StoreArchive / SwitchLoad / SwitchStore（↔ SaveChunk/save_data） |
| 下发 | SendOwnInfo（对象全量下发客户端） |

PERSISTID = 对象弱引用句柄（u64：id+serial），贯穿全部回调签名；IVarList/TVarList<8> 为回调参数列。

## 5. 持久化与外部消息链（全部对接已复刻组件）

```
模块写 Record/Var → CGameObject::StoreArchive / IKernel::SaveChunk(role,"chunk 名")
  → FsSoloMember store 通道（manage/save 族；internal/fssolostore 已复刻 67 型命令表）
    → FsSql → MySQL（save_data OBJ3 容器；玩家 schema 已解：save-data-OBJ3-player-schema.txt）
OnStore 事件（PlayerBaseModule）= 存档钩子；IKernel::SavePlayerData = 强制全量存。
外发：SendPublicMessage→FsSoloPublic / SendAuctionMessage→拍卖行 / SendChargeMessage→FsCharge /
      SendManageMessage→monitor / SendToGmcc→客服；world↔member 链路已复刻（fssoloworld/fsunite）。
```

## 6. 静态数据、Lua 与反外挂

- **静态配置**：`GetResourcePath()`（宿主 `..\Res`）+ `PreloadConfig("ini/item/ArmorItem.xml")` 等；
  Res/ini 下 40+ 域（Achievement/Ai/Arena/CreateRole/Drop/EventSpring/Fight/Guild/Item/ItemEnhance/
  Life/Map/Npc/PVPTask/Processor/Quest/Scene/Skill/Team/Title/Toplimit…），xml 解析为内联 tinyxml。
- **Lua**：宿主 FxCore 定步泵承载 VM；FsGame 侧 `DeclareLuaExt("nx_*", fn)` 注册 **473 个**服务器函数
  （各模块 nx_* 全列在 modules_index.json；LuaExtModule 一家 271 个）。脚本目录 `Lua\solounite`（合一形态）。
- **反外挂**：PlayerBaseModule 为 GPKitSvr 唯一消费者（GPKCreateSvrDynCode，hack10-79 码表，
  internal/gpkit 已复刻）；AddBuffToPlayerForGpk 桥接战斗侧。
- **源码树泄露**（__FILE__ 串，169 文件全列 src_files.txt）：`fsgame\{commonmodule, stagemodule,
  npcbasemodule, playerbasemodule, skillmodule, eventspringmodule, itembasemodule, trademodule,
  aimodule, warmodule, interactivemodule, scenebasemodule, questmodule, lifemodule, guildmodule,
  energywellmodule, logicclassmodule}` + `bg_source\tools\findpath` + `server\{public,utils}`。

## 7. Go 复刻方案（框架层）

落位 `internal/fsgame`（本轮只立架构与索引，玩法按 §8 表后补）：

```
internal/fsgame/
  doc.go            // 定位说明（本文件摘录）
  kernel.go         // Kernel 接口 = IKernel 507 槽的 Go 面（按 §2 分组裁剪方法集）
  kernel_slots.go   // 槽序常量（对齐 ckernel_slots.txt，供协议对照）
  module.go         // Module 接口 = ILogicModule 5 槽：Init/BeforeLaunch/LoadResource/Shut
  registry.go       // Creator 注册链（等价 IModuleCreator 链；Init 序 = §8 表 Init 序）
  eventbus.go       // 六类注册面：类回调/事件/整型命令/心跳/临界/记录钩子
  gameobj.go        // 对象模型：属性表+Record+子树+存档（对齐 IGameObj 120 方法分组）
  varlist.go        // IVarList/TVarList<8> 参数列
  staticdata.go     // PreloadConfig/xml 静态表（对接 Res/ini）
  luabridge.go      // nx_* 473 函数注册面（VM 沿 FxCore 里程碑二决策，先占位）
```

接驳点：kernel.SendPublicMessage→internal/fssolopublic；存档→internal/fssolostore 管理族；
碰撞→internal/fscollide；宿主生命周期→internal/fssolomember（其 [Logic] 装配位 LogicModules 已预留）。

**复刻顺序建议**（框架先行，玩法按表推进）：

1. registry/module/kernel 骨架 + eventbus 六类回调 + player 七事件空转；
2. gameobj 属性/Record + SaveChunk → store 链闭环（登录→OnEntry→OnStore→save_data 可回放）；
3. staticdata（Res/ini xml 子集：server_version/item/skill）+ 整型命令路由（先通 PlayerBase 4 命令）；
4. 心跳泵（对齐 FxCore 定步）+ Lua nx_* 占位；
5. 之后按 §8 表逐模块复刻（每模块套路：Init 注册面 → 命令/事件 → Record/心跳）。

## 8. 玩法模块索引表（241 个，位置齐全）

> 列说明：**Init序** = 宿主遍历创建链的初始化次序（1 最先＝LogicClassModule）；
> **creator构造** = `XxxModuleCreator::ctor` RVA（FsGame.dll）；**Init** = `?Init@Xxx@@UAE_NPAVIKernel@@@Z` RVA；
> 符号 = 该 obj 在 .map 的公共符号数（规模感）；命令 = AddIntCustomHook/AddIntCommandHook 注册的客户端
> 整型命令数（≥3 过滤布尔噪声）；心跳/Lua 同理。机器可读版 `tools/fsgame/modules_index.json`
> （含每模块命令号全列、心跳名全列、nx_* 全列、配置 xml 列）。
#### 框架·模块系统与配置（先复刻面，16 个）

| # | 模块 | obj | creator构造 | Init | Init序 | 符号 | 命令 | 心跳 | Lua | 命令号示例 |
|---|------|-----|------------|------|--------|------|------|------|-----|------------|
| 1 | `LogicClassModule` | LogicClassModule | 0x00c0e290 | 0x00ad21b0 | 1 | 27 | 0 | 0 | 0 |  |
| 2 | `StaticDataQueryModule` | StaticDataQueryModule | 0x00c0daf0 | 0x006cfe30 | 2 | 414 | 0 | 0 | 2 |  |
| 3 | `StaticInstanceModule` | StaticInstanceModule | 0x00c0d7a0 | 0x007cd940 | 3 | 2 | 0 | 0 | 0 |  |
| 4 | `SystemConfigModule` | SystemConf | 0x00c0d450 | 0x0020e5f0 | 4 | 30 | 0 | 0 | 0 |  |
| 47 | `DelayAction` | DelayAction | 0x00bded30 | 0x003ee7e0 | 47 | 19 | 0 | 0 | 0 |  |
| 60 | `FlowModule` | FlowModule | 0x00bd7a60 | 0x003f3c80 | 60 | 307 | 1 | 11 | 5 | 0x268 |
| 175 | `NotifyModule` | NotifyModule | 0x00b7c7b0 | 0x008957c0 | 175 | 149 | 3 | 1 | 0 | 0x97 0x3ab 0x3ac |
| 180 | `RequestModule` | RequestModule | 0x00b7b220 | 0x00877180 | 180 | 1193 | 8 | 0 | 0 | 8个(0x62…0x6d) |
| 190 | `SwitchManager` | SwitchManager | 0x00b75cf0 | 0x007f1e50 | 190 | 282 | 0 | 0 | 0 |  |
| 194 | `OpenPromptModule` | OpenPromptModule | 0x00b70270 | 0x007cb0c0 | 194 | 106 | 5 | 0 | 0 | 5个(0x6d…0x3c6) |
| 232 | `PubModule` | PubModule | 0x00b5a270 | 0x0076c080 | 232 | 192 | 7 | 0 | 10 | 7个(0x3e8…0x453) |
| 234 | `VirtualRecModule` | VirtualRecModule | 0x00b58960 | 0x007676e0 | 234 | 276 | 0 | 0 | 0 |  |
| 235 | `MsgBoxModule` | MsgBoxModule | 0x00b57670 | 0x00764490 | 235 | 146 | 2 | 0 | 0 | 0x5 0x6 |
| 236 | `ChannelModule` | ChannelModule | 0x00b57320 | 0x007640b0 | 236 | 35 | 0 | 0 | 3 |  |
| 238 | `AsynCtrlModule` | AsynCtrlModule | 0x00b55000 | 0x007604e0 | 238 | 142 | 0 | 3 | 0 |  |
| 239 | `LuaExtModule` | LuaExtModule | 0x00b54cb0 | 0x0075df00 | 239 | 1536 | 2 | 0 | 271 | 0x3 0x8 |

#### 框架·运维与后台（先复刻面，13 个）

| # | 模块 | obj | creator构造 | Init | Init序 | 符号 | 命令 | 心跳 | Lua | 命令号示例 |
|---|------|-----|------------|------|--------|------|------|------|-----|------------|
| 13 | `GmccModule` | GmccModule | 0x00bf3a10 | 0x007bf300 | 13 | 329 | 2 | 0 | 0 | 0xab 0x128 |
| 69 | `TestRobotModule` | TestRobotModule | 0x00bd3770 | 0x00466880 | 69 | 40 | 1 | 1 | 1 | 0x42 |
| 110 | `SceneObjectSeedEditor` | SceneObjectSeedEditor | 0x00babc60 | 0x00597f70 | 110 | 85 | 0 | 0 | 0 |  |
| 128 | `WordCheckModule` | wordcheckmodule | 0x00ba20a0 | 0x00277120 | 128 | 112 | 0 | 0 | 2 |  |
| 186 | `WebExBox` | WebExBox | 0x00b785d0 | 0x007f66b0 | 186 | 23 | 0 | 0 | 0 |  |
| 187 | `WebExchange` | WebExchange | 0x00b78270 | 0x007f78a0 | 187 | 439 | 0 | 0 | 0 |  |
| 188 | `WebExLock` | WebExLock | 0x00b77f20 | 0x007ffea0 | 188 | 445 | 0 | 0 | 0 |  |
| 189 | `WebExMsg` | WebExMsg | 0x00b77bd0 | 0x008057e0 | 189 | 47 | 0 | 0 | 0 |  |
| 191 | `DumpRoleModule` | DumpRoleModule | 0x00b75540 | 0x007ee030 | 191 | 176 | 1 | 0 | 0 | 0xc3 |
| 200 | `BackStageModule` | BackStageModule | 0x00b6d370 | 0x00790be0 | 200 | 1074 | 1 | 0 | 3 | 0x35b |
| 211 | `ManageModule` | ManageModule | 0x00b686c0 | 0x0066c9b0 | 211 | 9 | 0 | 1 | 0 |  |
| 240 | `LogModule` | LogModule | 0x00b547f0 | 0x0072f090 | 240 | 447 | 0 | 0 | 2 |  |
| 241 | `GMModule` | GMModule | 0x00b54330 | 0x0072dbf0 | 241 | 125 | 0 | 0 | 0 |  |

#### 玩家基础（先复刻面，19 个）

| # | 模块 | obj | creator构造 | Init | Init序 | 符号 | 命令 | 心跳 | Lua | 命令号示例 |
|---|------|-----|------------|------|--------|------|------|------|-----|------------|
| 52 | `EnergyModule` | EnergyModule | 0x00bdaae0 | 0x0040cd10 | 52 | 579 | 0 | 0 | 0 |  |
| 122 | `RevertPlayerModule` | RevertPlayerModule | 0x00ba5970 | 0x002439a0 | 122 | 562 | 2 | 0 | 1 | 0x2c8 0x2c9 |
| 129 | `SafeModeModule` | SafeModeModule | 0x00ba1d50 | 0x00279140 | 129 | 110 | 1 | 1 | 0 | 0x105 |
| 132 | `RoleExteriorModule` | RoleExteriorModule | 0x00b9f8f0 | 0x001d4b60 | 132 | 790 | 0 | 0 | 0 |  |
| 133 | `CreateRoleModule` | CreateRoleModule | 0x00b98bf0 | 0x001b4430 | 133 | 1505 | 1 | 0 | 0 | 0x375 |
| 134 | `MotionModule` | MotionModule | 0x00b988a0 | 0x0015cad0 | 134 | 116 | 4 | 1 | 0 | 0x3 0xf 0x37a 0x37b |
| 135 | `PlayerChangeModule` | PlayerChangeModule | 0x00b98550 | 0x00238660 | 135 | 137 | 0 | 2 | 1 |  |
| 137 | `RestoreModule` | RestoreModule | 0x00b96460 | 0x0019ecb0 | 137 | 91 | 0 | 11 | 0 |  |
| 141 | `ReliveModule` | ReliveModule | 0x00b91930 | 0x00173500 | 141 | 1382 | 4 | 5 | 2 | 0x16 0x122 0x13b 0x268 |
| 143 | `DeadModule` | DeadModule | 0x00b8faa0 | 0x001727c0 | 143 | 12 | 2 | 0 | 0 | 0x268 0x3e8 |
| 144 | `FreshmanModule` | FreshmanModule | 0x00b8f750 | 0x001f4070 | 144 | 284 | 3 | 4 | 0 | 0x91 0x268 0x3e8 |
| 148 | `PlayerBaseModule` | PlayerBaseModule | 0x00b8e7f0 | 0x00160ec0 | 148 | 152 | 4 | 0 | 0 | 0xc 0x33e 0x35c 0x3f7 |
| 192 | `ObjPropRefreshModule` | ObjPropRefreshModule | 0x00b73cb0 | 0x007d8330 | 192 | 724 | 0 | 0 | 0 |  |
| 195 | `OffLineCompensateModule` | OffLineCompensateModule | 0x00b6e600 | 0x007c6d10 | 195 | 184 | 1 | 0 | 0 | 0x374 |
| 206 | `SetPathModule` | SetPathModule | 0x00b6a710 | 0x006cac00 | 206 | 280 | 0 | 0 | 0 |  |
| 212 | `PathFindingModule` | PathFindingModule | 0x00b68370 | 0x0066ce40 | 212 | 444 | 0 | 2 | 0 |  |
| 215 | `DoSelectModule` | DoSelectModule | 0x00b66d60 | 0x0078fbc0 | 215 | 13 | 2 | 0 | 0 | 0x1b8 0x265 |
| 228 | `LevelModule` | LevelModule | 0x00b5ced0 | 0x00221e80 | 228 | 231 | 1 | 0 | 1 | 0x1d |
| 229 | `LocationModule` | LocationModule | 0x00b5b270 | 0x00779a40 | 229 | 140 | 0 | 0 | 0 |  |

#### 场景基础（先复刻面，10 个）

| # | 模块 | obj | creator构造 | Init | Init序 | 符号 | 命令 | 心跳 | Lua | 命令号示例 |
|---|------|-----|------------|------|--------|------|------|------|-----|------------|
| 12 | `WarFog` | WarFog | 0x00bf3d60 | 0x005f7810 | 12 | 1474 | 0 | 0 | 0 |  |
| 78 | `CreateMonsterModule` | CreateMonsterModule | 0x00bc3960 | 0x009c1090 | 78 | 183 | 1 | 0 | 0 | 0x3a7 |
| 111 | `MapModule` | MapModule | 0x00bab830 | 0x0027bc00 | 111 | 34 | 1 | 0 | 2 | 0x2df |
| 112 | `AreaObjModule` | AreaObjModule | 0x00baa920 | 0x002d71a0 | 112 | 794 | 0 | 0 | 0 |  |
| 113 | `MapMagModule` | MapMagModule | 0x00baa0c0 | 0x002e7030 | 113 | 2280 | 5 | 1 | 0 | 5个(0x13c…0x2dd) |
| 115 | `CloneSceneModule` | CloneSceneModule | 0x00ba9a10 | 0x002a6bd0 | 115 | 1528 | 14 | 11 | 11 | 14个(0x4…0x3e8) |
| 116 | `SceneBaseModule` | SceneBaseModule | 0x00ba9690 | 0x0027c3c0 | 116 | 202 | 0 | 1 | 0 |  |
| 169 | `CommonNpcModule` | CommonNpcModule | 0x00b81b60 | 0x008b56c0 | 169 | 23 | 1 | 0 | 0 | 0x16 |
| 171 | `VisibleNpcModule` | VisibleNpcModule | 0x00b80160 | 0x008b3ad0 | 171 | 139 | 2 | 0 | 1 | 0x266 0x3de |
| 225 | `NpcResourceModule` | NpcResourceModule | 0x00b5e470 | 0x008a3f60 | 225 | 106 | 0 | 0 | 0 |  |

#### 物品底座（先复刻面，13 个）

| # | 模块 | obj | creator构造 | Init | Init序 | 符号 | 命令 | 心跳 | Lua | 命令号示例 |
|---|------|-----|------------|------|--------|------|------|------|-----|------------|
| 64 | `CoolDownModule` | CoolDownModule | 0x00bd62c0 | 0x0036fbf0 | 64 | 72 | 0 | 0 | 4 |  |
| 70 | `ChargeModule` | ChargeModule | 0x00bd3420 | 0x00431690 | 70 | 715 | 1 | 11 | 1 | 0xc4 |
| 86 | `ItemNotify` | ItemNotify | 0x00bb5060 | 0x0010a8b0 | 86 | 22 | 0 | 0 | 0 |  |
| 95 | `ItemProCompute` | ItemProCompute | 0x00bb0ad0 | 0x000a83c0 | 95 | 479 | 0 | 0 | 0 |  |
| 96 | `ItemLimited` | ItemLimited | 0x00bb0780 | 0x0009bdc0 | 96 | 565 | 0 | 0 | 0 |  |
| 97 | `ItemBindModule` | ItemBindModule | 0x00bb0430 | 0x0009a6b0 | 97 | 27 | 0 | 0 | 0 |  |
| 100 | `ItemValidTimeModule` | ItemValidTimeModule | 0x00badf70 | 0x0008d710 | 100 | 48 | 0 | 1 | 1 |  |
| 101 | `ItemBaseModule` | ItemBaseModule | 0x00badc20 | 0x0007e810 | 101 | 158 | 0 | 0 | 0 |  |
| 197 | `ItemControlModule` | ItemControlModule | 0x00b6de60 | 0x007c6af0 | 197 | 9 | 0 | 0 | 0 |  |
| 216 | `CapitalModule` | CapitalModule | 0x00b64900 | 0x006553d0 | 216 | 992 | 3 | 1 | 7 | 0x2d 0x143 0x37c |
| 230 | `ModifyPackModule` | ModifyPackModule | 0x00b5aa50 | 0x0076f2a0 | 230 | 470 | 0 | 0 | 0 |  |
| 231 | `ContainerBaseModule` | ContainerBaseModule | 0x00b5a5f0 | 0x0071c760 | 231 | 584 | 11 | 0 | 0 | 11个(0x1e…0x29) |
| 237 | `AddContainerModule` | AddContainerModule | 0x00b56fa0 | 0x00716720 | 237 | 75 | 0 | 0 | 0 |  |

#### AI·世界模拟底座（先复刻面，9 个）

| # | 模块 | obj | creator构造 | Init | Init序 | 符号 | 命令 | 心跳 | Lua | 命令号示例 |
|---|------|-----|------------|------|--------|------|------|------|-----|------------|
| 39 | `AICharacterModule` | AICharacterModule | 0x00be0860 | 0x0004d350 | 39 | 14 | 0 | 0 | 0 |  |
| 40 | `AISkillControlModule` | AISkillControlModule | 0x00be0510 | 0x0005eaa0 | 40 | 586 | 0 | 0 | 0 |  |
| 41 | `AIOnCommandMotion` | AIOnCommandMotion | 0x00be0190 | 0x00045780 | 41 | 57 | 0 | 0 | 0 |  |
| 42 | `WarnModule` | WarnModule | 0x00bdfd80 | 0x00076b70 | 42 | 112 | 0 | 3 | 1 |  |
| 43 | `HateModule` | HateModule | 0x00bdfa00 | 0x0006e7b0 | 43 | 347 | 2 | 0 | 0 | 0x26a 0x26b |
| 44 | `AIChatWithOther` | AIChatWithOther | 0x00bdf6b0 | 0x0005c110 | 44 | 167 | 0 | 0 | 0 |  |
| 45 | `AIMotionModule` | AIMotionModule | 0x00bdf330 | 0x00042e90 | 45 | 47 | 0 | 1 | 0 |  |
| 46 | `AIModule` | AIModule | 0x00bdefe0 | 0x0000dcd0 | 46 | 223 | 0 | 0 | 0 |  |
| 48 | `TempStopAI` | TempStopAI | 0x00bdea80 | 0x003f20c0 | 48 | 19 | 0 | 0 | 0 |  |

#### 事件系统底座（先复刻面，25 个）

| # | 模块 | obj | creator构造 | Init | Init序 | 符号 | 命令 | 心跳 | Lua | 命令号示例 |
|---|------|-----|------------|------|--------|------|------|------|-----|------------|
| 11 | `TriggerDataModule` | TriggerDataModule | 0x00bfaff0 | 0x00555970 | 11 | 117 | 1 | 0 | 0 | 0x10e |
| 14 | `ProcessorCloneSceneBehavior` | ProcessorCloneSceneBehavior | 0x00bf3690 | 0x005a5760 | 14 | 108 | 0 | 0 | 0 |  |
| 15 | `ProcessorCommonBehavior` | ProcessorCommonBehavior | 0x00bf3310 | 0x005ab800 | 15 | 376 | 0 | 0 | 0 |  |
| 16 | `ProcessorAIBehavior` | ProcessorAIBehavior | 0x00bf2f90 | 0x0059b080 | 16 | 148 | 0 | 0 | 0 |  |
| 17 | `ProcessorRecordBehavior` | ProcessorRecordBehavior | 0x00bf2c40 | 0x005efef0 | 17 | 96 | 0 | 0 | 0 |  |
| 18 | `ProcessorPropertyBehavior` | ProcessorPropertyBehavior | 0x00bf28c0 | 0x005e5610 | 18 | 132 | 0 | 0 | 0 |  |
| 19 | `ProcessorObjectBehavior` | ProcessorObjectBehavior | 0x00bf2540 | 0x005d6a30 | 19 | 237 | 0 | 0 | 0 |  |
| 20 | `ProcessorConditionBehavior` | ProcessorConditionBehavior | 0x00bf21c0 | 0x005cae10 | 20 | 105 | 0 | 0 | 0 |  |
| 21 | `ProcessorComparisionBehavior` | ProcessorComparisionBehavior | 0x00bf1e40 | 0x005c7570 | 21 | 33 | 0 | 0 | 0 |  |
| 22 | `TriggerPlayerEvent` | TriggerPlayerEvent | 0x00bf1ac0 | 0x00581b20 | 22 | 173 | 0 | 0 | 0 |  |
| 23 | `TriggerSceneEvent` | TriggerSceneEvent | 0x00bf1740 | 0x005853d0 | 23 | 525 | 0 | 0 | 0 |  |
| 24 | `TriggerNpcEvent` | TriggerNpcEvent | 0x00bf13f0 | 0x0057e810 | 24 | 169 | 0 | 0 | 0 |  |
| 25 | `TriggerAreaEvent` | TriggerAreaEvent | 0x00bf10a0 | 0x0056ef40 | 25 | 272 | 0 | 0 | 0 |  |
| 26 | `TriggerBufferEvent` | TriggerBufferEvent | 0x00bf0d20 | 0x00577760 | 26 | 196 | 0 | 0 | 0 |  |
| 27 | `TriggerItemEvent` | TriggerItemEvent | 0x00befb10 | 0x0057b680 | 27 | 211 | 0 | 0 | 0 |  |
| 29 | `TriggerBaseModule` | TriggerBaseModule | 0x00bee560 | 0x00bee810 | 29 | 112 | 0 | 0 | 0 |  |
| 102 | `TriggerProcesserModule` | TriggerProcesserModule | 0x00bad840 | 0x00a70260 | 102 | 458 | 2 | 1 | 0 | 0x2f1 0x2f2 |
| 103 | `NoticeExecutor` | NoticeExecutor | 0x00bad4c0 | 0x00a8e6c0 | 103 | 65 | 0 | 0 | 0 |  |
| 104 | `TaskExecutor` | TaskExecutor | 0x00bad140 | 0x00a9aa10 | 104 | 118 | 2 | 1 | 0 | 0x39f 0x3a8 |
| 105 | `SwitchExecutor` | SwitchExecutor | 0x00bacdc0 | 0x00a978b0 | 105 | 48 | 1 | 0 | 0 | 0x2f6 |
| 106 | `RewardExecutor` | RewardExecutor | 0x00baca40 | 0x00a918f0 | 106 | 197 | 0 | 0 | 0 |  |
| 107 | `BehaviorExecutor` | BehaviorExecutor | 0x00bac6c0 | 0x00a7dad0 | 107 | 393 | 0 | 1 | 0 |  |
| 108 | `ConditionExecutor` | ConditionExecutor | 0x00bac340 | 0x00a79530 | 108 | 60 | 4 | 0 | 0 | 0x268 0x2f0 0x3a2 0x3a4 |
| 109 | `EventSpringModule` | EventSpringModule | 0x00babfb0 | 0x00a5ad20 | 109 | 817 | 9 | 6 | 15 | 9个(0x268…0x2fb) |
| 205 | `ConditionModule` | ConditionModule | 0x00b6b600 | 0x006f8f20 | 205 | 231 | 0 | 0 | 0 |  |

#### 战斗（玩法域，17 个）

| # | 模块 | obj | creator构造 | Init | Init序 | 符号 | 命令 | 心跳 | Lua | 命令号示例 |
|---|------|-----|------------|------|--------|------|------|------|-----|------------|
| 10 | `QteModule` | QteModule | 0x00bfb860 | 0x00705410 | 10 | 422 | 4 | 7 | 0 | 0x268 0x35c 0x3d7 0x3df |
| 49 | `Floating` | Floating | 0x00bde7d0 | 0x003eed20 | 49 | 22 | 0 | 0 | 0 |  |
| 50 | `MultiSpurt` | MultiSpurt | 0x00bde520 | 0x003efe30 | 50 | 92 | 0 | 0 | 0 |  |
| 53 | `SkillBreakManageModule` | SkillBreakManageModule | 0x00bda330 | 0x003d3df0 | 53 | 321 | 2 | 0 | 0 | 0x265 0x269 |
| 55 | `SkillEventManager` | SkillEventManager | 0x00bd8e00 | 0x003ce5e0 | 55 | 195 | 0 | 0 | 0 |  |
| 56 | `CSkillDataQuery` | ? | 0x00bd8b50 | 0x003db3c0 | 56 | 0 | 0 | 0 | 0 |  |
| 57 | `DamageCalculateModule` | DamageCalculateModule | 0x00bd88a0 | 0x003d3010 | 57 | 59 | 0 | 0 | 1 |  |
| 58 | `SkillUseLimitModule` | SkillUseLimitModule | 0x00bd85f0 | 0x003a37a0 | 58 | 295 | 0 | 0 | 1 |  |
| 59 | `CSideStepModule` | ? | 0x00bd82a0 | 0x003ee1b0 | 59 | 0 | 0 | 0 | 0 |  |
| 61 | `FightRelationModule` | FightRelationModule | 0x00bd7710 | 0x00338ba0 | 61 | 91 | 11 | 3 | 0 | 11个(0x124…0x2e5) |
| 62 | `PKModule` | PKModule | 0x00bd6960 | 0x0033d7c0 | 62 | 479 | 7 | 9 | 4 | 7个(0x123…0x399) |
| 63 | `FightModule` | FightModule | 0x00bd6610 | 0x0032bbc0 | 63 | 398 | 3 | 0 | 12 | 0x261 0x265 0x268 |
| 65 | `BufferModule` | BufferModule | 0x00bd4ac0 | 0x003491a0 | 65 | 1500 | 16 | 1 | 6 | 16个(0x5…0x3fd) |
| 67 | `SkillUpgradeModule` | SkillUpgradeModule | 0x00bd3fb0 | 0x0038f160 | 67 | 287 | 1 | 0 | 2 | 0xaa |
| 68 | `SkillModule` | SkillModule | 0x00bd3b20 | 0x00382620 | 68 | 582 | 6 | 34 | 0 | 6个(0xd3…0x366) |
| 72 | `FightStats` | FightStats | 0x00bcb790 | 0x003e8c10 | 72 | 90 | 0 | 0 | 0 |  |
| 149 | `SkillEffectNpcModule` | SkillEffectNpc | 0x00b8e4a0 | 0x00902060 | 149 | 29 | 0 | 0 | 0 |  |

#### 养成·装备与物品玩法（玩法域，24 个）

| # | 模块 | obj | creator构造 | Init | Init序 | 符号 | 命令 | 心跳 | Lua | 命令号示例 |
|---|------|-----|------------|------|--------|------|------|------|-----|------------|
| 51 | `CInscriptionModule` | ? | 0x00bde1d0 | 0x003faa80 | 51 | 0 | 0 | 0 | 0 |  |
| 54 | `UpgradeRuleModule` | UpgradeRuleModule | 0x00bd91b0 | 0x00316eb0 | 54 | 901 | 0 | 0 | 0 |  |
| 83 | `FashionModule` | FashionModule | 0x00bbe170 | 0x0010acd0 | 83 | 356 | 1 | 0 | 0 | 0x45 |
| 84 | `ItemLaveModule` | ItemLaveModule | 0x00bbc550 | 0x00080940 | 84 | 473 | 2 | 0 | 0 | 0x16 0x2e2 |
| 85 | `EquipmentModule` | EquipmentModule | 0x00bb53b0 | 0x000dfa50 | 85 | 1701 | 4 | 2 | 0 | 0x16 0x3b 0x3d 0xe9 |
| 87 | `ToolItemFunc` | ToolItemFunc | 0x00bb4d10 | 0x000dd900 | 87 | 85 | 0 | 0 | 0 |  |
| 88 | `ItemTreasure` | ItemTreasure | 0x00bb49c0 | 0x00111a30 | 88 | 361 | 0 | 0 | 0 |  |
| 89 | `ItemAppendModule` | ItemAppendModule | 0x00bb4670 | 0x000dcc70 | 89 | 29 | 1 | 1 | 0 | 0x3c2 |
| 90 | `InheritEnhance` | InheritEnhance | 0x00bb4240 | 0x000d0500 | 90 | 189 | 0 | 0 | 0 |  |
| 91 | `Strengthen` | Strengthen | 0x00bb3e10 | 0x000d4540 | 91 | 405 | 0 | 0 | 0 |  |
| 92 | `SlotItem` | SlotItem | 0x00bb2ee0 | 0x000c3220 | 92 | 651 | 0 | 0 | 0 |  |
| 93 | `EnchaseCompound` | EnchaseCompound | 0x00bb1250 | 0x000b8550 | 93 | 559 | 0 | 0 | 0 |  |
| 94 | `CompoundItem` | CompoundItem | 0x00bb0e20 | 0x000b3bf0 | 94 | 222 | 0 | 0 | 0 |  |
| 98 | `DropCapitalItemModule` | DropCapitalItemModule | 0x00baea60 | 0x0007a310 | 98 | 172 | 1 | 0 | 0 | 0x353 |
| 99 | `ToolItemModule` | ToolItemModule | 0x00bae2c0 | 0x0008e7a0 | 99 | 213 | 5 | 0 | 0 | 5个(0xa…0x3f4) |
| 126 | `ArmorChangeModule` | ArmorChangeModule | 0x00ba2740 | 0x002308f0 | 126 | 311 | 0 | 0 | 0 |  |
| 136 | `TransformJobModule` | TransformJobModule | 0x00b967b0 | 0x001a24e0 | 136 | 854 | 2 | 0 | 1 | 0x16 0x3ff |
| 145 | `DepotModule` | DepotModule | 0x00b8f2f0 | 0x0016f490 | 145 | 225 | 6 | 0 | 0 | 6个(0x16…0x393) |
| 146 | `ShortCutModule` | ShortCutModule | 0x00b8efa0 | 0x00166480 | 146 | 371 | 6 | 0 | 0 | 6个(0xa3…0xd1) |
| 147 | `ToolBoxModule` | ToolBoxModule | 0x00b8eb70 | 0x001633f0 | 147 | 183 | 6 | 0 | 0 | 6个(0x1d…0x395) |
| 196 | `FightBag` | FightBag | 0x00b6e1b0 | 0x006ffb00 | 196 | 248 | 0 | 0 | 0 |  |
| 198 | `Enchase` | Enchase | 0x00b6da10 | 0x00702ff0 | 198 | 96 | 0 | 0 | 0 |  |
| 208 | `PickUpModule` | PickUpModule | 0x00b69c20 | 0x006bc4c0 | 208 | 248 | 5 | 3 | 0 | 5个(0x17c…0x184) |
| 214 | `DropModule` | DropModule | 0x00b670b0 | 0x00632860 | 214 | 1511 | 1 | 0 | 7 | 0x44 |

#### 养成·职业与生活（玩法域，4 个）

| # | 模块 | obj | creator构造 | Init | Init序 | 符号 | 命令 | 心跳 | Lua | 命令号示例 |
|---|------|-----|------------|------|--------|------|------|------|-----|------------|
| 30 | `FormulaResearch` | FormulaResearch | 0x00becb90 | 0x00118460 | 30 | 445 | 0 | 0 | 0 |  |
| 31 | `LifeJobModule` | ? | 0x00be9850 | 0x0014c5b0 | 31 | 0 | 0 | 0 | 0 |  |
| 32 | `JobPropModule` | JobPropModule | 0x00be7ad0 | 0x00147a10 | 32 | 193 | 0 | 3 | 0 |  |
| 33 | `JobFormulaModule` | JobFormulaModule | 0x00be37b0 | 0x001282e0 | 33 | 1339 | 4 | 0 | 0 | 0x265 0x2de 0x2e0 0x2e3 |

#### 养成·宠物与骑乘（玩法域，9 个）

| # | 模块 | obj | creator构造 | Init | Init序 | 符号 | 命令 | 心跳 | Lua | 命令号示例 |
|---|------|-----|------------|------|--------|------|------|------|-----|------------|
| 36 | `MultiRideModule` | MultiRideModule | 0x00be1250 | 0x0020eb80 | 36 | 132 | 3 | 1 | 0 | 0x2f2 0x2f3 0x2f7 |
| 37 | `NormalRideModule` | NormalRideModule | 0x00be0f00 | 0x002123e0 | 37 | 6 | 0 | 0 | 0 |  |
| 38 | `RideModule` | RideModule | 0x00be0bb0 | 0x00212890 | 38 | 147 | 3 | 0 | 0 | 0xf3 0xf4 0x259 |
| 152 | `CPetMacStrengthen` | ? | 0x00b8d560 | 0x009379f0 | 152 | 0 | 0 | 0 | 0 |  |
| 153 | `PetOnItemBehavior` | PetOnItemBehavior | 0x00b8d210 | 0x00949e90 | 153 | 52 | 0 | 0 | 0 |  |
| 154 | `PetOnSkillBehavior` | PetOnSkillBehavior | 0x00b8cec0 | 0x0094b7a0 | 154 | 53 | 0 | 0 | 0 |  |
| 155 | `PetOrcSoulSpecial` | PetOrcSoulSpecial | 0x00b8cb70 | 0x0094d670 | 155 | 214 | 0 | 0 | 0 |  |
| 156 | `PetMagician` | PetMagician | 0x00b8c820 | 0x0093ae90 | 156 | 153 | 0 | 0 | 0 |  |
| 157 | `PetLogicModule` | PetLogicModule | 0x00b8c4d0 | 0x00922180 | 157 | 270 | 8 | 2 | 2 | 8个(0x261…0x3a9) |

#### 社交（玩法域，14 个）

| # | 模块 | obj | creator构造 | Init | Init序 | 符号 | 命令 | 心跳 | Lua | 命令号示例 |
|---|------|-----|------------|------|--------|------|------|------|-----|------------|
| 34 | `GuildStorageModule` | GuildStorageModule | 0x00be3370 | 0x00ad0e20 | 34 | 49 | 0 | 0 | 0 |  |
| 35 | `GuildModule` | GuildModule | 0x00be15a0 | 0x00abf480 | 35 | 379 | 3 | 2 | 3 | 0x14 0x16 0x3f6 |
| 176 | `ChatRoomModule` | ChatRoomModule | 0x00b7c460 | 0x00820030 | 176 | 180 | 17 | 0 | 0 | 17个(0x2ac…0x31d) |
| 177 | `LookModule` | LookModule | 0x00b7c110 | 0x0085ca00 | 177 | 115 | 9 | 1 | 2 | 9个(0x4…0x3ed) |
| 178 | `PostModule` | PostModule | 0x00b7bdc0 | 0x0080f7f0 | 178 | 579 | 3 | 8 | 0 | 0x16 0x136 0x15e |
| 181 | `ExchangeModule` | ExchangeModule | 0x00b7aed0 | 0x0080a410 | 181 | 170 | 8 | 2 | 0 | 8个(0xd2…0x145) |
| 182 | `TeamModule` | TeamModule | 0x00b7a580 | 0x0082a290 | 182 | 1506 | 53 | 12 | 6 | 53个(0x3…0x1c2) |
| 183 | `FriendModule` | FriendModule | 0x00b7a150 | 0x0085f390 | 183 | 742 | 10 | 9 | 1 | 10个(0x3…0x3e8) |
| 184 | `ChatModule` | ChatModule | 0x00b78c70 | 0x008514f0 | 184 | 436 | 2 | 1 | 0 | 0x392 0x3aa |
| 185 | `SpecialTeamModule` | SpecialTeamModule | 0x00b78920 | 0x00824b30 | 185 | 112 | 4 | 0 | 0 | 0xb 0x1cc 0x1cd 0x1ce |
| 203 | `TriggerTalkModule` | TriggerTalkModule | 0x00b6c420 | 0x006e61a0 | 203 | 101 | 1 | 0 | 0 | 0x16 |
| 204 | `OptionTalkModule` | OptionTalkModule | 0x00b6c0d0 | 0x006d80b0 | 204 | 1028 | 1 | 0 | 1 | 0x16 |
| 227 | `MenuModule` | MenuModule | 0x00b5d6a0 | 0x00789030 | 227 | 330 | 1 | 0 | 0 | 0x228 |
| 233 | `LetterModule` | LetterModule | 0x00b58d90 | 0x0077ce40 | 233 | 527 | 1 | 1 | 2 | 0x348 |

#### 任务与成就（玩法域，7 个）

| # | 模块 | obj | creator构造 | Init | Init序 | 符号 | 命令 | 心跳 | Lua | 命令号示例 |
|---|------|-----|------------|------|--------|------|------|------|-----|------------|
| 66 | `CollectModule` | CollectModule | 0x00bd4770 | 0x00117ee0 | 66 | 17 | 0 | 0 | 0 |  |
| 79 | `QuestModule` | QuestModule | 0x00bc1be0 | 0x0099c820 | 79 | 753 | 2 | 8 | 0 | 0x16 0x370 |
| 80 | `QuestCriteriaModule` | QuestCriteriaModule | 0x00bc1890 | 0x0098b8a0 | 80 | 280 | 5 | 1 | 0 | 5个(0x16…0x3a6) |
| 81 | `AchievementModule` | AchievementModule | 0x00bbfe90 | 0x009cb500 | 81 | 572 | 3 | 0 | 4 | 0x154 0x268 0x2cc |
| 82 | `AwardModule` | AwardModule | 0x00bbfb40 | 0x009b3c10 | 82 | 204 | 0 | 0 | 1 |  |
| 121 | `EverydayGuideModule` | EverydayGuideModule | 0x00ba6150 | 0x00260280 | 121 | 679 | 9 | 1 | 0 | 9个(0x10…0x3f2) |
| 124 | `MysticModule` | MysticModule | 0x00ba4dd0 | 0x0022ca30 | 124 | 48 | 4 | 3 | 0 | 0x268 0x384 0x385 0x3fd |

#### 活动与竞技（玩法域，19 个）

| # | 模块 | obj | creator构造 | Init | Init序 | 符号 | 命令 | 心跳 | Lua | 命令号示例 |
|---|------|-----|------------|------|--------|------|------|------|-----|------------|
| 28 | `StageModule` | StageModule | 0x00bee970 | 0x0054df80 | 28 | 312 | 3 | 3 | 2 | 0x268 0x379 0x7d0 |
| 74 | `ScapegoatModule` | ? | 0x00bc9700 | 0x00a3e290 | 74 | 0 | 0 | 0 | 0 |  |
| 75 | `PVPActivityModule` | PVPActivityModule | 0x00bc7d10 | 0x009d6a40 | 75 | 525 | 1 | 0 | 0 | 0x3d1 |
| 76 | `PVPTaskModule` | PVPTaskModule | 0x00bc55e0 | 0x00a02200 | 76 | 1931 | 29 | 11 | 1 | 29个(0x6…0x3f3) |
| 77 | `PVPTaskQuery` | PVPTaskQuery | 0x00bc5330 | 0x00a2f200 | 77 | 1161 | 0 | 0 | 0 |  |
| 114 | `GroupSceneModule` | GroupSceneModule | 0x00ba9d70 | 0x002cc970 | 114 | 471 | 1 | 0 | 2 | 0x326 |
| 117 | `FieldCloneModule` | FieldCloneModule | 0x00ba92b0 | 0x002cb910 | 117 | 42 | 0 | 1 | 0 |  |
| 125 | `CBladeRunnerModule` | ? | 0x00ba4470 | 0x00239b40 | 125 | 0 | 0 | 0 | 0 |  |
| 131 | `GunManModule` | GunManModule | 0x00ba16b0 | 0x002279b0 | 131 | 220 | 2 | 0 | 0 | 0x268 0x30c |
| 179 | `ActivityModule` | ActivityModule | 0x00b7b4d0 | 0x0088fee0 | 179 | 477 | 4 | 3 | 0 | 0x17a 0x334 0x335 0x39b |
| 217 | `CBattleWarGamePlayModule` | ? | 0x00b64650 | 0x0049aab0 | 217 | 0 | 0 | 0 | 0 |  |
| 218 | `BattleWarSceneModule` | BattleWarSceneModule | 0x00b64300 | 0x004c7a10 | 218 | 279 | 3 | 0 | 0 | 0x3 0x42 0x329 |
| 219 | `CBattleWarStatisticsModule` | ? | 0x00b63fb0 | 0x004cd980 | 219 | 0 | 0 | 0 | 0 |  |
| 220 | `CBattleWarModule` | ? | 0x00b63d00 | 0x004a9770 | 220 | 0 | 0 | 0 | 0 |  |
| 221 | `ArenaSceneModule` | ArenaSceneModule | 0x00b639b0 | 0x004e99a0 | 221 | 287 | 2 | 1 | 0 | 0x329 0x393 |
| 222 | `ArenaModule` | ArenaModule1 | 0x00b60580 | 0x004cef00 | 222 | 729 | 0 | 0 | 0 |  |
| 223 | `ArenaRankModule` | ArenaRankModule | 0x00b5ebb0 | 0x004e4b30 | 223 | 207 | 1 | 0 | 0 | 0x3c1 |
| 224 | `ChallengeModule` | ChallengeModule | 0x00b5e7f0 | 0x004927c0 | 224 | 257 | 1 | 7 | 2 | 0x266 |
| 226 | `CampaignStatusModule` | CampaignStatus | 0x00b5dfa0 | 0x007cda60 | 226 | 56 | 0 | 0 | 0 |  |

#### 经济（玩法域，5 个）

| # | 模块 | obj | creator构造 | Init | Init序 | 符号 | 命令 | 心跳 | Lua | 命令号示例 |
|---|------|-----|------------|------|--------|------|------|------|-----|------------|
| 71 | `AuctionHouseModule` | AuctionHouseModule | 0x00bcbae0 | 0x0041dd40 | 71 | 596 | 1 | 0 | 2 | 0xc5 |
| 73 | `ShopModule` | ShopModule | 0x00bcadb0 | 0x004425c0 | 73 | 1147 | 4 | 7 | 0 | 0x16 0x46 0x268 0x271 |
| 118 | `IncomeReduceModule` | IncomeReduceModule | 0x00ba8e60 | 0x00275a90 | 118 | 76 | 0 | 1 | 0 |  |
| 123 | `VipModule` | VipModule | 0x00ba5120 | 0x00216780 | 123 | 555 | 3 | 1 | 0 | 0x40 0x41 0x141 |
| 130 | `GiftModule` | GiftModule | 0x00ba1a00 | 0x00b531d0 | 130 | 43 | 1 | 1 | 0 | 0xbe |

#### 能量井（帮会资源战）（玩法域，6 个）

| # | 模块 | obj | creator构造 | Init | Init序 | 符号 | 命令 | 心跳 | Lua | 命令号示例 |
|---|------|-----|------------|------|--------|------|------|------|-----|------------|
| 5 | `EnergyWellGuildWarScene` | EnergyWellGuildWarScene | 0x00c0d1a0 | 0x005093f0 | 5 | 670 | 0 | 0 | 0 |  |
| 6 | `EnergyWellGuildWar` | EnergyWellModule | 0x00c0cef0 | 0x004f59d0 | 6 | 2452 | 6 | 2 | 1 | 6个(0x268…0x3fc) |
| 7 | `EWCarrierProduceModule` | EWCarrierProduceModule | 0x00c0b2e0 | 0x00b1de70 | 7 | 502 | 0 | 0 | 1 |  |
| 8 | `EnergyFillModule` | EnergyFillModule | 0x00c07fe0 | 0x00ad26f0 | 8 | 592 | 3 | 0 | 0 | 0x16 0x265 0x3e7 |
| 9 | `EnergyWellModule` | EnergyWellModule | 0x00bfe7f0 | 0x00ade600 | 9 | 2452 | 6 | 2 | 1 | 6个(0x268…0x3fc) |
| 170 | `BattleCarrierModule` | BattleCarrierModule | 0x00b804e0 | 0x009670f0 | 170 | 1071 | 14 | 6 | 6 | 14个(0xb…0x3b6) |

#### NPC 玩法载体（玩法域，17 个）

| # | 模块 | obj | creator构造 | Init | Init序 | 符号 | 命令 | 心跳 | Lua | 命令号示例 |
|---|------|-----|------------|------|--------|------|------|------|-----|------------|
| 150 | `NpcSwitchLocateModule` | NpcSwitchLocateModule | 0x00b8dc60 | 0x008a5780 | 150 | 211 | 1 | 0 | 2 | 0x2bc |
| 151 | `NpcChangeModule` | NpcChangeModule | 0x00b8d910 | 0x008a3650 | 151 | 14 | 0 | 0 | 1 |  |
| 158 | `TechnologyNpcModule` | TechnologyNpcModule | 0x00b8a620 | 0x00903e70 | 158 | 1072 | 5 | 7 | 0 | 5个(0x268…0x391) |
| 159 | `SkillNpcModule` | SkillNpcModule | 0x00b8a2d0 | 0x009026c0 | 159 | 24 | 0 | 1 | 0 |  |
| 160 | `SwitchDoorModule` | SwitchDoorModule | 0x00b89f50 | 0x00903390 | 160 | 52 | 1 | 0 | 0 | 0x12f |
| 161 | `OrganNpcModule` | OrganNpcModule | 0x00b88db0 | 0x008ee3a0 | 161 | 1335 | 3 | 5 | 0 | 0x25a 0x25b 0x379 |
| 162 | `MultiSkillNpcModule` | MultiSkillNpcModule | 0x00b88a60 | 0x008e8470 | 162 | 74 | 1 | 1 | 0 | 0x25b |
| 163 | `GivingNpcModule` | GivingNpcModule | 0x00b87900 | 0x008dad80 | 163 | 579 | 4 | 1 | 0 | 0x258 0x25a 0x25b 0x354 |
| 164 | `FlyNpcModule` | FlyNpcModule | 0x00b85f30 | 0x008ce570 | 164 | 466 | 5 | 0 | 0 | 5个(0x2bd…0x3d6) |
| 165 | `BannerNpcModule` | BannerNpcModule | 0x00b85bb0 | 0x008cbc60 | 165 | 86 | 1 | 1 | 0 | 0x25b |
| 166 | `FunctionNpcModule` | FunctionNpcModule | 0x00b85830 | 0x008d9ee0 | 166 | 56 | 0 | 0 | 0 |  |
| 167 | `SummonNpcModule` | SummonNpcModule | 0x00b85080 | 0x008c4ce0 | 167 | 331 | 0 | 0 | 0 |  |
| 168 | `ConvoyNpcModule` | ConvoyNpcModule | 0x00b81eb0 | 0x008b5e20 | 168 | 795 | 1 | 23 | 0 | 0x2bd |
| 172 | `FormationNpcModule` | FormationNpcModule | 0x00b7fe10 | 0x008a9f10 | 172 | 453 | 0 | 0 | 0 |  |
| 173 | `HiddenNpcModule` | HiddenNpcModule | 0x00b7fa90 | 0x008b34d0 | 173 | 3 | 0 | 0 | 0 |  |
| 209 | `MonsterOwnershipModule` | MonsterOwnershipModule | 0x00b698d0 | 0x0062ec10 | 209 | 57 | 5 | 1 | 0 | 5个(0x6e…0x268) |
| 210 | `MonsterAwardsModule` | MonsterAwardsModule | 0x00b68a10 | 0x0061f3f0 | 210 | 757 | 2 | 3 | 3 | 0x268 0x3e8 |

#### 世界规则与状态（玩法域，14 个）

| # | 模块 | obj | creator构造 | Init | Init序 | 符号 | 命令 | 心跳 | Lua | 命令号示例 |
|---|------|-----|------------|------|--------|------|------|------|-----|------------|
| 119 | `HonorValueModule` | HonorValueModule | 0x00ba8690 | 0x00271d70 | 119 | 171 | 1 | 2 | 0 | 0x268 |
| 120 | `AirportModule` | AirportModule | 0x00ba6cf0 | 0x0024d2d0 | 120 | 855 | 2 | 3 | 3 | 0x16 0x229 |
| 127 | `PunisherModule` | PunisherModule | 0x00ba23f0 | 0x0022e540 | 127 | 188 | 1 | 2 | 0 | 0x259 |
| 138 | `UnderWaterModule` | UnderWaterModule | 0x00b96110 | 0x0019d870 | 138 | 53 | 3 | 5 | 0 | 0x1de 0x1df 0x268 |
| 139 | `CampModule` | CampModule | 0x00b94740 | 0x0070ed80 | 139 | 329 | 0 | 0 | 0 |  |
| 140 | `ReputeModule` | ReputeModule | 0x00b92a10 | 0x0018f290 | 140 | 232 | 0 | 0 | 2 |  |
| 142 | `TitleModule` | TitleModule | 0x00b8fdf0 | 0x00194cd0 | 142 | 396 | 1 | 2 | 3 | 0x127 |
| 174 | `WorldRankModule` | WorldRankModule | 0x00b7cb00 | 0x00898140 | 174 | 441 | 3 | 0 | 1 | 0x320 0x374 0x3c3 |
| 193 | `ExileModule` | ExileModule | 0x00b706e0 | 0x007ce940 | 193 | 477 | 3 | 3 | 1 | 0x16 0x104 0x142 |
| 199 | `NewsModule` | NewsModule | 0x00b6d6c0 | 0x006ff490 | 199 | 20 | 0 | 0 | 1 |  |
| 201 | `ForceStateModule` | ForceStateModule | 0x00b6d0c0 | 0x006ea5f0 | 201 | 528 | 0 | 0 | 0 |  |
| 202 | `ForceModule` | ForceModule | 0x00b6c850 | 0x006e6a90 | 202 | 165 | 0 | 0 | 0 |  |
| 207 | `MovieModule` | MovieModule | 0x00b69f70 | 0x006c55f0 | 207 | 220 | 5 | 5 | 1 | 5个(0x326…0x3f1) |
| 213 | `UnenthrallModule` | UnenthrallModule | 0x00b68020 | 0x0066bec0 | 213 | 29 | 2 | 1 | 0 | 0x115 0x132 |

## 9. 工具与数据文件（tools/fsgame/）

| 文件 | 内容 |
|---|---|
| `maparse.py` | FsGame.map 解析（publics → RVA/符号/obj） |
| `fdis.py` | capstone 反汇编助手（RVA→指令，符号交叉解析） |
| `regscan2.py` | 全模块 kernel 槽调用扫描 → regs_calls.json（2917 条注册） |
| `mkindex.py` | 241 模块索引合成（Name/Init/obj/统计） |
| `gendoc.py` | 玩法表生成（分类域 → 本文 §8） |
| `creators.txt` | 241 创建器（ctor RVA 升序 = 构造序） |
| `ckernel_slots.txt` | IKernel 507 虚槽有序全表（FsSoloMember CKernel 解 thunk） |
| `modules_index.json` | 每模块 class/name/ctor/Init/LoadResource/obj/符号/命令全列/心跳全列/临界/Lua 全列/配置 |
| `regs_calls.json` | 调用级注册元组（slot/site/args，含回调函数 RVA） |
| `cmd_map.json` | 359 个整型命令号 → 注册模块映射 |
| `src_files.txt` | 169 个泄露源码路径（bg_source 树） |
| `gameplay_table.md` | 本文 §8 表源 |
| `objs.txt` / `src_paths.txt` | obj 符号统计 / 路径串中间产物 |

## 10. 遗留与下一步

- 玩法本体复刻按 §8 表逐模块推进；建议第一批（登录可玩最小闭环）：
  PlayerBase / SceneBase / Location / Motion / ContainerBase / ItemBase / Capital / CreateRole / Level / RequestModule。
- `g_pCreator`（ICreator*）运行期写入点未定位（全文仅 GetCreator 一处读）——装载主路径走 GetModuleCreator，不影响复刻。
- IKernel 槽序的 FsMember.dll（网络形态）未逐槽 diff（预期同构；FsSoloMember 已解全表）。
- Lua VM 承载沿用 FxCore 里程碑二决策；473 个 nx_* 随所属模块逐个 Go 化。
- regs_calls.json 中少量噪声（布尔实参 0/1/2 误计为命令号）已在统计侧过滤（≥3）；
  精确命令语义以各模块 Init/注册函数反汇编复核为准（site 字段给出扫描点 RVA）。
