# NPC 作用分类（源自 stringname_npc/npctalk/npcclientinfo 抽取）

> 数据全量：`data/npcs.jsonl.gz`（87,559 条）、`data/talks.jsonl.gz`（35,533 条）、`data/gather_codex.jsonl.gz`（179 条）。
> 名称 `stringname_npc.idres`；台词关联 `string_npctalk.idres` 的 `task_<小写key>_<序号>`（19,677 个 NPC 有关联台词）；采集物图鉴来自 `string_npcclientinfo.idres` 四键一组（story/usepart/planed/usetype）。

## 种类规模与作用

| 种类 | key 形态 | 数量 | 作用 |
|---|---|---:|---|
| 世界NPC | `WorldNpcNNNNN`（`_1` 后缀=别名/职业名） | 16,313 | 主城/野外常驻居民、功能NPC；带台词（任务/闲聊） |
| 城市/剧情功能NPC | `npc_*`、拼音缩写 `ly/jl/cd/sz/yyzNNN`（洛阳/金陵/成都/苏州…） | 6,912 | 城市商贩、剧情角色 |
| 场景怪物 | `mon<m><sceneNNNN>`（f/rm/m/c/rc/v/rf/_tm 等）、`khdnpc` | 5,339 | 野外/副本怪物命名池；`monrf` 带精英 `_1` 变体 |
| Boss/掉落宝箱 | `boss_*`、`BossBoxNNNN` | 4,937 | 副本/世界 Boss 与其掉落宝箱 |
| 副本克隆NPC | `NpcCloneNNNN`/`npcclone*`/`npcclonetl`(爬塔) | 4,239 | 副本内怪物/NPC（按 clone 编号归组） |
| 新世界NPC | `newworld_*`/`newmp_*` | 3,546 | 西域等新地图势力 NPC（峨眉势力/武当势力…） |
| 功能克隆NPC | `FunnpcCloneNNNN` | 2,728 | 副本功能交互（杨小磊等对话 NPC；"巴豆酒/破攻酒"=酒类增益交互物） |
| 采集点/采集物 | `Gather_*`、`cj_NNNNNN` | 2,505 | 采集点（毒蝎子/毒蛇…任务采集与生活材料）；`cj_` 编号有独立图鉴 |
| 事件触发点 | `EventNpc*/Event*` | 1,614 | 场景跳转/剧情触发（向阳坡、冲古寺、"进入地宫"） |
| 帮会建筑/帮会NPC | `GuildBuildingNPC*`、`GB_*` | 2,281 | 帮会建筑（练功台："帮众修炼内功…阵型变化对内功提升非常有效"） |
| 护送/镖车NPC | `ConvoyNNNNN` | 1,327 | 押镖护送对象（严夫人、马） |
| 雇佣商店NPC | `HireShopNpc*`/`hireshop` | 670+1,606 | 雇佣/商店服务 |
| 农田NPC | `FarmlandNpc*` | 400 | 农田种植交互 |
| 家园物件 | `home_*` | 335 | 家园交互物（团圆宴等） |
| 召唤物 | `SummonNNNNN` | 350 | 玩家/机关召唤单位（罗汉堂棍僧、十八铜人阵） |
| 奇遇NPC | `qy_*` | 396 | 奇遇事件角色（窃书贼、白暮云） |
| AI功能NPC | `AIFuncNpcNNNNN` | 70 | 场景氛围 NPC（小胖和尚、水牛、更夫、小鸡） |
| 箱子/宝箱NPC | `BoxNpc*`、`boxtg` | 1,026 | 可开启箱子（"秋夫人的箱子"） |
| 建筑克隆 | `MacClone*` | 571 | 副本建筑（山寨牢房） |
| 事件/猎杀目标 | `hunter_*` | 20 | 猎杀任务目标（人形傀儡、独目异蟒） |
| 其余长尾 | 活动专属前缀 | ~25,789 | 各期活动/玩法专属（wssz 无双山庄、tianshan 天山、khd 快活岛…） |

## 采集物图鉴（gather_codex.jsonl.gz）

`cj_NNNNNN` 采集物带四项档案，是生活技能原料知识库：

| 字段 | 含义 | 示例（cj_020001 铜矿） |
|---|---|---|
| story | 背景与功效 | "青铜主要原料…锻造兵器" |
| usepart | 成因/使用部位 | "岩层堆叠积压而成" |
| planed | 用途 | "炼铜、铸造兵器" |
| usetype | 出没地 | "多见高山裸地、贫瘠之地" |

## 关键机制线索

- **`_1` 后缀双命名**：`WorldNpc00001_1=樵夫` 同一 NPC 显示别名/职业，服务端下发的名字 key 需带后缀区分。
- **事件点即传送/触发器**：`EventNpc_*` 值常为地名或动作（"进入地宫""继续前往""赶回秦王府"），是任务链路的节点。
- **酒类交互物**：`FunnpcClone015077 巴豆酒/015078 破攻酒`——副本中以 NPC 形态存在的消耗型增益。
- **怪物命名池与场景绑定**：`mon<类><场景4位><编号>` 结构（如 monrc013911=慕容世家门人），可直接从 key 反解出生场景。

检索示例：

```bash
# 查带台词的世界NPC
python -c "
import gzip, json
for l in gzip.open('docs/text-extract/data/npcs.jsonl.gz','rt',encoding='utf-8'):
    r = json.loads(l)
    if r['kind']=='世界NPC' and r.get('talk_sample'):
        print(r['id'], r['name'], r['talk_sample'][0][:30]); break
"
```
