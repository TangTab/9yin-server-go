# res1 客户端资源包总览（ini / lua64 / map / text）

> 来源：`resources/`（完整客户端资源，共 777MB）。
> 版本：**官方版**（`ini/res_ver.ini` 中 `VersionType=1`，0=蜗牛GM 1=盛大GM 2=华人服GM）。
> 清单数据：`data/res1_inventory.json`（overview / share / ini / map / lua64 五部分）。

## 顶层结构

| 目录 | 文件数 | 大小 | 职责 |
|---|---|---|---|
| `text/` | 598 | 113MB | 多语言文本（`chineses/` 为主，另 `language.ini` 声明简/繁/英） |
| `share/` | 9,732 | 267MB | **玩法逻辑配置表**（服务端同构数据，详见 [share-configs.md](share-configs.md)） |
| `ini/` | 18,548 | 81MB | **客户端表现配置**（UI/NPC模型/过场/特效/声音） |
| `map/` | 7,444 | 226MB | 场景资源（灯光/粒子/模型 + 寻路 path） |
| `lua64/` | 2,297 | 26MB | 客户端 Lua 脚本（界面/流程/触发器） |

## text/chineses 相对上一版的差异

上一轮整理源为 `E:/Workspace/九阴服务端/resources/modern/text/chineses/`；res1 版本更新：

- 18 个 idres 有内容更新（desc/desc2/desc3/interface/stringname/stringname_npc/string_buff_desc/string_skill_desc/scene/systeminfo/tips/stringtalk/string_conditiondesc/string_npctalk/string_taskmenu）。
- 新增 4 个子目录：`stringtask_normal/`(277 文件) + `stringtask_special/`(278 文件) 任务文本、`form_helper_text/`(玩法帮助 name+desc)、`string_player_name/`(姓 + 男名/女名库)。

## ini/（客户端表现配置）

- 根目录 82 个 ini：全局开关与功能入口（`res_ver.ini` 版本开关、`func_btns*.ini` 功能按钮、`world_boss_info.ini`、`wuxueshop.ini` 武学商店、`shopmount.ini`、`rolecreate.ini` 角色创建、`gmconfig.ini` GM 配置等）。
- 重点子目录：

| 子目录 | 文件数 | 内容 |
|---|---|---|
| `npc/` | 7,303 | NPC 客户端模型配置（`[COMPOSITE] Model=obj\npc\...xmod`） |
| `quest/` | 7,497 | 过场动画工程（`project_info.ini`：相机/时间轴，编辑器工程文件） |
| `scenario/` | 1,467 | 场景脚本/剧情编排 |
| `movie/` | 1,427 | 影片/CG 引用 |
| `ui/` | 222 | 界面贴图与布局 |
| `card/` | 125 | 卡牌 |
| `create/` | 55 | 角色创建资源 |
| `form/` `form_outland/` | 48 | 界面表单配置 |
| `effect/` `particles/`(在 map) | 62+ | 特效引用 |
| `func_guide/` `new_helper/` `freshmanhelp/` | 42 | 新手引导 |
| `skill/` `school_convert/` `school_destroy/` `origin/` `selectbook/` | 少量 | 武学表现/转门派/灭派/起源/选书 |

## lua64/（客户端脚本，2,297 文件）

- `form_stage_main/`：主界面全部窗体（`form_activity/`、`form_9yinzhi/`、`form_adventure/`…每个系统一个 `form_*.lua`）。
- `form_stage_login/` `form_stage_create/` `form_stage_roles/`：登录/创建/选角流程。
- `define/`：常量定义；`form_common/`：通用控件。
- `player_state/`：玩家状态机；`dynamic_weather/`：动态天气；`trigger/`：触发器；`share/`：与 share 配置对齐的脚本；`terrain/`、`ocean_edit/`、`haiwai/`（海外版差异）。
- 入口：`client.lua`（主循环）、`console.lua`、`custom_effect.lua`、`custom_sender.lua`（与服务端消息对应的发送器）。

## map/（场景资源）

- `map/ini/`：`lights.ini` 灯光、`particles*.ini` 粒子（5 组）、`model_weather.ini` 天气模型、`mdl/` 模型引用、`group/`、`effect/`。
- `map/path/`：560 个 `.path` 寻路文件，按场景前缀分组（`clonezone001/002` 各 46、`battle` 18、`jianghu` 12、`clone004` 6、`adv155/168/176` 各 4、其余为各场景 `_main.path` + 分路径）。文件名与 `scene.idres` 场景 ID 同空间（如 `adv031`）。

## 与 docs/text-extract 其他数据的关联

- 场景 ID：`map/path` 前缀 ↔ `system_rules.json:scene`（`area_NNNN` 区域名）。
- NPC：`ini/npc/<ID>.ini`（模型）+ `share/npc/*`（数值）↔ `npcs.jsonl.gz`（名称/类型/对话）。
- 任务：`share/task/pre_file.txt` 索引 ↔ `quests_text.jsonl.gz`（27,486 任务文本）。
