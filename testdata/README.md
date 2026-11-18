# testdata — 规格输入与对拍资产

本目录内容**仅限研究用途**（见根 README 许可证声明），来源与用途如下：

## struct/ — 对象属性定义（structgen 的输入）

原版服务端 `Res/struct/` 的全量副本（208 个 XML）。这是对象模型的事实源：
每个游戏对象的属性名/类型/`public/private/save/realtime` 标志。
`tools/structgen` 读取本目录生成 `internal/component`。

注意：文件为 GBK（gb2312）编码，解析前需转 UTF-8。

关键文件：

- `struct_template.xml` — 属性定义模板，含各标志语义的官方注释
- `player/` — 玩家对象属性（含 Record 表定义）
- `skill/skillbase.xml` — 技能对象属性

## ini/ — 数值表抽样（config 包的格式样例）

从原版 `Res/ini/` 抽样的代表性文件，用于驱动 `internal/config` 的解析器
开发与测试（不追求全量，全量数据仍以原版目录为准）：

- `scenes.xml` — 场景注册表（含 Clonable/ReliveLocation 语义）
- `Skill/Skill_Config.xml` + `skill_static.xml` + `buff_static.xml` —
  技能四件套的注册与 lua 引用关系
- `CreateRole/RoleBaseProp.xml` — 阵营/种族/职业基础属性
- `Level/Level.xml` — 等级经验曲线

## lua/ — 行为逻辑脚本全量副本（script 层的移植目标与 API 语料）

原版服务端 `Res/lua/` 全量副本（361 个脚本，GBK 编码）。按引用状态分三类
（引用计数来自 ini/Skill/skill_static.xml + buff_static.xml 的 211 处引用）：

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
