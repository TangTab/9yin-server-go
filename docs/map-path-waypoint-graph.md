# 路点图(.path)格式逆向与 NPC 追击寻路方案

> 2026-11-09 完成。服务端 `.path` 二进制格式破解 + 数学级校验 + 几何空间评估 + NPC 追击穿墙/穿地根因定位。
> 状态:**§6 全部落地（2026-11-09）**——`internal/mapnav`（parse/index/astar/follow 四文件）+ `cmd/protocol-probe/mapnav_chase.go` 追击重接线，测试 22 项全绿（mapnav 13 + 追击集成 6 + 批测 2）。实现新发现见 §6.5。配套文档:`map-region-are0-rgn1.md`(客户端区域多边形)、`jingmai-cd-qinggong.md`(轻功)。

## 1. 数据资产

`E:\Workspace\project\game\9yin_server-go\resources\modern\map\path\`,共 **557 个 `.path`**(~214MB)。

| 前缀 | 含义 | 数量级 |
|---|---|---|
| `scene01~64` | 主世界场景 | 大文件(scene14 达 7MB) |
| `school01~31` | 门派(含 `_mie` 灭门、`war` 门战变体) | 中 |
| `adv0xx` | 副本(含 `_xue` 血战变体) | 小 |
| `battle_*` | 战场 | 极小(几十节点) |
| `born01~04` | 出生地 | 小 |
| `*_main.path` | 同名地图的**主干道骨架图**(极稀疏) | 如 adv031_main 仅 32 节点 vs 主图 513 |

地图类型全覆盖:主城/门派/副本/战场/出生点。服务端资源**没有任何碰撞几何**(全目录无 .scene/.mdl/.col/.nav/.mesh,仅 ini/xml/lua/path)——碰撞从来只在客户端。

## 2. 二进制格式(已破解并验证)

小端,无文件头魔法:

```
int32  节点数 N
节点 × N:
    float32 x, y, z        // y = 高度
    int32   邻边数 E
    边 × E:
        int32   目标节点下标   // 0 基,双向(邻接表互指)
        float32 欧氏距离       // 与两点坐标欧氏距离完全一致
```

Go 解析伪代码:

```go
count := int32(binary.LittleEndian.Uint32(b[0:4])); off := 4
for i := 0; i < count; i++ {
    x, y, z := leF32(b[off:]), leF32(b[off+4:]), leF32(b[off+8:]); off += 12
    e := int32(le(b[off:])); off += 4
    for j := 0; j < e; j++ {
        target := int32(le(b[off:])); dist := leF32(b[off+4:]); off += 8
        _ = target; _ = dist
    }
}
// 校验:off == len(b) 必须成立
```

### 2.1 验证方法与结果

两步验证,6 个样本跨类型(scene 主城/school 门派/adv 副本/battle 战场 + `_main`):

1. **字节级**:全部精确解析到文件尾,无剩余字节、无截断;
2. **数学级**:把每条边存储的距离与两端节点坐标的欧氏距离比对。

| 文件 | 节点 | 边 | 距离不匹配 | 字节精确 |
|---|---|---|---|---|
| scene14.path | 166,948 | 1,243,430 | **0** | ✅ |
| school01.path | 15,271 | 81,798 | 0 | ✅ |
| scene04.path | 6,133 | 29,998 | 0 | ✅ |
| adv031.path | 513 | 2,244 | 0 | ✅ |
| adv031_main.path | 32 | 60 | 0 | ✅ |
| battle02.path | 33 | 112 | 0 | ✅ |

合计 135 万条边零不匹配 → (x, 高度, z) 坐标解释**不是猜测,是数学确认**。

## 3. 几何性质(实测)

| 指标 | scene14(主城) | school01(门派) | adv031(副本) |
|---|---|---|---|
| 节点/边 | 166,948 / 1,243,430 | 15,271 / 81,798 | 513 / 2,244 |
| 连通分量 | 62 个,最大占 **99.8%** | 439 个,最大 8,109(53%) | 19 个,最大 385(75%) |
| 度 | min1 / avg7.4 / **max8** | min1 / avg5.4 / max8 | min1 / avg4.4 / max8 |
| 边长 | 10.00~19.57m,p50=10.45 | 10.00~22.37m | 10.00~20.78m |

- 度上限恒为 8 + 边长下限恒 10.0m → 典型 **8 邻接网格采样路点网**,即可行走面的 ~10m 均匀采样;
- 主城尺度 ~4km×4km(x/z ∈ [-1915, 2165]),高度差近 200m(三维结构完整);
- school 的多分量 = 多层建筑/轻功区天然形成子空间;**A* 失败(跨分量)必须有兜底**(见 §6.4)。

## 4. 架构决策:服务端不做精确碰撞

原版即客户端权威运动(客户端自报告位置,服务端 0x21/0x1F 转发);`share/area/areainfo.ini` 的 `forbid_qinggong/forbid_run/forbid_ride/forbid_drop_down_hurt` 用**规则开关**管运动能力,而非物理模拟。

服务端防作弊/校验采用分层(成本从低到高),现有数据全够:

1. **速度钳制** — 每 tick 位移 ≤ 速度×时间×容差;
2. **路点吸附** — 位置离最近路点 > ~15m 判定在墙里/虚空(需空间索引);
3. **图距离校验** — 两次心跳间位置变化 vs 路点图最短路,穿墙抄近路必暴露(图绕墙、外挂走直线);
4. **区域规则** — areainfo forbid_* + ARE0/RGN1 多边形;
5. **统计累积** — 单次违规忽略,累积阈值再处置,防网络抖动误伤。

技能 LOS 用距离+朝向即可;讲究点在路点图上做射线连接性判断(10m 精度够用)。

## 5. NPC 追击穿墙/穿地:根因(代码已定位)

`cmd/protocol-probe/combat_ai.go`,追击分支:

**穿墙** — `combat_ai.go:408`:每 tick `stepToward` 沿"NPC→玩家"**水平直线**推进 `MoveSpeed`,无任何寻路,中间有墙有山直接切过去。

**飘/穿地** — `stepToward`(`combat_ai.go:781-795`)只插值 X/Z:

```go
result := from                 // Y 继承出发点的旧高度,追击全程永不更新
result.X += (to.X - from.X) * ratio
result.Z += (to.Z - from.Z) * ratio
```

玩家往坡上跑 NPC 钻地,往坡下跑 NPC 悬空 —— 与观察症状完全吻合。

另:每 tick 重算直线目的地本身也造成走位抖动(客户端按 0x21 目的地插值)。

## 6. 实现方案:`internal/mapnav` 模块

### 6.1 模块构成

```
internal/mapnav/
    parse.go    二进制解析(§2 格式)+ 字节级校验(off == len)
    index.go    场景级缓存:XZ 均匀网格(建议 32m 格)→ 节点桶;最近路点查询
    astar.go    A*(边权即存储距离;节点数最大 16.7 万,需二叉堆 + 距离矩阵惰性)
    follow.go   折线跟随:推进 MoveSpeed、相邻路点 y 线性插值、末段直线
```

单次加载、`sync.Once`/LRU 缓存;scene14 解析预算 ~100ms 级、内存 ~30MB(可只留 XZ 索引 + y,边表紧凑 int32)。

### 6.2 场景 → path 文件映射

挂接点已有:`role.Scene.Resource`(由 `door_routes.go:175 resourceForDoorScene` 从场景配置推导地形名,如 scene14/school01)。
规则:取 `<Resource>.path`;文件不存在 → 无图场景走兜底。**实现前先用真实会话的 Resource 值核对一遍命名**(尤其 `_mie`/`war`/`_2` 变体是否各有独立 .path)。

### 6.3 追击分支重接线(combat_ai.go)

替换 `combat_ai.go:408` 的直线推进为:

1. **重算节流**:每 300~500ms,或目标偏离上次规划终点 > 8m 时重算 A*(起点=NPC 最近路点,终点=玩家最近路点);
2. **折线跟随**:每 tick 沿路径折线推进 `MoveSpeed`,**Y 取相邻两路点高度的线性插值** → 穿地/悬空消失;
3. **末段直线**:距玩家 < ~12m(约一个路点间距)切直线逼近,视觉无缝;
4. **兜底(无图场景)**:保留现有直线 `stepToward`,但 Y 从最近可图吸附;完全无图则维持现状并打点日志;
5. **跨分量兜底**:起终点不同连通分量 → A* 失败 → 沿直线走但每 ~2s 重试吸附(玩家可能走进另一分量可达范围);连续失败 N 次放弃仇恨回 Home。

注意保持 `scheduleCombatMoveArrival`(`combat_ai.go:560`)的到达确认节奏:折线路径长 > 直线距离,到达延时按**路径长/速度**计算,否则提前确认又回到"漂移"。

`0x21/0x1F` 广播结构不动 —— 目的地沿折线走后,客户端插值出的轨迹自然绕墙。

### 6.4 测试计划

- 解析:全部 557 文件离线批量解析(字节精确 + 距离校验,即 §2.1 脚本转 Go 单测,挑 6 个代表文件入 testdata);
- A*:scene14 上随机 1000 对起终点,断言路径存在率 ≥ 主连通分量占比、路径长 ≥ 欧氏距离;
- 跟随:构造坡地路径断言 Y 单调插值无跳变;穿墙回归:在已知有墙场景(需实测选图)断言路径绕行;
- 追击集成:复用 `NINEYIN_NPC_CHASE` 方言 + `combatTickForTest` 驱动。

## 6.5 实现落地记录（2026-11-09）

### 模块与接线

| 文件 | 内容 |
|---|---|
| `internal/mapnav/parse.go` | 二进制解析（两遍扫描 CSR 邻接表）+ 字节精确校验 + `ValidateEdgeDistances` 数学级校验 + 越界/截断防御 |
| `internal/mapnav/index.go` | 32m XZ 均匀网格（counting-sort 前缀桶）+ `Nearest`（**3D 距离**环形扩张剪枝——同 x/z 多楼层必须命中同层）+ `Library`（进程级缓存，缺失负缓存只探一次盘） |
| `internal/mapnav/astar.go` | 二叉堆 A*（手写堆 + 代章懒失效数组复用；启发式 3D 欧氏可采纳） |
| `internal/mapnav/follow.go` | `Follower` 折线进给（y 线性插值）+ `StepTowardGround`（直线贴地）/`LerpOrientFreeStep`（末段直线，旧 stepToward 语义+y 插值） |
| `cmd/protocol-probe/mapnav_chase.go` | 追击状态机（`npcChaseState`/`advanceChaseLocked`/`planChasePath`）+ `warmNav`（sync.Once 后台预热，r.mu 内落盘） |
| `cmd/protocol-probe/combat_ai.go` | 追击分支替换为 `advanceChaseLocked`；栓绳/放弃共用 `appendNPCReturnHomeLocked`；`npcMoveUpdate.delay`=到达确认延时（剩余路径长/速度，旧固定 1s 保留给重生/栓绳瞬移） |
| `testdata/mapnav/` | 6 个代表文件（adv031/_main、battle02/_main(4B 空图)、scene04、school01） |

### 关键实现决策（与 §6.3 的差异/补充）

1. **规划折线丢弃 path[0]（NPC 最近路点）**：战斗 tick 1s > 重规划节流 400ms，即追击期间几乎每拍重规划；若折线首段是“当前位置→最近路点”，NPC 位于两路点中间时每拍都先折返 → **原地振荡永不前进**（穿墙回归测试实测）。改为当前位置直接走向 path[1]，弦切 ≤ 一格路点间距（~10m），网格拓扑仍守。
2. **Nearest 用 3D 距离而非水平距离**：school 类多层建筑图同 x/z 上下叠层，水平最近会吸附错楼层；网格环形剪枝的下界是水平量恒 ≤ 3D 距离，正确性不受影响。
3. **导航状态三处生命周期钩子**：清仇恨（栓绳/目标死亡）、重生、跨分量放弃——全部 `meta.chase = nil`。
4. **sceneRun 异步预热**：`newSceneRun` 即 `go warmNav()`（scene14 解析 ~400ms 全量批测口径），首个追击 tick 在 run.mu 内零阻塞；未就绪当拍直线兜底。
5. **测试钩子 `mapnavLoaderForTest`**：合成图注入必须先于 sceneRun 创建（预热 Once 抢跑会负缓存）；追击测试驱动用 `chaseTickForTest`（`combatTickForTest` 每次调用挂新成员、旧成员位置冻结——仅测试现象，生产连接循环无此问题）。

### 批测实证（F 盘部署 557 文件，2026-11-09）

- **字节精确 557/557**（351 万节点/2085 万边，解析总耗时 ~400ms）；scene14 A* 随机 1000 对：成功率 0.995（理论 Σ(分量占比)²=0.996）、平均 **2.6ms/对**、主分量 99.8%（62 分量）——与 §3 分析一致；
- **官方手编非欧边权（§2.1 的修正）**：508/557 文件在 0.1m 内全净；49 文件含少量官方 authored 边权（groupscene* 系恒 52 边/≤1.5m、test*/clone* 副本图最高 ~25m、25% 边）。**存储权重即权威代价**，A* 照用；批测硬闸只拦“maxDev>50m 或过半边偏差”（格式误读的签名），authored 噪声记日志不判败。

### 后续

- 真机验证：city05/scene 野外拉怪观察绕墙/贴坡；追击总闸已上 GM 面板（怪物刷新分组 `set_npc_chase` 0/1，进程级覆写、重启回落启动 env `NINEYIN_NPC_CHASE`——env 仍作起服初值方言）；
- `_main` 图接入（§7.1）与防作弊层（§4 的 2/3/5 层）未做，属独立批次。

## 7. 开放问题

1. `_main.path` 的真实用途未证实(推测:大世界行路/巡逻主干),实现 mapnav 时顺带在官服抓包验证,暂不接入;
2. 路点间距 ~10m,坡度大处两路点高差可达数米,y 插值已够视觉;若日后要更准,可结合 ARE0 地形(`map-region-are0-rgn1.md`,完成度 ~80%);
3. school 类多分量图占 47% 节点在次级分量,A* 兜底策略(§6.3.5)需实测调参;
4. `idres` 文件(608 个)未排查是否含额外空间数据,低优先级。

