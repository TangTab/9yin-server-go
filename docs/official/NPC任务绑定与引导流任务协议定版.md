# NPC 任务绑定与引导流任务协议定版

> 对象：NPC 对话框的任务行下发与接取/提交受理（internal/fssoloworld/npctask.go +
> npcinteract.go）、任务数据链审计（npcgen/taskgen）、引导抓包任务协议
> （资料/初始账号登陆-完整引导-女.pcapng）、LZ47 解码链（tools/lz_exact.py）。
> 2026-11-23 定版。相关：《NPC服务端XML构建链与黑金样板.md》（NpcConfig 构建）、
> 《外服0x29批量预载通道解析.md》（登录簇 0x29/LZ 池种子）、《初始账号外服教程态
> 方言.md》（Task_Completed 位图）、2433 分支《fxnet2-LZ压缩方言全量定案.md》
> （lz_exact 算法权威）。

## 一、数据链审计结论（2026-11-23）

| 数据 | 状态 | 规模 | 说明 |
|---|---|---|---|
| `ini/Quest/Task/Task.xml` | ✅ 齐 | 3785 任务 | BGserverCJ txt→XML 全量转换（npcgen -tasks 域） |
| `ini/Npc/NpcConfig` + `NpcTemplate.xml` | ✅ 齐 | 77093 模板 + 6272 战斗行 | 76 表 + npctemplate*.tab 合并 |
| `ini/scene/*/npc_creator.xml` | ✅ 齐 | 405 场景 268507 实例 | 44655 唯一 ID 中 887 个绑有 Task.xml 任务 |
| `ini/Text/TaskText.xml` | ✅ 齐 | 239672 键 | stringtask_normal + special 全收集 |
| `ini/Text/NpcTalk.xml` | ✅ 新增 | 52845 键 | string_npctalk/string_taskmenu/stringtalk 三散档（npcgen convertNpcTalk） |
| `taskdata/tasks.json` | ✅ 部署 | 17534 任务 + 3226 绑定 + 2171 NPC | storytasklist/qianjiu_tasknpc.ini 派生（cmd/taskgen） |

不可补的源缺口（非转换遗漏）：
- Task.xml 423 任务（11%）核心对话键（title_/context_/complete_）在 modern 客户端
  提取档中本不存在；
- 9 个 NPC 引用（WorldNpc00690 等）不在 NpcConfig，影响 21 任务（0.6%）；
- m5 标题键 `task_WorldNpc01230`/`task_FuncNpc00107` 同样缺席（官服版本差异，
  键形正确时客户端解析失败仅标题空，窗照开）。

**两套任务 ID 体系（架构事实）**：BGserverCJ `Task.xml`（ID 2001~38xxx，SearchID
s010101 公共）与客户端本地任务链 `storytasklist`（ID 40~77xxx）**任务 ID 不相交**，
但 NPC 命名空间一致（交集 350）。客户端全屏窗 m5 `task_<ConfigID>` 由客户端按
**本地 storytasklist 体系**聚合——服务端要与客户端显示一致的任务行，必须用
tasks.json（storytasklist 派生），不是 Task.xml。

## 二、运行时绑定（npctask.go）

数据装载：env `NINEYIN_TASK_SNAPSHOT` > `<数据根>/taskdata/tasks.json`（正式 Res）
> `<数据根>/../artifacts/local/tasks.json`（仓内 testdata 布局）；进程级缓存，
缺档降级不发任务行（不阻塞对话）。**taskdata 快照 v2**（2026-11-23）：
Binding 增剧情段字段（StoryID/ContextID/TitleID/LeaveTextID）且多段行全保留
（bindings 3226→6350）——storytasklist 每行即剧情一段（同 StoryID 多行按
行序为段序，引导抓包 story_482_1..5 与源表 5 行互证）。

- `npcTaskLines`：`Catalog.TasksForNPC(ConfigID)`（绑定且已加载文案的任务，
  TID 升序，同任务多剧情行只计一次）→ 三态行：
  - 未接 → 100 接取行；
  - 已接 + 有剧情段 + 未走完 → 500 进行中行；
  - 已接 + 剧情走完/无剧情段 → 200 可提交行；
  上限 `NINEYIN_NPC_TASK_LIMIT`（缺省 8——客户端菜单每页 4 行翻页；<0 关停）。
- `npcTaskMenuFrames`：m6 任务行，**双文本键**（首键=行文案，次键 title 作
  TextManager 格式化参数）：
  - 接取 `[100×10⁶+TID, "CanAcceptMenu_<TID>", "title_<TID>"]`
  - 提交 `[200×10⁶+TID, "CompleteMenu_<TID>", "title_<TID>"]`
  - 进行中 `[500×10⁶+TID, "story_menu_<sid>", "title_<TID>"]`
- mark 受理（handleNPCMenuSelect → handleNPCTaskMark）：
  - flag 100 接取：首次=推接取窗（见 §五.1）；同 TID 二次（pendingTask
    状态机）=真接取 → 写 TaskAccepted → 0x29 重放 → 重发菜单（行翻 500）；
  - flag 200/201 提交（201=奖励单选二次确认，官方恒随 200 后到达）：
    移除+清剧情态 → 0x29 重放 → **m8×2 关窗**（官方口径，不刷新菜单）；
  - flag 500 进行中：发剧情对话窗第 1 段；
  - flag 50N 剧情续：发段 N+1；链尾=剧情完成（`Player.taskStoryDone`）+m8×2；
  - flag 60N 剧情离：m8×2 关窗；
  - 300/400/700 留痕。
- 0x29 重放走 `entergen.TaskAcceptedBatchFrame`（m327 sub6 同通道，全量幂等）。

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
| 100 | 接取（本流未出现，主城抓包待补） | 推测 CanAcceptMenu_ + title_ |
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

1. **接取行（100 域）官方形态**：三份抓包（引导女/跳过引导男/npc.pcapng）
   均无手动接取样本——按客户端 lua 链路推断实现并标注待真机验证：
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
（prize 数据接入）；官方链推进（NextTaskLimit——提交后自动接取下一环，
数据在 Task.xml/drama_cite，未接）；DialogID_<sid>/okmenu_<sid> 双窗模式
（第二窗紧随问候窗，npctask 现单窗）。
