# docs/ 索引

研究笔记与协议知识库总目录。状态列含义：

- **现行权威** —— 该主题的最终定版结论，以本文为准
- **现行参考** —— 仍有效的参考资料（可能含未完成余项）
- **历史记录** —— 已落地工作的实现记录，留作实现依据回溯
- **部分作废** —— 含已被后续文档推翻的结论，入链前先看文首横幅
- **冻结快照** —— 旧方言时代的快照数据，再生脚本已不在本仓库

> 2026-11-14 官方表切换终局背景：本分支已切换为 **official2433 唯一方言**
> （尾追 61 行 → 2494 行，`NINEYIN_PROPERTY_TABLE` 三路门控已删除）。
> 凡描述 "fork 表默认 / hybrid2433 / 双表门控" 为现行状态的段落均已按终局更新或加横幅。
> 清理前原始快照：`docs-backup-20261114/`（仓库根，不入库）。

## 一、official/ —— 外服方言（官方协议逆向）

| 文档 | 内容 | 状态 |
|---|---|---|
| [外服方言B线路线收束](official/外服方言B线路线收束.md) | B 线（外服家族客户端）渲染全链收束：16 轮真机、idx1468 MaxVisuals 注册闸、当前 NINEYIN_REPLAY_* 生产布局 | 现行权威 |
| [外服模式NPC渲染收束](official/外服模式NPC渲染收束.md) | A 线（fork 客户端）NPC 渲染五环因果链 + 官方表终局（尾追 61 行方案） | 现行权威（演进史含自勘误） |
| [外服登录帧对照与缺口清单](official/外服登录帧对照与缺口清单.md) | 登录帧逐帧对照总账（§一~§15）：缺口/差异/未应答台账，逐轮追加 | 现行权威（总账） |
| [fxnet2-op-dispatch-table](official/fxnet2-op-dispatch-table.md) | fxnet2.dll 引擎 op 分发器静态解码定版（0x00-0x37 主表 + 0x27-0x30 压缩族二级表） | 现行权威 |
| [fxgamelogic-s2c-custommsg-table](official/fxgamelogic-s2c-custommsg-table.txt) | fxgamelogic 722 槽 custommsg ID→handler 原始提取表 | 数据附录 |
| [fxgamelogic格挡门与CustomSend双路架构](official/fxgamelogic格挡门与CustomSend双路架构.md) | 格挡门逆向定版：CustomSend lua 白名单闸（54 不在名单）vs 原生无闸路、m292 发送器+RoleStateIndex、InParry/fightparry/ClaspParryLockTime 生态、m54 链定位 VMP 区、下一步三选 | 现行权威（格挡门逆向） |
| [外服抓包交互时间线分析](official/外服抓包交互时间线分析.md) | a-i.pcapng（8 抓包/4 会话/7071-7074 线）交互时间线主报告 | 现行参考 |
| [附录-骨干时间线](official/附录-骨干时间线.md) · [精选](official/附录-精选时间线.md) · [全量合并](official/附录-全量合并时间线.md) | 三粒度原始时间线 dump（98K/530K/670K） | 数据附录 |
| [外服0x29批量预载通道解析](official/外服0x29批量预载通道解析.md) | 0x29 登录簇玩家记录批量预载通道语法 + LZ 池种子定案 | 现行参考 |
| [外服压缩值消息族0x27-0x2a全景](official/外服压缩值消息族0x27-0x2a全景.md) | 压缩值族 op→handler 命名全景（-0x18 规则） | 现行参考 |
| [外服玩家档案0x28全量帧](official/外服玩家档案0x28全量帧.md) | 玩家档案 0x28 大帧（292~393 字段）方言 + 技能行 17 列模板 | 现行参考 |
| [外服NPCm229物化方言定版](official/外服NPCm229物化方言定版.md) | NPC 物化帧 c130 16 槽方言 + npcdata 数据源/native 移植 + reactor 队满丢帧事故修复 | 现行权威 |
| [初始账号外服教程态方言](official/初始账号外服教程态方言.md) | 新号教程态登录全解：记录双通道、Task_Completed 位图、第三 0x09 方言（2086 行） | 现行参考 |
| [op28-fxnet2-lz-decoder](official/op28-fxnet2-lz-decoder.md) | 0x28 四连崩根因：fxnet2 LZ77 字典解码器全逆向 + v3 生成器 | 现行参考 |
| [NPC任务绑定与引导流任务协议定版](official/NPC任务绑定与引导流任务协议定版.md) | NPC 对话任务行域表/双键语义/接取提交受理、任务数据链审计（两套任务 ID 体系）、引导抓包任务协议、lz_exact 落地与官方 0x29 任务记录帧布局 | 现行权威 |
| [自定义配置705推送卡顿排查](official/自定义配置705推送卡顿排查.md) | 重登/切图卡顿排查：首卡提交 548e616（二分）、线路抓包实证 107 早于 0x09（「抢跑位」不成立）、与 2433 的 ClientReady 位 705 位置差异、候选 A（Ready 簇去 705）测试判读表 | 现行参考（真机待验） |
| [技能服务端XML构建链与msg211受理链](official/技能服务端XML构建链与m211受理链.md) | 技能配置按黑金模型离线构建（`tools/skillgen`：11 表 18.3 万行 → `testdata/ini/Skill`，源表脏数据处理审计）+ `internal/skilldata` 按需索引装载 + msg211 受理链（普通/锁定直收/冲锋暂存→c61→m212 落地/超距/冷却·MP 闸门，女号抓包 41 次施放逐帧定版）+ 冷却表模型与套路规则（同套路按招式 CD、换套路才共享 7 秒）+ Lua 边界查证（挂 Lua 的 177 个技能全是特殊技能，内功零 Lua） | 现行权威（命中/伤害结算未接） |

## 二、combat/ —— 战斗与技能

| 文档 | 内容 | 状态 |
|---|---|---|
| [冲锋与打坐外服方言总集](combat/冲锋与打坐外服方言总集.md) | 冲锋+移动+打坐四文档合并定版：c61 逐技窗口授予协议、武器系 flag/anim、受理簇 9 帧、四桶模型、fork 自走差集终局、打坐回复帧、22 条开放项、11 条勘误索引 | 现行权威（冲锋终局） |
| [寒心恨雪与太极剑舞引导机制](combat/寒心恨雪与太极剑舞引导机制.md) | 引导技三形态定版（怒招预扣/逐跳索敌/演出锁死）、通道引擎、脚本化 | 现行权威 |
| [combat-rewrite-hardcoded-audit](combat/combat-rewrite-hardcoded-audit.md) | 战斗硬编码三分审计（A 已替/B 对齐保留/C 本地占位） | 历史记录（属性索引为 fork 表位，见文首横幅） |
| [record-table-official-and-qinggong-economy](combat/record-table-official-and-qinggong-economy.md) | 0x0A 记录表官方化（447 schema）+ 轻功经济对齐 | 历史记录（记录表侧索引不受 0x09 切换影响） |
| [jingmai-cd-qinggong](combat/jingmai-cd-qinggong.md) | CD 不下发根因（记录表方言）+ 经脉 + 滑翔 buff | 历史记录（部分被上行文档覆盖） |

## 三、systems/ —— 玩法系统实现

| 文档 | 内容 | 状态 |
|---|---|---|
| [装备外观与风物志挂载](systems/装备外观与风物志挂载.md) | 三模式外观 + 风物志皮肤 + 切换同步（Java→Go 移植） | 历史记录（已落地） |
| [镇派兵器与娃娃宝物随机属性](systems/镇派兵器与娃娃宝物随机属性.md) | 镇派兵器动态属性 + 娃娃/宝物随机属性管线、洗炼 688 | 现行参考（tersure 未实现清单开放） |
| [坐骑与待机站姿](systems/坐骑与待机站姿.md) | 坐骑三技能受理簇 + 马背技能栏 + 待机站姿 | 现行参考（真机二轮待验） |
| [item-use-msg31](systems/item-use-msg31.md) | msg31 物品使用链：客户端语义、盒子数据模型、掉落包 | 现行参考 |
| [dungeon-clone-system](systems/dungeon-clone-system.md) | 副本实例生命周期 + 波次引擎 + Lua 玩法钩子 | 现行参考 |
| [livepoint-conversion](systems/livepoint-conversion.md) | 历练→修为转化全链（235/GM/flag/快照） | 历史记录（属性索引为 fork 表位） |
| [neigong-bind-persist](systems/neigong-bind-persist.md) | 内功栏绑定（C2S 199 / 属性 201）持久化 | 历史记录（索引为 fork 表位） |
| [ninjiuzhi-gm-gating](systems/ninjiuzhi-gm-gating.md) | 九阴志全局阶段门控（jyz_gate.go，蓝本=Java 被删 NinJiuZhi） | 现行参考（门控接线待做） |
| [npc-optiontalk-shop](systems/npc-optiontalk-shop.md) | 点击 NPC → 菜单（0x1C/0x1D）→ 商店 View61 四阶段 | 历史记录（LastObject idx414 为 fork 位） |
| [npc-mappath-patrol](systems/npc-mappath-patrol.md) | NPC 官方巡逻路径（mappath ini） | 现行参考 |
| [fork-p2-shop-title-jingmai](systems/fork-p2-shop-title-jingmai.md) | P2 商店/称号/经脉 + 邮件/坐骑回放 | 历史记录（属性索引为 fork 表位） |
| [fork-p3-exchange-school](systems/fork-p3-exchange-school.md) | P3 兑换商店（type-3）/新生选择/入学 | 历史记录 |

## 四、guides/ —— 攻略转写（游戏侧事实）

| 文档 | 内容 | 状态 |
|---|---|---|
| [attributes-beginner-guide](guides/attributes-beginner-guide.md) | 属性篇（十一源，防御曲线/暴击闪避公式/面板七色等定论） | 现行参考 |
| [neigong-beginner-guide](guides/neigong-beginner-guide.md) | 内功篇（四大类/层级/兑换/怒气） | 现行参考 |
| [combat-mechanics-guide](guides/combat-mechanics-guide.md) | 战斗篇（第二篇到稿：招式五类/冲锋/轻功/控制/霸体/防反，含服务端对照） | 现行参考 |

## 五、client/ —— 客户端资产与知识库

| 文档 | 内容 | 状态 |
|---|---|---|
| [lua64-analysis](client/lua64-analysis.md) | 客户端 lua64 脚本价值评估（协议语义/战斗线索） | 现行参考 |
| [lua64-protocol-dict](client/lua64-protocol-dict.md) | custom 通道字典：710 C2S ID + 184 引擎直连 + 665 S2C handler（custom 数字 ID 通道与表方言无关，仍有效） | 现行参考 |
| [map-region-are0-rgn1](client/map-region-are0-rgn1.md) | ARE0/RGN1 区域多边形格式逆向（~80%） | 现行参考 |
| [mpq-resource-system](client/mpq-resource-system.md) | MPQ 三层资源仓（res-base/res-patch/local） | 现行参考 |
| [text-extract/](client/text-extract/README.md) | GM 客户端（盛大 res1）文本抽取知识库 16 文件 | 冻结快照（再生脚本已不在本仓库） |

## 六、infra/ —— 工具与基建

| 文档 | 内容 | 状态 |
|---|---|---|
| [server-luascript](infra/server-luascript.md) | 服务端内嵌 Lua 5.5 宿主（internal/luascript）设计与用法 | 现行参考 |
| [skill-script-framework](infra/skill-script-framework.md) | scripts/ Lua 契约：formula.lua、效果数据、内功钩子、物品脚本 | 现行参考 |
| [skill-desc-ir](infra/skill-desc-ir.md) | 技能描述 IR（desc_effects.lua）规范与路线 | 现行参考（路线未完） |
| [gm-console-axui-redesign-design](infra/gm-console-axui-redesign-design.md) | GM 面板 AXUI 重写：45 板块、框架行为、冒烟测试 | 现行参考（旧版下线待验收） |
| [gm-service-rpc-split-design](infra/gm-service-rpc-split-design.md) | GM 面板拆分独立进程 + internal API（19063/19064） | 历史记录（API 契约参考） |
| [map-path-waypoint-graph](infra/map-path-waypoint-graph.md) | .path 路点二进制格式逆向 + A* 追击重接线 | 现行参考（格式稳定） |

## 七、fork-era/ —— fork 方言存档

fork exe（9yin-game-native-menu.exe, fb21001f）时代逆向记录。fork 客户端仍是
A 线连接目标，方言知识保留作兼容参考；**属性索引/帧方言均为 fork 表位**
（如 LastObject=414，official2433 位 →416），现行权威见
[fxnet2-op-dispatch-table](official/fxnet2-op-dispatch-table.md)。

| 文档 | 内容 | 状态 |
|---|---|---|
| [fork-binary-message-inventory](fork-era/fork-binary-message-inventory.md) | fork exe C2S custom msgId 分发清单 + fork 专属子系统 + 恢复优先级 | 历史存档 |
| [fork-login-crypto](fork-era/fork-login-crypto.md) | fork DES/AES 登录加密逆向 + /api/register 移植 | 历史存档（加密层仍在用） |
| [fork-object-id-migration](fork-era/fork-object-id-migration.md) | fork 线上对象 ID 迁移（0x11000000\|idx） | 历史存档（"远程成员同段"方案已被 id-system §6.5.2 撤销） |
| [id-system-analysis](fork-era/id-system-analysis.md) | ID 体系全景：wire ident 链、fork vs 旧硬编码、§6.5 落地权威 | 部分现行（§6.5 为 fork 方言权威；外服线已另用官方 ID 对） |

## 八、spec/ —— fork 协议冻结快照

| 文档 | 内容 | 状态 |
|---|---|---|
| [spec/README](spec/README.md) | fork 方言协议知识库（51 opcodes/1548 属性/297 记录，JSON+MD） | 冻结快照 |
| [spec/opcodes](spec/opcodes.md) | fxnet2 消息目录（IDA sub_* 地址对照） | 冻结快照 |

## 九、草稿

| 文档 | 内容 | 状态 |
|---|---|---|
| [bug.txt](bug.txt) | 草稿本：历史变更日志 + 未结构化待办（含 2433 索引平移等待办项） | 草稿（未治理） |

---

## 已合并/移除文档（2026-11-14 清理）

以下文档已合并入《[冲锋与打坐外服方言总集](combat/冲锋与打坐外服方言总集.md)》（原文可从
git 历史或 `docs-backup-20261114/` 检索）：

- 冲锋与打坐外服抓包分析及Go移植.md（18 轮移植实录；4 处结论被推翻）
- 技能pcapng移动交互与冲锋外服分析.md（§二距离结论被推翻，§一并入总集 §二）
- 冲锋自走触发链外服定版与差集.md（P0 系列两处被修正；终局并入总集 §四）
- 技能1pcapng多套路冲锋外服方言.md（定版主体并入总集 §一）
- lua64-analysis2.md（纯摘要，内容已被 lua64-analysis.md 全覆盖）
