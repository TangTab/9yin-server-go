# 技能服务端 XML 构建链与 msg211 受理链定版（2026-11-23）

> 数据源：`E:\Game\9yin\9yin-server-go2\资料\初始账号登陆-完整引导-女.pcapng`
> （8032 线，197~398s 共 41 次施放：普攻 `zs_normal_00`、长拳
> `CS_jh_cqgf01/04/06`、飞镖 `hw_normal_fb`）。设计态表：`go2 Res/share/skill`。
> 结论先行：
> 1. **技能配置按黑金模型离线构建为服务端 XML**（`tools/skillgen` →
>    `testdata/ini/Skill`），服务端只读产物（`internal/skilldata`）。
> 2. **msg211 受理链已接入**（`internal/fssoloworld/skill*.go`）：普通技 /
>    锁定技直收 / 冲锋暂存→c61 授予→m212 落地 / 超距拒绝 / 冷却·MP 闸门，
>    帧序按抓包逐帧定版。
> 3. **Lua 不是技能主链的一环**：17,855 个技能里挂 Lua 的 177 个全是特殊
>    技能，普通技能全走「内核流程 + 表」。
> 本阶段不含命中/伤害结算（见 §七）。

## 一、为什么不能直接搬 2433 线的实现

2433 分支 `cmd/protocol-probe` 有约 500KB 战斗 Go 码，但与官服抓包**不是同一
方言**：普通施法它发整块 0x10 属性面板 + Unix 纪元整行 CD、用 rec446
active_skill；官服是 `CurSkillID → CurSkillTarget → 0x11 空行+0x13 逐格 →
0x1F → m337 → m20`，另有 `m24 57/80` 与收招 `m707`。本仓走官服逐字节复刻
路线，故只借其帧布局与表字段知识，流程按抓包新写。其 Lua 是自家
`modulevm` 框架（`M.compile/M.on_hit`），也不是官服 `nx_*` 宿主。

## 二、黑金模型 → 九阴产物

官服运行时本就读 XML：FsGameLogic `SkillModule::LoadResource @0x387310`
是 `sprintf("ini/skill/skill_new_%d.xml", GetMemberId())` +
`ini/Skill/Skill_Normal_Attack.xml`；`share/skill/*.ini` 是设计态表。

| 黑金样本（`ini/Skill/heijin_backup/`） | 九阴产物（`testdata/ini/Skill/`） |
|---|---|
| `skill_new.xml.xml` 技能对象行 | `Skill_New/<逻辑类>.xml`（`SkillNormal` 14,843 / `SkillLock` 3,014 行；按 `script` 逻辑类分档＝服务端对象类） |
| `Skill_Config.xml` 引用注册表 | 不需要——九阴包按数值 ID 列引用（`Consume=1440` → `Skill_Consume` 行 1440） |
| `skill_static.xml` 静态包行 | `skill_static.xml`（17,923 行，ID=StaticData） |
| `Skill_Main_*.xml` 深层包 | `Skill_VarProp/<逻辑类>.xml`（46,190 / 13,624 行）+ `Skill_Consume`(37,726) / `Skill_TargetShape`(183) / `Skill_HitShape`(313) / `Skill_Flow`(45,684) / `Skill_LockData`(3,028) / `Skill_UseLimit`(543) / `Skill_RushRange` |

- 共 11 表 183,070 行 → 12 个文件约 30MB（NPC 产物先例 126MB）。
- 再生：`go run ./tools/skillgen -in <Res>/share/skill -rule <Res>/share/rule -verify`
  （`-verify` 用 `config.LoadObjectFile` 回读校验行数、ID 序、首行属性）。
- 黑金原档让位：首跑把 `<out>` 根下 7 个黑金样本整体移入 `heijin_backup/`
  （与 `npcgen` 同款）；`internal/fsgame` 的 `readResource` 规范位缺席时
  回退该目录（它的技能/buff 夹具仍是黑金真档）。
- `skill_static.xml` 沿用已跟踪路径的原名（小写）：换大小写命名会被 git
  当作该路径的内容修改，日后区分大小写的系统上检出会打不开。

### 源表脏数据处理（全部入审计报告 `artifacts/local/skillgen-report.json`）

| 现象 | 处理 |
|---|---|
| `[-1]` 列序头 / `[0]` 列宽头（装载器元数据） | 丢弃 |
| `skill_new.ini`（68178/69774）与 `skill_normal_varprop.ini`（223220/227866）残留 SVN 合并冲突标记 `<<<<<<< .mine` / `>>>>>>> .r759612`（单侧、无 `=======`，内含 233/426 节） | 标记行跳过、其间数据保留 |
| 非法 XML 属性名的键（3 个） | 丢弃 |
| 节内重复键（`skill_use_limit` 每行两个 `r=`，共 435 处） | 输出 `r` / `r_2` |
| 审定改名 | `UseLimit → Skill_Use_Limit`（`struct/skill/skillbase.xml` 属性名） |
| 审定丢弃 | 静态包 `Photo`（客户端图标路径） |

### 属性词汇

技能源表是**服务端**设计态表，列名即官服运行时词汇——官服 Lua 按源表列名取
技能对象属性（`lua/buf/buf_jn_lmsj_03.lua:31`
`nx_query_property(cur_skill, "AConsumeMP")`）。struct schema 里的
`AConsumeCurMP` 等是黑金词汇，**不得据其改名**。工具按三处真档给出每列出处
（只入报告、不改输出）：direct 21（struct skillbase+includes）/ heijin 30 /
define 4（`skill_static_data_define.lua`）/ native 165 / rename 1 / dropped 1。

## 三、装载层（internal/skilldata）

产物 30MB/18 万行，整表解成 map（`config.LoadObjectFile` 口径）要占数百 MB。
做法：每个文件首次用到时读入，一遍扫描 `<Property` 只取 ID 建「ID → 元素
偏移」索引（`bytes.Index`，不拷贝缓冲）；行属性取用时才解析；联查结果按
(技能, 等级) 缓存。实测真实产物首次装载 + 四层联查 **53ms**。

`Definition` = 技能对象行 → 静态包行（`Min/MaxVarPropNo` 选逐级行，等级不
符回落首行）→ 逐级可变属性行 → Consume / TargetShape / HitShape / Flow /
LockData 包。`Script` 非 `SkillNormal/SkillLock`（内功、轻功…）返回
`ErrUnsupported`。数据根下无产物时回退仓库 `testdata`。

## 四、冷却表模型（记录 109 cooldown_rec）

行 = `[类别, 起, 止, 队]`，起止为 1601(FILETIME) 纪元毫秒。用
`tools/pcapskill/cdsim.py` 把整段抓包的表逐次复原后得到：

| 行 | 类别 | 时长 | 写法 | 拦截 |
|---|---|---|---|---|
| 个人行 | 静态包 `CoolDownCategory` | 逐级 `CoolDownTime` | 每次受理追加新行 | 总是 |
| 窗口行 | `1` | 恒 1000ms | 首放插入，之后只改写起止两格 | （出招锁另计） |
| 公共行 `CoolEffectType=1`（5414 行） | `50000` / 队=`CoolDownTeam` | `AddCoolDownTime` | 每次受理都写；未过期原地改写，过期删后新插 | **仅换套路** |
| 公共行 `CoolEffectType=2`（249 行，普攻族） | `=CoolDownTeam` / 队 `0xFFFF` | `AddCoolDownTime` | 每次受理追加新行 | 总是 |

**套路规则**（用户定案 + 抓包 384~388s 实证）：同一套路内只按招式自身 CD，
公共 7 秒行只在换套路时拦截。普攻是另一套机制（组内 2 秒），不受套路公共
行约束（抓包 216.188 / 390.880：长拳的 7 秒行仍在，普攻照样受理）。

线上方言（抓包逐字节，`entergen.RecordInsertRowFrame` / `RecordGridFrame`）：
插入 = `0x11` 空行 + 4×`0x13` 逐格；改写 = 2×`0x13` 只写起止；清扫 = `0x12`。
**清扫对每次被处理的按键都发生**（受理 / 超距拒绝 / 冷却中静默三者同款——
397.17s 那次单发删除就是冷却中被拒的按键留下的），只清本技能类别（个人/
公共/共享）的过期行、每次至多 2 行。官服长会话里刷新公共行用的缓存行号会
失效（381s 时表里已积三行 50000、写到了别的行上），本实现始终按类别定位
真行，另加 16 行水位保险。

## 五、msg211 受理链（internal/fssoloworld）

`[211, 技能, x, y, z, 朝向, 0, 0, 0, "", [目标, 目标x, 目标y, 目标z]]`

| 形态 | 帧序（抓包时刻） |
|---|---|
| 普通技 | `[清扫] → CurSkillID → [CurSkillTarget] → 冷却行 → [MPRatio/MP] → 0x1F → m337 → m20`（197.729 / 209.389） |
| 锁定技直收带内（≤ castRange+0.5） | `[清扫] → CurSkillID → CurSkillTarget → m284 → 冷却行 → MPRatio/MP → 0x1F → LastWGHit → m24 目标 40 → m19 → CantMove/CantParry=1`（355.780） |
| 锁定冲锋技超直收带 | 暂存：`CurSkillID → CurSkillTarget → m25[PauseTime,"skill_pause"] → PauseSkill → m707[15]`（206.558） |
| 超距（非冲锋） | `[清扫] → m24 目标 80 [目标位, 朝向]`（205.644） |

细节：

- `CurSkillTarget` 值变才发（无目标技能在之前有目标时补发 None，198.879）。
- 朝向 = 指向服务端目标位的 `atan2(dx,dz)`；无目标技能取 m211 自报朝向。
- m20：普通技 `[自己, 动作, 目标|None, 0, 0.0]`，动作取流程包 `HitAction`
  列表（普攻连招段按技能各自轮转）。
- **闸门**（顺序即官服语义）：出招进行中再按 → `m707[9033]`；冷却中 →
  **零应答**（客户端按冷却表自行拦键）；MP 不足 → `m707[9021]`；需目标而
  无目标 → `m707[9030]`。结果码 = 9000 + `skill_error_define.lua`
  `USESKILL_RESULT_*`。
- 需目标的技能在 m211 不带目标时回落最近一次 msg431 选靶；自身范围技不回落。
- 收招只清 `CurSkillID`（`CurSkillTarget` 保持）；锁定位先清。出招锁：锁定
  技取 `Lock.LockSrcTime`，其余取流程包 `LockBuffTime`，缺省 500ms（普攻实测
  473ms）。
- MP：上限 = 基础值 + 首本内功加成（登录面板同口径）；`MPRatio` 截断
  （52/55 → 94）。回复 = 登录面板已下发的 `MPUpSpeed = ceil(上限/10)` 每 2 秒
  一档（55 上限 → 3/秒，抓包 386.5/391.6/396.6s 三次 +3），变化即推 MP 帧。

### 冲锋链（CanRush=1）

```
暂存（冷却/MP 此刻不结）
 → 授予：客户端接近上报（m15 行走 / m220）进入冲锋带 rush_range.ini
   GRushRange+Add（6.5）：
   m24[自己,61,落点 xyz,档 1,LockGRushAction] → CantMove=1 → CantParry=1 →
   冷却行 → MPRatio/MP → [QingGongPoint]
   落点 = 目标身前 Lock.BindDistance（缺省 1.5m）、高度 = 目标 y + 1.0
 → 落地：客户端 m212（锚 = 自报位）：
   CantMove/CantParry=0 → 0x1F → CurBreathState=0 → PauseSkill="" → m26 →
   0x1F → m337 → [LastWGAttackNpc] → LastWGHit=100 → m24 目标 40 → m19 →
   CantMove/CantParry=1 → 0x1F → 按 LockSrcTime 收招
```

m19（整帧 144B，23 值）：`[攻击者, 目标, 技能, LockSrcTime, LockTargetTime,
打击点1 xyz, 朝向, LockSrcDist, 打击点2 xyz, 常量 2.6166286,
LockTargetDist, 0, 0.0, 1, 1, 0.0, 目标位 xyz]`；打击点 = 出招位 / 目标位
各沿攻击方向前进对应位移（抓包两样本逐位吻合）。

兜底：授予后 1.5s 未收到 m212 → 在玩家当前上报位就地释放；暂存窗
（`PauseTime`，30000）到期 → 三清 `CurSkillID="" → m26 → PauseSkill=""`；
暂存中被新按键覆盖 → 同样三清后按新技能受理。选靶处理（msg431）的 5 秒
`m26` 看门狗在暂存中不发（暂停归暂存链所有）。

## 六、Lua 的边界（查证数据，tools/luaaudit）

| 范围 | 总数 | 挂 Lua | 分布 |
|---|---|---|---|
| 技能（`skill_static.LuaScript`） | 17,855 | **177（1.0%）** | 副本机制技 94、坐骑/骑乘 27、测试/模板库 39、武学特效 25 等 |
| buff | 28,532 | 314（1.1%） | 坐骑 60、副本机制约 130、武学 9 等 |
| 内功 | 398 | **0** | `neigong_static.ini` 只有 Photo/VarPropNo/BelongType 四列；1,093 个 `buf_ng_*` 同样零 Lua |

- 抓包里的 5 个技能 `LuaScript` 全为空——整条链是纯内核流程。
- 内功在 Lua 里只有**经验**部分（`monster_awards.neigong_awards` /
  `faculty_convert` / `sitcross.practice_neigong`），效果是 `buf_ng_*` +
  `neigong_varprop.ini` 数据驱动。
- buff 钩子：`buffer_active / buffer_deactive / buffer_timer / buffer_break`；
  另有 203 个被引用的脚本文件不在这份 `Res/lua` 里。
- 宿主规模：战斗相关 `nx_*` 约 50（技能）+ 23（buff）个；整个 `Res/lua`
  636 个 / 361 文件。
- 结论：Lua 宿主留给特殊技能 / buff / 内功经验的钩子，不是技能主链的前置。

## 七、本阶段未覆盖

- **命中/伤害结算**：需 NPC 战斗态（HP、仇恨、死亡）与 `damage_calculate` /
  `formula` 联查；`Definition` 已带 `DamageCalculate` / `EventProcess` 包 ID。
  抓包里的伤害簇（`m24 17 / m205 / FP·SP / rec114`）与锁定结果的闪避/格挡
  分支（`m24 7` + `m20 *_dodge`、`m707[14]`）待接。
- MP 回复只在站立每秒 tick 里做；打断（`skill_break`）、引导、空中档冲锋、
  `HeightDiff` 未接。
- 锁定技 1.5~3m 的「原地普通施放」变体（抓包 392.306）未区分，统一走直收簇。
- **部署同步**：`go2 Res/ini/Skill` 目前仍是黑金 7 文件。把产物同步过去：
  `go run ./tools/skillgen -out <Res>/ini/Skill -verify`（首跑会把黑金样本
  移入该目录的 `heijin_backup/`）。未同步时服务端回退仓库 `testdata`。

## 八、工具索引

| 工具 | 用途 |
|---|---|
| `tools/skillgen` | 设计态 ini → 服务端 XML（黑金模型），审计报告 + `-verify` |
| `tools/pcapskill` | 抓包按时间窗/方向/opcode/消息号逐帧 hex；`cddecode.py` 冷却表可读时间线，`cdsim.py` 冷却表逐次状态模拟 |
| `tools/pcapevents`（原 m54scan） | 抓包全事件时间线（`dumpall`）；`rushband.py` / `staged.py` / `m337check.py` 文本分析 |
| `tools/luaaudit` | `audit_skills.py` / `audit_buffs.py` Lua 覆盖面审计，`colunion.py` 源表列并集，`heijin_shape.py` 黑金样本形状 |

复跑：

```sh
go run ./tools/skillgen -in "E:/Game/9yin/9yin-server-go2/Res/share/skill" \
  -rule "E:/Game/9yin/9yin-server-go2/Res/share/rule" -verify
go run ./tools/pcapskill <女号.pcapng> 8032 197.72 197.74 "op11,op13"
go run ./tools/pcapskill <女号.pcapng> 8032 197.0 398.0 "op11,op12,op13,op10" \
  | python -I tools/pcapskill/cdsim.py
go test ./tools/skillgen ./internal/skilldata ./internal/entergen ./internal/fssoloworld
```
