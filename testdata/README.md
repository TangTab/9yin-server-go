# testdata — 规格输入与对拍资产

本目录内容**仅限研究用途**（见根 README 许可证声明），来源与用途如下。

## 存储形态（2026-11-24 起）

本目录是一个**多层资源仓**的数据根：绝大部分内容在层文件 `b001.rep`～`b003.rep` 里，散文件
盖在层之上。读取顺序固定为

```
根下散文件  >  b00N.rep  >  …  >  b001.rep
```

资源名就是本目录下的相对路径，所以下文各节写的路径都不变，只是不再以散文件
形式出现在工作区。

保持散文件、不进层文件的：

- `mpq/` —— `internal/mpq` 自身的测试夹具（读归档的测试不能依赖归档）。
- `references/` —— **参考资源根**，自带层文件 `h001.rep`（黑金原档全量，6489 个
  文件：ini 5708 / lua 368 / struct 209 / dumpconfig 142 / map 49 …）。它是嵌套在
  本目录里的另一个数据根：`references/…` 下的路径按它自己的层解析，与外层互不
  相干；`h001.rep` 是层容器，不会被收进 `b` 层。
- `README.md` —— 本文件。

**读取**：代码一律经 `internal/mpq/res` 的路径式入口读本目录——`res.ReadFile` /
`res.Stat` / `res.ReadDir` / `res.WalkDir` / `res.Glob`，用法与 `os.*` /
`filepath.*` 完全相同。直接用 `os.ReadFile` 之类读本目录的代码，在散文件并入层
之后就找不到文件了。散文件改完即生效；层文件的内容要重新打开才更新。

**构建数据**：只有 `item`（物品配置）、`skill`（技能配置）、`action`（招式判定节拍）**必须**经统一构建器
`tools/datagen` 离线构建——它们是模型重建。NPC（模板 / 摆放 / 战斗）、任务域、
任务文案的装载器从 2026-11-25 起可以直接读层里的客户端 `share` 树，不必先转
XML（开关与保险见 `docs/official/NPC与任务数据来源直读落地.md`）；`tools/datagen
npc` 与 `patrol` 仍可运行，产物（`ini/Npc`、`ini/Scene`、`ini/Quest`、
`ini/Text`）留在层里作回退路径与 A/B 对拍基线（`NINEYIN_DATA_SOURCE=xml`）。
构建器只需要两类输入：客户端资源解压根（`-client`）与黑金参考资源根（上面的
`references/`，缺省就是它）：

```bash
go run ./tools/datagen                      # 构建全部
go run ./tools/datagen item skill action    # 只构建指定的（npc / patrol / item / skill / action）
go run ./tools/datagen -client <解压根>     # 别的机器：指定客户端资源根
```

生成物先以**散文件**落在对应目录（`ini/Npc`、`ini/Scene`、`ini/Quest`、
`ini/Text`、`ini/Item`、`ini/Skill`、`ini/Action`、
`share/creator/mappath`），验证无误后再并入资源层。
写出时行尾跟随层里的同名现有文件，所以内容没变的文件重建后与层里逐字节相同，
并入时会被去重（`patrol` 是字节透传，本就与源逐字节相同）。

本目录层里没有 `text/chineses`（任务文案），所以任务域在这里缺省走 XML 产物；
要验任务域直读，把文案目录指到客户端树：
`NINEYIN_TASK_TEXT_DIR=<客户端资源根>/text/chineses go test ./internal/... -run Parity`。

**更新**：改动的文件先以散文件形式放在本目录，跑完验证后并入新的增量层：

```bash
# 只收与下层内容不同的文件 → 下一个增量层，并清掉已并入的散文件。
# config-bootstrap / lua-bootstrap / mysql / share.7z 是受 git 跟踪的散文件，须排除，
# 否则会被吞进层并从工作区删掉。
go run ./cmd/mpq store pack testdata -prefix b -prune \
  -exclude mpq -exclude README.md \
  -exclude config-bootstrap -exclude lua-bootstrap -exclude mysql -exclude share.7z
```

**查看**：

```bash
go run ./cmd/mpq store info   testdata               # 各层与合并视图统计
go run ./cmd/mpq store list   testdata ini/Skill     # 每个文件所在的层
go run ./cmd/mpq store read   testdata scenes.ini    # 命中层 + 内容预览
go run ./cmd/mpq store unpack testdata               # 合并视图解到同级的 testdata-unpacked/
```

模型、CLI 与接入规则详见 `docs/client/mpq-resource-system.md`。

## 编码说明（已全量转为 UTF-8）

本目录全部 `.xml` / `.lua` 文件已由原版 GBK **转码为 UTF-8**，XML 头声明
同步改写为 `encoding="utf-8"`（共 610 文件转码、217 声明改写）。

- 所有解析入口（structgen / protogen / config）已改为**编码自适应**：
  合法 UTF-8 直通、否则按 GBK 转码——因此本快照（UTF-8）与
  `NINEYIN_DATA` 直读原版（GBK）两条路径均可工作
- 转码已验证无损：转码前后生成器产物逐字节一致
- 将来从原版刷新快照时，拷入后需重跑一次转码（GBK 检测 + 声明改写）

相对原版副本的本地修正（转码之外）：

- `player/PetShow.xml:166` — 装饰分隔注释整行横线（注释体含 `--`），
  改为 `=` 线（原文件不合法 XML，原版引擎解析器容忍）
- `rule/transtool.xml:18` — `maxrows=1` 属性值未加引号，补为 `maxrows="1"`

structgen 的容错链（剥非法注释 / 补裸数字属性）保留未删——将来换入九阴
正版 struct 仍可能有同类问题，作为兜底。

## struct/ — 对象属性定义（structgen 的输入）

原版服务端 `Res/struct/` 的全量副本（208 个 XML）。这是对象模型的事实源：
每个游戏对象的属性名/类型/`public/private/save/realtime` 标志。
`tools/structgen` 读取本目录生成 `internal/component`。

关键文件：

- `struct_template.xml` — 属性定义模板，含各标志语义的官方注释
- `player/` — 玩家对象属性（含 Record 表定义）
- `skill/skillbase.xml` — 技能对象属性

## ini/Skill/ — 技能配置（九阴构建产物）

- `Skill_New/<逻辑类>.xml`、`skill_static.xml`、`Skill_VarProp/`、
  `Skill_Consume/TargetShape/HitShape/Flow/LockData/UseLimit/RushRange.xml`
  — **九阴构建产物**：`tools/datagen skill` 把客户端设计态表
  `share/skill/*.ini` + `share/rule/flow_config.ini` 离线构建为服务端 XML
  （黑金模型：技能对象行 + 引用包分档，运行时联查），服务端只读产物、
  不读设计态表（装载器 `internal/skilldata`）。再生命令：
  `go run ./tools/datagen skill`
- 黑金原档 7 个样本（`Skill_Config.xml`、`skill_static.xml`、`buff_static.xml`
  等）在参考资源根 `references/ini/Skill/`（原字节，`gb2312` 声明）：仅作
  `internal/fsgame` 测试夹具与属性词汇对照，`fsgame` 的 `readResource`
  在规范位缺席时回退到那里取同名黑金样本。

- 命中结算包（2026-11-25 战斗接入，同一构建器）：`Skill_DamageCalculate.xml`、
  `Skill_Formula.xml`、`Buff_New/`、`buff_static.xml`、`Buff_VarProp/`、
  `Skill_StaticToId.xml`（伤害计算包 → 公式 → buff 族，装载面
  `internal/skilldata/{damage,buff}.go`）。源表脏数据的**审定修正**在构建期
  改写（`tools/datagen/skill/mapping.go` 的 `Fixups`，入审计报告）：产物是层里的
  生成物，手改会被下次重建覆盖，手修定版一律落 `Fixups`。

## ini/Action/ — 招式判定节拍（九阴构建产物）

`tools/datagen action` 把客户端招式表 `ini/action/zhaoshi_*.ini` 离线编译为
`ini/Action/zhaoshi_hits.xml`（一招一行：`Actions` 各段动作名 / `Frames` 各段帧长 /
`Hits` 判定时刻毫秒 / `Length` 总时长）。判定帧 = `trigger_hitbreak ∪ hitback`（同刻
去重），30fps，按 `trigger_action_speed` 变速折算；12,770 招、1.3MB。服务端只读
产物（装载器 `internal/actiondata`）——战斗链按技能的 `HitAction` 取判定时刻调度
每击结算；产物缺席时全部查询落空，按流程包 `HitTime` 回退。再生命令：
`go run ./tools/datagen action`。

## ini/Item/ — 物品配置（九阴构建产物，黑金物品配置格式）

`tools/datagen item` 把客户端设计态表 `share/item/*.ini` 离线构建为服务端 XML
（黑金 `ini/Item` 同构：`path_config.xml` 登记表 + 11 个模板档
`Equipment/Weapon/Material/ItemUse/Rune/ItemMountUse/TaskItemUse/
TaskMaterial/ItemBox/SkillBook/add_money.xml` + 9 个静态包
`item_func_static/item_logic_pack/ItemArtStatic/item_use_skill_static/
item_learn_skill_static/item_task_static/item_ingredient_static/
item_prop_modify_static/item_seed_static.xml` + `ItemTypeList.xml`；22 文件，
约 12.6 万行）。服务端只读产物（装载器 `internal/itemdata`；产物缺席回退
设计态 `share/item`，只有目录面）。再生命令：`go run ./tools/datagen item`
（`-out <目录>` 可把产物先生成到别处）。详见
`docs/official/物品服务端XML构建链与msg31使用链.md`。

## share/creator/mappath/ — NPC 巡逻路线（官方路线表原样镜像）

巡逻路线在数据根的 `share/creator/mappath/`——与客户端资源树同路径，客户端
`share` 整树并入时随之到位（2026-11-25 起装载器读这里；此前的顶层 `mappath/`
不再读取，层里残留的那份是死数据，重铺层时可清）。`tools/datagen patrol` 也可
把客户端 `share/creator/mappath/*.ini` 单独搬进来
（251 档约 13.6 MiB；GBK 与行尾都不动——装载器 `internal/npcfeed/patrol.go`
读的就是这份 `r=` 行方言）。节名 = 摆放 `PathId`，文件名 =
`<场景短名>_<类别>.ini`；含 `offline` 的是离线态路线，服务端装载器按文件名
排除。**缺这个目录 = 带 PathId 的 NPC 全部站立**（2026-11-24 Res 层化后曾因此
巡逻全无）。构建器报告按服务端解析器统计路线 / 路点 / 坏行。详见
`docs/official/NPC巡逻mappath路线链.md`。

## ini/ — 数值表抽样（config 包的格式样例）

从原版 `Res/ini/` 抽样的代表性文件，用于驱动 `internal/config` 的解析器
开发与测试（不追求全量，全量数据仍以原版目录为准）：

- `scenes.xml` — 场景注册表（含 Clonable/ReliveLocation 语义）
- `Skill/Skill_Config.xml` + `skill_static.xml` + `buff_static.xml` —
  技能四件套的注册与 lua 引用关系（黑金原档，在参考资源根
  `references/ini/Skill/`；本目录 `Skill/` 根下是九阴构建产物，见上）
- `CreateRole/RoleBaseProp.xml` — 阵营/种族/职业基础属性
- `Level/Level.xml` — 等级经验曲线

## lua/ — 行为逻辑脚本全量副本（script 层的移植目标与 API 语料）

原版服务端 `Res/lua/` 全量副本（361 个脚本，已转 UTF-8）。按引用状态分三类
（引用计数来自黑金原档 `references/ini/Skill/skill_static.xml` + `buff_static.xml` 的 211 处引用）：

- **活脚本**（黑金运行时实际加载，也是本项目 `internal/script` 的移植目标
  与回归用例）：`skill/jn_normal/`（复活/中毒/AOE 等 47 处引用）、
  `skill/ride/`、`skill/clone/`、`buf/` 大部分
- **死脚本·九阴血统**（0 引用，但是九阴系技能脚本写法的直接参考，
  对本项目价值最高）：`skill/jn_menpai/`（少林 mp_sl、武当等门派技能）、
  `skill/jn_jianghu/`（江湖技能，jn_jh_0h~8h）
- **define 语料**（协议旁证）：`share/server_custom_define.lua` 的
  `SERVER_CUSTOMMSG_*` 表、`define.lua`、`command_define.lua`——
  protogen 规格的第三线索源（另两个：FsGame.pdb 符号、probe 抓包）

2026-11-25 起这些脚本同时是**运行时脚本根**：Lua 宿主（`internal/script`）
缺省从本目录 `lua/` 装载技能 / buff 的钩子脚本（技能表 `LuaScript` 列按路径
指到这里；`NINEYIN_LUA_DIR=<目录>` 可改指别处）。每份钩子脚本有独立的函数
环境，`require` 进来的模块全局各归各；内置的伤害公式引擎不在这里。详见
`docs/official/战斗接入与技能Lua宿主.md`。

## lua-bootstrap/ — 各进程引导脚本

原版顶层 `Lua/` 目录副本（36 个），每个进程一套 main/echo 引导，
展示了进程实体装配顺序（SoloMember/SoloPublic/SoloEntry/... 的创建链），
是 engine 装配与 proc 角色定义的参考。

## mysql/ — 原版 MySQL DDL 快照（persist 里程碑事实源）

原版服务端 `Res/mysql/` 全量副本（7 个文件，与 `BGserverCJ\Res\mysql\`
逐字节一致），覆盖三库两形态：

- `nx_base` / `nx_district` / `nx_log`：正式库名
- 同名 `*cj.sql`：黑金 CJ 分支部署库名变体（表结构相同，仅库名不同）
- `create_account.sql`：建账号库脚本

用途：里程碑 7（persist）的建库 DDL 与 `nx_chunks` blob 逆向的表结构
参照（FsSql.pdb 挂符号后交叉验证）。快照只含 DDL，不含表内数据。

## captures/ — 原服抓包与 golden 用例

`cmd/probe` 对原服抓包的落盘位置。约定：

- 原始字节与元信息（消息号、时序、触发动作）成对存放
- 每个切片应可被 proto 的 golden 测试逐字节断言
- 命名建议：`<流程>-<步骤>-<编号>.bin`（如 `login-enter_scene-001.bin`）

## 血统说明

struct XML 与 ini 样本源自黑金（Black Gold）CJ 分支服务端。其中：
ini 数值本体是黑金数据；struct/属性体系与引擎框架为九阴同源，
作为九阴系服务端的对象模型规格使用。背景详见
`E:\Workspace\project\game\九阴真经ol\BGserverCJ\目录分析报告.md`。
