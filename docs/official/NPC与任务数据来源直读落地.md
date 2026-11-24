# NPC 与任务数据来源：share 直读落地（2026-11-25）

> 结论先行：
> 1. **NPC（摆放 / 模板 / 战斗）、任务域、任务文案**的装载器可以直接读数据根里的
>    客户端 `share` 树，不再必须先经 `tools/datagen npc` 转成服务端 XML。XML 产物
>    退为回退路径和 A/B 基线。**物品、技能不变**，仍读 `ini/Item`、`ini/Skill`
>    （那是模型重建，不是换包装）。
> 2. **缺省行为**：数据根带 share 源就直读；不带、或 share 看起来比 XML 产物旧，
>    就读产物——不带 share 树的旧数据根行为完全不变。
> 3. **验收口径 = 两种来源读同一份数据根，结果逐行相同**（对拍测试常驻，见 §四）。
>    仓内 testdata 实测：摆放 405 场景 / 268,507 行、模板 77,093 行、战斗 6,272 行、
>    任务域 15 域 102,088 行、文案 239,672 键、任务目录 14,925 任务 / 23,208 绑定，
>    全部一致。
> 4. **唯一的行为变化**：4 个 NPC 的静息姿态从 `stand` 变成站姿变体（§五）。
> 5. **部署根（go2）现状**：层内 `share` 仍是旧快照，缺省模式下的保险会让它继续读
>    XML 产物，行为与现在一致；要让部署根走直读，先把 share 换成当前客户端树（§六）。
>
> 取舍的来龙去脉见 [客户端share直读可行性分析](客户端share直读可行性分析.md)。

## 一、数据从哪里来

| 域 | share 源（客户端设计态表） | XML 产物（回退 / A/B 基线） | 装载入口 |
|---|---|---|---|
| NPC 摆放 | `share/creator/npc_creator/<场景>/*.xml` | `ini/Scene/<场景>/npc_creator.xml` | `npcfeed.LoadPlacements` |
| NPC 模板 | `share/npc/npcconfig/**/*.txt` | `ini/Npc/NpcConfig/**/*.xml` | `npcfeed.LoadStore`、`fsgame` NpcResourceModule |
| 战斗模板 | `share/task/npctemplate*.tab` | `ini/Npc/NpcTemplate.xml` | `npcfeed.LoadStore` |
| 任务域 | `share/task/task/<域>/*.txt` | `ini/Quest/<域>/<域>.xml` | `taskdata.LoadQuest`、`fsgame` `loadPropRows` |
| 任务文案 | `text/chineses/stringtask_{normal,special}/*.idres` | `ini/Text/TaskText.xml` | `fsgame` `loadPropRows` |
| 巡逻路线 | `share/creator/mappath/*.ini` | （无；一直直读） | `npcfeed.LoadPatrolRoutes` |
| 物品 | — | `ini/Item`（模型重建） | `itemdata` |
| 技能 | — | `ini/Skill`（模型重建） | `skilldata` |

读入口统一在 `internal/sharedata`：编码自适应（合法 UTF-8 直通、否则 GBK、剥 BOM）、
三种表头的 TAB 表、creator XML 展开、`.idres` 键值档、来源开关。

## 二、缺省怎么选，怎么切

开关 `NINEYIN_DATA_SOURCE`：

| 取值 | 行为 |
|---|---|
| （未设，缺省 auto） | 某域的 share 源在数据根里就直读；不在就读 XML 产物。另有一道保险：share 树与 XML 产物并存、而 share 摆放树比 `ini/Scene` **缺的场景比多的多**（产物按新树建、share 是旧快照的形态）时，判 share 为旧快照，全部回落 XML |
| `share` | 强制直读；某域的 share 源不在时该域仍回落 XML |
| `xml` | 强制读 XML 产物（A/B 基线） |

- 保险的判据只有场景目录集一项，因为层内文件没有可靠的时间戳；share 比产物**多**
  场景（产物过期）不拦，照常直读。
- 任务域的 share 源要求 `share/task/task` 与文案目录**两者都在**（主表 Title 要靠
  文案档解析）。文案目录缺省取 `<数据根>/text/chineses`，可用 `NINEYIN_TASK_TEXT_DIR`
  指到数据根之外（须含 `stringtask_normal`、`stringtask_special`）。
- 巡逻路线不归这个开关管，一直直读 `share/creator/mappath`。

启动日志（首个场景目录装载时各 trace 一次）：

```
npc data: 来源 摆放=share 模板=share 战斗=share（NINEYIN_DATA_SOURCE=""；…）
npc task: 任务目录 14925 任务 / 23208 绑定，来源 share（…）
```

被判旧快照时另有一行说明缺口与办法：

```
npc data: 数据根的 share 摆放树比 XML 产物少 16 个场景（另有 0 个产物没有的），像是旧快照——已回落 xml（读构建产物）；把数据根的 share 树换成当前客户端树后自动改直读
```

`go run ./tools/npcprobe -root <数据根> -all` 的头两行给出同样的信息。

## 三、直读的口径（与 XML 产物逐条对齐）

**摆放**：同场景各档按文件名排序；静态 `<item>` 一行；随机组 `<creator><item>`
按已给出的 `<position>` 逐点成实例（属性合并、`No=组:项:序`、`Random=1`），没给
点的保留一行；`File=来源档名`（下发类别白名单按它判）；空值属性不收；
`eventtrigger.xml` 不算摆放；坏档整档跳过。

**模板**：行 = `ID`、`script`、`Name`、`Desc`（=Name）+ 按源列序的映射列。属性名
与产物一致：审定绑定（`npcfeed/load_share.go` 的 `npcColumnBindings`）直通或改名
6 条（`PathID→PathId`、`PowerNo→FightID`、`SelfForceID→Force`、`DropID→WorldDropID`、
`FacultyExp→Exp`、`table@SkillRec→SkillPackID`），其余 `table@` 前缀列去前缀保留
（撞名加 `_2`）。**未映射的列入不了行**——这是刻意的：物化帧的模板层按官方场景
域字段名同名列直连，未映射列里有一批正是场景域字段（`SpringEffect`、`Material`、
`ActionSoundMode`、`FixedBeHitAct`、`DirectSpring`…），放开等于一次改掉几万个 NPC
的下发帧。要让某个列进模板行，在 `npcColumnBindings` 加一条，重启即生效，不用
重建数据。

**模板表序**（决定同 ID 多表时哪一行胜出，76 张表里有 44 个这样的 ID）：按
「构建器会把它放到哪」排，与产物树 `ini/Npc/NpcConfig` 的目录遍历序逐位一致
（`sharedata.NPCConfigTables`）。细节都出在大小写：`npc_type.ini` 登记串里目录名
大小写不统一（`NpcConfig` / `npcConfig`），构建器在不分大小写的文件系统上落盘时
并进首次建出的目录。按客户端表的全小写名直接遍历会换掉 7 个 ID 的胜出行。

**战斗**：`npctemplate.tab` 主档序前，同目录 `npctemplate*.tab` 按名补充，同 ID
首行胜出。

**任务域**：同域各档按文件名排序后合并——列取并集、行按列键对齐；列头须齐含域
签名才归属本域（挡错置档）；行 = `ID` 置首、空 cell 省略。主表 `Title` 按
`TitleId` 到文案档解析（留空回落 `title_<ID>`），解析不到的任务不进任务目录。

**文案**：两个串目录并表，先 normal 后 special，各档按名排序，首见胜出；空文本的
串键保留 `ID` 行（3,098 个）。

## 四、对拍（测试即验收）

数据根同时带 share 源与 XML 产物时，下列测试逐行比较两种来源；真档不随 git 时
跳过。任务域与文案的对拍另需文案目录（`NINEYIN_TASK_TEXT_DIR`）。

```bash
go test ./internal/... -run 'Parity|FollowSource|TableOrder'
# 仓内 testdata 没有 text/chineses，任务域直读的对拍要指向客户端树：
NINEYIN_TASK_TEXT_DIR=F:/Game/Online/9yin/九阴资源/resources/text/chineses \
  go test ./internal/sharedata/ ./internal/taskdata/ ./internal/fsgame/ -run Parity
# 指定别的数据根（如部署根）：
NINEYIN_DATA_ROOT=F:/Game/Online/9yin/9yin-server-go2/Res go test ./internal/npcfeed/ -run Parity
```

| 测试 | 覆盖 | 2026-11-25 结果（仓内 testdata） |
|---|---|---|
| `npcfeed.TestPlacementSourceParity` | 全部场景的摆放属性行（含随机组展开）与过滤后列表 | 405 场景 / 268,507 行 / 98,205 条一致 |
| `npcfeed.TestNPCConfigTableOrderMatchesXMLTree` | 模板表装载序 | 76 张逐位一致 |
| `npcfeed.TestTemplateSourceParity` | 模板行与战斗行（全部属性） | 77,093 / 6,272 一致 |
| `npcfeed.TestCatalogSourceParity` | 端到端：全部场景联查目录（物化帧输入）逐个 NPC | 405 场景 / 82,837 NPC，仅 4 个静息姿态不同 |
| `sharedata.TestQuestDomainParity` | 15 个任务域 + 根级散表（含行序） | 102,088 行一致 |
| `sharedata.TestQuestTextParity` | 任务文案、带文案的主表 | 239,672 键一致；主表 15,434 行，Title 解析 14,925 |
| `taskdata.TestLoadQuestSourceParity` | 任务目录（任务表 + 绑定序列） | 14,925 任务 / 23,208 绑定一致 |
| `fsgame.TestNpcResourceSourceParity` | NPC 资源登记 | 76 档 / 77,093 NPC 一致 |
| `fsgame.TestNineyinQuestSourceParity` | 引擎词汇任务两表、奖励、NPC 索引、统计、文案 | 15,434 / 28,026 / 424 / 239,672 一致 |

`npcprobe -all` 两种来源的合计：

| 数据根 | 来源 | 场景 | 物化 | walkers | 姿态 |
|---|---|---|---|---|---|
| 仓内 testdata | share | 405 | 82,837 | 2,513 | 15,695 |
| 仓内 testdata | xml | 405 | 82,837 | 2,513 | 15,691 |
| 部署根（缺省） | 判旧快照 → xml | 405 | 82,837 | 2,513 | 15,691 |
| 部署根（强制 share） | share（旧快照） | 389 | 80,735 | 2,513 | 15,662 |

## 五、已知差异与遗留

1. **4 个 NPC 的静息姿态**（直读比产物多出来的唯一变化）：层内 XML 落后于工作区
   构建器的 `BornAction` 直列绑定（出生姿态那次改动有意没重建数据，判断是只多
   4 个姿态）。直读让当前映射即时生效：
   - `city05_chengdu` WorldNpc09633：`stand` → `1_stand`
   - `school08_shaolin` WorldNpc00374、WorldNpc00201：`stand` → `b_stand_1`
   - `test_add` WorldNpc00007：`stand` → `2_stand`

   涉及 68 个模板行、260 个 NPC 的模板行多出 `BornAction`，其中出生状态原始值
   不同 14 个、静息姿态不同 4 个。对拍测试只放行这一种形态。
2. **7 个同名异构的模板 ID**保留 XML 链的胜出行（§三表序）：
   `ngsj_npc_xhq_016`（一张表里是「静心清泉」物件，另一张是战斗怪）、
   `WorldNpc08672`、`WorldNpc09081`、`WorldNpc01302`、`WorldNpc10589`、
   `WorldNpc09955`、`WorldNpc10027`。哪一张才是对的没有依据，这次迁移不改。
3. `npcfeed.BindPatrol` 的「模板 PathID 缺省」读的是 `Template["PathID"]`，而模板行
   的键是 `PathId`（两条来源都如此），这个回落从未生效过。实测无影响：摆放无
   `PathId` 的 80,061 个 NPC 里模板带 `PathId` 的 659 个，没有一条能在本场景路线
   表里找到路线。没改。
4. `tools/gen_scene_waypoints`（离线生成器）仍读 `ini/Scene` 产物。
5. 层内的 XML 产物没有删（部署根里 `ini/Npc`、`ini/Scene`、`ini/Quest`、`ini/Text`
   约 177 MiB）：它们还是回退路径与 A/B 基线。`tools/datagen npc` 保留，用途变为
   审计报告与基线重建。

## 六、部署与运维

**部署根 `F:\Game\Online\9yin\9yin-server-go2\Res`（2026-11-25 05:00 核对）**

- 层内 `share` 仍是旧快照：比 `九阴资源\resources\share` 少 108 档（`npc_creator`
  66、`task/task` 7…）、多 11 档、同名不同内容 201 档。缺省模式下保险生效，
  行为与现在一致。
- 要让部署根走直读：把重铺 `Data001.rep` 所用源目录里的 `share` 换成
  `F:\Game\Online\9yin\九阴资源\resources\share` 后重铺（文案档
  `text\chineses\stringtask_*` 也有 11 档不同，一并换）。换完 `npcprobe -all` 的
  来源行应变成三项 `share`，合计与上表第一行相同。
- 也可以加一层增量层而不重铺 `Data001`：把客户端 `share` 拷进一个临时目录（保持
  `share\…` 相对路径），`go run ./cmd/mpq store pack <Res> -prefix data -from <临时目录>`
  只收与下层内容不同的档。代价是以后再换更新的树重铺 `Data001` 时，这一层会压住
  新数据，要记得一并处理。
- 该层在 11-25 03:03 重铺后已没有顶层 `mappath\`，路线只在
  `share\creator\mappath`（252 档）：**旧 exe 读顶层路径，会读不到路线、NPC 全部
  站立；必须部署新 exe**（`build\game_solounite.exe`，11-25 05:09）。

**带 share 树的数据根要齐备什么**

| 域 | 必需 |
|---|---|
| 摆放 / 路线 | `share/creator/npc_creator`、`share/creator/mappath` |
| 模板 / 战斗 | `share/npc/npcconfig`、`share/task/npctemplate.tab` |
| 任务域 | `share/task/task` + 文案目录 |

缺哪项哪项回落 XML 产物；产物也没有时该域为空（NPC 不下发 / 任务目录缺席），
启动日志会说明来源。

## 七、代码索引

| 文件 | 职责 |
|---|---|
| `internal/sharedata/sharedata.go` | 来源开关、auto 保险（`ShareOlderThanXML`）、编码自适应、`ReadTextFile` |
| `internal/sharedata/table.go` | TAB 表读取（模板 3 行头严格 / 任务域与战斗 1~2 行头宽松） |
| `internal/sharedata/creator.go` | 摆放 creator XML 展开 |
| `internal/sharedata/npc.go` | 模板表清单与装载序（`npc_type.ini` 映射、目录并档、遍历序） |
| `internal/sharedata/quest.go` | 任务域合并、主表 Title 解析、文案并表、`BuiltRows`（产物路径改道直读） |
| `internal/sharedata/idres.go` | 键值行档 |
| `internal/npcfeed/source.go`、`load_share.go`、`load.go` | NPC 三域的来源选择与直读装载；`npcColumnBindings` |
| `internal/taskdata/questxml.go` | `LoadQuest`：任务目录两路来源共用 `buildCatalog` |
| `internal/fsgame/questnineyin.go` | `loadPropRows` 改道；`npcmods.go` NpcResourceModule 直读；`scenemods.go` / `stagemods.go` / `playermods.go` 三个 share ini 读点改走编码自适应入口 |
| `internal/fssoloworld/npc.go`、`npctask.go` | 来源 trace |
| `tools/npcprobe` | 头部标明来源与保险说明 |
