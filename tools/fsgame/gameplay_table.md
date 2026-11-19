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
