# res1/share 配置表体系（玩法逻辑数据）

> 来源：`resources/modern/share`（客户端共享配置，39 个子系统、9,732 文件、267MB）。
> 全量清单：`data/res1_inventory.json`（每张表的段数/字段集/示例段/表说明）。
> 扫描脚本：`tools/text-extract/extract_res1_configs.py`。

## INI 通用惯例

- `[section]` 即配置主键：道具/装备 ID（如 `CLOTH_SZ001`）、技能 ID（`Clone007_bingkuai`）或数字索引（`[3]`）。
- `[-1]` 段是**字段顺序图例/表说明**（字段名→列号），如 `skill_static.ini`、`npc/staticprop.ini` 开头。
- `//` 与 `;` 为注释；行内说明常写 `Comment=` 字段（不计入字段集合）。
- **双层指针结构**：`*_new.ini` 存 `ID → StaticData=<数字索引>` + `script=<逻辑脚本名>`，实际数值在同目录 `*_static.ini` 按索引展开（如 `skill_new` → `skill_static`、`buff_new` → `buff_static`）。
- **varprop 表**：按等级/成长逐段展开的变量属性（`buff_varprop` 5.5 万段、`skill_normal_varprop` 4.6 万段、`skill_consume` 3.8 万段）。

## 子系统清单（按规模）

| 子系统 | 文件数 | 用途 | 代表表 |
|---|---|---|---|
| creator | 2607 (71MB) | 场景生成器：NPC/任务路径点、场景→路径映射 | `mappath/city03_npc.ini`、`scene_creator_path.ini` |
| rule | 2062 | 通用规则：条件、成就、AI事件、装备属性规则 | `condition.ini`(4.1万段)、`equip_prop_rule.ini`(5.0万)、`flow_config.ini`、`ai_editor_event.ini`、`achievement/` |
| task | 3679 | 任务支撑配置（定义在服务端，此处为文本索引/剧情引用/托管路径） | `pre_file.txt`、`task_drama_cite.ini`、`offline_trusteepath.ini`、`npctemplate.tab` |
| skill | 111 (52MB) | 技能/Buff/内功/经脉/穴位/轻功/阵法全量表 | 见下节 |
| item | 97 (40MB) | 道具/装备/合成/兑换/铭刻 | `tool_item.ini`(7.7万段)、`equipment.ini`(2.9万)、`exchangeitem.ini`、`fuse_*`(合成)、`inscription_*` |
| npc | 95 (21MB) | NPC 属性基值与刷新 | `staticprop.ini`(745段 属性基值)、`worldbossrefresh.ini`、`jhscenebossrefresh*`、`transpathrec.ini`(传送路径) |
| war | 389 | 战场/帮战/据点战 | `battlefield_group.ini`、`balance_war_grow.ini` |
| life | 108 | 生活职业（制造/采集产出/熟练度） | `lifeexpinfo.ini`(4.2千段)、`lifedropinfo.ini`、`alchemy.ini` |
| karma | 34 | 善恶/江湖声望/奇遇好感 | `karma_pack.ini`(6.2千段)、`prestige_prize.ini` |
| guild | 70 | 帮会建筑/商店/领地 | `guild_domain_info.ini`、`guildfuncnpc_*` |
| modifypack | 7 (11MB) | 属性修改包（buff/装备/起源打包下发） | `buffpack.ini`、`equippack.ini`、`originpack.ini` |
| force / forceschool | 30 / 51 | 门派势力活动/禁地/门派 BOSS | `bossdata.ini`、`dmp_*`(独门秘技) |
| newschool | 34 | 新门派(昆仑等)养成 | `newschoolcharacter.ini`、`newschoolgrow.ini` |
| home | 59 | 家园/宅邸 | `homebaseconf.ini`、`homeworkshopitemconfig.ini` |
| activity / interactive | 43 / 61 | 运营活动 / 交互玩法(婚宴、年兽、杀BOSS) | `activity_condition.ini`、`year_boss_act.ini` |
| ride | 6 | 坐骑 | `new_ride.ini`、`new_ride_chest.ini` |
| worldwar / worldauction / worldevent / seekemperorcity / sealosttemple / skyhill / challengeclone / championwar / clone / sns / sweetemploy / trade / schoolextinct / area / drama / faculty / globalrank / playerstatue / relive | — | 世界战争/世界拍卖/世界事件/寻皇城/海市蜃楼/凌霄/挑战副本/冠军赛/副本/SNS/甜蜜雇佣/交易/灭派/区域/剧情/名士/全局排行/玩家雕像/复活 | 见 `data/res1_inventory.json` |

## 重点表结构

### skill/（技能全量）

| 表 | 内容 |
|---|---|
| `skill_new.ini` | 技能 ID → `StaticData`(指向 skill_static 行) + `script`(SkillNormal 等) |
| `skill_static.ini` | 每技能静态参数：CoolDownCategory/DamageType/TargetType/LauchType/TaoLu(套路)/SkillType/LuaScript 等（`[-1]` 段为字段图例） |
| `skill_normal_varprop.ini` / `buff_varprop.ini` | 按等级变化的伤害/效果数值（4.6万 / 5.5万段） |
| `skill_consume.ini` | 出招消耗（内力/气血） |
| `damage_calculate.ini` | 5.1万段伤害计算参数 |
| `attack_hitshape.ini` / `attack_targetshape.ini` | 范围形状（`[3]`=以自身为圆心半径2米，Comment 有中文说明） |
| `neigong/` `jingmai/` `xuewei/` `qinggong/` `canwu/` `huihai/` `binglu/` | 内功、经脉、穴位、轻功、领悟、恢复、冰络子系统表 |
| `zhenfa_*.ini` | 阵法（阵图/属性/技能/位置） |
| `cooldown_big.ini` `cooldownteam.ini` | 大冷却/共享冷却组 |

### item/（道具装备）

- `tool_item.ini`（7.7万段）：普通道具主表；`equipment.ini`（2.9万段）：装备 ID → 模型/耐久/装备位/基础属性包（如 `IneBasePropId=ine_main_dragon_yu`）。
- ID 空间与文本表一致：`equipment.ini` 的 `CLOTH_SZ001` ↔ `stringname.idres:CLOTH_SZ001`（名称）+ `desc.idres:desc_CLOTH_SZ001`（描述），见 `data/items.jsonl.gz`。
- `fuse_formula*.ini` 合成配方、`exchangeitem.ini` 兑换、`inscription_*` 铭刻、`item_learn_skill_static.ini` 学习类道具。

### npc/（NPC 数值）

- `staticprop.ini`：745 段属性基值表（HPBase/StrBase/DexBase/各防御 Base），`[-1]` 为字段图例。
- `worldbossrefresh.ini` / `jhscenebossrefreshinfo.ini` + `...point.ini`：世界 BOSS / 江湖场景 BOSS 刷新点与时间。
- `transobj/`、`transpathrec.ini`：传送物件与路径。

### task/（任务索引，定义本体在服务端）

- **`pre_file.txt`：游戏自带的任务字段前缀索引表** —— 声明 `title_/target_/context_/menu_/CanAcceptMenu_/CompleteMenu_/complete_/next_` 等前缀按 TID（任务 ID）左连接到 `stringtask_normal` 等文本表。`data/quests_text.jsonl.gz` 即按此规则聚合（共 27,486 任务）。
- `task_drama_cite.ini` / `task_zx_cite.ini`（3.7千段）：任务 ↔ 剧情引用；`npctemplate.tab`：任务 NPC 模板（TAB 分隔）。

## 与服务端（9yin-server）的关联

- 任务 ID：服务端任务定义 ↔ 客户端 `stringtask_*` 文本（`data/quests_text.jsonl.gz`）。
- 道具/装备 ID：`share/item/*` ↔ `data/items.jsonl.gz`（名称+描述+数值参数）。
- Buff ID：`share/skill/buff_new.ini` ↔ `data/buffs.jsonl.gz`（`buf_xxx` 名称与效果描述）。
- NPC ID：`share/npc/*` + `ini/npc/*`（表现）↔ `data/npcs.jsonl.gz`。
