# NPC 官方巡逻路径（mappath 路线巡逻）

取代原 Go 复刻的占位巡逻（前 3 个非战斗 NPC 每 5s X±1.5 摆动），
按官方 `creator/mappath` 路线数据驱动 NPC 沿路点往返行走。对应
`bug.txt` V2 "添加NPC官方巡逻路径"。

## 数据链（9yin_server1 数据已验证）

```
creator XML <item PathId="AI_sz_001">        ← 实例级绑定（优先）
npcconfig 模板列 PathID                      ← 模板级缺省
        ↓
share/creator/mappath/<场景短名>_npc.ini/_mst.ini/...   （GBK）
        [AI_sz_001]
        r=59,458.7,0.629,637.8,-10,0,stand,nil,walk,nil,1,
        ↓
PatrolRoutes.bind：就近吸附 ≤35m + 按速度属性构建 PatrolSchedule
        ↓
SceneLifecycle 时间表广播（0x21 ServerAllDest）
```

场景短名 = creator 目录名去掉首个 `_` 后缀（`city02_suzhou` → `city02`），
加载该前缀的全部 `.ini`（stem 含 `offline` 的排除）。列序已用苏州 NPC
sz051（公振鹏）交叉验证：出生点 (459.139, 0.630, 637.788) ↔ 路线首点
(458.7, 0.629, 637.8)。

### 三种行格式（对齐尾部 `stand,nil,walk/run,nil,1` 固定列）

| 格式 | 样例 | x,y,z | 停留列 |
|---|---|---|---|
| 前导空列（主流，15.5 万行） | `r=,900.36,17.48,117.49,0,5000,…` | field 1..3 | field 5 |
| 整数导航点 id 前缀（约 1 万行） | `r=59,458.7,0.629,637.8,-10,60000,…` | field 1..3（**跳过 id**） | field 5 |
| 无 id 列（少量） | `r=318.19,3.79,632.01,-10,4000,…` | field 0..2 | field 4 |

> 教训：旧 Java 备份（bak/PatrolPaths.java）把格式 2 的 id 列当 x 读，
> 坐标整体错位——这正是"偶尔上天入地/穿墙"的根因。`coordinateBase()`
> 按"首列含小数点=坐标，否则为 id/空列"区分。

各列语义：
- **朝向列**：`0`=未设、`-10`=哨兵 → 两者按行进方向 `atan2(dx,dz)` 计算；
  `0<|o|≤2π` 才采用路点朝向
- **停留列**（毫秒）：到达该路点后的驻留，实测值 0/2000/4000/5000/15000/
  20000/30000/**60000**，钳 0..120s —— 路点停留由数据驱动（哨卫换岗等）
- **walk/run 标记**（field 8 固定列）：该腿到达点为 run 则用 RunSpeed
- 旧实现的 28m 跳腿上限已移除（正确解析后长腿是合法数据，走全程，
  行程钳 300..90000ms）；保留高度差 |dy|>6m 路点剔除（构建时一次完成）

## 时间表（PatrolSchedule）与多连接一致性

每条绑定在目录加载时展开成绝对时间线：段序 w0→…→w(n-1)→…→w0，
每段 = 行程(距离/速度) + 到达停留；周期 = 全部行程 + 停留。

- **共同时钟**：epoch = 绑定时刻（场景目录缓存加载），所有连接共享
  同一份 schedule + epoch → 同场景不同玩家看到**一致的 NPC 位置**
- **重进续走**：视口重进（120m 出入）时 `registerNpc` 按
  `positionAt(now)` 采样当前位置（行程中线性插值），不再回到路线起点
- **绝对时间步进**：每步按时间表时刻推算（剩余时长 = 段末 - now），
  调度抖动不累积漂移；SceneLifecycle 时钟可注入（测试确定性驱动）
- **速度同步**：行程 = 距离 / 该 NPC WalkSpeed（walk 腿）或 RunSpeed
  （run 腿），与 0x0D 下发属性同值（客户端动画据此播放），缺失回退
  1.25/4.0，钳 0.5..15 m/s —— 速度不同步会造成到点 0x1F 橡皮筋

## 绑定与步进策略

- **绑定**（`PatrolRoutes.bind`）：路线 ≥2 点（<0.5m 去重）且出生点到
  最近路点 ≤35m → 就近吸附；速度属性构建时间表（不可用则站立）。
  `attacknpc`/`bossnpc` 不绑定（敌对巡逻与战斗互斥，留待敌对 AI 期）
- **广播**：1s 周期 `0x21 ServerAllDest` 批量帧（`op | u16 count |
  N×44B`：`id | owner | destXYZ | orient | speed | 0 | 0 | 0 | flags`）。
  行进中记录 speed=该腿速度、flags=1（驱动客户端走动画），停留中
  speed=0、flags=0（到达）；每帧 ≤8 条分帧，首拍 5s（避开物化 0x1F
  重放窗口 250..5000ms）；行进实体的服务端权威坐标每拍同步为时间表
  插值位置（combat 快照/重进采样同源）
- **视口重进**：物化后新注册的巡逻 NPC 立即入巡（登记进广播集，
  ticker 幂等不重启）；时间表绝对时刻采样，重进续走不回起点

## 真实数据干跑（9yin_server1/resources/modern/share）

| 场景 | 路线 | 摆位 | 绑定巡逻 | 带停留路线 | 周期 min/avg/max |
|---|---|---|---|---|---|
| city02_suzhou 苏州 | 314 | 7135 | 189 | 79 | 6.8s / 8.4min / 48min |
| born03_YanYuZhuang 出生 | 385 | 5149 | 176 | 21 | 0.6s / 2.7min / 16min |
| city05_chengdu 成都 | 370 | 5355 | 221 | 47 | 5.4s / 6.5min / 31min |

短周期（<5s）路线仅 born03 2 条，无帧风暴风险。

## 代码索引

- `world/npc/PatrolRoute.java` 路点/路线（dwellMs 朝向哨兵 walk/run）
- `world/npc/PatrolRoutes.java` 加载（三格式/offline 排除/去重）+ 绑定
  吸附 + 时间表构建（速度属性/共享 epoch）
- `world/npc/PatrolSchedule.java` 绝对时间线（行程+停留）、positionAt/
  actionAt 采样、dy 过滤、行程钳制
- `world/scene/SceneLifecycle.java` 0x21 广播 ticker（注入时钟）、注册
  采样（重进续走）、`startPatrol/ensurePatrolBroadcastLocked/
  patrolBroadcastTick`
- 测试：`world/npc/PatrolRoutesTest.java`（格式回归/停留解析/吸附/速度钳
  制）、`world/npc/PatrolScheduleTest.java`（时间线数学/回绕/run 腿/过滤/
  motionAt 官方语义）、`game/scene/SceneLifecycleTest.java`（广播拍/停留/
  弹回/重进续走）

## 0x20 → 0x21 帧型更替（2026-11-03 真机取证）

- **症状**：04611f4 时间表版部署后服务端 0x20 帧持续下发（22:56 抓包
  230 帧、53 NPC 目标点正常往返），但真机 NPC 全部站立
- **取证**：官方真机抓包（`9yin_server/logs/capture-official` 与
  `capture-fxgame-live`，两服一致）**实体移动全部走 `0x21 movement`
  （每秒一拍批量），0x20 一帧都没有**；Go 参考服的 45B `0x20
  ServerMoving` 是未经真机验证的帧型（Go 源码注释自称 proven 但 Go 服
  日志中无任何巡逻输出）
- **0x21 布局**（44B/记录，47/91/179/223B 四种实测帧长 = 3+44N 钉死）：
  `id u32 | owner u32 | destX/Y/Z f32 | orient f32 | speed f32 |
  0 | 0 | 0 | flags u32`；玩家样本 slot5 出现 -8π..2π（转向类）、
  slot6=23、slot7=9.8（重力），NPC 样本三槽全 0 —— 未证槽位一律填 0
- **官方节拍语义**（移动中 NPC 0x000151 三点往返样本）：移动开始帧
  spd=WalkSpeed(1.25) flags=1；到达帧 spd=0 flags=0；每秒全量重播
  （含停留中实体）；NPC 移动**不配 0x1F 确认**（0x1F 仅玩家自己 15 帧）
- **验证**：`tools/smoke/patrol_probe.py` 登录物化后录 20s——city05
  15 拍 × 4 帧 = 60 帧 0x21、465 记录、31 NPC、速度=各家 WalkSpeed、
  flags 移动/到达分布正常

## 已知限制

- `moveNpc`（0x20+到点 0x1F）保留给非巡逻显式移动，但同样是未验证
  帧型，勿用于巡逻；战斗 AI 若需移动应一并切换 0x21
- 0x21 未证槽位（slot5/6/7 与 flags 高位）全填 0；玩家抓包出现的
  slot5 转向值/骑乘值未跟进（巡逻 NPC 无此需求）
- 服务器重启后 epoch 重置，NPC 在路线上重新"随机"分布（各连接仍一致）
- 停留中的朝向沿用到达段行进方向（数据无站立朝向列）
