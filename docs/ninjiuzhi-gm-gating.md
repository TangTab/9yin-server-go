# 九阴志全局门控（GM 修复 · Go 移植）

蓝本 = Java 参考服已被删除的 `NinJiuZhi.java`（`G:\pojie\ai\9yin-server` git `3d1df4a`「新增九阴志全服阶段门控并接入场景切换」，原设计文档 `docs/ninjiuzhi-gating.md`）。本文件记录 Go 侧实现与 Java 蓝本的差异。

## 核心组件

`cmd/protocol-probe/jyz_gate.go`（`jyzGate` 全服单例，`main()` 经 `loadJyzGate` 装载）：

- **阶段**：1..9 单向刻度（1=1内 … 9=6内，对应客户端 power_guide.ini 的 step 主步骤）。`setStage` 拒绝回退（原因文本回给面板 409）；`reset` 回 initial、点数/事件清零。
- **事件点**：kill/task 双计数 + 总点数，累计阈值自动升阶（可连跳）：升到 2..9 阶 = 10/30/60/100/150/210/280/360 累计点，`thresholds` 可覆写。
- **落盘**：`<server root>/data/ninjiuzhi.json`（字段 stage/active/points/events/revision，与 Java 及 data-import 的 `server_ninjiuzhi` 表同构），每次变更即落盘；**损坏档拒绝覆写**（loadFailed 闸，比 Java 的自愈口径更保守——避免静默重置全服阶段）。

## 初始阶段优先级

`data/ninjiuzhi.json` 存档 > env `NINEYIN_JYZ_STEP`（>0 生效，钳 1..9）> requirements `initial_stage` > **6**。

**Go 差异 ①**：缺省 6 而非 Java 的 1——保住登录 419/414 gamestep 帧与官服 c.pcapng #111（gamestep=6）逐字节一致的性质（2026-11-08 拍板）。要测 1..5 低阶段：设 env 或 requirements，再 reset 回落。

## 数据源（均容缺，缺省 = 阶段 6、全部开放）

| 文件（`<server root>/data/` 下） | 作用 |
| --- | --- |
| `ninjiuzhi.json` | 运行时状态（装载 + 落盘同一文件） |
| `ninjiuzhi_requirements.json` | 五类门控覆写 + `initial_stage` + 可选 `thresholds`（对象 `{"2":30}` 或数组双形态） |

```json
{
  "features":    {"smelting": 5},
  "tasks":       {"1346": 2},
  "npcs":        {"npc_locked": 3},
  "scenes":      {"book8": 2},
  "inner_books": {"ng_wd_001": 4},
  "initial_stage": 1,
  "thresholds":  {"2": 10, "3": 30}
}
```

**Go 差异 ②**：五类门控只装载 + `isOpen` 查询，**尚未接线**到场景切换/任务解锁/NPC 可见性/内功 View（Java 蓝本的五处接入点回归面大，留待后续批次；未配置覆写的键一律开放，行为与接线前完全一致）。

## 协议：阶段帧的三处下发

- **登录链**（既有）：quest49 簇 → 419 步骤（`jyzGate.config()` 取当前阶段）→ data_record 58 行 → switch sync → 414 'gamestep' 收尾（帧序铁律不变，逐字节与官服对齐的基线 = 阶段 6）。
- **GM 阶段生效**：发起连接内联写 419 → 414 gamestep → `worldInfo(0, "九阴志推进：阶段 N（name），江湖暗流涌动")` → `worldInfo(0, 状态文本)`（Java broadcastNinJiuZhiStage + GmActions 同源帧）。
- **其他在线连接**：**Go 差异 ③**——不跨 goroutine 写连接（单写者纪律），由各会话 1s ticker 的 `stageRevision` 漂移检查补推同组帧（≤1s 延迟；`jyzGameStepSent` 门槛保证不会抢在 switch sync 前发 gamestep）。晚接入一致性由登录链天然保证。

data_record 事件行与阶段正交（env `NINEYIN_JYZ_EVENT_STATE`），不随阶段变化重发。

## 事件点来源

- **击杀**：`recordNPCDeathLocked`（npc_refresh.go，单体/群伤两条死亡路径的咽喉点）→ `jyzGate.onKill()`（Java SceneFlow 同款）。锁内只动门控状态（gate.mu 不再取场景锁，无锁序环），广播交给 ticker。
- **任务**：任务系统未接入前由 GM 注入（`ninjiuzhi_event task`）。

## GM 动作（面板「任务与身份 → 九阴志全局门控」板块，前端零改动）

| action | 参数 | 行为 |
| --- | --- | --- |
| `ninjiuzhi_set` | stage 1..9 | 单调设阶；生效即落盘 + 本连接三帧组；回退返回「阶段只增不减：…」409 |
| `ninjiuzhi_event` | eventKind=kill/task | 事件点 +1（可触发自动升阶连跳） |
| `ninjiuzhi_reset` | - | 回 initial、点数/事件清零；阶段回落同样广播 |
| `ninjiuzhi_status` | - | worldInfo 回显状态文本 + 会话 LastAction 供面板轮询 |

状态文本口径（Java status 逐句对齐）：`九阴志：阶段 6/9（5内后期），距下一阶还差 210 事件点；累计 0 点；门控覆写 0 条`。

## 与既有机制的关系

- 副本波次（NpcProgress）是**场景内**刻度，与九阴志**全服**刻度相互独立。
- funcbtns 全局门（26001=暮色初章）走 Task_Completed 1346 位（选角链 grantFunctionUnlock，已铺），与本门控正交——九阴志阶段影响的是 419/414 携带的 gamestep 值与（后续批次接线后的）内容门控。
