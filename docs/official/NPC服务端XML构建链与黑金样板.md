# NPC 服务端 XML 构建链与黑金样板

2026-11-22 定案。NPC 数据链改为「离线构建服务端 XML → 服务端只读构建产物」，
NPC 相关服务端代码（装载/联查/帧/下发）整体重写为 `internal/npcfeed`，
帧配方参考 2433 线 `cmd/protocol-probe`（npcAddObject / officialNPC0x28Frame）。

## 一、黑金数据的构建方式（样板分析）

黑金原档：`go2 Res/ini/Npc/heijin_backup/NpcConfig/CommonNpc/common_npc{,_1,_2,_3,_fb,_pvp}.xml`
+ `NpcConfigAuto/member0_clonedoor_npc.xml`（黑金档在 npcgen 首跑时整体移入
heijin_backup 留存）。五点结论：

1. **三件分离**——NpcConfig 模板（视觉/交互面）、场景摆放 creator、NpcTemplate
   战斗模板（数值面）各自独立，运行时联查。
2. **FightID 引用解耦**——模板行不带战斗数值，`FightID` 引用战斗模板 ID
   （黑金 `FightID="ZS_PT_0010"` 实力包式；同表还有 `SkillPackID="SkillPack3_1"`、
   `WorldDropID="W_00001"`——技能/掉落同样引用式外包）。
3. **按玩法域分组**——黑金 CommonNpc 下 common_npc 主档 + `_1/_2/_3` 分档 +
   `_fb`（副本）+ `_pvp` + NpcConfigAuto（自动生成档）；九阴版对应
   `WorldNpc/CloneNpc/BattleNpc/LifeNpc` 四组（npcconfig 家族映射）。
4. **属性词汇以服务端 schema 为准**——`testdata/struct/npc/*.xml`
   （VisibleNpc 基类 + 子类 include）是权威词汇源；客户端列名经 npcgen
   审定映射表（direct/rename/tableRec/dropped，`tools/npcgen/mapping.go`）
   转服务端词汇，没把握的列宁可 dropped 不发明。
5. **摆放行保持 creator 源名**——`File/x/y/z/ay/id/RefreshTime/PathId/...`
   原样透传，不做词汇转换。

## 二、九阴版数据源与构建

构建器：`tools/npcgen`（本仓已有，产物已就位）。

| 输入 | 来源 |
| --- | --- |
| 模板+摆放 | `F:\Game\Online\9yin\9yin-server-go\resources\modern\share\`（客户端解压资源：npc/npcconfig TSV + creator/npc_creator XML） |
| 战斗模板 | `F:\Game\Online\9yin\9yin-server-go2\Res\bak\ini\Task\npctemplate.tab`（GBK TSV，1 行头，6272 行） |

产物（服务端 XML，`testdata/ini` 与 `go2 Res/ini` 两处同源已投）：

```
ini/Npc/NpcConfig/<组>/<表>.xml    NPC 模板（行带 FightID=PowerNo）
ini/Npc/NpcTemplate.xml            战斗模板（npctemplate.tab 逐行转换，6272 行）
ini/Scene/<场景>/npc_creator.xml   每场景摆放（406 场景）
```

再生成命令（2026-11-22 起两个输入源均可传参，缺省探测链自动命中本机
部署路径——零参数直接 `go run ./tools/npcgen` 亦可）：

```
go run ./tools/npcgen -in F:\Game\Online\9yin\9yin-server-go\resources\modern ^
  -npctemplate F:\Game\Online\9yin\9yin-server-go2\Res\bak\ini\Task ^
  -out testdata\ini\Npc -scenes-out testdata\ini\Scene
```

输入参数：

| flag | 语义 | 缺省探测链 |
| --- | --- | --- |
| `-in` | 客户端解压资源根（模板+摆放） | `NINEYIN_MODERN` > 仓库兄弟目录 > `F:\...\9yin-server-go\resources\modern` |
| `-npctemplate` | 战斗模板源：**npctemplate.tab 文件 / 所在目录 / `npctemplate*.tab` glob 三态**；多档合并（主档优先、同 ID 首档胜出、npctemplate2.tab 空壳 warning 跳过） | `-tasks` 目录 > `F:\...\9yin-server-go2\Res\bak\ini\Task` |
| `-tasks` | 任务域数据根（Task/ 子目录；缺席自动跳过任务域转换） | `NINEYIN_TASKS` > 仓库兄弟目录 |

最近一次构建基线：76 表 77394 行（唯一 ID 77093）/ 405 场景 268507 摆放
（孤儿 705）/ 战斗模板 6272 行 / verify 483 文件回读一致。

## 三、联查键实证（FightID == npctemplate ID）

客户端模板 `PowerNo` 列（实力包编号）→ npcgen rename 为服务端 `FightID` →
`NpcTemplate.xml` 行 ID。两样本互证：

- 鳄鱼（CloneNpc/AttackNpc.xml）`FightID="114" Level="70"` ↔ NpcTemplate
  `ID="114" Level="70"`（朦胧子行，等级吻合——战斗行名是设计期名，数值面
  以行 ID 为准）。
- monv010200 `PowerNo=34 Level=1` ↔ NpcTemplate `ID="34" Level=1`。

## 四、服务端链路（internal/npcfeed，全新实现）

```
load.go     三件装载（模板树/战斗表/场景摆放；<Property> 行方言，进程级缓存）
join.go     摆放→模板(FightID)→战斗 联查 + 闸门 + ident
resolve.go  分层属性合成（默认值→模板→战斗→摆放，按 0x09 官方表场景域表序）
frame.go    2433 帧配方：0x0D（63B 头 + OrderedProperties + 官方 16-prop 块）
            / 0x28 mode=0x46（v3 分形态：#0 普通 26 字段 / #1 带武器 23 字段）
```

- ident：`obj=0x11000000|(idx+2)`、`own=0x00660000|低16位`（2433 定案）。
- 闸门（2433 npcSpawnEligible 同款）：EventNpc 类、TaskSubId、SubSceneNo+
  战斗类不注册。
- 类别白名单：commonnpc/funcnpc/itemnpc/lightnpc/guildbuildingnpc/
  gathernpc/other（attacknpc 族与随机组不发——需 AI/战斗面）。
- 战斗层映射：MaxLife→MaxHP/HP、MaxMana→MaxMP、Level（战斗行权威）。
- 宝箱族（NpcType=161/BoxNpc*）与采集族（153/GatherClone*）不发
  CollideRadius，尾行交互半径 OpenRange=2.0 / GatherRange=3.0。
- 0x28 物化帧 = 2433 npc_detail_v3 分形态方言（npc.pcapng #0/#1 官方样本
  逐字节定版）：#0 普通 NPC ~26 字段（含 CursorShape/MPRatio/HasKarma/
  Character/KarmaLevel/OriginPara1-3 建档常量族）；#1 带武器 NPC ~23 字段
  （Fixed/WeaponName/FixedBeHitAct/SelfForceID，无 Photo/CursorShape/
  MPRatio/karma 族）；RotatePara 缺省 "0.0,1.000,0.0"；整数值按 0x09 表
  条目线型定宽；判别开关 `NINEYIN_NPC_RESOURCE_OVERRIDE`（全同模型实验）。

### 真机定案：MaxVisuals=AddObject 注册闸（2026-11-22）

0x0B 进场帧 idx1468 MaxVisuals 是**客户端角色渲染数据池容量**
（GameVisual::CreateRoleUserData 注册闸）。历史上恒发 230：

- 全量物化 born03 1674 条 0x28 → 帧全部到达，但池满拒建 **1236 条**
  （trace.log `CreateRoleUserData can't create any more` ×1236，仅 ~438
  建成）→ 症状「NPC 变少 / 只有角色周围显示且只一个」。
- 2433 线全城渲染成功的隐含前提：视口物化单次 ~134 条 < 230。
- 判别实验佐证：`NINEYIN_NPC_RESOURCE_OVERRIDE=obj\char\b_baixing01`
  全同模型后**全灭**——NPC 物化通道只认 `npc\` 族资源路径（obj\char 是
  玩家化身族）；客户端模型包 obj_npc（索引加密）/obj_newnpc（明文 1823
  模型）对 born03 Resource 表观命中率 71.4%（缺的多为宝箱/物件族，
  疑在加密包内实际存在）。

修复：`maxVisualsValue` 230 → **10000**（entergen/frames.go 0x0B 帧 +
internal/login/entry.go 两处同步）——born03 全量 1674、city05 全量 2509
均落池内。残留观察项：个别 NPC 呆立无动画（模型动画资源面，后续按
newnpc 库做 Resource 映射）。

fssoloworld 接线：`sceneNPCFrames(scene, origin)`（npc.go 薄壳）→
`entergen.Context.NPCFrames`（进场序：玩家 0x0D→记录行→NPC 物化×N）。

## 五、开关

| env | 缺省 | 语义 |
| --- | --- | --- |
| `NINEYIN_DATA_ROOT` | 探测 | 数据根（含 ini/Npc/NpcConfig；探测链 Res→testdata） |
| `NINEYIN_NPC_PUSH` | on | `=off` 一键关停 NPC 下发 |
| `NINEYIN_NPC_0X28` | on | `=off` 判别切裸 0x0D 大档 |
| `NINEYIN_NPC_RADIUS` | 0（全量） | 出生点平面半径过滤 |
| `NINEYIN_NPC_LIMIT` | 0（不截断） | 距离升序截断上限 |
| `NINEYIN_NPC_RESOURCE_OVERRIDE` | 空（真值） | 全 NPC 同模型判别实验（只认 `npc\` 族路径） |

诊断工具：`go run ./tools/npcprobe -root <数据根> -scene born03`
（摆放/联查/战斗命中率、类别分布、帧抽查 hexdump）。

真档基线（2026-11-22，go2 Res）：born03 摆放 1907→物化 1674（战斗命中
71.9%，帧均值 297B）；city05 摆放 3768→物化 2509（70.7%）。
