# 副本系统（实例生命周期 + 波次引擎 + Lua 玩法接线）

2026-11-13 分两批落地：①副本实例生命周期（进入/重进/销毁/断线清场/通关收尾），
②`creator.NpcProgress` 数据驱动的默认波次引擎 + `game.on("dungeon_*")` Lua 玩法层。
GM 面板「副本」三按钮（前端早已就位）自此有后端实现。

## 数据源（官方资源树，容缺）

| 文件 | 作用 | 消费点 |
| --- | --- | --- |
| `share/clone/cloneinfo.ini` | 副本目录：节名即 config（`ini\scene\clone003` 原样），`Desc` + `Easy/Difficult/HellPowerLevel` → 难度档 1/2/3 | `loadGMCloneRows()`（gm_catalog_dirs.go，`/api/clones` 与 enter_clone 执行侧校验共用） |
| `share/creator/npc_creator/<clone>/ *.xml` | 副本 NPC 摆放：`<creator NpcProgress="N">` 波次刻度（经 `npc_resolver.go` 自动透传为 `resolved.Extensions["creator.NpcProgress"]`，解析层零改动）；`refreshtime="-1"` 不刷新 | 波次分组 `npcWaveValue`（dungeon_wave.go） |
| `share/clone/clonecolinfo.ini` | 关卡结构（`col_change` 波次→关卡映射）、通关时间分/奖励/声望 | **本期未接**（见后续方向） |

实测口径（clone003 暮色之村）：波次值域 **{2,3,4,5,7}，有间隔**——推进一律
"跳到目录中下一个存在的值"，不做 +1。无任何 `NpcProgress` 的副本照旧全量
注册（引擎不开启，行为与共享场景一致）。

## 实例模型

`sceneKey{config, resource, instance}`（scene_run.go）：`instance` 空 = 共享
世界场景（现状不变），非空 = 副本实例 ID（`dn-<roleID>-<序号>`）。副本实例
进入同一 `sceneRegistry`，同 config 多实例并存互不干扰。`sceneLifecycle.begin`
保留原签名作薄包装，`beginKey(sceneKey, ...)` 承载实例键（`sceneDestination`
加 `instance` 字段经 `SwitchScene` 透传，完整复用 0x0C 重入序列）。

`sceneRun` 上的副本标记（attachRun 时置位）：

- **`managed`**：末成员离开**不**自动 shutdown——实例打开期间常驻，供
  `reenter_clone` 反复进入；销毁只由 dungeonManager 显式执行。
- **`noRespawn`**：副本怪杀完不复活——combatTick 每怪复活分支与 refreshTick
  批量/重注册扫描一并跳过（尸体 TTL 清除照常，掉落宝箱不受影响）。

### dungeonManager（cmd/protocol-probe/dungeon.go）

全局单例 `dungeons`，一人一实例（无组队系统）：

```go
type dungeonInstance struct {
    id         string        // 即 sceneKey.instance
    key        sceneKey
    difficulty int           // 1=普通 2=困难 3=地狱
    ownerRole  role.RoleID
    returnTo   role.Location // 进本前位置：销毁回跳 + 存档锚点
    entry      role.Position // 入口落点（全部战斗 NPC 质心 +4m 偏移，数据驱动无硬编码表）
    finished   bool          // 末波全灭置位，属主会话 ticker 执行回城+销毁
    run        *sceneRun
}
```

状态机：`create`（已有打开实例拒绝）→ `attachRun`（回填 run + 置标记）→
`open` / `take`（摘除）/ `markFinished` / `dropOnDisconnect`。

### 位置持久化口径（重要）

进副本期间**不落库** clone 场景位置：移动路径有"在场闸门"（main.go，
`dungeons.open(...) && runtime.world.run == instance.run` 时跳过 2s 节流
落库并清 `pendingLocation`），存档锚点始终保持 `returnTo`——防重启后落进
共享态 clone 场景；destroy/通关回城后才正常持久化。断线时实例空场销毁、
不回跳，重登自然回到存档锚点。

## 生命周期流程

| 路径 | 流程 |
| --- | --- |
| `enter_clone` | 目录+难度档校验 → `create`（returnTo=当前位置）→ `SwitchScene`（带 instance 键）→ `attachRun`（managed/noRespawn + scripts/finishHook 注入）→ 发 `dungeon_enter` 事件 |
| `reenter_clone` | 有实例 → SwitchScene 回实例入口（进度保留：死过的怪不复活） |
| `destroy_clone` / 通关收尾 / 断线 | 共用 `returnDungeon` 闭包：发 `dungeon_destroy` 事件 → 在场则 SwitchScene 回 returnTo 并落库 → `shutdownInstance`；断线路径（handle 的 defer）只销毁不回跳 |
| 通关 | 末波全灭 → 10s 缓冲（捡拾掉落）→ `finishHook`（sceneRun 定时器，epoch 防穿越）→ `markFinished` → 属主会话 1s ticker 发现后走 `returnDungeon`（SwitchScene 必须在属主会话 goroutine，单写者纪律） |

## 波次引擎（cmd/protocol-probe/dungeon_wave.go）

- **装填**：`populate` 时 `initWaveGroupsLocked` 归纳合格出生点的波次组；
  只注册首波（最小值）+ 无波次常驻 NPC（funcnpc/commonnpc 等场景设施）。
  NPC ID 沿用 `(目录下标+2)|0x11000000` 布局，`nextWireID` 已预留全场，中途
  注册不冲突（见 [id-system-analysis.md](id-system-analysis.md) §6.2）。
- **在场集合**：击杀咽喉点 `recordNPCDeathLocked` 维护 `waveAlive`（删除）+
  `dungeonKills`（累计）+ 发 `dungeon_npc_killed` 事件。
- **推进**：`tickOnce` 的 waveTick 段——当前波全灭（`waveAlive` 空）且在场
  有成员时结算一次（`waveClearedFired` 水位防重发）：默认物化下一波
  （`spawnWaveLocked`，幂等）或末波排程收尾；`viewsDirty` 置位后**锁外**
  `reconcileMemberViews` 对账全体视口——站着不动的玩家也能立即收到新波
  0x0D/0x28/0x1F 物化帧簇。
- **空场暂停**：成员为 0 时 waveTick 直接返回（切出副本期间不推进）。

## Lua 玩法层（引擎在 Go，玩法在脚本）

事件与能力回调（宿主契约见 [server-luascript.md](server-luascript.md)，
ctx 字段见 [../scripts/README.md](../scripts/README.md)）：

| 事件 | 触发点 | ctx 特有 |
| --- | --- | --- |
| `dungeon_enter` | enter_clone 成功进入后（main.go，会话 goroutine） | `ctx:player()` 快照 |
| `dungeon_npc_killed` | 副本内 NPC 死亡（recordNPCDeathLocked，持 run.mu） | `ctx:npc()` 死者 |
| `dungeon_wave_cleared` | 当前波全灭结算点（waveTick，持 run.mu） | 见下 |
| `dungeon_destroy` | returnDungeon 销毁前 | - |

`ctx:dungeon()` → `{ config, resource, instance, difficulty, wave, next_wave, killed }`
（`next_wave`=0 即末波）。能力回调：

- `ctx:spawn_wave(n)`：立即物化波次组 n（引擎 `spawnWaveLocked`，幂等）；
- `ctx:after(seconds, event)`：声明延时事件（秒 ∈ (0,600]，定时器归
  `sceneRun.timers` 持有、epoch 防世代穿越，触发时携带全套副本回调）；
- `ctx:notify(brief, chapter?)`：剧情提示卡**广播全体成员**。

**接管语义**：`dungeon_wave_cleared` 处理器返回真值 = 接管推进节奏（默认
引擎跳过），脚本用 `after`+`spawn_wave` 自己控场；返回 `false`/无处理器 =
默认立即推进。末波收尾不受接管抑制（防悬挂实例）。锁纪律沿用 npc_service
口径：事件在持 `run.mu` 的路径发起，宿主 mu 是叶子锁，回调（写帧/注册）
在调用方锁纪律下执行；脚本错误按"已接手"记录日志，VM 自动重建。

试水脚本：[../scripts/host/dungeon_clone003.lua](../scripts/host/dungeon_clone003.lua)
（进本剧情卡、每波肃清播报、波 3 接管演示：2.5s 后放下一波、末波通关提示）。

**已知限制**：`game.on` 每事件名单处理器（后者覆盖前者）——第二个副本
脚本落地前须先加"按 `ctx:dungeon().config` 分发"的路由层。

## GM 命令（面板「副本」板块，前端零改动）

| action | 参数 | 行为 |
| --- | --- | --- |
| `enter_clone` | config + fun 1..3 | 契约层形状校验；执行侧查 cloneinfo 目录 + 难度档存在性；已有打开实例拒绝 |
| `reenter_clone` | - | 回到打开的实例（入口落点） |
| `destroy_clone` | - | `returnDungeon`：回进本前位置并销毁 |

命令链不变：`gmapi.Validate` → `hub.enqueue` → 会话循环 `processGMCommand`
（见 [gm-service-rpc-split-design.md](gm-service-rpc-split-design.md)）。

## 验证记录（2026-11-13, windows/amd64）

- `go build ./...` + `go test ./... -count=1` 全绿；副本链路 `-race` 通过。
- 单测：实例键隔离、managed 常驻、noRespawn 对照、状态机、入口质心、
  波次分流/推进/收尾、Lua 接管（真 Host + 临时脚本）、npc_killed 快照、
  真机脚本目录装载冒烟（`TestDungeonScriptsLoad` 拦截 scripts/host 语法错）、
  gmapi 三命令校验（cmd/protocol-probe/dungeon*_test.go、internal/gmapi、
  internal/luascript/dungeon_ctx_test.go）。
- 存量提示：`TestChannelWavesDrainSPAndEmitPerWaveFrames` 在 `-race` 下有
  数据竞争，经干净 HEAD worktree 验证为**基线既有**（引导技能未提交改动），
  与副本无关。
- 真机路径：起服（`-scripts scripts/host`）→ GM 进 clone003 普通 → 进本
  剧情卡（Lua）→ 仅波 2 在场 → 杀光即冒波 3 → 波 3 清完 2.5s 放波 4（接管
  演示）→ 末波 7 清完提示 → 10s 自动回城销毁；reenter/destroy 同路径可验。

## 后续方向

- **脚本路由层**：`game.on` 单 handler 限制，多副本脚本前先做按 config 分发。
- **clonecolinfo 关卡层**：`col_change` 波次→关卡映射、时间分/通关奖励/
  声望（数据已勘察，落在本层之上的结算模块）。
- **难度数值缩放**：难度档目前只随实例记录，未参与 NPC 属性推导。
- **AdventureDoor 准入**：副本入口门协议未还原（door_data.ini 无 clone 条目），
  本期只能 GM 进本；官服进副本播报 `m1 c1=3` 同样待逆向。
- **多人/组队**：byOwner 单人归属；sceneRun 成员模型天然支持多人，补"加入
  他人实例"的 join 语义即可。
