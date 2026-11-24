# NPC 巡逻（mappath 路线）数据链与广播定版（2026-11-24）

> 结论先行：
> 1. **巡逻代码链一直在**（2026-11-22 自 2433 `cmd/protocol-probe/patrol.go`
>    移植到 `internal/npcfeed` / `internal/fssoloworld`）。「巡逻没了」的真因是
>    **路线数据没进数据根**：2026-11-24 Res 层化（079f866）后 `Res/mappath`
>    没有随迁，装载器对缺目录静默返回空表，带 PathId 的 NPC 全部站立。
> 2. 路线数据接入统一构建器 **`tools/datagen patrol`**：客户端
>    `share/creator/mappath/*.ini` → 数据根 `share/creator/mappath/`（与客户端
>    资源树同路径；2026-11-25 前落在数据根 `mappath/`），`.ini` 原样字节透传。
>    客户端 `share` 整树并入数据根时路线随之到位，无须单独搬运。
> 3. 覆盖面由 born03 一档扩到**全场景**：251 档 / 路线 9044 条 / 路点
>    134,655；全场景绑定 **2513 个巡逻 NPC、64 个场景**。
> 4. **本次未真机验证**（验证面 = 装载器在仓内与部署根回读 + 单测/集成测试，
>    见 §七）。

## 一、数据链

```
客户端 share/creator/mappath/<场景短名>_<类别>.ini   （官方路线表，GBK）
        │  客户端 share 整树并入数据根，或 tools/datagen patrol 单独搬运
        │  （.ini 原样字节透传，报告走服务端解析器）
        ▼
数据根 share/creator/mappath/                          （与客户端资源树同路径；层内或散目录）
        │  npcfeed.LoadPatrolRoutes(mappathDir, 场景目录)
        ▼
npcfeed.BindPatrol：摆放 PathId → 路线 → 最近路点吸附（≤35m）→ 按模板 WalkSpeed/RunSpeed 建时间表
        │  （fssoloworld.sceneNPCCatalog 目录缓存内一次完成，epoch = 装载时刻）
        ▼
sceneNPCSpawns（物化时按时刻采样坐标） + npcPatrolTick（每秒 0x10 State + 0x21 ServerAllDest）
```

- 服务端只读数据根，不直读客户端资源（与 Npc / Item / Skill / Quest 同一原则）。
- **不转换格式**：装载器读的就是 `r=` 行方言，黑金服务端自带的
  `ini/Scene/creator/MapPath/*.ini` 同形；逐字节相同，并入资源层时才能与既有层去重。
- 路线目录可由 `NINEYIN_MAPPATH_DIR` 改路径（缺省 `<数据根>/share/creator/mappath`；
  2026-11-25 前为 `<数据根>/mappath`，该旧路径已不再读取）。

## 二、路线表格式

节名 = 摆放 `PathId`（creator `PathId` 优先，模板 `PathID` 缺省），每行一个路点：

```
[AI_yyz_001]
r=,450.847992,14.162,1028.557007,3.095,200,stand,nil,walk,nil,1,
```

列定义出自同目录 `default.txt`：

| # | 列 | 含义（缺省） |
|---|---|---|
| 0 | CircleIndex | 区域编号（导航点 id；主流档留空） |
| 1..3 | PosX / PosY / PosZ | 坐标 |
| 4 | Orient | 面向角度（`-10` 哨兵 = 按行进方向） |
| 5 | StopTime | 停留时间（毫秒） |
| 6 | ActionName | 到达后播放的动作（`stand`）——**停留态**：是动作名且非 stand / walk / run 时，到点停留就发这个 State（`ty_saodiloop` 扫地、`interact50`…） |
| 7 | EffectName | 到达后播放的特效（`nil`） |
| 8 | RoadActionName | 走向下点路上的动作（`walk`）——**现行判跑依据：`run`** |
| 9 | RoadEffectName | 路上的特效（`nil`） |
| 10 | MoveType | 官方注明「走向下点是走还是跑」（`1`）；本仓未用，见 §八 |
| 11 | talkID | 对话 id |

行有三种列格式（差别只在有没有第 0 列）：

| 行格式 | 样例 | x,y,z 起始列 | 停留列 |
|---|---|---|---|
| 前导空列（主流） | `r=,900.36,17.48,117.49,0,5000,…` | 1 | 5 |
| 整数导航点 id 前缀 | `r=59,458.7,0.629,637.8,-10,60000,…` | 1（**跳过 id**） | 5 |
| 无 id 列 | `r=318.19,3.79,632.01,-10,4000,…` | 0 | 4 |

- 朝向列 `0` / `-10` 是哨兵 → 按行进方向 `atan2(dx,dz)`；否则取路点朝向。
- 停留列钳 0..120s；行进动作 `run` 的腿跑 RunSpeed（现行实现记在到达腿，
  见 §八）。
- 相邻点 <0.5m 去重，不足 2 点不成路线；与前一保留点高度差 >6m 的路点剔除。
- **把格式 2 的 id 当 x 读 = 坐标整体错位**（旧服「上天入地」根因）。
- 文件选择：stem = `<场景短名>_<类别>`，短名须在 `_` 边界处结束（纯前缀匹配会让
  `adv040` 吞进邻场景的 `adv040a_lead.ini`）；stem 含 `offline` 的不装载
  （49 档 12791 节，实测无摆放引用）。

真档统计（`tools/datagen patrol` 报告，客户端资源树
`F:\Game\Online\9yin\9yin-server-go\resources\modern`）：251 档 13.6 MiB，装载面
202 档；路线 9044 条 / 路点 134,655；坏行 1（`adv101_clone.ini` 第 179 行朝向列
`--3.023`，官方数据笔误，服务端丢弃该路点）；点数不足弃 9 节。

## 三、绑定与时间表

- **吸附**：摆放点到路线最近路点 ≤35m 才绑定，否则站立（吸附后出生坐标改写为
  该路点）。超出即「过远」。
- **时间表**：路线展开为「行程 + 停留」绝对时间线（去程 w0→…→wn-1、回程
  wn-1→…→w0），行程 = 平面距离 / 速度，钳 300..90000ms。epoch = 目录装载时刻，
  同场景所有玩家共享一份 → 位置跨连接一致，重进视口从中断处续走。
- **速度同步**：行程速度与下发给客户端的 WalkSpeed / RunSpeed 同值（缺失回退
  1.25 / 4.0 m/s，钳 0.5..15）——不同步会到点橡皮筋。
- 敌对脚本类（attacknpc / bossnpc）不绑定（巡逻与战斗互斥）。

## 四、广播

- **物化**：进场 / 重入视口时 Spawn 坐标按时刻采样（`sampleSpawns`），不闪现回
  出生点；采样时刻在走则物化帧 `State=walk`（官服同款），否则是静息态。
- **首拍延迟 5s**（`patrolFirstBroadcastDelay`）：物化落地窗口内不发巡逻帧，防把
  刚物化的 NPC 拽回出生点。
- **每秒一拍**（`npcPatrolTick`，挂 `timerCheck` 尾部）：对每个在线玩家的视口内
  巡逻 NPC——
  - 状态变了才先发 `0x10 State`（属性按名解析，fork 表位 183；先置动画状态）：
    行走 `walk`；停留 = 路点到达动作，否则 NPC 的静息态（出生姿态或 `stand`，见
    [NPC出生姿态与状态定版](NPC出生姿态与状态定版.md)）；
  - 全体组 `0x21 ServerAllDest`（`op | u16 count | count×44B`，≤8 条/帧）：
    行进 `spd=WalkSpeed/RunSpeed`、`flags=1`、`rotate=+2π`；停留 `spd=0`、
    `flags=0`。每拍全量重播（含停留中的实体）；State 按玩家各自的水位（状态名）
    决定是否发。
- **对话暂停**：点击行人弹菜单即停在暂停时刻的采样位（静息态 + `spd=0`），菜单
  关闭恢复；60s 兜底过期。
- **AOI**：`NINEYIN_NPC_RADIUS>0` 时入圈按采样位置判定（150m 迟滞圈），巡逻中
  走出视口的 NPC 发 `0x0E` 移除。
- obj / own 取目录分配值，与物化帧同源（官服 ident 形态下 own 是场景内序列号，
  不可由 obj 反推）。

## 五、开关、投放与诊断

| 项 | 说明 |
|---|---|
| `NINEYIN_NPC_PATROL=off` | 关停巡逻（路线不绑定、不发巡逻帧）；缺省开 |
| `NINEYIN_MAPPATH_DIR` | 路线目录覆盖（缺省 `<数据根>/share/creator/mappath`） |
| `go run ./tools/datagen patrol` | 构建路线：`share/creator/mappath/*.ini` → `<out>/share/creator/mappath/` |
| `go run ./tools/npcprobe -root <数据根> -scene born03` | 单场景：`巡逻 walkers N \| 站立 N \| 路线缺席 N \| 离路线过远 N` |
| `go run ./tools/npcprobe -root <数据根> -all` | 全场景扫描 + 合计 |
| 服务端 trace | 进场看 `npc catalog: … （巡逻 walkers=… stand=… missingPath=… far=…）`；路线目录缺席时另有一条 `npc patrol: 巡逻路线目录缺席 …` |

投放：数据根 `share/creator/mappath/`（与客户端资源树同路径）。仓内 testdata 层
（`b002.rep`）与部署根 `F:\Game\Online\9yin\9yin-server-go2\Res\Data001.rep` 均已含；
也可散目录投放（`Res\share\creator\mappath\`，散文件实时可见，无须重打层）。
场景目录缓存按进程缓存——**已装载过的场景要重启服务才重新绑定**。

2026-11-25 核对部署根：03:29 前层内 `share\creator\mappath` 只有 249 档（缺
`groupscene081_npc.ini` / `groupscene083_npc.ini` / `groupscene086_1.ini`，36 个 NPC
由「过远」改记「路线缺席」，行为不变）；**11-25 03:03 重铺后已补全到 252 档**，
同时顶层旧 `mappath\` 已从层里清掉。因此部署根现在只有 `share\creator\mappath`
这一处路线：**旧 exe 读顶层路径，会读不到路线、NPC 全部站立，必须部署新 exe**
（`npcprobe -all` 新 exe 实测：场景 405 / walkers 2513 / 路线缺席 111 / 过远 143，
与仓内相同）。

## 六、2026-11-24「巡逻消失」复盘

| 时间 | 事件 |
|---|---|
| 11-22 | 巡逻在 npcfeed 架构落地；路线数据双投放（仓内 `testdata/mappath` 仅 born03 两档 + 部署根 `Res/mappath`） |
| 11-24 | 079f866 testdata 并入层（born03 两档进 `b001.rep`）；部署根 Res 层化为 `Data001.rep` + bak/bak2/share，`Res/mappath` **未随迁** |
| 症状 | 全城 NPC 站立；日志只有一行 `walkers=0 stand=N missingPath=M`，没有说清原因 |

处置：路线数据接入 `tools/datagen patrol` 并全量投放；装载器对缺目录补一条一次性
trace；`npcprobe` 增巡逻绑定输出；`TestSceneNPCCatalogPatrolMultiScene` /
`TestPatrolRoutesMissingTraced` 把这两种形态钉成回归。

## 六之二、2026-11-25 路线目录改读 `share/creator/mappath`

客户端 `share` 整树并入数据根后，路线已在 `share\creator\mappath\`，再单独搬运到
顶层 `mappath/` 就是同一份数据的两处副本。读取缺省改为
`<数据根>/share/creator/mappath`（`fssoloworld.resolveMappathDir`、
`tools/npcprobe` 同序；`tools/datagen patrol` 输出同路径）。顶层 `mappath/` 不再读——
仓内 `b002.rep` 与部署层里残留的那份成了死数据，下次重铺层时可清。

## 七、验证状态（2026-11-24）

- `tools/datagen patrol` 真实资源干跑：251 档 verify 逐字节回读一致；产物与
  `testdata/b002.rep` 层内 251 档逐字节相同。
- 2026-11-25 改读 `share/creator/mappath` 后复验（`npcprobe -all`）：仓内 testdata
  **与旧路径逐场景相同**（场景 405 / 有路线场景 172 / 有巡逻场景 64 / walkers
  2513 / 路线缺席 111 / 过远 143）；部署根 walkers 2513 / 有巡逻场景 64 同数，
  仅 `groupscene081/083` 因层内缺 2 档由「过远」改记「路线缺席」（见 §五）。
- 装载器回读（`npcprobe -all`，仓内 testdata 与部署根 Res 同数）：
  **场景 405 / 有路线场景 172 / 有巡逻场景 64 / walkers 2513 / 路线缺席 111 /
  过远 143**。
- 主要场景 walkers：金陵 371、成都 197、洛阳 185、武当 161、苏州 134、丐帮 117、
  君子堂 103、极乐谷 102、燕京 96、少林 90、烟雨庄 64、恶人谷 51、千灯镇 36。
- 单测 / 集成测试（`internal/npcfeed`、`internal/fssoloworld`、`tools/datagen/...`）
  全绿：路线三格式 / 短名边界 / 缓存键 / 解析统计 / 绑定面；多场景绑定；
  巡逻拍全链（首拍闸 → State 先于 0x21 → 记录 = 视口内 walker → 水位 →
  对话停走与恢复 → 兜底过期 → timerCheck 挂接）；部署形态（半径 150 / 上限 230 /
  官服 ident）下出生点视口 230 条含 28 条巡逻。
- **未验证**：真机（客户端实际播放行走动画、到点停留、重进视口续走、多连接
  位置一致）。0x21 + State 配方沿用 2433 记载的真机定版，但本仓物化形态已换
  0x28-only，组合效果须真机确认。

## 八、已知限制与待真机确认

- **判跑依据待真机确认**：现行规则沿 2433——第 8 列 `RoadActionName == "run"`
  才跑 RunSpeed。真档里它与官方注明「走还是跑」的第 10 列 `MoveType`
  （`1` 走 / `0` 跑）并不同义：装载面 134,701 行中 4,232 行不一致（动作名留空 +
  `MoveType=0` 2,465 行、自定义跑姿如 `h_run2_1`/`qg_W_run` + `MoveType=0`、
  `run`+`MoveType=1` 858 行、`walk`+`MoveType=0` 721 行）。落到绑定到的 NPC
  上很少：千灯镇 欢欢 / 乐乐 / 笑笑（各 20 条腿，按 MoveType 应全程跑，现按走）、
  成都巡逻家丁 cd071（1 条腿）；烟雨庄、金陵无此情况。真机若见「跑姿慢放 /
  该跑在走」，把 `parseWaypoint` 的 Run 判定改读第 10 列再对拍。
- **判跑标记的归属腿待真机确认**：现行实现把一行的 run 记在**到达**该点的腿上
  （2433 同款）；而列定义写的是「走向**下**点路上的动作」，即应记在从该点
  **出发**的腿上——两者差一腿（两点往返路线正好相反）。只影响 walk / run
  混排的路线（主城与八大门派里只有二妞 qdzai005：九腿里三腿跑；大黄
  qdzai016 二十腿全跑不受影响），纯步行路线无差别。
- 行进腿一律发 `State="walk"`。官服抓包（2026-11-25 姑苏两包）里巡逻 NPC 的运行时
  状态是 `walk`/`stand` 或 `offactwalk`/`offactstand`（服务端 AI 驱动的那批），跑腿
  没有独立状态样本；本仓不发 `offact*`（见状态定版文档 §四）。
- 「路线缺席 111」：摆放带 PathId 但装载面里没有同名路线（如 `schoolwar*`、
  `world_war_02` 没有对应路线档）；「过远 143」：有路线但摆放点离最近路点
  >35m（如 `groupscene081/083`、`city04` 各 18 条）。两者均站立，属数据现状。
- 服务器重启后 epoch 重置，NPC 在路线上重新分布（各连接仍一致）。
- 停留中的朝向沿用到达段行进方向。
- 敌对 NPC 的巡逻 / 追击移动未接（战斗 AI 另期）。

## 九、代码索引

| 文件 | 职责 |
|---|---|
| `internal/npcfeed/patrol.go` | 路线域：`ParseRouteTable`（解析 + 统计）/ `LoadPatrolRoutes`（目录装载 + 缓存）/ `BuildSchedule`（时间表）/ `BindPatrol`（绑定） |
| `internal/entergen/alldest.go` | 帧族：`ServerAllDestFrame`（0x21）/ `NpcStateFrame`（0x10 State）/ `SceneRemoveObjectFrame`（0x0E） |
| `internal/fssoloworld/npc.go` | 目录缓存内的巡逻绑定、`sampleSpawns`、`npcPatrolTick` / `patrolFrames`、对话暂停、`resolveMappathDir`（缺省 `<数据根>/share/creator/mappath`）、缺目录 trace |
| `tools/datagen/patrol/` | 构建器：路线表原样搬运 + 报告 |
| `tools/npcprobe/main.go` | 诊断：单场景绑定 / `-all` 全扫 |
