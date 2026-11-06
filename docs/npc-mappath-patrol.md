# NPC 官方巡逻路径（mappath 路线巡逻）

> 2026-11-06 Go 实现（本仓库 `cmd/protocol-probe`）。数据/协议真值源为
> `resources/modern/share/creator/mappath`（252 个 ini）与官方真机抓包。
> 对应 `bug.txt` V2 "添加NPC官方巡逻路径"。

取代原 Go 复刻的占位巡逻（前 3 个非战斗 NPC 每 5s X±1.5 摆动，
`-npc-patrol-count` flag 已随之移除），按官方 mappath 路线数据驱动 NPC
沿路点往返行走。

## 数据链（资源已验证）

```
creator XML <item PathId="AI_sz_001">        ← 实例级绑定（优先）
npcconfig 模板列 PathID                      ← 模板级缺省
        ↓
share/creator/mappath/<场景短名>_npc.ini/_mst.ini/...   （GBK）
        [AI_sz_001]
        r=59,458.7,0.629,637.8,-10,0,stand,nil,walk,nil,1,
        ↓
bindPatrolRoutes：就近吸附 ≤35m + 按速度属性构建 patrolSchedule
        ↓
sceneRun 中央 tick：0x21 ServerAllDest 每秒广播
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

> 教训：把格式 2 的 id 列当 x 读，
> 坐标整体错位——这正是"偶尔上天入地/穿墙"的根因。`patrolCoordinateBase`
> 按"首列含小数点=坐标，否则为 id/空列"区分。

各列语义：
- **朝向列**：`0`=未设、`-10`=哨兵 → 两者按行进方向 `atan2(dx,dz)` 计算；
  `0<|o|≤2π` 才采用路点朝向
- **停留列**（毫秒）：到达该路点后的驻留，实测值 0/2000/4000/5000/15000/
  20000/30000/**60000**，钳 0..120s —— 路点停留由数据驱动（哨卫换岗等）
- **walk/run 标记**（field 8 固定列）：该腿到达点为 run 则用 RunSpeed
  （注意 run 标记归属**到达点**：去程 run 腿跑、回程到达 w0 为 walk 腿走）
- 长腿是合法数据（走全程，行程钳 300..90000ms）；高度差 |dy|>6m 路点
  剔除（构建时一次完成）

## 时间表（patrolSchedule）与多连接一致性

每条绑定在目录加载时展开成绝对时间线：段序 w0→…→w(n-1)→…→w0，
每段 = 行程(距离/速度) + 到达停留；周期 = 全部行程 + 停留。

- **共同时钟**：epoch = 绑定时刻（场景目录缓存加载），场景运行时
  （sceneRun）重建后共享同一份 schedule + epoch → 同场景不同玩家看到
  **一致的 NPC 位置**，服务器重启前 NPC 位置跨场景重建续走
- **重进续走**：视口重进（120m 出入）时 `registerNPCLocked` 按
  `positionAt(now)` 采样当前位置（行程中线性插值），不再回到路线起点
- **绝对时间步进**：每拍按时间表时刻推算（`floorMod(now-epoch, cycle)`），
  调度抖动不累积漂移
- **速度同步**：行程 = 距离 / 该 NPC WalkSpeed（walk 腿）或 RunSpeed
  （run 腿），与 0x0D 下发属性同值（客户端动画据此播放），缺失回退
  1.25/4.0，钳 0.5..15 m/s —— 速度不同步会造成到点 0x1F 橡皮筋

## Go 实现（cmd/protocol-probe）

### 文件与职责

| 文件 | 内容 |
|---|---|
| `patrol.go` | 全部巡逻域：`loadPatrolRoutes`（三格式/offline 排除/<0.5m 去重/≥2 点成路线）、`buildPatrolSchedule`（绝对时间线/dy 过滤/行程钳制）、`bindPatrolRoutes`（PathId 绑定/≤35m 吸附/速度钳制）、`serverAllDest`（0x21 编码）、`patrolTick`（中央 tick 巡逻段）、`patchAddObjectTransform`（0x0D 载荷坐标重打） |
| `patrol_test.go` | 7 个测试：行格式解析、路线加载、时间表采样（插值/停留/run 腿/回绕/dy 过滤/行程钳）、绑定闸门、0x21 字节级编码、端到端 patrolTick、注册采样 |
| `scene_catalog_registry.go` | `catalogFor` 加载目录后 `bindPatrolRoutes`（epoch 进目录缓存，一次绑定永久共享） |
| `scene_run.go` | `tickOnce = combatTick + patrolTick`；`registerNPCLocked` 时间表采样 + `patrolIDs` 登记；`activateMember` 设 5s 首拍延迟；占位巡逻（startPatrol/patrolStepOnce 家族）已删除 |
| `scene_lifecycle.go` | `sceneEntity.schedule`（替换 patrolSign）；Adds 时 0x0D 载荷坐标重打为当前权威位置 |
| `main.go` | `npcSpawn` 增 route/routeStartIdx/schedule；`-npc-patrol-count` 移除；单目录回退模式（`-npc-all-chengdu=false`）同样绑定 |

### 实现的架构

- Go 并入共享 sceneRun 的 **每秒中央 tick**（战斗与巡逻各一段，"锁内结算、锁外写帧"同节奏）
- Go 按成员视口 `npcVisible` 过滤 + ≤8 条/帧分帧，重放窗口抑制按成员（`patrolReadyAt` =
  clientActivity+5s，避开 0x1F 重放 250..5000ms）
- 插值位置只写 ECS Transform 组件（`writeNpcTransform`），**不走**
  `setNPCTransformLocked` → 不抬升可视权威 Scene 的 revision，视口
  对账不因巡逻产生 0x1F 更新（与官方一致）；重进视口的 0x0D/0x1F
  采样自然续走，0x0D 落点由 `patchAddObjectTransform` 重打防闪现

### 互斥与战斗衔接

- `attacknpc`/`bossnpc`（`patrolCombatScriptClass`）bind 时即不绑定巡逻；
  非战斗 NPC 无 Brain 不参战——两套系统在数据层面天然无交集
- 战斗 AI 的追击移动仍是 0x20+0x1F（另一条通路，见"已知限制"）

## 真实数据干跑（resources/modern/share，Go 同包同数）

| 场景 | 路线 | 摆位 | 绑定巡逻 | 带停留路线 | 周期 min/avg/max |
|---|---|---|---|---|---|
| city02_suzhou 苏州 | 314 | 7135 | 189 | 79 | 6.8s / 8.4min / 48min |
| born03_YanYuZhuang 出生 | 385 | 5149 | 176 | 21 | 0.6s / 2.7min / 16min |
| city05_chengdu 成都 | 370 | 5322 | 221 | 47 | 5.4s / 6.5min / 31min |

city05 已在 Go 侧复测（`mappath table scene=city05 files=3 named_paths=370`
+ `patrol bind: walkers=221 stand=5101 missingPath=2 farOrShort=16`）。
短周期（<5s）路线仅 born03 2 条，无帧风暴风险。

## 0x20 → 0x21 帧型更替（2026-11-03 真机取证）

- **症状**：时间表版部署后服务端 0x20 帧持续下发（抓包 230 帧、53 NPC
  目标点正常往返），但真机 NPC 全部站立
- **取证**：官方真机抓包（`capture-official` 与 `capture-fxgame-live`
  两服一致）**实体移动全部走 `0x21 movement`（每秒一拍批量），0x20 一帧
  都没有**；Go 参考服的 45B `0x20 ServerMoving` 是未经真机验证的帧型
- **0x21 布局**（44B/记录，47/91/179/223B 四种实测帧长 = 3+44N 钉死）：
  `op u8 | u16 count |` 每条 `id u32 | owner u32 | destX/Y/Z f32 |
  orient f32 | speed f32 | 0 | 0 | 0 | flags u32`；玩家样本 slot5 出现
  -8π..2π（转向类）、slot6=23、slot7=9.8（重力），NPC 样本三槽全 0 ——
  未证槽位一律填 0
- **官方节拍语义**（移动中 NPC 0x000151 三点往返样本）：移动开始帧
  spd=WalkSpeed(1.25) flags=1；到达帧 spd=0 flags=0；每秒全量重播
  （含停留中实体）；NPC 移动**不配 0x1F 确认**（0x1F 仅玩家自己 15 帧）
- **验证**：登录物化后录 20s——city05 15 拍 × 4 帧 = 60 帧 0x21、465 记录、
  31 NPC、速度各家WalkSpeed、flags 移动/到达分布正常

## 代码索引

**Go（本仓库）**：`cmd/protocol-probe/patrol.go`（域实现）、
`patrol_test.go`（测试）、`scene_run.go`（tick/注册/首拍延迟）、
`scene_catalog_registry.go`（bind 挂点）、`scene_lifecycle.go`
（sceneEntity.schedule/载荷重打）。

## 验证

1. `go build ./...`；`go test ./...`（patrol 测试 7 项 + 全仓回归）
2. 启动后日志：`mappath table scene=<短名> files=N named_paths=N` 与
   `patrol bind: walkers=N stand=N missingPath=N farOrShort=N`
3. 客户端进城观察：带 PathId 的平民 NPC 沿路线往返、路点停留
   （哨卫换岗）、重进视口续走；战斗 NPC 原地不动；多连接位置一致

## 已知限制

- 战斗 AI 的追击移动仍走 0x20+0x1F（未验证帧型）；巡逻一律 0x21。
  战斗移动若需修正应一并切换 0x21
- 0x21 未证槽位（slot5/6/7 与 flags 高位）全填 0；玩家抓包出现的
  slot5 转向值/骑乘值未跟进（巡逻 NPC 无此需求）
- 服务器重启后 epoch 重置，NPC 在路线上重新"随机"分布（各连接仍一致）
- 停留中的朝向沿用到达段行进方向（数据无站立朝向列）
- 可见集按吸附后的出生坐标静态过滤（refreshVisibility 读目录），
  路线远端超出视口半径的巡逻段不会提前出视口——待可见性改按权威坐标再修
