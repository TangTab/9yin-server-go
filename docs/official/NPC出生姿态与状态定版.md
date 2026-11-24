# NPC 出生姿态与状态定版（2026-11-25）

> 结论先行：
> 1. **官服物化帧里 `State` 与 `BornAction` 恒同值**，取 NPC 的「出生状态」：姿态 NPC
>    两者都是动作名（`ty_sitchair_drinkloop` 坐椅喝茶 / `sitloop` / `ty_lie_sleep` /
>    `ty_clap` …），普通站立为 `stand`，行进中进视口为 `State=walk` + `BornAction=stand`。
>    两包官服抓包（姑苏 `npc.pcapng` 152 个 NPC + `npc-点击对话.pcapng` 420 个 NPC，
>    合计约 70 个姿态物化）无一例外。
> 2. 此前本仓物化帧把 `State` 写死 `stand`、`BornAction` 只读模板列——而出生状态基本
>    都挂在**摆放行**上（模板 8.4 万行里只有 1 行带 `BornAction`），所以坐 / 喝 / 倒地
>    的 NPC 全站着。
> 3. 现按 摆放 `BornAction` > 模板 `BornAction` > 模板 `BornActionRec` 取值：物化面
>    82,837 个 NPC 里 **15,691 个带姿态**（423 种动作，189 个场景）。
> 4. **未真机验证**（验证面 = 装载器回读 + 与官服抓包样本逐个对拍的单测，见 §六）。

## 一、官服证据

抓包：`F:\Game\Online\9yin\9yin-server-go2\资料\npc.pcapng`（姑苏，0x28 物化帧 320 条）
与 `npc-点击对话.pcapng`（447s，195 次点击 NPC）。解码：`tools/pcapdump` 落帧 → 会话
自己的 0x09 表（2433 项，`State@183` / `BornAction@1604` 均为串）→ 0x28 mode 0x47 体经
`tools/lz_exact.py` 解压。

| 物化形态 | State | BornAction | 姑苏样本数 |
|---|---|---|---|
| 普通站立 | `stand` | `stand` | 88 |
| 行进中进视口 | `walk` | `stand` | 20 |
| 姿态 NPC | 动作名 | **同一动作名** | 12 |

- 12 个姿态 NPC 的 `ConfigID` 在本仓 `city02_suzhou` 摆放行上一一对应同值
  （`WorldNpc01499=ty_kneel_entreat`、`WorldNpc01108=ty_talk`、`FuncNpc00104=ty_nod`、
  `WorldNpc01064=ty_lie_sleep`、`npc_mmzw_other_003=ty_sitground_dozeloop` …）。
  同一模板在别的图没有 `BornAction`（`FuncNpc00104` 在燕京为空）→ **摆放级属性**。
- 模板 `BornActionRec=stand` 的 NPC 官服发 `stand`/`stand`。
- **姿态 NPC 物化后零 State 帧**（抓包内物化后 30~120s）：姿态由客户端自持，服务端
  不收势。
- **点击对话不给被点击的 NPC 发 State 帧**（195 次点击无一例外）。
- 运行时官服给 NPC 切状态用 `offact*` 命名（`offactwalk` / `offactstand` /
  `offact148`…，客户端 `ini/action/offlineaction.ini` 的离线动作）或 `walk`/`stand`；
  玩家自己才是成对的 `stand`→`walk`。

## 二、出生状态的数据来源

客户端 npcconfig 表里同一个语义有三种列名，列注释都是「**出生状态**」，没有多值单元格：

| 列名 | 出现在 | 本仓 XML 属性名 |
|---|---|---|
| `BornAction` | `worldnpc/born_commonnpc`、`city_commonnpc`、`jobnpc_commonnpc`、`school_commonnpc`、`other_commonnpc`、`scene_commonnpc` | `BornAction`（2026-11-25 起入映射；此前被丢弃） |
| `table@BornActionRec` | `clonenpc/*`、`worldnpc/newworldnpc`、`newschoolnpc` … | `BornActionRec`（tableRec 去前缀） |
| `table@BornAction` | `worldnpc/monsternpc_commonnpc` | `BornAction` |
| creator 摆放行 `BornAction` 属性 | 场景摆放 | `Spawn.BornAction`（优先级最高） |

分层（`npcfeed/pose.go`）：摆放行 > 模板 `BornAction` > 模板 `BornActionRec`。

**取值规则**（`RestAction`，不移动时的 `State`）：

- 是动作名（`^[A-Za-z0-9_]+$`）且不是 `stand` / `walk` / `run` / `nil` / `0` → 原样用。
- 其余 → `stand`。`walk` / `run` 表示「出生即在走」，不是停留姿态；非动作名是客户端
  表里的脏值（`scene_commonnpc` 的 `BornAction` 列混有 `icon\monster\*.png`）。
- 分号列表（摆放行 60 条：`stand;sitloop`、`h_stand_2;ty_bukuai_guard`、
  `坐喝茶;坐拉二胡`…）语义未知，官服抓包里没有样本——不猜：去空后恰剩一个动作
  （`stand;` / `diedownloop;` 这类尾分号）就取它，多候选按 `stand`。

巡逻 NPC 的出生状态（约 2,500 个里约 100 个有值）：`walk` / `run` 28 个、`stand` 9 个，
其余约 60 个是真姿态（`standfree1`、`ty_stand`、`ty_talk4`、`sitloop`、
`ty_bukuai_guard`、`ty_kneel_eat`…）——它们停留时回各自的姿态。

全量分布（`npcprobe -all`，仓内与部署根同数）：

| 类别 | 数量 |
|---|---|
| 物化面 NPC | 82,837 |
| 姿态 NPC（静息态≠stand） | 15,691（摆放行 15,340 + 模板 `BornActionRec` 351） |
| 主要取值 | `h_stand_2` 1331、`diedownloop` 676、`sitloop` 608、`closeloop` 441、`ty_talk` 429、`ty_bukuai_guard` 387、`ty_sitchair_drinkloop_free01` 323、`ty_sitchiar_eatloop` 170 … |
| 烟雨庄 / 成都 | 490 / 476 |

## 三、状态模型

```
物化帧      State = BornAction = 静息态（巡逻 NPC 采样时刻在走则 State=walk）
巡逻每秒拍  行走 → walk；停留 → 路点到达动作，否则静息态；状态变了才发 0x10 State
对话暂停    停在暂停时刻的位置，置静息态
不巡逻 NPC  只靠物化帧，不发 State 帧
```

- **路点到达动作**：mappath 每行的 ActionName 列（`default.txt` 注「到达后播放的
  动作」，缺省 `stand`）。是动作名且不是 stand / walk / run 时作为停留态——巡逻路线
  975 个停留路点里 27 个带动作（`ty_saodiloop` 扫地 ×6、`interact50`、`ty_clean`、
  `ty_think`、`ty_nod` …）。
- **行进中进视口**：`sampleSpawns` 把采样时刻的状态标到 `Spawn.State`，物化即 `walk`
  （官服同款）。
- State 水位按玩家记**状态名**（原为「是否在走」布尔）。
- `entergen.NpcStateFrame(id, own, state)` 改为直接传状态名；空状态拒绝。
- **A/B 开关 `NINEYIN_NPC_POSE=off`**：关停出生姿态（静息态与物化帧一律 `stand`，
  即此前行为），路点到达动作不受影响。真机上某类动作表现不对时，加这个环境变量
  重启即可对照，不必回滚代码。

## 四、没有做的（及原因）

- **官服 `offactNNN` 周期空闲动作**：抓包里有一批 NPC 周期性切 `offact148↔offact149`
  这类成对动作（铁匠抡锤、摊贩叫卖之类）。这是官服服务端 AI 驱动的，模板表头有
  「表现AI包编号 `AIParaNo`」一列，但没找到 NPC → 动作组的可解码通路
  （`offlineaction.ini` 只列了 22 组生活动作且无人引用；AI 编辑器事件号与 offact 号
  对不上）。未实现。
- **随机空闲动作**（参考端 `9yin-go-server2` 的 idle emote：按 AI 编辑包里
  EventType=1 + OpEvent=31 的 37 条事件，随机间隔播放后回 stand）：姑苏包 152 个物化
  NPC 里有 112 个物化后再没收到 State 帧，收到帧的 16 个全是巡逻 NPC 的走 / 停——
  没有证据支持，未实现。
- **对话状态**：官服不发，保持不发。
- **分号列表型出生状态的多候选取法**：见 §二，没有官服样本，暂按 `stand`。

## 五、判跑列的小修正

`parseWaypoint` 判跑原先固定读 `parts[8]`，没跟随坐标起始列——无 id 列的行格式（坐标
从第 0 列起，装载面里约 6 行）会读成特效列，永远判走。现按 `base+7` 取；到达动作按
`base+5`。

## 六、验证状态（2026-11-25）

- `TestFramePoseMatchesOfficialSamples`：官服姑苏 14 个样本 NPC（12 姿态 + 2 普通）的
  物化帧 `State`/`BornAction`（0x28 与 0x0D 两个形态）逐一等于抓包值。
- `TestFramePoseCoverage`：烟雨庄 490 个姿态 NPC 的静息态等于其出生状态。
- 分层 / 校验 / 列表型 / 到达动作 / 无 id 列判跑（`pose_test.go`、`patrol_test.go`）；
  巡逻状态时间线（去程 walk → 到点扫地 → 回程 walk → 站立；带姿态的巡逻 NPC停留回
  姿态）、物化采样、对话停走（`npc_patrol_test.go`）。
- `npcprobe -root <数据根> -all` 末行有 `姿态 N（M 种动作）`；单场景有 `姿态` 行与
  动作名分布（也认 `NINEYIN_NPC_POSE=off`）。
- 回归：`go build ./...`、vet、`internal/npcfeed` / `internal/entergen` /
  `internal/fssoloworld` / `tools/datagen/...` 全绿；NPC 相关用例 40 轮压测稳定（顺带
  修了测试服务器迟到日志的 panic：选靶 5s 看门狗比用例活得久）。
- **未验证**：真机（客户端对各类动作名的实际表现，尤其 `ty_nod` / `ty_clap` 这类
  一次性动作作出生状态时是否保持——官服同样这样发且不收势）。
- NPC 构建器映射补了模板 `BornAction` 列；在临时目录重建核对过：77 个模板文件里
  71 个与现有产物一致、6 个只多 `BornAction` 属性（另有 `city_CommonNpc.xml` 一行
  `sz466` 的任务列表多 `74580;`，是现有产物落后于客户端任务表的既有漂移）。重建后
  姿态只多 4 个，**这次不需要重做数据**。

## 七、代码索引

| 文件 | 职责 |
|---|---|
| `internal/npcfeed/pose.go` | 出生状态分层（`bornActionOf`）、静息态（`RestAction`）、动作名校验、`Motion.State` |
| `internal/npcfeed/join.go` | `Spawn.BornAction` / `Spawn.State` |
| `internal/npcfeed/patrol.go` | `Waypoint.Arrival`、`Motion.Arrival` |
| `internal/npcfeed/frame.go` / `resolve.go` | 物化帧 State/BornAction（0x28 值对、0x0D 头行与官方追加块） |
| `internal/entergen/alldest.go` | `NpcStateFrame(id, own, state)` |
| `internal/fssoloworld/npc.go` | `sampleSpawns` 标 State、巡逻拍状态、对话停走 |
| `tools/npcprobe/main.go` | 姿态统计 |
| `tools/datagen/npc/mapping.go` | 模板 `BornAction` 列映射 |
