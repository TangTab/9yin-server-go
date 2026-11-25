# NPC 任务绑定与引导流任务协议定版

> 2026-11-26 更新：真机反馈「接取后目标 NPC 不见了、物品也没有给」——查出接取影像
> （m104）是一场客户端剧情副本，客户端进副本会清场景对象、官服在影像开始 0.6 秒后
> 重发视口 NPC，本服此前只发影像不重发（§六·接取影像）；任务奖励（步骤物品、完成
> 奖励的碎银/物品/声望/历练）落地（§六·任务奖励）。未提交未部署，待真机验证。
>
> 2026-11-25 更新：任务记录同步层落地（§六）——客户端三张任务记录表按官服帧序
> 行级同步（接取/步进/完成/追踪回置/NPC 任务标记），§二/§三的运行时规则随之
> 重写；订正 §3.1「全程无 m327」、§3.3 提交帧序、§4.2 的 0x29 内层头布局。未提交
> 未部署，待真机验证。

> 2026-11-24 更新：任务目录只从服务端任务域 XML（`ini/Quest`）装载。旧版 JSON 快照
> `taskdata/tasks.json`、它在运行时的回退与 `NINEYIN_TASK_SNAPSHOT` 环境变量、以及生成它的
> task 构建器均已移除；§一里涉及快照链的描述是定版当时的记录。


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
  TID 升序，同任务多剧情行只计一次）→ 行规则按**玩家进度**（2026-11-25，
  进度取 `roleState.TaskTracks`，见 §六）：
  - 已接且当前步是本 NPC 的剧情步 → 500 进行中行（`story_menu_<剧情组 ID>`）；
  - 已接且可提交（各步记满且非 `AutoSubmit`；没有步骤数据的任务接取即可
    提交）→ 200 提交行；
  - 已接的交互步**不出主窗行**：点 NPC 时主窗之后追加步骤确认窗（下）；
  - 未接 → **仅当任务有接取文案键**（主表 `CanAcceptMenu`）才发 100 接取行
    ——客户端没有该键时发出去只会显示原始键名（引导链任务全都没有）；
  - 已交（`roleState.TaskCompleted`）→ 不出行；已在已接表但没有进度行的
    （建号种子的功能门行）→ 不出行；
  上限 `NINEYIN_NPC_TASK_LIMIT`（缺省 8——客户端菜单每页 4 行翻页；<0 关停）。
- 步骤确认窗（`npcTaskStepWindowFrames`，`sendNPCTalkMenu` 在主窗帧之后连发）：
  任务当前步是本 NPC、带对话的交互步时追加第二个对话窗
  `m4 / m5 "<DialogID>" / m6 [500×10⁶+TID, "okmenu_<步骤号>"] / m6 [600000000,
  "ui_form_talk_leave"] / m7`（官服 103.635s：问候窗只有 ui_exit，紧接
  `DialogID_558` 窗）。无对话的交互步（`DialogID` 空，如 1346 的事件步 568）
  不出窗。
- 链根自动接取（`autoAcceptChainStarts` / `npcAcceptSceneGuideRoots`）：无前置、
  无接取文案、且不是任何任务的 `AutoAcceptID` 目标的任务。两种触发：进场景后
  延时 5 秒接取「接取 NPC 摆放在当前场景」的新手引导链根（官服 1340 在进场约
  15 秒后自动接取）+ 首谈即接（车夫抓包：1648@Transcity05D 首次点击即进行中
  行）。27509 虽无前置但是 1648 的链式目标，只随 1648 提交接入。
- `npcTaskMenuFrames`：m6 任务行，**双文本键**（首键=行文案，次键 title 作
  TextManager 格式化参数）：
  - 接取 `[100×10⁶+TID, "<主表 CanAcceptMenu 键>", "title_<TID>"]`
  - 提交 `[200×10⁶+TID, "CompleteMenu_<TID>", "title_<TID>"]`
  - 进行中 `[500×10⁶+TID, "story_menu_<剧情组 ID>", "title_<TID>"]`
- mark 受理（handleNPCMenuSelect → handleNPCTaskMark）：
  - flag 100 接取：首次=推接取窗（见 §五.1）；同 TID 二次（pendingTask
    状态机）=真接取 → 接取簇（§六）→ 重发菜单；
  - flag 200 提交：任务可提交时**只回提交窗参数**
    `m209 [2, NPC, 201×10⁶+T, T, "complete_T", "submit_T", 0, "", Line]`，客户端
    弹提交窗；flag 201（窗内确认）= 完成簇（§六）；目标未完成的提交被拒；
  - flag 500：任务当前步是本 NPC 的交互步=确认完成该步；是剧情步=剧情窗第 1 段；
  - flag 501..599 剧情续：窗 N 的续行点击发窗 N+1；末窗续行=剧情步记满进度
    （末步且 `AutoSubmit` 即进完成簇，其余置可提交并紧接提交窗参数）；
  - flag 601..699 剧情离：关窗；
  - 300/400/700 留痕。
- 回归锚点：`TestNPCGuideDialogMatchesOfficial` 把 funnpcborn003031 主窗五帧
  （m4/m5/m6 进行中/m6 ui_exit/m7）与官服引导抓包 98.16s 的明文逐字节比对；
  `TestTaskSyncOfficialOrder` 把 1340→1347→1341 的接取簇、剧情窗五段、完成簇、
  步骤确认窗、追踪回置、提交窗逐帧摘要与官服时间线对拍；
  `TestTaskSyncGuideChainToEnd` 模拟客户端从 1340 走到链尾 1346。

## 三、引导抓包任务协议（女号完整引导流，8032 world 连接）

抓包拆解：`tools/pcapsplit <pcapng> 8032` → EE 去转义 + XOR 0xFA6D3BCC 解帧
（c2s 3809 / s2c 13272 帧，任务相关事件 461 条）。工具与产物留存
`tmp/guide_f/`（parse_tasks.py / timeline.py / taskline.py）。

### 3.1 总体形态

- 引导任务**全部由服务端自动接取**：全程无 100 接取域 mark、无 task_msg 字符串
  路由——接取窗/提交窗均由客户端从任务记录**本地渲染**，服务端只做记录帧状态
  同步（§六）。客户端每接一环上报一次追踪列表 `m327 [6,"0,<TID>"]`（lua 拼串
  里混进的 0 占位忽略；2026-11-25 订正：旧稿「全程无 m327」有误）。
- 任务链（官服时间线，2026-11-25 订正）：
  **1340→1347→1341→1342→1343→1344→1348→1349→1597→1345→1346**，引导结束
  （1346 无后继）；进城（场景 17）时进场自动接取 1648（车夫链根）与 34096。
- 双窗模式（交互步）：NPC 问候窗（m5 `task_<ConfigID>`，仅离开行）+ 紧接
  **步骤确认窗**（m5 `DialogID_<步骤号>`，行=`[500×10⁶+TID, okmenu_<步骤号>]`
  + `ui_form_talk_leave`），玩家点一次推进一步。剧情步走主窗的 500 行进
  `story_<组 ID>_<段>` 窗（§3.2）。

### 3.2 菜单任务行域表（m6 funcid = flag×10⁶+TID，form_talk_data.lua g_menu_id）

| flag | 语义 | 行文本键（按序） |
|---|---|---|
| 100 | 接取（多样本抓包实证 `[100081264, "CanAcceptMenu_81264", "title_81264"]`、`[100000616, …]`；只对有 CanAcceptMenu 文案的任务发） | `CanAcceptMenu_<TID>` + `title_<TID>` |
| 200 | 可提交（任务目标完成后出现） | `CompleteMenu_<TID>` + `title_<TID>` |
| 201 | 提交+奖励单选（lua `funcid+singleselect*base`，恒随 200） | — |
| 500 | 进行中（已接任务行）／步骤确认窗的确认行 | 主窗 `story_menu_<组 ID>` + `title_<TID>`；确认窗 `okmenu_<步骤号>` |
| 50N/60N | 剧情窗 N 的续/离行（N=1..99）——**窗 N 的续行点击发窗 N+1，末窗续行=剧情步完成** | `storymenu_<组 ID>_N` / `9607` |
| 600 | 离开 | `ui_exit` / `ui_form_talk_leave` |

（2026-11-25 订正：旧实现把续行 50N 点击当成「发窗 N」，点一次重发同一窗、
剧情永远走不完——这是「对话正常但接不了任务」的直接病因之一。）

npc.pcapng（主城，玩家无任务在身）佐证：任务域行一条不发，全是服务行
（805 商店/807 仓库/898 通用/600 离开）——**服务端只按玩家实际任务状态发任务行**。

### 3.3 提交流帧序（TID=1347 实证，2026-11-25 按净时间线订正）

```
S2C  m4 / m5 "task_funnpcborn003035" / m6 [200001347, "CompleteMenu_1347", "title_1347"]
     / m6 [600000000, "ui_exit"] / m7          ← 可提交主窗（146.728s）
C2S  op07 mark=200001347                        ← 点"完成任务"行
S2C  m209 [2, NPC, 201001347, 1347, "complete_1347", "submit_1347", 0, "", 9]
                                                ← 提交窗参数，客户端弹窗（147.593s）
C2S  op07 mark=201001347                        ← 提交窗内确认（148.054s）
S2C  完成簇（§六）：奖励提示 → 完成位图 → 删目标行/已接行 → m209 [8,1347] →
     m1 "9631" → 1341 接取簇 → m24 "task_complete" → m18 [NPC,""] → m8
```

剧情对话流（TID=1340）：主窗 500 行 → 点后 5 段剧情窗（story_482_1..5，每窗
50N 续行 + 60N "9607" 离行）→ 末窗续行（505）点击=剧情步记满，末步自动提交
进完成簇（m8 单发关窗）。

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
[0x00][obj u32][own u32][recIdx u32][rows u16] + 行体（逐行拼接，无长度前缀）
```

（2026-11-25 订正：旧稿记作 `[0x00][recIdx u16][obj u32][own u32][row u16]
[rows u16]`——那是把 obj 的低 16 位 0x00f3 误读成了「recIdx=243」。净时间线
解码：`#1538 op=29 rec=29 rows=1 id=1340`，每环接取各一张单行批；与
entergen.recordBatch0x29 的 29B 头方言不同——两者客户端均可解，2433 实证。
`tmp/cf/lz29.py` 按 `<BIIIH` 解全部 0x27..0x30 族 LZ 批。）

行体 Task_Accepted 28 列（客户端记录表列型，`internal/entergen/taskrecords.go`
逐列编码并与官服行体逐字节对拍）：

```
id i32 | line i32 | "Server" | submit_scene i32 | submit_npc 串 | title 串 |
status i32 | track i32 | step i32 | time i64 | plot i32 | can_giveup byte |
target 串 | context 串 | accept_npc 串 | accept_scene i32 | submit_npc_ex 串 |
prize_base 串 | prize_ex 串 | prize_over 串 | accept_limit 串 | submit_limit 串 |
repute_limit 串 | vip byte | sceneid i32 | prize_cond 串 | 0 i64
```

官服行里 status/track/step/time 是玩家进度（接取时 0/1/0/Unix 秒，随后 0x13
写 step=1）；prize_base 形如 `"1000,0,repute_jianghu,10,…,5,5000,0,0"`。

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
2. **500 进行中行 + 进度系统**：2026-11-25 由任务记录同步层取代——进度是
   持久化的 `roleState.TaskTracks`（追踪标志/当前步序/各步已记次数），菜单
   行、步骤确认窗、NPC 任务标记全由它推导（§六）；旧的连接期
   `Player.taskStoryDone` 布尔态已删。
3. **剧情对话窗**：已实现（抓包帧族逐字节复刻）——500 mark 发段 1 窗
   （m4/m5 "story_<组 ID>_N"/m6 [50N×10⁶+TID, "storymenu_<组 ID>_N"]/m6
   [60N×10⁶+TID, LeaveTextID‖"9607"]/m7）；50N 发段 N+1（2026-11-25 订正：
   旧实现差一位、点了重发同一窗）；末窗续行=剧情步记满进度（末步自动提交
   进完成簇，否则置可提交并紧接提交窗参数）；60N 中断=关窗（未实证，按离开
   语义）。
4. **接取窗参数族**：已实现（见 1）；奖励三槽无数据发空串（prize 数据源
   在官方 0x29 帧体/BGserverCJ Award.xml，未接）。
5. **15B 头对齐：不做**——recordBatch0x29 现行 29B 头方言客户端实证可解
   （四连崩根因是池回引非头布局），对齐需动 Task_Accepted 全通道（登录
   挂载行推/m327 sub6/批量重放）+ golden 测试，收益（字节级对拍）小于风险。

**新遗留**：接取窗/剧情窗/步骤确认窗真机验证（需 fork 客户端联调）；奖励里
未建模的部分不发也不提示（官银 CapitalType2 等其余奖励串字段、1345/1346 完成
时的真气/经脉/内功类奖励簇）；杀怪/特殊/事件交互步只能占位推进（§六）；接取行
前置门（PreTaskLimit——已交历史不可追溯，现未校验）；新号初始物品未做（官服
新号首次登录的就绪段里下发 35 条物品行：轻功书 zs_normal_00~14、16 共 16 本、
CS_jh_cqgf01/04/06/04_sky 4 本、taolu_zhenfa_wzyx/cfxz、CS_light_rad_81、布衣
三件、box_wnhd_dllb01/box_card_310/box_guj_20200808_001/box_20220401_mpwjb_fc_001
4 个盒子——书与武学类在我方物品目录里缺席，只有 4 个盒子能走现有入包链；本服
新号背包为空）。
已落定（2026-11-24）：链推进（AutoAcceptID——提交后自动接取下一环）、链根
自动接取、AutoSubmit 目标达成即交、通用行按业报门控。已落定（2026-11-25）：
任务记录行级同步、DialogID 双窗交互步骤流、200→提交窗→201 结算、追踪列表
回置、NPC 任务标记（§六）。已落定（2026-11-26）：接取影像的视口重发与结束
收尾、任务奖励发放（§六）。引导链 NPC（funnpcborn00303x）摆放在
born_single_01（场景 300）与 adv056，不在出生场景 born03（烟雨庄）。
真机悬案：`story_menu_482` 在用户客户端显示为原始键名——我方该组 m4/m5/
m6/m7 与官服抓包逐字节一致、文案键也在客户端语言包里（stringtask_special_
yd_01.idres），非服务端帧差异；待确认官服上同一行是否同样显示键名。

## 六、任务记录同步协议（2026-11-25 定版，internal/fssoloworld/taskrecord.go）

净时间线解码器 `tmp/cf/tasksync.py`（按 TCP seq 增量重组 + 帧内 `EE 00` 转义
还原 + XOR；早期按抓包序拼接的解码会因子段乱序到达把帧序打乱——曾误读出
「提交确认先于完成簇发下一环通知」「34 字节掩码 op07」两个假象，均已订正）。
玩家进度三态：`roleState.TaskAccepted`（已接序＝客户端已接表行号）、
`roleState.TaskTracks`（每个流程接取任务的追踪标志/步序/各步已记次数）、
`roleState.TaskCompleted`（完成位图来源）。建号种子的功能门行
（`SeedTaskIDs`：1069~1073、1344、1555~1562、1140~1147、51002）在已接表里
但没有进度行——不是玩家接的任务，不出行不推目标行；链推进遇到它们按
「已接表已有该行」处理（1344：目标行照发，已接行逐格写追踪/接取时刻/步序），
已交的（1346）照常重新接取。

客户端行布局（服务端按同一规则推算行号；重登簇 `entergen.taskRecordRows`
同源，测试 `assertRecordLayout` 对拍）：

- `Task_Accepted`：行号＝已接序下标；
- `Task_Record`：每个有进度行的已接任务占「步骤数」行，按已接序拼接。

事件簇（帧序即下发序；奖励与 NPC 标记见下）：

```
接取 T      0x11 Record row=start+i × n（零进度行）
         → 0x11 Accepted row=a（已接表已有该行则改逐格写 track=1、time）
         → 0x13 step=1 →（首步杀怪）0x13 Record 首行 flags=1
         → NPC 标记 → m209 [9,T] → m24 [玩家,40,"task_accept"] → m1 [1,"9628","title_T"]
            （官服的已接行走 LZ 0x29 单行批，小行明文 0x11——等价，本服恒明文）
            （官服同时置 CantMove/CantBeAttack=1 冻结玩家；T 有 AcceptMovie 约
              半秒后 m8 + m104 [0/0, 影像号, 4]——见「接取影像」）
客户端     m327 [6,"0,T"] → 列表内的已接行连写 0x13 track=0、track=1
一步完成    （记满且有步骤奖励）入包帧 → m1 [1,"9645","<物品>",<数量>]
         → 0x13 curnum →（非末步）0x13 step=k+1 ／（末步且非自动提交）0x13 status=2
         → 0x13 flags=32+h → 0x13 flags=48+h →（下一步杀怪）0x13 下一行 flags=1
         → NPC 标记 → m24 "task_ok" → m18 [NPC,""] →（末步非自动提交）m209 提交窗参数 → m8
            h＝杀怪步带位 1（写 33/49）；杀怪步不关对话（无 m8）
            末步且自动提交：不发 step/status/flags 写与 task_ok，curnum 后直接进完成簇
完成 T      奖励（见「任务奖励」）→ 完成位图（客户端没有的位图行先 0x11 加零值
            行，再 0x13 写格；行＝T/32、位＝T%32）→ 0x12 Record row=start × n
         → 0x12 Accepted row=a
         → m209 [8,T] → m1 [1,"9631","title_T"] → NPC 标记
         → 后继任务接取簇（AutoAcceptID）→ m24 [玩家,40,"task_complete"]
         → 对话中：剧情窗/交互确认窗 m8，提交窗确认 m18 → m8；杀怪无
```

NPC 任务标记 `m209 [15, obj, state]`（引擎侧消费，NPC 头顶任务图标；位义按
官服引导抓包各簇取值归纳）：1＝当前步在此 NPC、2＝可提交／任务待回此 NPC
（当前步在别处，与 1 同现＝3）、4＝此 NPC 有可接取的任务、8＝当前步是此 NPC
的剧情步（与 1 同现＝9）。只发与上次不同的 NPC（首推同官服：视口内每个 NPC
各一帧）；采集对象的标记 100 未接。

占位推进：服务端没有 NPC 战斗/伤害、打坐/经脉、场景事件这些通路，杀怪步
（1341/1343/1346/1349）、特殊步（1342 的打坐 2061、1348 的经脉 2091）、无对话
的事件交互步（1346 的 568）无从达成——`NINEYIN_TASK_STUB=guide`（缺省，只
占位新手引导链任务）｜`all`｜`off`：特殊步到步即过，其余每 1.5 秒记一次进度
（帧序同官服击杀）。

进场：`normalizeTaskTracks` 给旧流程只记了任务号的已接任务补进度行（旧流程
没有目标行/追踪标志，任务行也不出）；`npcTaskEnterScene`（NPC 物化后）补发
标记并延时 5 秒接取本场景的引导链根。引导链根自动接取只认 `IsGuide`
（SearchID `s980301`）且接取 NPC 摆放在当前场景的任务——全表链根 7543 个
（场景 17 就有 415 个），官服靠未建模的 AcceptLimit 等条件筛选，不能全接。

### 接取影像（2026-11-26，internal/fssoloworld/taskmovie.go）

主表 `AcceptMovie` → 接取约半秒后 `m8` + `m104 [0/0, 影像号, 4]`（1340→473、
1341→474）。这不是一小段动画，而是一场**剧情副本**：客户端
`ini/movie/movie_clone003_<号>.ini` → `scenario=clone003\<脚本>.ini`（473
「开场」把玩家变成 NPC 演员放在副本地点、敌人按脚本表演、`esc_key=0` 禁跳过、
`restore_pos=1` 结束传回；官服实录 473 约 27 秒、474 约 5 秒）。客户端进副本
会清掉场景对象——官服在影像开始约 0.6 秒后把视口内 NPC 的物化帧整批重发
（33.486 m104 → 34.095 六帧 0x28，与进场首推同一组、隔 1 秒再补第 7 个），玩家
回到原位时 NPC 还在。本服此前只发影像不重发 → 副本结束 NPC 全没了（2026-11-26
真机「接取任务后目标 NPC 不见了」）。

```
官服  T+0.0   接取簇，同时 0x10 CantMove=1、0x10 CantBeAttack=1（#2439/#2440）
      T+0.5   m8 → m104 [0/0, 影像号, 4]（#1559/#1560）→ FinishFreshmanRecStr
      T+1.1   m98 145 → 视口 NPC 0x28 重发
      副本中   客户端照常每 6 秒上报站立态位移（坐标即出生点），约 +1.9 秒发
               m440、m431 [obj 0/0]
      结束     C2S m806 [影像号] → 0x10 CantBeAttack=0 → 0x10 CantMove=0 → m107
               （#1789..#1791）→ 客户端报 m950 [0,"<新手引导步>"] → 0x10 HelpStepID
本服  不冻结（客户端不上报结束也不会卡住）；副本期间位移不收编（等价官服冻结时的
      服务端口径），行走态位移（state 7）视作玩家已离开副本、立即收尾并照常收编；
      影像在途态 3 分钟超时兜底；自动步（占位推进）在影像期间顺延、结束后接着走
      （官服 1341 的怪是 474 影像结束后才刷的）。影像只在进场自动接取与完成簇内的
      链式接取后播，对话中接取不播（半秒后的 m8 会把刚开的对话窗关掉）。
开关  NINEYIN_TASK_MOVIE=off 不发影像（彻底不进副本）；
      NINEYIN_TASK_MOVIE_REPUSH_MS 重发延迟（缺省 600；0＝不重发）。
未做  官服接取时的 m24 [玩家,56]、0x31 对象列表、FinishFreshmanRecStr 与 HelpStepID
      （新手引导 UI 步骤链 m950）、副本后 scripted 刷怪（1341 的村民/怪物）。
```

### 任务奖励（2026-11-26，internal/fssoloworld/taskreward.go）

取数面 `internal/taskdata/prize.go`，帧编码 `internal/entergen/taskrewards.go`
（原字节对拍官服引导抓包）。只发已到账的奖励：物品目录里没有的物品、背包放不下
的那一件不发也不提示；奖励串里未建模的字段（官银 CapitalType2 等）不发。

```
步骤奖励  交互步记满时、目标行 curnum 写入之前（InteractiveList GetItems/LostItems）
          入包帧（0x19 预清 + 0x18 行，叠堆/新行走 npcshop 的入包链）
          → m1 [1,1,"9645","<物品>",<数量>] 逐件 → m1 [1,1,"9697","<物品>",<数量>]（扣除）
             （#2089..#2092：1347 步骤 558 给武学书；#6196..#6200：1344 步骤 566 给内功书扣暮蓼）
完成奖励  完成簇开头、完成位图之前（Prize 域奖励串 prize_base）
          物品：入包帧 → m1 9645 逐件 → m1 [1,8,"10012,<物品>,<数量>|…"]（整单一张）
          碎银：0x10 CapitalType1=新余额 → m1 [1,1,"1303",<数量>]（「您获得碎银 N」）
          声望：Repute_Record 行（名称、数值、标志 1；新声望 0x11 加行，已有 0x13 写列 1）
                → m1 [1,1,"9693","<声望名>",<数量>]
          历练：0x10 LiveGroove_<奖励串 LiveGroove 下标>=新余额
                → m1 [1,1,"8055",<LiveGrooveCount/1000>]（「您获得历练 N 点」）
             （#1931..#1939：1340＝碎银 1000+江湖声望 10+LiveGroove_5 +5000；
               #6827..#6852：1344＝book_qinggong_7_fc+碎银 3000+LiveGroove_1）
状态      roleState.Money（原有）、LiveGrooves（下标 1..5）、Reputes——随 0x04 存档；
          重登簇下发 LiveGroove_1..5（原写死 0）与 Repute_Record 行
引导链各环  1340 碎银1000/声望10/历练5点；1347 1000/20/10；1341 5000/80/5；
          1342 2000/40/5；1343 5000/250/5（历练记 LiveGroove_2）；1344 物品
          book_qinggong_7_fc+3000/–/5（LiveGroove_1）；1345 物品 additem_0020×2+10000/–/7；
          1346 物品 additem_0011+box_cdjh_001+20000/–/10。步骤物品：1347 的 558/559/560
          给 tbook_CS_jh_cqgf03/02/05，1342 的 562/570/139 给 book_zs_default_01/
          book_hw_normal/hw_fb_rw_01003×60，1344 的 564/566 给 book_qinggong_4/
          ng_tbook_jh_001。
```

验证：`go test ./internal/... ./cmd/...` 66 包通过（2026-11-26）。未做：真机
验证；官服脚本事件（`m1 [2,"Borninfo_<T>_11"]` 黄字广播、村民逃跑等场景编排）；
采集对象点击；打坐/经脉等特殊步的真实条件；官银及真气/经脉/内功类奖励
（1345/1346 完成簇里的 m1 8065/8095、m24 zhenqi 族）；新号初始物品。
