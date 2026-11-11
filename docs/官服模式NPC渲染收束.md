# 官服模式 NPC 渲染收束（2026-11-11 定案）

> **结论一句话**：fork 客户端的 NPC 视觉管线消费属性块里的**对象头行**（fork 表 0..173：
> Type@0/Sex@1/Photo@2/Camp@3/Race@4/Job@5/Level@6/ConfigID@7/Resource@8/
> PosiX@9/PosiY@10/PosiZ@11/Orient@12/Hair@13…）。官服 2433 整表把头块换血为
> 家园/帮会/VIP 域后，这 35 个头行名无行可落 → `OrderedProperties` 静默丢弃 →
> 客户端对象添加成功但拿不到对象头数据 → **NPC 不渲染**。
> 修复 = `hybrid2433` 混合表（fork 头块 0..173 + 官服共享域 174..2432）。
>
> 真机定案会话 56232（07:46）：`dialect=hybrid2433 entries=2433`、NPC 0x0D=83 字段
> （头行回归）、134× 0x28 v3 全发存活、**NPC 渲染成功、无断连**。

本工程全程五环因果链，每环独立定案。以下按追凶时间线记录，供后续官服方言工作引用。

---

## 一、0x28 四连崩：LZ 字典压缩（2026-11-10 深夜 → 11-11 上午）

### 症状
`=3` 模板身份 / v2 精确切片 × 0x0D 前后两种时序，四种组合真机全崩。

### 证据链（三重闭环）
1. **崩溃现场**：bin64 五个 minidump（`[]DUMP2026111*.dmp`）纯 Python 解析
   （tools/op28/parse_dumps.py）——五崩全部同址 **fxnet2.dll+0x10b0，AV READ 0x0**
   （栈 ←+0x125c←+0x1bdae←fxgame.exe+0x65978）。
2. **反汇编**（fxnet2 x64 .text 明文，capstone）：崩点=LZ77 式字典压缩解压器
   （RVA 0x1000-0x1260）。位流=[flags u32 内联重填(哨兵 1)]交替 token；
   bit0=字面量（unary 计数 1/01/001/000→1-4B）；bit1=匹配（u16：
   `idx=(v>>4)&0xfff` 12 位池索引 + `nib=v&0xf` 4 位长度 2..17，nib=0 扩展长度）；
   字典池 .data@0x502e0（4096 槽全局持久，注册哈希 `((u32@pos>>12)^u32@pos)&0xfff`，
   窗口滞后 3）；尾部 write≥outlen-0xb 后逐字节直拷；包装层 0x11f0
   （mode bit1=4B/1B 长头、bit0=解压/纯 memcpy）。
3. **帧字节实解**（tools/op28/decode_capture.py：EE 去转义→XOR 0xFA6D3BCC→抽帧）：
   崩溃帧身份串全在场、**模板保真段携带 `80 02` 串池回引 token**——客户端会话
   字典只见过我方 fork 方言串，无此槽 → NULL → 必崩。同帧+官方字典加持可完整
   解出（outlen=0x122）→ 帧结构合法、死因纯属字典状态。

### 根因
0x28 的 0x47 帧 = LZ 压缩流；官方压缩器跨帧引用会话字典状态（官方 224 帧顺序
解码 141 OK / 83 FAIL=跨帧回引），重放必须全量时序——模板路线天然死路。

### 修复
**v3 生成器**（npc_detail_v3.go，提交 81dc007）：mode=**0x46 纯 memcpy 路径**
（官方 l.pcapng 自有 6 帧先例，#2537 等）程序化构造明文帧，零 LZ/零字典依赖：

```
帧 = [op 0x28][mode 0x46][len-1 u32][明文体长 u32]
body = [obj u32][own u32][52B 变换块(posA xyz+yaw / posB 同 / scale 1.25 /
        rot 0 / 保留 0×3 / 原始 u32 1)]
       [u16 字段数][字段×N：[idx u16][0x09 表列型定宽值]]
串值 = [len u32][GBK+NUL]（len 含 NUL）
```

**勘误**：旧"[6:10]=模板号"实为 **LZ 解压输出长度**（f6=290 系巧合）；
旧"[u8 len][01 00][串]"串定律系 LZ 流标志位交织的误读，明文真身无 01 00。
tag 0x27..0x30 整族=压缩值消息（op 0x28 处理器=fxnet2 ServerAddObject，
对象键 `"%u-%u"`(obj,own)，需 len≥0x3f，this+0x38→+0x68 场景容器，
no scene 才静默）。

---

## 二、进场 Out range：0x0B 尾属性宽度（2026-11-11 06:29）

### 症状
官方模式进场，客户端 trace.log（bin64/trace.log，每进程覆盖，**排障第一站**）：
`(GameReceiver::RecvProperty)get message error / Out range / (GameReceiver::ServerEntryScene)property error`。

### 根因
fxnet2 0x9a60 带流读取原语（u8/u16/u32/u64/f32 各自 cursor+size≤end）按
**0x09 表列型定宽**读取；0x0B 尾属性 idx99 恒写 u32——fork 活动表该行
`MaxVisuals` int32（4B 恰好合法，fork 模式真机无恙的谜底）、官服 2433 表该行
`LastCowPetSigninTime` int64 → 客户端读 8B 越出帧尾。

### 修复（cc2aa3d）
尾宽按当前表线型编码；官方 0x0B 真身（l.pcapng #126，84B）为 5 属性形
（178 Name wstr 中文名/1468 u16/1439 u64/1447/1448，无 99 尾）——留作后续对齐项。

**附带认知**：0x0B bug 存在于此前**每一次**官方模式测试——11-10 深夜 sparse 二分
"官服方言 NPC 渲染必需 0x28"的结论是带病会话得出的（后经验证 0x28 非必需条件之一）。

---

## 三、fork+0x28 静默断连：AIEffect panic（2026-11-11 07:07-07:18）

### 症状
0x28 env 开启的会话（55663/58600/59855）全部死在**首帧 NPC 0x0D 之后**、
0x28 一帧未落 capture；客户端 `InnerSend send failed → on_close`、无任何
GameReceiver 错误。关闭 env 的会话（52629）跑完 135 NPC 全流。

### 根因
v3 的 `officialNPCEntityProperties` 用 `MustPlayerProperty("AIEffect")`——
官方表@84 在场、**fork 表玩家域缺席** → 会话 goroutine panic → 连接静默关闭。
本地一测即崩：`panic: property name absent from official table (player): AIEffect`。

### 修复（0c3109e）
AIEffect 改 `PlayerProperty` 可选查（缺席跳过，值为常量 "nil" 无害）；
补 fork 方言回归测试（此前 v3 测试只跑 official2433——**测试方言覆盖教训**）。

---

## 四、NPC 不渲染：头块理论 → hybrid2433（2026-11-11 全天主线）

### 定案路径
1. **史上首次干净官方会话**（07:00，Out range 已修）：进场全绿仍无 NPC →
   排除 0x0B 干扰后的真问题浮出；
2. **83 vs 48**：同一 NPC 的 0x0D，fork 方言 83 字段、官方方言 48 字段——
   差的 35 个全是头行名（PosiX/Y/Z/Orient/CollideType/NPCCanSelect/AITemplate/
   PatrolMode/ChaseRange/RefreshTime/NpcNickName/Material/ActionSoundMode…，
   属"64 缺名"子集）；
3. **头块 diff**：两张活动表的 0..173 完全不同——fork=对象头（上表），
   官服=家园/帮会/VIP 域（171/174 行全异）；
4. 官方会话里 0x28 对象与 0x0D 对象**零重叠**（206:28）——官方 0x28 对象走
   独立通道，但 fork 客户端两条路都吃头行。

### 修复（64ba565）
**hybrid2433 混合表**（`NINEYIN_PROPERTY_TABLE=hybrid2433`）：
fork 头块 0..173 + 官服共享域 174..2432（`splicePropertyTables` 升序位置绑定保持）。
玩家域（末现）解析落官服行号、场景域（首现）回落头块——NPC 0x0D 头行通道
恢复且共享域保持官服方言。

---

## 五、hybrid"不生效"：启动脚本硬编码（2026-11-11 07:33）

用户 shell 设了 hybrid，服务端日志却打出 `dialect=official2433`——
`start-9yin.bat` 第 11 行硬编码 `set NINEYIN_PROPERTY_TABLE=official2433`
覆盖用户 env（bat 的 set 在 shell env 之后执行）。二进制安全改 bat→hybrid2433
（**GBK+CRLF 必须保持**，见 bat-gbk 记忆）。F: 侧 bat 手改不回写仓库，
deploy.bat 对 server-cmd\* 是 skip-existing。

---

## 工具资产（全部入仓 tools/op28/）

| 文件 | 用途 |
|---|---|
| lzdec.py | LZ 解压器模拟器（位流/池/窗口注册，World 模型；DecodeError 带 partial） |
| decode_capture.py | captures/local 裸 TCP 解码（EE 去转义→XOR 0xFA6D3BCC→抽帧） |
| parse_dumps.py | minidump 崩溃栈解析（纯 Python，免 WinDbg） |
| lz_session_decode.py | 全量会话解码（0x27-0x30 族，共享池） |
| parse_template46.py | 0x46 明文帧字段图（0x09 表列型驱动走查） |

**走查器语义**（逐帧验证的标准 walker）：串=[len u32][GBK+NUL]；
wstr len=字节数；WireObject=8B；0x10=[op][flags][obj u32][own u32][count u16][pairs]；
0x0D=[61B 固定头][count u16@0x3d][pairs@0x3f]；0x18 视图行列号走 0x0A schema
（不是属性 idx——按属性表走查会误报）。

## 客户端侧参考（逆向定版）

- fxnet2 x64 副本：artifacts/fxgame/fxnet2_x64.dll（.text 明文可 capstone；
  .themida 6MB 壳段不影响 0x1000-0x12000 带）
- fxnet2 GameReceiver 错误串表：rva 0x402b8-0x41600（每 op 的
  size/property/no scene 错误名 + on_* lua 事件名，与 fxgamelogic
  GameMessageHandler vtable 槽一一对应——vtable@0x12e02178 全 78 槽图见
  ~/.zcode/analysis/fxgamelogic-0x28/GAMEMSGHANDLER-MAP.md）
- 客户端 trace.log = 排障第一站（每进程覆盖，需跨会话对比时先备份）

## 余项（可选）

1. **0x28 必要性复验**：hybrid + 关 0x28 是否仍渲染（旧"必需"结论出自带病会话）；
2. 真·官服还原（纯 official2433 无头块）：~~需 0x29 批量预载通道~~
   【2026-11-11 勘误，0x29 已全解见 `官服0x29批量预载通道解析.md`】：0x29 是
   **玩家记录行集**批量预载通道（任务/称号/好友/装备方案…），obj=玩家自身，
   **与 NPC 无关**——"需 0x29"系未解时把它误当对象预载通道。NPC 物化仍只走
   0x28（0x46 明文路径即可，池种子已证含客户端本地态、LZ 精确重放不可行）。
   本条仍然成立的部分：官方 0x28 对象从不走 0x0D；疑 0x09/0x0A 表内容参与
   ——已证部分成立（表帧滑窗可命中多数 miss 槽，但非唯一源）；
3. m655 登录帧方言差异（official_login_align 在 fork 形下=紫泉+618 vs 旧版
   account 串）——低优先级观察；
4. 默认方言是否翻 hybrid2433：待拍板（现默认 fork）。

## 提交清单（feat/op28-official）

| 提交 | 内容 |
|---|---|
| d5a1870 | 分析资产入仓（lzdec/decode_capture/parse_dumps + 权威报告） |
| 81dc007 | v3 生成器 mode=0x46 明文方言 + v2 退役标记 |
| cc2aa3d | 0x0B 尾属性按表线型编码（官方方言 Out range 根因） |
| 64ba565 | hybrid2433 混合表方言 |
| 0c3109e | v3 AIEffect 可选化（fork 表 panic 静默断连根因）+ fork 方言回归测试 |
