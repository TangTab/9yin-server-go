# 9yin ID 体系全景分析（9yin-game-native-menu.exe 逆向 + 抓包实证）

> 分析对象：`E:\Workspace\project\game\adata\fork\9yin-game-native-menu.exe`
> （Go 1.26.5 PE64，本项目 `github.com/local/9yin-go-server/cmd/protocol-probe` 的 fork 构建，带完整 DWARF/pclntab 符号）
> 抓包：同目录 `127-0-0-1-57378_{c2s,s2c}.bin`（该 fork 服务端与真实客户端的一次完整会话，**能正常进入游戏**）。
> 方法：DWARF 结构体提取 + `go tool objdump` 带行号反汇编 + 当前项目源码对照 + 抓包全量解码（c2s 403 帧 / s2c 4353 帧全部解开）。
>
> **状态**：fork 的 wire ID 方案已于 2026-11-09 落地到本项目（§6 实施记录），全仓构建与测试通过。
> §1-§5 中"当前项目写死 1"的描述为改造前快照，行号以 §6 为准。

---

## 0. TL;DR：为什么"改了 ID 就进不了游戏"

客户端进入场景的硬条件是**一条自洽的 wire 身份链**，而不是数据库里的 roleID：

```
0x0B EntryScene ──┬─ [1:5]  场景对象ID（fork: 0x1F6A3 常量）
                  ├─ [5:9]  玩家 objectID   ←┐ 必须与下述所有消息
                  └─ [9:13] 玩家 ownerID   ←┘ 完全一致：
0x0D 玩家档案 (objectID, ownerID)   ← stage_main 开图的 1-1 对象
0x1F ServerLocation (objectID, ownerID)
0x10 Vital 属性帧 (objectID, ownerID)
0x11 Record 增删行 (objectID, ownerID)  ← 快捷栏/轻功/任务记录容器按此键索引
0x0D NPC 档案 ((index+2)|0x11000000, owner=1)
```

客户端 Lua（stage_main / form_main_player）把"EntryScene 之后紧跟的那对 ident"登记为本地玩家，
之后**所有 C2S 交互都原样回传这对 ident**（0x07 ObjRequest、custom obj 参数），
服务端再按回传值路由。任何一环改了而其他环没改（下发混合 ident），客户端对象表映射即断裂 → 黑屏/卡加载/无法进入。

当前项目（9yin-go-server1）把 `playerObjectID=1 / playerOwnerID=1` 写死（44 处引用、8 个文件），
另有 3 处**不引用常量的字面量 1** 是最容易漏改的地雷（见 §5 清单）。
fork 的成功改造 = 把整条链统一换成 `(0x11000001, 0x7A5C0B)` 并给 NPC 加 `0x11000000` 类型位。

---

## 1. 传输层与抓包解码（分析工具链）

- 帧加密：4 字节循环 XOR，密钥 `0xFA6D3BCC`（LE 字节 `CC 3B 6D FA`），见 `internal/transport/key.go`。
- 成帧：XOR 后转义（`EE 00`→`EE`），帧终止符 `EE EE` **明文追加、不参与 XOR、每帧独立从 key[0] 起**。
- 登录消息（opcode 0x02）的账号/密码槽在传输层之下还有一层固定字节旋转异或（密文区呈现 `36 F7 56 97` = 两层 key 异或的残影）——服务端不解除它，只对槽密文做 SHA256。
- c2s 附加帧型：
  - `0x18` 心跳：`18 | u32 tick(GetTickCount类毫秒,单调) | u16 seq(每连接递增) | 尾零`，服务端回 `0x38` 原样回显。
  - `0x0A` custom：`0A | 15字节保留 | u32 msgSeq(每条递增,首个=888) | u16 值数 | 类型化值...`；类型 2=int32、3=int64、4=float、5=double、6=string、7=widestr、**8=obj(8字节=objectID u32+ownerID u32)**。

## 2. 服务端内部 ID 的生成（全部还原）

| ID | 生成方式 | 代码位置 |
|---|---|---|
| 账号 key | `"acct:v1:" + hex(SHA256("nineyin/login-account/v1\0" + LE32(acctSlotLen) + 账号槽密文))` | `internal/auth/login_identity.go`（fork exe 内确认同一常量串） |
| AccountID / RoleID | 自增：`nextAccountID`/`nextRoleID` 从 1 起，JSON 持久化（MySQL 后端为 AUTO_INCREMENT + `0001..0006_*.sql` 迁移） | `internal/role/json_repository.go:301 ensureIdentityLocked` |
| 会话 ID | `session.Machine.id` = 进程级原子计数 `nextID.Add(1)` | `internal/session/state.go:90` |
| wire 玩家 ident | **常量**。当前项目 `(1,1)`；fork `(0x11000001, 0x7A5C0B)` | 当前 `player_actor.go:20`；fork `player_actor.go:6145` |
| wire NPC ident | 当前 `(index+2, 1)` 且注册拒绝 0/1（"collides with reserved player identity"）；fork `((index+2)\|0x11000000, 1)` | 当前 `scene_run.go:305`；fork `main.go:3422`（`ADDQ $0x2; ORL $0x11000000`） |
| EntryScene 对象ID | 当前 `1`；fork 常量 `0x1F6A3`（128419） | 当前 `scene_transition.go:20`；fork `main.go:1808` |
| 物品唯一实例ID | `fmt.Sprintf("7100568-%03d-%010d-%04d", (h>>20)%1000, h, h%10000)`，`h = FNV-1a32(configKey)`（内联哈希：basis 0x811C9DC5、prime 0x01000193） | fork `equip_wear.go:1200 bagUniqueID`（格式串在 exe 字符串区实证） |
| 属性索引 | 协商属性表（0x09）内固定枚举 1..227+（propCurSkillID=221 等） | `player_actor.go:23+` |
| 记录容器名 | 字符串 ID（"QingGongRec"、"ShortcutRec"...），挂在玩家 ident 下 | fork 字符串区/`qinggong_records.go` |

**注意**：数据库 RoleID（自增大数）与 wire ident 是**两套体系**——抓包证明客户端选角回包（0x04, 0x54 字节）几乎全零、不含 roleID，fork 服务端也不校验它；`oneRoleLoginSuccess` 的 `msg[0x25]=1` 在 fork 中**仍未改**（角色槽位恒 1）。所以改 RoleID 本身不影响进图，影响进图的是 wire ident 链。

## 3. 登录→选角→进世界的 ID 传递（抓包实测序列）

```
C→S 0x02 ClientLogin (121B)
     [0]=0x02 [0x15:0x19]=u32 账号槽长(16) 槽区(账号16B|密码48B|clientType|loginString|u32 loginType)
     → 服务端算 acct key → FindByKey/ProvisionLegacy(首次建号 AccountID=1)
S→C 0x04 ServerPlayerRoles (noRole 0x29B / oneRole 追加外观)
     [1]=1(恒) [0x25]=u32=1(角色槽,写死) 后接 6/2/2/2/7(名字UCS2)/7(rolePara)/2/5/8 的属性元组
C→S 0x18 心跳 ×N     (登录后立即, 12B: tick+seq)
C→S 0x04 ClientChooseRole (0x54B, 内容几乎全零——不回传 roleID)
S→C 0x04 角色列表(811B) + 0x1E custom(自定义状态推送) ×N
C→S 0x0A custom ×N   (msgSeq 888.., 就绪前活动)
C→S 0x09 ClientReady (1B)
S→C 0x0B EntryScene: [1:5]=场景对象ID(0x1F6A3) [5:9]=玩家objectID(0x11000001) [9:13]=玩家ownerID(0x7A5C0B) + 属性7=场景config 属性8=资源名 + u16 99 + u32 MaxVisuals(0xF4240=1000000) + 属性9=场景名宽字符
S→C 0x0D 玩家档案 (玩家ident) ← stage_main 开图的 1-1 本地对象
S→C 0x1F ServerLocation (玩家ident, 坐标) ；0x10 Vital ×N (ident, 属性档案)
S→C 0x11 记录行 ×N (玩家ident, QingGongRec/ShortcutRec/...)
S→C 0x0D NPC 档案 ×N ((index+2)|0x11000000, owner=1)
C→S 0x0A ... 进入 PhaseInWorld
```

会话状态机（`internal/session/state.go`）：Connected --0x02--> RoleSelection/Creation --0x04--> EnteringWorld --0x09--> AwaitingActivity --0x0A--> InWorld；0x0C ExitScene → 回 EnteringWorld（重入时 EntryScene 后 750ms 延迟再发玩家档案，0x1F 推迟到目标 0x09）。

## 4. 后续交互中 ID 的传递

| 交互 | 消息 | ID 字段 | 服务端用法 |
|---|---|---|---|
| 点击 NPC/对象 | C2S 0x07 (33B) | seq@17, **objectID@21, ownerID@25**, funcID@29；均校验非 0 | `npcMeta(EntityID(objectID))` 直查注册表 → 对话菜单/传送门/功能 |
| custom 里指定目标 | C2S 0x0A 值类型 8 | 8B = objectID+ownerID | 原样回传 fork 下发的 ident，按值路由（如选怪放技能） |
| 对象移动 | S2C 0x1F / 0x2x 移动帧 | (objectID, ownerID) + 变换头13值 | NPC 巡逻广播带 NPC ident |
| 属性更新 | S2C 0x10 | `flag@1, objectID@2, ownerID@6, count@10, 属性[]` | 玩家 ident（HP/MP/技能状态 221-225 等） |
| 记录增删 | S2C 0x11 | objectID@2, ownerID@6, 表ID, 行号 | 快捷栏/轻功/任务按 ident+表名索引 |
| 菜单 | S2C 0x1C | objectID@1, ownerID@5 | NPC 对话菜单回显点击对象的 ident |
| 心跳 | C2S 0x18 / S2C 0x38 | u32 tick + u16 seq | 回显，不参与路由 |
| 移除对象 | S2C 0x2B(sceneRemoveObject) | objectID@1, ownerID@5 | 视口收缩/场景重入 |

## 5. fork（能进游戏）vs 当前项目（写死 1）对照 & 改造清单

fork 对 wire ident 的改造（exe 内全部实证）：

| 项 | 当前项目 | fork | fork 证据 |
|---|---|---|---|
| 玩家 objectID | 1 | `0x11000001` | `player_actor.go:6145` 等 20+ 处 `MOVL $0x11000001` |
| 玩家 ownerID | 1 | `0x7A5C0B` | 同上 `MOVL $0x7a5c0b`；组合常量 `MOVQ $0x7a5c0b11000001` |
| EntryScene 对象ID | 1 | `0x1F6A3` | `main.go:1808` |
| EntryScene [9:13] | 1 | 玩家 ownerID | `main.go:1810` |
| NPC objectID | index+2 | `(index+2)\|0x11000000` | `main.go:3422` |
| NPC ownerID | 1 | 1（不变） | 抓包 0x0D [5:9]=1 |
| 记录/菜单/属性 ident | (1,1) | (0x11000001, 0x7A5C0B) | `qinggong_records.go:954` 等 |
| MaxVisuals | 200/1000 注释摇摆 | `0xF4240`(1000000) | `main.go:1813` `MOVL $0xf4240` |
| roleID 下发 (0x25) | 1 | 1（**未改**） | `messages.go:75` `MOVL $0x1` |

**改造必须同步改的位置全集**（当前项目）：

1. `cmd/protocol-probe/player_actor.go:20-21` — `playerObjectID/playerOwnerID` 常量（44 处引用自动跟随）。
2. **不引用常量的字面量点（最易漏）**：
   - `messages.go:47` `oneRoleLoginSuccess` `PutUint32(msg[0x25:], 1)`（角色槽，可保持 1，但若引入多角色必须与 0x04 回包解析联动）；
   - `scene_transition.go:20-22` `sendEntryScene` 的 `entry[1:]/entry[5:]/entry[9:]`（objectID/玩家objectID/玩家ownerID 三处独立写入！）；
   - `main.go:1398 npcAddObject` `PutUint32(msg[5:], 1)`（NPC owner，语义是"场景/所有者标识"）。
3. NPC 分配与冲突保护：`scene_run.go:305 registerNPCLocked` 拒绝 0/1 —— 若玩家 objectID 改为动态/大数，须同步扩保护区间或像 fork 一样用类型位前缀隔离；
   `scene_lifecycle.go:293` `uint32(index)+2` 的裸 ID 同时用于注册表 key 和帧 ident，两处必须同公式（fork：同一点 `ADDQ $2; ORL $0x11000000`）。
4. 回传路由：`client_object_request.go`（obj@21/owner@25 直查 map）、custom type-8 obj 解析、
   `scene_lifecycle.go` 多处 `EntityID(request.ObjectID)`——下发 ident 公式一变这些自动跟随，但**任何手写的 `== 1` 判定**（如 `if request.ObjectID == 1` 视为玩家）必须同步。
5. 0x0C 场景重入链：`SwitchScene` 的 EntryScene→玩家档案→0x1F 延迟序列使用同一对 ident，重入后客户端对象表按新 EntryScene 重建，必须整链一致。
6. 持久化层（可不动）：bagStore/currencyStore/facultyStore 等按 RoleID 键（fork 有 `bag store: zero role id` 防御）；wire ident 改造与 RoleID 无耦合。

**结论**：fork 的可行方案 = 玩家 ident 换成一对非零且不与 NPC 冲突的常量（`(0x11000001, 0x7A5C0B)`），NPC 用 `|0x11000000` 类型位与玩家共用前缀并保留 +2 偏移，EntryScene 三字段与玩家档案严格同源；roleID 只存在于数据库与登录响应的槽位展示，不参与 wire 身份链。

---

## 6. 改造实施记录（2026-11-09 已落地）

上述 fork 方案已按"照抄 fork，抓包已验证"原则落地到本项目，全仓 `go build ./...`、`go vet`、`go test ./...` 全部通过。

### 6.1 生产代码改动（5 处）

| 文件:行 | 旧 | 新 | 说明 |
|---|---|---|---|
| `cmd/protocol-probe/player_actor.go:25-26` | `playerObjectID=1 / playerOwnerID=1` | `0x11000001 / 0x7A5C0B` | 常量定义；全项目 44 处引用（0x0D/0x1F/0x10/0x11/快捷栏/轻功/战斗/内功）自动跟随 |
| `cmd/protocol-probe/scene_transition.go:23-25` | `[1:5]=1, [5:9]=playerObjectID, [9:13]=1` | `[1:5]=0x1F6A3, [5:9]=playerObjectID, [9:13]=playerOwnerID` | EntryScene 三处独立写入统一同源——上次"改了进不去"的首要雷区 |
| `cmd/protocol-probe/scene_run.go:127` | `uint32(index)+2` | `(uint32(index)+2)\|0x11000000` | NPC 注册 ID（fork main.go:3422 `ADDQ $2; ORL $0x11000000`） |
| `cmd/protocol-probe/scene_run.go:307` | `id==0 \|\| id==1` | 追加 `\|\| id==playerObjectID` | 保留身份冲突保护 |
| `cmd/protocol-probe/scene_lifecycle.go:294` | `next[uint32(index)+2]` | `next[(uint32(index)+2)\|0x11000000]` | 可见集公式与 populate 严格一致——两个独立计算点必须同步 |

### 6.2 ID 空间布局（改造后）

```
0x00000002 .. len(catalog)+1        远程玩家 wireID（nextWireID 自增，低位无前缀空间）
0x001F6A3                            EntryScene 场景对象ID（常量）
0x7A5C0B                             玩家 ownerID（所有者标记，非零）
0x11000001                           本地玩家 objectID（=1|类型位，独占低位）
0x11000002 .. 0x11000000|len+1       NPC objectID（=(目录下标+2)|类型位，最低 0x11000002）
```

三个空间互不重叠：NPC 与玩家共用 `0x11000000` 前缀但玩家独占低位 1；远程玩家 wireID 落在无前缀低位空间，理论上限约 2.8 亿目录条目。

### 6.3 测试更新（9 个用例）

| 断言类型 | 旧 | 新 | 文件 |
|---|---|---|---|
| BufferInfo hex（`bufferInfo` 用 `%X`） | `1742,1,1,3,` 等 | `1742,11000001,7A5C0B,3,` | player_progress_test ×3 |
| | `21EA,1,1,1,` | `21EA,11000001,7A5C0B,1,` | qinggong_records_test ×1 |
| BufferListStr 十进制（`%d-%d`） | `…,1-1,` | `…,285212673-8018955,` | player_progress ×2、qinggong ×3 |
| NPC 数字 ID | `2`/`3` | `0x11000002`/`0x11000003` | main_test、patrol_test、scene_run_test（npcMetaByID/patrolIDs/npcTransform/Vitals/npcVisible/scene.Get） |

**刻意保留旧值的测试**：`TestServerAllDestEncoding`、`TestSceneObjectByteProperty`、`TestSceneRemoveObject` 为编码函数自洽测试（传入 objectID=2 断言回写 2），验证的是字段编码而非 ID 分配方案，不应随方案变更。

### 6.4 刻意不改的项（与 fork 实测行为一致）

- `messages.go` `oneRoleLoginSuccess` `msg[0x25]=1`：角色槽位恒 1，fork 未改；客户端选角回包（0x04, 0x54B）几乎全零不含 roleID，服务端不校验——角色 ID 不参与 wire 身份链。
- NPC ownerID=1：语义为"场景/所有者标识"（npcAddObject 注释），fork 保持 1。
- 远程玩家 `nextWireID`：`len(catalog)+2` 起自增（scene_run.go:135），与 NPC 前缀空间天然隔离。
- EntryScene 附加字段（属性数 3、MaxVisuals 等）：当前值可正常进图，fork 的属性 36 宽字符串场景名属可选增强，未纳入本次 ID 改造。

### 6.5 后续验证要点（真客户端实测）

1. 登录→选角→进图后，抓包核对 0x0B 的 `[5:9]/[9:13]` 与玩家 0x0D、0x1F、0x10、0x11 的 ident 是否全链一致（应全为 `01000011 0b5c7a00`）。
2. 点击 NPC：C2S 0x07 的 obj@21 应为 `0x11000000|下标+2`、owner@25=1；服务端日志应能路由到对话菜单。
3. 快捷栏/轻功记录：0x11 增删行后客户端 UI 应即时反映（记录容器按玩家 ident 索引）。
4. 多连接场景：第二连接的远程玩家物化应使用 nextWireID 低位 ID，不与 NPC/本地玩家冲突。
