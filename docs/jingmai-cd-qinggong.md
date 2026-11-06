# 经脉修炼 + CD 下发修复 + 三段跳缓降（2026-11-02）

> 数据权威源：`9yin_server1/captures/local/127-0-0-1-52795_{c2s,s2c}.bin`（F 盘野猪端参考服
> 与真机客户端的完整会话，21,479 帧 S2C）。本文档的每一条 wire 结论均出自该抓包的直接解码。

## 一、冷却没有正常下发——根因与修复

### 根因

战斗一期（db884de）的 cooldown 行帧字节级对齐抓包（38B、team=u16、cols [3,4,4,2]），
但客户端 CoolLogicManager 仍不启动 CD。**真正的差异在 0x0A 记录表协商**：

| | 官方（F 参考） | 本服（修复前） |
|---|---|---|
| 0x0A 表 | **447 schema 整表**（10,449B） | 自协商 8 张表 |
| cooldown_rec 下标 | **109** | 5 |
| Task_Accepted | 29（**28 列**） | 2（7 列前缀） |
| ActiveQGSkillRec | 117（[7,1]） | 1（[7]） |
| shortcut_rec / Task_Completed | 93 / 31 | 4 / 3 |

客户端 C++ 原生消费者（CoolLogicManager 等）对 cooldown_rec 的绑定方式（按名/按官方
下标 109）无法从 lua 侧证伪——自协商下标是唯一的未验证假设。**修复：整表切换为官方
447 schema 方言逐字节透传**（`resources/nineyin/protocol/official_record_table.bin`，
由抓包 0x0A 帧提取；与 `PropertyTable` 的 0x09 官方属性表同一策略），行帧索引全部
换官方下标（`RecordMessages` 常量）。行帧必须满宽（官方 Task_Accepted 行 144B/28 列
实证）：`recordAddCells` 对缺失尾列按线型自动补零/空串。

### 同批修复

1. **GCD 行 team 回退**：无 CoolDownTeam 的技能原回退 0xFFFF——客户端
   `gui.lua SetCommonTeamCoolRange(50000,50001,51000)` 只认 50001-51000（50001-50100
   无特效段），0xFFFF 落在段外。改回退 **50001**（`SkillCastHandler.GCD_TEAM_FALLBACK`）。
   抓包实证 GCD 行 team 恒为 skill_static.CoolDownTeam（50260）。
2. **msg294 v2 纪元**：官方 v2 = **1601-01-01 UTC 纪元毫秒**（Windows FILETIME 基准
   ÷10⁴）。抓包首帧 v2=13434237233402 恰为 v3=1789763633 的 unix ms + 11644473600000。
   此前误发 unix 毫秒（口径差 1.16e13ms），已对齐（`LoginFlow.timeSyncFrame`）。

### 官方记录下标速查（docs/official-record-schemas.txt 全表）

Task_Accepted=29 / Task_Completed=31 / shortcut_rec=93 / **cooldown_rec=109** /
fight_relation_rec=114 / QingGongRec=116 / ActiveQGSkillRec=117 /
**active_jingmai_rec=162** / **PepareJingMaiRec=204**（8 串列）/ active_skill_rec=446。

## 二、经脉修炼（修炼经脉/修炼周天/修炼进度/经脉总等级）

### 客户端显示契约（form_main_player.set_jingmaiinfo_tips + 抓包实证）

| 显示项 | 数据源 | 通道 |
|---|---|---|
| 修炼经脉 | 玩家属性 `CurJingMai`（串，idx 849） | 0x10 |
| 修炼周天 | **View45 行** `Level`（byte，idx 1465） | 0x18/0x10 |
| 修炼进度 | View45 行 `CurFillValue`(764)/`TotalFillValue`(765) | 0x18/0x10 |
| 经脉总等级 | 玩家属性 `JingMaiTotalLevel`（idx 819） | 0x10 |
| 今日真气 | `ZQPLValue`(855)/`ZQActlValue`(856，官方拼写)/`ZhenQiDayValue`(852)/`ZQUnUsedValue`(857) | 0x10 |

View45（VIEWPORT_JINGMAI）行 = 6 属性：ConfigID(1440)/ItemType(1889)/StaticData(2237)/
Level(1465,byte)/MaxLevel(2083)/TotalFillValue(765)。已激活经脉 = active_jingmai_rec
（162）行，抓包行值 'jm_dantian' 与 CurJingMai 同源。

### 配置来源（resources/modern/share）

- `skill/jingmai/jingmai.ini`：ConfigID → StaticData/ItemType=1003。
- `skill/jingmai/jingmai_static.ini`：StaticData → Min/MaxVarPropNo，**区间宽 = MaxLevel**
  （丹田 [141001,141180]→180、正经→216、跷脉→225）。
- 每周天所需真气 = **平曲线 750**（F 端行为；wuxuelevelinfo.ini 的正经曲线 20100+/级
  未被 F 采用，速度列 43/分钟被采用为修炼速率）。
- `skill/jingmai/zhenqi.ini`：战斗真气 damage_ratio=0.1（每 10 点伤害 1 真气）。

### 实现（com.nineyin.server.game.avatar.jingmai）

- `MeridianCatalog`：上述三表目录（ResourceCache 缓存，测试可注入 share 根）。
- `PlayerMeridian`：14 条初始经脉（丹田+十二正经+阴阳跷·隐，抓包同序）；
  整分钟结算 43 真气（advanceFaculty 同款时钟模型）；战斗伤害即时折算真气；
  满 750 → 周天+1（溢出结转、MaxLevel 封顶）、JingMaiTotalLevel=ΣLevel；
  ZQPLValue 按 UTC 日切清零。
- **每日真气硬闸（GM 面板「经脉」分类 → `set_zhenqi_cap`）**：是否开启 +
  每日阈值（1..10⁷）。开启后当日 ZQPLValue 达阈值即拦截入账（被动/战斗同闸，
  超额笔按剩余额度截断、零帧静默），ZhenQiDayValue 属性随开关显示阈值/回落
  官方缺省 500；角色级设置随 meridians.json/role_meridians 持久化。默认关闭。
- `JingMaiMessages`：View45 容器/行帧、进度/升级单帧、玩家经脉属性帧、162 记录行。
- `JingMaiCustomHandler`：**C2S 150** sub=1 开始/停止修炼（`[150, 1, ConfigID|""]`，
  未学静默拒）、sub=2 真气刷新（回属性帧）。首次开始修炼补 active_jingmai_rec 行。
- `MeridianStore`：`data/meridians.json` 主档 + MySQL `role_meridians` 快照双源
  （FacultyStore 同款）。断线/变化落盘。建表走权威存档增量脚本
  `admin/sql/0004_add_role_meridians_20261102.sql.gz`（deploy.bat 第[3]步按序重放；
  基线 role_faculty 同形态：snapshot json + FK roles ON DELETE CASCADE），
  `attachDatabase` 另有同款 `CREATE TABLE IF NOT EXISTS` 运行期兜底。

### 下发时机

- onChooseRole：读档恢复（无档按 14 条初始落档）。
- onClientReady（每连接一次）：View45 容器+行 → active_jingmai_rec（若在修）→
  玩家经脉属性帧。
- 1s ticker（facultyTicker 同拍）：整分钟结算变化时 View45 行 CurFillValue 单帧
  （升级加发 Level/TotalFillValue 行帧）+ 玩家属性帧 + 落盘。
- 出招命中：伤害×0.1 真气即时入账（帧随命中序列）。

## 三、三段跳后按住——缓慢降落 buff（qinggong_9 轻身术）

### 抓包解码（C2S 216 / S2C msg169）

```
C2S 216: [216, 'qinggong_9', 1]   按住跳跃键开始缓降
C2S 216: [216, 'qinggong_9', 0]   松开结束（qinggong_8 等冲刺类恒 flag=0）
S2C 169: [169, 0, 'buf_qg_light', obj(施), obj(承), 宽名, 宽名, 到期ms(i64), 2048(i32)]
         v1=0 挂 / v1=1 摘；v7=到期 unix 毫秒；v8=buff_static[218].Category=2048
```

缓降物理（GravityAdd 负值重力）与三段跳存在 buff（buf_DHadd_3rd，DropHeightAdd+4）
全在客户端引擎本地；服务端职责 = buff 图标链 + 轻功点权威。

### 实现（world/fight/QingGongHandler + PlayerActor）

- 216 flag=1：`FightMessages.timedBuff(0, buf_qg_light, 2048, now+600s)`（九值形态，
  buff_new LifeTime=600000）+ BufferInfoN 空闲槽帧（槽 4/11 等保留位自动避开）。
- 216 flag=0：msg169(v1=1) + 空槽帧（游离 release 也回帧，客户端状态重同步）。
- 1s ticker：每秒扣 10 轻功值（buff_varprop[23001] L1 CCQGP=-10）+ 0x10 单帧推送；
  耗尽/租约到期强摘。扣点走 `Actor.drainQingGong`（不触发跳跃的 <50 自动回满）。
- 其它轻功（4/5/8 跳跃冲刺）：本地物理，零帧放行。

## 四、遗留（后续批次）

1. `custom_jingmai_msg` 下行的数字 ID 未知（引擎注册表在 C++ 侧，客户端 lua 无映射表；
   抓包无该帧样本）——它只承载真气增长动画(sub2)/满级弹窗(sub3)，装饰性，当前缺省不发。
2. C2S 172（一步经脉 PepareJingMaiRec 预激活方案）未实现：0x13 单格写串格式无抓包
   样本，暂不受理。
3. 经脉等级的属性加成（jingmai_varprop → PropModifyPackRec → 面板）未接入 effectStats
   叠加链；激活经脉目前只有展示语义。
4. JsonRoleRepositoryTest.scenarioMatchesGoByteForByte 在干净 HEAD 上也失败
   （6821 vs 6561 字节，golden 与生成器漂移，早于本次改动），待单独排查。
5. GmRemote 白名单此前漏挂四期面板动作（set_power_level/set_alignment/set_vitals/
   wear_title/cancel_title——LoginFlow 已实现但入口即拒），本次已顺手补齐；
   面板的 popup_gm_panel / bot_* 动作服务端本就未实现（积压按钮，未动）。

## 五、测试

`MeridianAndQingGongTest`（11 用例）：目录解析（MaxLevel 180/216、平曲线 750、速度 43）/
状态机（整分钟结算、战斗折算、升级结转、总等级、日切）/ View45 帧布局（6 属性索引序
逐字节）/ 150 修炼设置与停修 / 216 缓降挂-扣-摘-耗尽强摘。
`ProtocolGoldenTest`：官方 447 表 verbatim、Task_Accepted 144B 满宽、cooldown 38B 双行
（个人 team=0xFFFF / GCD team=CoolDownTeam）。golden 已再生
（`-Dnineyin.golden.regen=true`）。
