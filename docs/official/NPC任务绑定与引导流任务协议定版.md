# NPC 任务绑定与引导流任务协议定版

> 对象：NPC 对话框的任务行下发与接取/提交受理（internal/fssoloworld/npctask.go +
> npcinteract.go）、任务数据链审计（npcgen/taskgen）、引导抓包任务协议
> （资料/初始账号登陆-完整引导-女.pcapng）、LZ47 解码链（tools/lz_exact.py）。
> 2026-11-23 定版；同日晚任务域改由客户端 `share/task` 树重生成（§一 订正）。
> 相关：《NPC服务端XML构建链与黑金样板.md》（NpcConfig 构建）、
> 《外服0x29批量预载通道解析.md》（登录簇 0x29/LZ 池种子）、《初始账号外服教程态
> 方言.md》（Task_Completed 位图）、2433 分支《fxnet2-LZ压缩方言全量定案.md》
> （lz_exact 算法权威）。

## 一、数据链审计结论（2026-11-23；任务域晚间重生成后订正）

| 数据 | 状态 | 规模 | 说明 |
|---|---|---|---|
| `ini/Quest/Task/Task.xml` | ✅ 重生成 | 15435 任务（14926 有客户端文案） | 客户端 `share/task/task/task_*.txt` → XML（npcgen；取代 BGserverCJ 2014 快照） |
| `ini/Quest/{StoryTaskList,SubFunction,HuntList,…}` | ✅ 重生成 | 15 域 + 3 根级散表 | 同一客户端任务域树并档：StoryTaskList 8430 行 / SubFunction 28327 行 / InteractiveList 12839 行 / Prize 10194 行（原空壳）等 |
| `ini/Npc/NpcConfig` + `NpcTemplate.xml` | ✅ 齐 | 77093 模板 + 6272 战斗行 | 76 表 + npctemplate*.tab 合并 |
| `ini/scene/*/npc_creator.xml` | ✅ 齐 | 405 场景 268507 实例 | 44653 唯一 ID 中 4223 个绑有 Task.xml 任务 |
| `ini/Text/TaskText.xml` | ✅ 齐 | 239672 键 | stringtask_normal + special 全收集 |
| `ini/Text/NpcTalk.xml` | ✅ 新增 | 52845 键 | string_npctalk/string_taskmenu/stringtalk 三散档（npcgen convertNpcTalk） |
| `taskdata/tasks.json` | 回退源 | 17534 任务 + 3226 绑定 + 2171 NPC | storytasklist/qianjiu_tasknpc.ini 派生（cmd/taskgen）——ini/Quest XML 缺席时才用 |

不可补的源缺口（非转换遗漏）：
- Task.xml 509 任务（3.3%）的 TitleId 在客户端文案表无键（无标题即无行文案），
  装载链不入目录；
- 27 个 NPC 引用（clonenpc_snailphone_task_01、jjc_* 竞技场 NPC 等）不在
  NpcConfig，影响 57 任务（0.4%）；
- m5 标题键 `task_WorldNpc01230`/`task_FuncNpc00107` 同样缺席（官服版本差异，
  键形正确时客户端解析失败仅标题空，窗照开）。

**任务 ID 体系（2026-11-23 晚订正）**：客户端 `share/task/task/task_*.txt`
（275 档入表，15435 任务，ID 40~87227）是任务主表正典——ID 与客户端文案键
（`title_<TID>`/`CompleteMenu_<TID>`…）、官服 0x29 记录帧同一体系：引导链
1340→1347→1341→1342→1343→1344→1345→1346→1648 的接/交 NPC、前后置、剧情组
（1340→story 482 五段）与 §4.2 解码表逐项吻合。此前 Task.xml 取自
BGserverCJ（2014 服务端快照）：4156 任务只覆盖客户端表的 2886 个同号 ID，
引导链等 12549 个任务缺席、1270 个 ID 客户端已无；同号任务中 886 个的接/交
NPC 或前后置已漂移，HuntList 同号 899 行仅 17 行一致。旧稿「两套任务 ID
体系不相交、须用 tasks.json」的结论作废——易误判点：表内 `Title` 列是过期的
设计期备注（与客户端文案七成不一致），拿它对文案会把同号任务误判成异任务；
显示名须按 `TitleId` 解析（npcgen 已回填进 Title）。`storytasklist` 派生的
tasks.json 降为 XML 缺席时的回退源。

## 二、运行时绑定（npctask.go）

数据装载（候选源序）：env `NINEYIN_TASK_SNAPSHOT` 显式快照 > 服务端任务域
XML `<数据根>/ini/Quest`（正典，2026-11-23 起：Task.xml 主表 Title 为 TitleId
解析的客户端显示文本，无文案任务不入目录；SubFunction(story)×StoryTaskList
合成剧情段）> 旧 JSON 快照 `<数据根>/taskdata/tasks.json` >
`<数据根>/../artifacts/local/tasks.json`（仓内 testdata 布局）；进程级缓存，
缺档降级不发任务行（不阻塞对话）。剧情段语义两条链一致——**每行即剧情一段**
（同 StoryID 多行按行序为段序，引导抓包 story_482_1..5 与源表 5 行互证；
JSON 链 taskdata 快照 v2 的 Binding 增剧情段字段且多段行全保留）。源表只在
一组剧情的首段行写 NpcID，续段行沿用组内上一行的 NPC（装载链续填——否则
引导链 1340 只剩第 1 段）。

- `npcTaskLines`：`Catalog.TasksForNPC(ConfigID)`（绑定且已加载文案的任务，
  TID 升序，同任务多剧情行只计一次）→ 行规则（2026-11-24 按官服三份抓包
  订正，见 §3.5）：
  - 已接 + 有剧情段 + 未走完 → 500 进行中行；
  - 已接 + 剧情走完/无剧情段 → 200 可提交行；
  - 未接 → **仅当任务有接取文案键**（主表 `CanAcceptMenu`）才发 100 接取行
    ——客户端没有该键时发出去只会显示原始键名（引导链任务全都没有）；
  - 已交（`roleState.TaskCompleted`）→ 不出行；
  上限 `NINEYIN_NPC_TASK_LIMIT`（缺省 8——客户端菜单每页 4 行翻页；<0 关停）。
- 链根「首谈即接」（`autoAcceptChainStarts`，`sendNPCTalkMenu` 算行之前）：
  无前置、无接取文案、且不是任何任务的 `AutoAcceptID` 目标的任务，绑定 NPC
  首次对话即由服务端接取并重放 0x29——官服引导链首环 1340@funnpcborn003031、
  车夫的 1648@Transcity05D 首次点击即进行中行。27509 虽无前置但是 1648 的
  链式目标，只随 1648 提交接入（官服车夫窗进行中行只有 1648）。
  建号种子（`StarterSet.xml` TaskGates）是「引导已完成」态（1346 已交、1344
  已接），1340 既未接也未交——旧实现因此发了接取行。
- `npcTaskMenuFrames`：m6 任务行，**双文本键**（首键=行文案，次键 title 作
  TextManager 格式化参数）：
  - 接取 `[100×10⁶+TID, "<主表 CanAcceptMenu 键>", "title_<TID>"]`
  - 提交 `[200×10⁶+TID, "CompleteMenu_<TID>", "title_<TID>"]`
  - 进行中 `[500×10⁶+TID, "story_menu_<sid>", "title_<TID>"]`
- mark 受理（handleNPCMenuSelect → handleNPCTaskMark）：
  - flag 100 接取：首次=推接取窗（见 §五.1）；同 TID 二次（pendingTask
    状态机）=真接取 → 写 TaskAccepted → 0x29 重放 → 重发菜单（行翻 500）；
  - flag 200/201 提交（201=奖励单选二次确认，官方恒随 200 后到达）：
    `completeNPCTask`＝移出已接+清剧情态+记 `TaskCompleted`+按 `AutoAcceptID`
    链式接取下一环 → 0x29 重放 → **m8×2 关窗**（官方口径，不刷新菜单）；
  - flag 500 进行中：发剧情对话窗第 1 段；
  - flag 50N 剧情续：发段 N+1；链尾=剧情完成+m8×2——`AutoSubmit` 任务
    （1340：官服 505 点击后直接奖励+关窗、随后上报 m327 "0,1347"）走完即
    `completeNPCTask` 自动提交并链推进；其余置 `Player.taskStoryDone`，
    任务行翻 200 待提交；
  - flag 60N 剧情离：m8×2 关窗；
  - 300/400/700 留痕。
- 0x29 重放走 `entergen.TaskAcceptedBatchFrame`（m327 sub6 同通道，全量幂等）。
- 回归锚点：`TestNPCGuideDialogMatchesOfficial` 把 funnpcborn003031 主窗五帧
  （m4/m5/m6 进行中/m6 ui_exit/m7）与官服引导抓包 98.16s 的明文逐字节比对。

## 三、引导抓包任务协议（女号完整引导流，8032 world 连接）

抓包拆解：`tools/pcapsplit <pcapng> 8032` → EE 去转义 + XOR 0xFA6D3BCC 解帧
（c2s 3809 / s2c 13272 帧，任务相关事件 461 条）。工具与产物留存
`tmp/guide_f/`（parse_tasks.py / timeline.py / taskline.py）。

### 3.1 总体形态

- 引导任务**全部由服务端自动接取**：全程无 100 接取域 mark、无 m327、无
  task_msg 字符串路由——接取窗/提交窗均由客户端从 REC_TASK_ACCEPT 记录
  **本地渲染**，服务端只做 0x29 记录帧状态同步。
- 任务链：**1340→1347→1341→1342→1343→1344→1345→1346→1648**（引导结束进
  千灯镇，见 §4.2 时间线）。
- 双窗模式：NPC 问候窗（m5 `task_<ConfigID>`，仅离开行）+ 紧接剧情窗
  （m5 `DialogID_<sid>` / `story_<sid>_N`，行=任务行+okmenu/离开）。

### 3.2 菜单任务行域表（m6 funcid = flag×10⁶+TID，form_talk_data.lua g_menu_id）

| flag | 语义 | 行文本键（按序） |
|---|---|---|
| 100 | 接取（多样本抓包实证 `[100081264, "CanAcceptMenu_81264", "title_81264"]`、`[100000616, …]`；只对有 CanAcceptMenu 文案的任务发） | `CanAcceptMenu_<TID>` + `title_<TID>` |
| 200 | 可提交（任务目标完成后出现） | `CompleteMenu_<TID>` + `title_<TID>` |
| 201 | 提交+奖励单选（lua `funcid+singleselect*base`，恒随 200） | — |
| 500 | 进行中（官方已接任务行） | `story_menu_<sid>` + `title_<TID>` |
| 50N/60N | 剧情第 N 段续/离（N=1..5） | `storymenu_<sid>_N` / `9607` |
| 600 | 离开 | `ui_exit` / `ui_form_talk_leave` |

npc.pcapng（主城，玩家无任务在身）佐证：任务域行一条不发，全是服务行
（805 商店/807 仓库/898 通用/600 离开）——**服务端只按玩家实际任务状态发任务行**。

### 3.3 提交流帧序（TID=1347 实证）

```
S2C  m4 / m5 "task_funnpcborn003035" / m6 [200001347, "CompleteMenu_1347", "title_1347"]
     / m6 [600000000, "ui_exit"] / m7          ← 可提交主菜单
C2S  op07 mark=200001347                        ← 点"完成任务"行（提交窗客户端本地开）
C2S  op07 mark=201001347                        ← 提交窗选奖励后确认
S2C  0x29 记录帧（任务状态迁移 + 链推进自动接取下一环）
S2C  m8 ×2                                      ← 关窗（无菜单刷新）
```

剧情对话流（TID=1340）：主窗 500 行 → 点后 5 段剧情窗（story_482_1..5，每窗
50N 续行 + 60N "9607" 离行）→ 末段 m7 后 0x29 推下一环。

### 3.4 文本键数据归属

| 键型 | 归属（全覆盖验证 2026-11-23） |
|---|---|
| title_/CompleteMenu_/CanAcceptMenu_/story_menu_/storymenu_/story_/DialogID_/okmenu_ | TaskText.xml（stringtask_normal+special） |
| task_<ConfigID>（NPC 问候标题） | NpcTalk.xml（string_npctalk，23356 键） |
| ui_form_talk_leave 等 ui_* | 客户端本地 interface.idres（服务端发键即可） |

### 3.5 多样本对话窗（npc-点击对话.pcapng，7072，81 窗）

拆解工具：`tools/pcapsplit <pcapng> 7072` → `tools/pcapevents … dumpall` →
`tmp/cf/dialogs.py`（以 S2C m4 为界切窗，输出 m5 标题 + m6 行清单）→
`tmp/cf/join.py`（联客户端 npcconfig 模板列）。

**「赠送物品 / 侠恶查询」通用行（898000024 / 898000023+1000）不是每窗必带**——
只在模板 `Character`（善恶参数）有值且 `KarmaLevel`（地位评分）≥1 的 NPC 上
出现（世界 NPC、钱庄、部分商铺）；引导 NPC、车夫驿站、生活职业、多数商店
都没有。两行可独立出现（WorldNpc01081 只有侠恶查询，唯一单行样本，未找到
区分列，现两行同发）。捕快类 NpcType 235/236 虽有业报值但走专属行
（`1000 ui_apply_police` / `2000 ui_query_arrest`）。例外样本：FuncNpc00644
（114，业报 1/2）在车夫抓包（8031，另一角色）里未发两行——疑为角色侧条件
（新号/侠恶系统未开），未建模。实现＝`npcKarmaDialog`（npcgen 已把两列折进
NpcConfig；客户端自己按本地 npcconfig 的 Character 列做关系界面，不读服务端
下发的这两个字段值）。

| NPC | NpcType | Character | KarmaLevel | 官服行 |
|---|---|---|---|---|
| WorldNpc01230 常鞅 / 01231 / 01240 | 96 | 1 | 3 | 898×2 + 离开 |
| WorldNpc01108 | 96 | 1 | 1 | 898×2 + 离开（KarmaLevel=1 即有） |
| WorldNpc01635 | 96 | 4 | 5 | 接取行 100081264 + 898×2 + 离开 |
| FuncNpc00107 钱百魁 | 67 | 1 | 3 | 807 仓库×2 + 898×2 + 离开 |
| FuncNpc00102 / 00101 / 00111 | 63/62/114 | 1 | 3/3/2 | 805 商店 + 898×2 + 离开 |
| WorldNpc01081 易吉 | 96 | 1 | 3 | 仅 898000023 + 离开 |
| WorldNpc01244 / 01245 | 235/236 | 1 | 6 | 1000/2000 捕快专属行 |
| Transcity05D 车夫 | 69 | - | 0 | 900 骑乘 + 805 + 进行中行 + 830 传送×2 + 离开 |
| JobSuZ002 | 123 | - | - | 806 生活×5 + 897 + 899 + 805 + 离开 |
| yqd_npc_001 / hd_dw_001 / JinybNpc001 | 61/95/60 | - | -/0 | 805 / 805 / 2300 + 离开 |

其它行形态（待接，现仅记录）：服务行文本键多为 `ui_buy<ConfigID>`（NPC 专属
键）而非通用 `ui_shop`；驿站 `830100000 ui_trans_scene 0` + `830000000
ui_trans_city 0`；闲聊 `890000895 ui_gossip_talk`；二级窗（点服务行后）
`m4 + m6 [2, "ui_return"] + m7`，YD_017 的层级菜单用 `841…` 域并以父级
funcid 作返回行；m5 标题在有 `NpcTalkRule` 的 NPC 上可为 `talk_<规则>_<n>_<m>`；
交互步骤窗＝「问候窗（仅 ui_exit）+ `DialogID_<步骤ID>` 窗（`[500×10⁶+TID,
okmenu_<步骤ID>]` + `ui_form_talk_leave`）」双窗连发，玩家点一次推进一步。

## 四、0x29 记录帧解码（LZ47）

### 4.1 解码器

**2433 分支 `tools/lz_exact.py` 为终版**（fxgame.DMP 运行时抠取 fxnet2_rebuilt.dll
逐指令反汇编定案，2026-11-15；取代旧 lzdec.py——后者三处语义偏差仅 81% 通过率）。
已落地本分支 tools/ 同路径。要点：

- mode 0x47=LZ+9B 头（`[op][0x47][帧总长 u32][outlen u32][位流]`）；
- flags u32 LSB 先消费，`==1` 重填；**bit0=1 是 match 回引**（peek u32：
  idx=(v>>4)&0xfff 12 位池索引，nib=v&0xf 长度——≠0 时 len=nib+2 消费 2B，
  =0 时 len=byte[+2] 消费 3B）；
- literal：n=flags&0xf 查 T=[4,0,1,0,2,0,1,0,3,0,1,0,2,0,1,0]，**先写 peek 的
  u32 共 4 字节**（溢出字节参与后续 hash 注册）；
- 全局池 4096 槽（.data+0x32F0），hash=((u32@pos>>12)^u32@pos)&0xfff；
  **官服 S2C 编码器跨帧复用池** → 必须会话模式（`Session().decode`）按线序
  喂全部 0x27-0x30 族帧；
- 尾段（write≥outlen-0xb）逐字节 1 bit，重填=跳过 u32、flags=0x80000000。

本仓发送端 `entergen.WrapAs47Literal` 的全字面量编码是官方解码器接受的
安全子集（4bit 型 0010/0100/1000/0000 ↔ T[2]/T[4]/T[8]/T[0]，bit0 恒 0）。

### 4.2 官方 0x29 任务记录帧明文布局（本抓包定版）

```
[0x00][recIdx u16][obj u32][own u32][row u16][rows u16] + 行体
```

（与 entergen.recordBatch0x29 的 29B 头方言不同——现行官服 15B 头；两者客户端
均可解，2433 实证。）行体 Task_Accepted 列型同构（recIdx 本流=243，2433 快照表
idx=29 Task_Accepted）：

```
TID i32 | 步骤 i32 | "Server" 串 | 奖励id i32 | NPC 串 | title 串 | …int 列… |
接取时间戳 i32（Unix） | target 串 | context 串 | NPC 串 | i32 | SubmitNpcEx 串 |
prize 串（"钱,0,repute_jianghu,声望,…,物品,数量,…,5,5000,0,0"） | 尾 int 列
```

引导链解码结果（41/41 帧 fail=0，含丢包重同步点之后）：

| TID | NPC | 接取时刻 | 奖励 |
|---|---|---|---|
| 1340 | funnpcborn003031 | 00:51:58 | 1000 钱 + 声望 10 |
| 1347 | funnpcborn003035 | 00:53:07 | 1000 钱 + 声望 20 |
| 1341 | funnpcborn003036 | 00:53:53 | 5000 钱 + 声望 80 |
| 1342 | funnpcborn003032 | 00:55:13 | 2000 钱 + 声望 40 |
| 1343 | funnpcborn003033 | 00:56:34 | 5000 钱 + 声望 250 |
| 1344 | funnpcborn003033 | 00:58:06 | 3000 钱 + 轻功书 book_qinggong_7_fc |
| 1345 | funnpcborn003037 | 01:01:57 | 10000 钱 + additem_0020×2 |
| 1346 | funnpcborn003034 | 01:02:18 | 20000 钱 + additem_0011 + 宝箱 |
| 1648 | Transcity05D | 01:06:19 | 声望 20 + 坐骑 mount_2_001yd |

（注：早期启发式字节扫描曾把 #2425 误读为 1342，LZ 精确解码实为 1341——
启发式结论须以 lz_exact 解码为准。）

## 五、遗留处置（2026-11-23 第二轮，全部落定）

1. **接取行（100 域）官方形态**：2026-11-24 多样本抓包（§3.5）实证菜单行
   `[100×10⁶+TID, "CanAcceptMenu_<TID>", "title_<TID>"]`，且**只对有
   CanAcceptMenu 文案的任务发**（81264/616 两个样本；点击后的接取窗帧序
   仍未采到——样本里玩家直接点了离开）。接取窗本身仍按客户端 lua 链路
   推断实现并标注待真机验证：
   首次 100=服务端推接取窗（`KeyedMessageFrame("task_msg", 1, npc, funcid,
   task_id, "context_<TID>", "task_menu_accept_<TID>", prize×3 空串, 0,0,0,
   "", title 富文本, 0, 0)`——show_accept_form 15 参族）；窗内「接受任务」
   行同 funcid 二次到达（菜单簿记 pendingTask 状态机）=真接取。entergen 新增
   `KeyedMessageFrame`（0x1E 字符串路由）。
2. **500 进行中行 + 最小进度系统**：已实现——storytasklist **每行即剧情一段**
   （同 StoryID 多行、行序即段序；taskdata v2 快照 bindings 3226→6350 多段
   全保留）。已接+有剧情段+未走完=500 行（双键 story_menu_<sid>+title_<TID>）；
   剧情走完/无剧情段=200 可提交行；`Player.taskStoryDone` 布尔进度态（连接
   生命周期，不落库）。
3. **剧情对话窗**：已实现（抓包帧族逐字节复刻）——500 mark 发段 1 窗
   （m4/m5 "story_<sid>_N"/m6 [50N×10⁶+TID, "storymenu_<sid>_N"]/m6
   [60N×10⁶+TID, LeaveTextID‖"9607"]/m7）；50N 发段 N+1；链尾（段数+1）=
   剧情完成+m8×2 关窗；60N 中断=m8×2 关窗（未实证，按离开语义）。
4. **接取窗参数族**：已实现（见 1）；奖励三槽无数据发空串（prize 数据源
   在官方 0x29 帧体/BGserverCJ Award.xml，未接）。
5. **15B 头对齐：不做**——recordBatch0x29 现行 29B 头方言客户端实证可解
   （四连崩根因是池回引非头布局），对齐需动 Task_Accepted 全通道（登录
   挂载行推/m327 sub6/批量重放）+ golden 测试，收益（字节级对拍）小于风险。

**新遗留**：接取窗/剧情窗真机验证（需 fork 客户端联调）；接取窗奖励预览
（prize 数据接入——Prize 域已随客户端树出档，Prize.xml 10194 行）；
DialogID_<步骤>/okmenu_<步骤> 双窗交互步骤流（第二窗紧随问候窗，点击一次
推进一步——1347 的四个交互步骤 558~561；需按 SubFunction 行序建步骤游标，
npctask 现无步骤进度，无剧情段任务接入后直接出 200 提交行）；杀怪步骤
（hunter，1341/1343/1346）需战斗事件接线；接取行前置门（PreTaskLimit——
已交历史不可追溯，现未校验）。已落定（2026-11-24）：链推进（AutoAcceptID
——提交后自动接取下一环，1340→1347、1648→27509）、链根首谈即接、
AutoSubmit 剧情走完即交、通用行按业报门控。引导链 NPC（funnpcborn00303x）
摆放在 born_single_01（场景 300）与 adv056，不在出生场景 born03。
真机悬案：`story_menu_482` 在用户客户端显示为原始键名——我方该组 m4/m5/
m6/m7 与官服抓包逐字节一致、文案键也在客户端语言包里（stringtask_special_
yd_01.idres），非服务端帧差异；待确认官服上同一行是否同样显示键名。
