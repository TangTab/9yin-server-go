# 客户端 share 直读可行性分析（2026-11-25）

> **已落地**（同日）：NPC 摆放 / 模板 / 战斗、任务域、任务文案的装载器已可直读
> share 树，XML 产物退为回退与 A/B 基线；开关、保险、对拍结果与部署步骤见
> [NPC与任务数据来源直读落地](NPC与任务数据来源直读落地.md)。本文保留作取舍
> 依据；下文「本文只做分析，代码未动」是分析当时的状态。文中的四步建议
> （§七）即实际做法，其中「前提：先换部署根的 share」改由缺省模式下的保险
> 代替（share 看起来比产物旧即回落 XML），部署根不必为此先重铺。
>
> 问题：客户端 `share` 整树已经进了数据根，NPC、任务这些域**还有必要先转成服务端
> XML 再读吗**？
>
> 结论先行：
> 1. **NPC（模板 / 摆放 / 战斗模板）、任务域、文案：不必要。** XML 在这些域只是
>    「换包装」——把 TAB 表、creator XML、键值档包成 `<Object><Property k="v"/>`，
>    外加一层列过滤。服务端三个消费方要的都只是「字符串键值行」，直读 `share/`
>    就能给出同样的行。
> 2. **物品、技能：仍然要。** `ini/Item`、`ini/Skill` 不是包装，是另一个模型
>    （设计态 INI → 登记表 + 模板分档 + 静态包 / 引用包），转换本身就是业务逻辑。
> 3. 现行 XML 链的代价是实打实的：**有损**（列位 57% 被丢，含服务端正在读的
>    `Photo`）、**易过期**（此刻就有三处不同步，见 §五）、每加一个 NPC 字段要走
>    「改映射 → 重跑构建器 → 并层 → 重启」。实测直读还**更快、不更费内存**（§四）。
> 4. 直读不是零成本：要先把部署根的 `share` 换成指定那份（层内现在是旧快照）、
>    读入口统一编码自适应、把构建器里的合并 / 展开逻辑挪进装载器。建议分四步切，
>    每步与 XML 链对拍（§七）。
>
> 本文只做分析，**代码未动**。复测工具 `tmp/shareprobe`（见 §八）。

## 一、客户端 share 树是什么格式

`F:\Game\Online\9yin\九阴资源\resources\share`：9732 档 / 267.5 MiB / 39 个一级域。

| 扩展名 | 档数 | 体量 | 哪些域 |
|---|---|---|---|
| `.ini` | 4045 | 171.4 MiB | 物品、技能、规则、任务散表… |
| `.txt` | 3633 | 33.8 MiB | NPC 模板表、任务域表 |
| `.xml` | 2039 | 59.3 MiB | NPC 摆放（`creator/npc_creator`）、mapcircle |
| `.tab` / `.idres` | 2 / 13 | 3.0 MiB | `npctemplate*.tab` 等 |

也就是说 **share 里绝大多数本来就不是 XML**，「转 XML」是把三种方言
（TAB 表 / INI / creator XML）归一到服务端一种方言。

各域的原始形态：

| 域 | 原始位置 | 形态 |
|---|---|---|
| NPC 模板 | `share/npc/npcconfig/**/*.txt` | 76 张 GBK + TAB 表，3 行头（中文注释 / 类型 / 列键）；**几乎每张表一套列头**（76 张里只有 3 张共用一套）；同名列（WeaponName / FreeTalk…）一表内出现两次 |
| NPC 摆放 | `share/creator/npc_creator/<场景>/*.xml` | 407 个场景目录 / 2299 档；`<staticcreator><item/>` 静态项 + `<randomcreator><creator><item><position/>` 随机组；UTF-8 BOM 为主（1 档 GBK），2 档损坏 |
| 战斗模板 | `share/task/npctemplate*.tab` | 1 行头 TAB 表，多档同 ID 首档胜出 |
| 任务域 | `share/task/task/<域>/*.txt` | 15 个域目录 / 3543 档，2 行头（类型 / 列键）；**同域各档列头不统一**，有错置档 |
| 文案 | `text/chineses/stringtask_{normal,special}/*.idres` 等 | `key=value` 行档（UTF-8 BOM / GBK 自适应）——注意**不在 share 下** |
| 物品 | `share/item/*.ini` | 设计态 INI（`[-1]` 列序头 / `[0]` 列宽头 / `[configID]` 节） |
| 技能 | `share/skill/*.ini` + `share/rule/flow_config.ini` | 设计态 INI |

客户端自带的 `npc_type.ini` 把来源写作 `ini\Npc\NpcConfig\…\X.txt`——**客户端表就是
服务端表的镜像**，官方服务端读的也是这些 `.txt`。

## 二、现行 XML 构建链在每个域做了什么

`tools/datagen npc`（模板 / 摆放 / 任务 / 战斗 / 文案）、`item`、`skill`。

| 域 | 输出 | 实际做的事 | 性质 |
|---|---|---|---|
| NPC 模板 | `ini/Npc/NpcConfig/<组>/<表>.xml`（`ini/Npc` 共 77 档 39.6 MiB，含 NpcTemplate.xml） | 行包成 `<Property>`；**列过滤**：审定映射 80 个列键直通 / 6 个改名（PathID→PathId、PowerNo→FightID、SelfForceID→Force、DropID→WorldDropID、FacultyExp→Exp、table@SkillRec→SkillPackID）、`table@` 前缀去前缀、**其余全部丢弃** | 换包装 + 有损过滤 |
| NPC 摆放 | `ini/Scene/<场景>/npc_creator.xml`（406 档 84.2 MiB） | 同场景多档并一档；随机组按 `<position>` 逐点展开（`No=组:项:序`）；加 `File` 溯源属性；排除 `eventtrigger.xml` | 机械展开 |
| 战斗模板 | `ini/Npc/NpcTemplate.xml` | 多档合并包装 | 换包装 |
| 任务域 | `ini/Quest/<域>/<域>.xml`（19 档 18.4 MiB） | 同域各档按列键并集合并、行按列键对齐；域签名过滤错置档；主表 `Title` 按 `TitleId` 到文案档解析（无文案留空→装载链不进目录） | 合并 + 一次联查 |
| 文案 | `ini/Text/{TaskText,NpcTalk}.xml`（2 档 38.8 MiB） | 键值表包装 | 换包装 |
| 物品 | `ini/Item`（22 档 37.1 MiB） | 重建黑金物品模型：`path_config` 登记表 + 11 模板分档 + 9 静态包 + 类型树（由 lua 定义生成） | **模型重建** |
| 技能 | `ini/Skill`（12 档 29.3 MiB） | 重建运行时模型：技能对象行按逻辑类分档 + 引用包分档 | **模型重建** |

服务端消费方全部只吃「字符串键值行」：`npcfeed.LoadStore` / `LoadPlacements`、
`taskdata.LoadQuestXMLDir`、`fsgame` 的 `questnineyin.go` / `questmods.go`
（`loadPropRows`）。没有任何一方需要 XML 本身。

**已经在直读 share 的服务端代码**：`share/creator/mappath`（巡逻）、
`share/trade/shop.ini`（商店）、`share/modifypack/proppack.ini`、
`share/area/areainfo.ini`、`share/clone/cloneinfo.ini`（fsgame）、`share/item`
（物品目录回退）、`share/rule/door_data.ini`（门）。

## 三、列过滤丢了什么

`artifacts/local/npcgen-report.json`（2026-11-25 00:42 那次构建）：

| 项 | 数 |
|---|---|
| 表 / 行 / 唯一 ID | 76 / 77,394 / 77,093（ID 冲突 45） |
| 列位：直通 / 改名 / table@ / **丢弃** | 1372 / 166 / 175 / **2309** |
| 丢弃占比 | 2309 / 4022 = **57%** |
| 被丢的不同列键 | **534 个**（入表的只有 80 个） |

丢掉的大头：`LuaScript`（67 表）、`AIEditorPackID`、`CKindID`、`Photo`、`School`、
`NotPositive`、`BossLevel`、`Material`、`RandomBorn`、`AddPropID`、`NormalSkill`、
`SkillHoldTime`、`CallHelpID`、`CantBeStick`、`RelivePos`…… 其中不少是服务端迟早要用的
（AI、脚本、出生规则），不只是客户端表现列。

**现成的例子**：`internal/npcfeed/frame.go` 的 `resourceTrio()` 读 `tpl["Photo"]` /
`tpl["Material"]`，而 `Photo`（55 表）和 `Material`（23 表）都在丢弃名单里——客户端
`city_commonnpc.txt` 有 6440 行带真实 `Photo`（如 `icon\funcnpc\worldnpc233.png`），
产物 `city_CommonNpc.xml` 里 `Photo=` 出现 0 次，服务端只能用拼出来的兜底路径。

## 四、实测对照（`tmp/shareprobe`，仓内 testdata，热缓存）

| 域 | 现行链：读 XML 产物 | 直读 share 原始表 |
|---|---|---|
| NPC 模板 + 战斗模板 | 567–588 ms | **130–147 ms** |
| NPC 摆放（全部 405 场景） | 1134–1157 ms | **754–756 ms** |
| 任务域 | 176–190 ms | **60–70 ms** |
| 模板域常驻内存 | 204.9 MiB（77,093 + 6,272 行，每行一个 map） | **185.7 MiB**（全部 9,014,946 个列位都留——上界） |

规模对得上：模板带 ID 行 77,393 ↔ 产物唯一 ID 77,093；摆放实例 268,507 ↔ 构建报告
268,507。

TAB 表比 XML 紧凑（没有逐格重复的属性名），所以直读反而快；内存上按表共享列索引的
行切片，即使全部列都留，也不到「每行一个 map」的开销。**性能不是保留 XML 的理由。**

体积：部署根 `ini/` 产物共 581 档 244.5 MiB，其中 Npc + Scene + Quest + Text 四项
约 177 MiB 是 share 同一份数据的第二份副本。

构建器本身（`datagen npc`，解析 + 写盘，不含 verify）一次 13.65 s。

## 五、此刻就存在的三处不同步

XML 链是「一份源 + 一份产物」，产物不会自己跟上源。

1. **部署根层内的 share 是旧快照，而层内 XML 是新的。** `Data001.rep` 的 `share/` 与
   `九阴资源\resources\share` 相比：少 111 档（`npc_creator` 66、`mappath` 3、
   `task/task` 7…）、多 11 档（`share/drop/*`、`*.bak`）、同名不同内容 201 档
   （`npc/npcconfig` 24、`task/task` 24…）。直读对账：模板 76,441 行（↔ 77,393）、
   场景目录 389（↔ 405）、摆放实例 264,549（↔ 268,507）、任务主表 15,391（↔ 15,434）。
   而层内 XML 产物与仓内 testdata 完全同数（77,093 / 6,272 / 405 / 14,925）——
   说明 XML 是按新树构建的，share 是另一份旧树。**部署根若直接改成读 share，
   NPC / 任务会退回旧快照，必须先换 share。**
2. **仓内 testdata 层内 XML 落后于当前构建器。** 工作区 `tools/datagen/npc/mapping.go`
   新增了 `BornAction` 直列绑定（出生姿态那次改动），但层里没重跑：用当前代码重建，
   6 张模板档共 68 行多出 `BornAction="…"`（`ty_cry2`、`diedownloop`、
   `ty_kneel_peccavi`、`appear` 等值；`scene_commonnpc` 另有 7 行是图标路径脏值）。
   层内这 68 行的模板没有任何出生状态值（摆放行另带 `BornAction` 的除外）。这一处是
   **有意的**——当时的判断是重建后只多 4 个姿态，不值得为一趟并层重启；它说明的是
   「产物与构建器可以各走各的」，而不是谁忘了。
3. **全量测试有 1 条红测，正是编码问题。** `internal/fsgame` 的
   `TestCloneInfoAndAreaRules`：`b002.rep`（11-25 00:10 重铺）把客户端原始 **GBK** 的
   `share/clone/cloneinfo.ini` 盖到了原先转成 UTF-8 的快照上（079f866 之前入库的那份是
   UTF-8），而 `LoadCloneInfo` 走 `fskit.ParseItems` 不转码，`Desc` 是 GBK 字节、比不上
   UTF-8 字面量。直读 share 之后这类问题会从「一条测试」变成「每个读入口」。

## 六、逐域结论

| 域 | 还要转 XML 吗 | 理由 |
|---|---|---|
| NPC 模板 | **不要** | 纯包装 + 有损过滤；直读拿到全部列，加字段不用再改映射重建 |
| NPC 摆放 | **不要** | 源就是 XML；展开逻辑约 100 行，放装载器即可 |
| 战斗模板 | **不要** | 一行头 TAB 表直读 |
| 任务域 | **不要** | 合并逻辑进装载器；15 个域 → 现有 `Catalog` 接口不变 |
| 文案 | **不要** | 键值档直读（需 `text/chineses` 在数据根，部署根已有、仓内 testdata 没有） |
| 物品 | **要** | 模型重建；`itemdata` 已是按产物做偏移索引的懒加载（10.5 万件约 90 ms），产物缺席时的 `share/item` 回退只有目录面 |
| 技能 | **要** | 模型重建；源表脏数据（SVN 冲突标记、表头元数据行）在构建期清理更合适 |

另一个选择是**保留 XML 但把构建器改成不丢列 + 每次换 share 必重跑**：改动最小，
但「一份源两份副本、产物会过期」的结构性问题还在，§五的 1、2 两条以后还会出现。

## 七、如果要直读：要做的事与建议步骤

前提（不做就不能切）：

- **部署根的 share 换成 `九阴资源\resources\share`**（层内是旧快照，见 §五-1）。
- **统一的读入口做编码自适应**（合法 UTF-8 直通、否则 GB18030）：TAB 表、INI、
  creator XML（有 1 档 GBK）、`.idres` 全部走它。顺手把 `LoadCloneInfo` 这类已有
  直读点修掉。

装载器要接手的构建期逻辑（现在在 `tools/datagen/npc` 里）：

- 摆放：多档合并 + 随机组 `<position>` 展开 + `eventtrigger.xml` 排除（`creator.go`）。
- 模板：同名列取首见、无 ID 行跳过、6 个改名映射在装载时套（消费方读的是
  `PathId` / `FightID` / `Force` / `WorldDropID` / `Exp` / `SkillPackID`）。
- 任务：域签名过滤错置档、列键并集、短行补齐 / 超宽截断、主表 `Title` 解析。
- 脏数据从「构建期报告 + 跳过」变成「装载期告警 + 跳过」：2 档损坏的 creator XML、
  `npctemplate2.tab` 无列键、45 个 ID 冲突。

建议顺序（每步都可独立上线、与 XML 链对拍）：

| 步 | 改动 | 对拍口径（切前切后应相同） |
|---|---|---|
| 1 摆放 | `npcfeed.LoadPlacements` 改读 `share/creator/npc_creator/<场景>/*` | `npcprobe -all`：物化 82,837 / 有巡逻场景 64 / walkers 2,513 |
| 2 模板 + 战斗 | `npcfeed.LoadStore` 改读 TSV | 模板 77,093 / 战斗 6,272；`npcprobe -all` 物化数、姿态数 |
| 3 任务 + 文案 | `taskdata` 与 `fsgame` 的任务读取改走共享读入口 | 带文案任务 14,925；NPC↔任务绑定数 |
| 4 收尾 | `datagen npc` 只留审计报告（不再产出这四类 XML）；层内旧产物登记删除（约 177 MiB） | — |

过渡期留一个开关（如 `NINEYIN_NPC_SOURCE=share|xml`）做 A/B，与现有
`NINEYIN_NPC_POSE=off` 同一风格。

## 八、复测

```bash
go run ./tmp/shareprobe -root testdata                               # 耗时对照（A 现行链 / B 直读）
go run ./tmp/shareprobe -root testdata -mem                          # 模板域常驻内存
go run ./tmp/shareprobe -root testdata -build <临时目录> -client F:/Game/Online/9yin/九阴资源/resources
                                                                      # 层内 XML 与当前构建器是否同步
go run ./tmp/shareprobe -root F:/Game/Online/9yin/9yin-server-go2/Res # 部署根
```

`-build` 把 npc 构建器跑到临时目录（不改数据根、不写仓内报告），逐档与层内同名产物
比对（行尾归一）。

## 九、代码索引

| 文件 | 职责 |
|---|---|
| `tools/datagen/npc/{npcgen,creator,mapping,table,manifest,tasks,text}.go` | 现行构建器：各域转换逻辑（直读时要挪进装载器的部分） |
| `internal/npcfeed/load.go` | 模板 / 战斗 / 摆放装载（XML 消费方） |
| `internal/taskdata/questxml.go` | 任务域装载（XML 消费方） |
| `internal/fsgame/questnineyin.go`、`questmods.go` | 框架层任务模块（XML 消费方） |
| `internal/fsgame/scenemods.go` | `LoadCloneInfo`：已有直读点，GBK 未转码 |
| `tmp/shareprobe/main.go` | 本文 §四、§五-2 的实测工具（不入库） |
