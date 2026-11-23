# testdata — 规格输入与对拍资产

本目录内容**仅限研究用途**（见根 README 许可证声明），来源与用途如下。

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

## ini/Skill/ — 技能配置（九阴构建产物 + 黑金原档备份）

- `Skill_New/<逻辑类>.xml`、`skill_static.xml`、`Skill_VarProp/`、
  `Skill_Consume/TargetShape/HitShape/Flow/LockData/UseLimit/RushRange.xml`
  — **九阴构建产物**：`tools/skillgen` 把官服设计态表
  `share/skill/*.ini` + `share/rule/flow_config.ini` 离线构建为服务端 XML
  （黑金模型：技能对象行 + 引用包分档，运行时联查），服务端只读产物、
  不读设计态表（装载器 `internal/skilldata`）。再生命令：
  `go run ./tools/skillgen -in <Res>/share/skill -rule <Res>/share/rule -verify`
- `heijin_backup/` — 黑金原档 7 个样本（`skillgen` 首跑整体移入）：仅作
  `internal/fsgame` 测试夹具与属性词汇对照，`fsgame` 的 `readResource`
  在规范位缺席时回退到此目录。

## ini/Item/ — 物品配置（九阴构建产物，黑金物品配置格式）

`tools/itemgen` 把九阴设计态表 `share/item/*.ini` 离线构建为服务端 XML
（黑金 `ini/Item` 同构：`path_config.xml` 登记表 + 11 个模板档
`Equipment/Weapon/Material/ItemUse/Rune/ItemMountUse/TaskItemUse/
TaskMaterial/ItemBox/SkillBook/add_money.xml` + 9 个静态包
`item_func_static/item_logic_pack/ItemArtStatic/item_use_skill_static/
item_learn_skill_static/item_task_static/item_ingredient_static/
item_prop_modify_static/item_seed_static.xml` + `ItemTypeList.xml`；22 文件
12.5 万行）。服务端只读产物（装载器 `internal/itemdata`；产物缺席回退
设计态 `share/item`，只有目录面）。再生命令：`go run ./tools/itemgen -verify`
（部署根同步加 `-out <Res>/ini/Item`）。详见
`docs/official/物品服务端XML构建链与msg31使用链.md`。

## ini/ — 数值表抽样（config 包的格式样例）

从原版 `Res/ini/` 抽样的代表性文件，用于驱动 `internal/config` 的解析器
开发与测试（不追求全量，全量数据仍以原版目录为准）：

- `scenes.xml` — 场景注册表（含 Clonable/ReliveLocation 语义）
- `Skill/Skill_Config.xml` + `skill_static.xml` + `buff_static.xml` —
  技能四件套的注册与 lua 引用关系（黑金原档，已移入
  `Skill/heijin_backup/`；`Skill/` 根下现为九阴构建产物，见下）
- `CreateRole/RoleBaseProp.xml` — 阵营/种族/职业基础属性
- `Level/Level.xml` — 等级经验曲线

## lua/ — 行为逻辑脚本全量副本（script 层的移植目标与 API 语料）

原版服务端 `Res/lua/` 全量副本（361 个脚本，已转 UTF-8）。按引用状态分三类
（引用计数来自 ini/Skill/heijin_backup/skill_static.xml + buff_static.xml 的 211 处引用）：

- **活脚本**（黑金运行时实际加载，也是本项目 `internal/script` 的移植目标
  与回归用例）：`skill/jn_normal/`（复活/中毒/AOE 等 47 处引用）、
  `skill/ride/`、`skill/clone/`、`buf/` 大部分
- **死脚本·九阴血统**（0 引用，但是九阴系技能脚本写法的直接参考，
  对本项目价值最高）：`skill/jn_menpai/`（少林 mp_sl、武当等门派技能）、
  `skill/jn_jianghu/`（江湖技能，jn_jh_0h~8h）
- **define 语料**（协议旁证）：`share/server_custom_define.lua` 的
  `SERVER_CUSTOMMSG_*` 表、`define.lua`、`command_define.lua`——
  protogen 规格的第三线索源（另两个：FsGame.pdb 符号、probe 抓包）

这些脚本是**开发参考与测试夹具**，不是运行时资源——本项目自己的脚本
（移植/重写后）另行存放，不从 testdata 加载。

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
