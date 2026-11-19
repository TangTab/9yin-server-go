# FsVerCli / FsVerEnc 全解与 Go 复刻

> 快照：`F:\Game\Online\9yin\BGserverCJ\Debug`（2014-03-22，均带 PDB 全符号）
> 复刻：`internal/fsvercli`、`internal/fsverenc`、`cmd/verencode`（2026-11-19 全绿）

## 0. 三件套关系（本次定案）

```
FsVerEnc.dll（离线生成器，FsLoader 宿主 Lua 实体 "GameEncode"）
   │  GenEncodeFiles(id, complexity) → ver<id>\{<id>Mes.h, <id>Encode.h, <id>Encode.cpp}
   ├────────────────────────────┐
   ▼                            ▼
FsVerify.dll（服务端校验插件）   FsVerCli.dll（客户端校验钩子）
 Mes.h 宏编译进 CVerify          Encode.cpp/h 编译成 FxCli.dll 旁挂模块
 g_encode_order/cond_value/      encode_func_N(bool 签名+cond 分支+反调试诱饵)
 client_code + void encode_func_N
```

- **编码器等价性（本次全量实证）**：FsVerCli 的 `encode_func_N` 是独立生成系
  （bool 返回、`content[i]` 槽位、g_cond 分支、`_CORE_API`/反调试门），但在
  FsVerify `GetClientCond(id)` 下发的 cond16 种子下，其输出与
  `fsverify.EncodeFunc(id, in)` **逐字节一致**——unicorn oracle 对拍
  2048×3 输入零失配（`tools/fsvercli_oracle.py`，假 VAPI 桩需返回非零堆缓冲）。
  Go 客户端复刻因此直接复用 internal/fsverify 程序表。

## 1. FsVerCli.dll（客户端）

### 1.1 导出面与注入模型

| 导出 | 实现 | 语义 |
|---|---|---|
| `FxModule_GetType` | → `"FxClient"` | 模块类型名 |
| `FxClient_SetApi(VAPI* core, send_t send)` | 存 `_CORE_API`@0x102E1D28、`g_send_msg_func`@0x102E1D2C | 客户端注入核心 API 与原始发送函数 |
| `FxClient_GetSendApi` | → `SendMessageFunc` | 替换客户端发送路径 |
| `FxClient_GetRecvApi` | → `RecvMessageFunc` | 替换客户端接收路径 |

`EncodeMessageInit`（客户端开机调用）：LoadLibrary("kernel32") + 取
`IsDebuggerPresent`（串面 `3c 66 37 58 …f3` 每字节 +0x0D 解码）存 `g_isdp`；
`CreateMutexA(NULL,FALSE,"FxClient_Dll_Mutex")`。

### 1.2 传输层（EncodeMsgBuf/DecodeMsgBuf @0x10259320/0x10259410）

32 位静态 XOR：dword 道 `buf32[i] ^= key`，尾字节按道内下标 XOR；
`g_encode_key == g_decode_key == 0xAE5C37A5`（.data 固定常量，全镜像仅此二函数引用）。

### 1.3 发送路径（SendMessageFunc @0x10259680，跳表 @0x10259C44）

先按 `buf[0]` 注令牌再整体 XOR 交 `g_send_msg_func`：

| op | preimage | 令牌写入 |
|---|---|---|
| 1 CLIENT_RET_ENCODE | `m_nDynamicKey ‖ msg[0x21](4B) ‖ msg[1..0x21](32B)` | `msg[0x25..0x35)` |
| 2 ClientLogin | `key ‖ msg[1..0x43](0x42B)` | `msg[0x43..0x53)`（帧拓至 0x53） |
| 4 choose_role | `key ‖ serial ‖ msg[0x1d] ‖ msg[0x15] ‖ msg[0x19]` | `msg[1..0x11)` |
| 7 / 10 | `key ‖ serial ‖ word@0x15 ‖ 载荷(≤8 全拷，否则首4+尾4)` | `msg[1..0x11)` |
| 3/5/6/8/9 | — | 直通 |

- `msg[0x11]=g_msg_serial++`（op4/7/10）。
- **令牌语义（双重 md5）**：`m_EncodeFunc(md5.Digest())` —— 即
  `token = encode_func_id(md5(preimage))`，而 encode_func 内部还有一次 md5。
  服务端（FsSoloEntry VerifyMessage）对同一 preimage 取摘要后以
  `fsverify.EncodeFunc(id, digest)` 比对。
- `m_EncodeFunc == NULL`（未握手）时不注令牌直通。

### 1.4 接收路径（RecvMessageFunc @0x10259DF0）

先 `DecodeMsgBuf`；仅处理 `buf[0]==2`（SERVER_SET_ENCODE）且 `len≥0x3E`，
其余恒返回 1（放行给客户端原逻辑）。op02 帧布局（与 FsSoloEntry
SendClientEncode 对合）：

```
[0]=2  [1..0x21)=回显 0x20  [0x21]=m_nDynamicKey  [0x25]=encode id
[0x29]=应答回显 dword(f28)  [0x2d..0x3d)=cond16  [0x3d..0x4d)=code16 头
[0x41..)=记录表 {u32 目标, u32 跳过字节数, u32 dword 数, 变长体}*（官方仅游走）
```

处理序：`m_EncodeFunc=NULL` → 取动态键/id（2048 项跳表选编码器，越界不选）→
`set_cond_value(msg+0x2d)` → Open/Wait `"FxClient_Dll_Mutex"` → 记录表游走
（`GetModuleHandleA("FxCli.dll")` 参与但产物不落地）→ Release/Close →
回 0x35 字节 `[1]‖回显32‖f28`。应答经客户端发送钩子环回，从而获得
令牌与 XOR（官方 golden 线路帧 login-connect-001.bin 的 op01 令牌可被
服务端验通即证）。

### 1.5 set_cond_value（@0x100F7280）：恒等 LSB 位映射

128 组 `g_cond_N = (p[N/8] >> (N%8)) & 1`（PDB 符号位序全量核对；另有
`g_chaos_N` 家族共享同函数写入）。Go：`Client.SetCondValue/CondBit`。

### 1.6 login_send（@0x100F89C0）

```
[0]=2  [1:5)=0x12C(300)  [0x15]=u32 账号槽长 ‖ 槽  ‖ u32 密码槽长 ‖ 槽
```

槽加密 = 块序偶/奇 XOR 掩码（field1Even/Odd、field2Even/Odd）+ `des2key/
D2des` 阶梯，键/掩码与 internal/fsverify 的 `field1Key/field2Key` 及掩码
**逐字节同源**（生成器侧字符串面 MES_ACCOUNT_*/MES_PASSWORD_* 互证）。
Go：`LoginSend` = 版本头 + `fsverify.EncodeAccountFields`。

## 2. FsVerEnc.dll（生成器）

### 2.1 Lua 面（FsLoader 宿主，GameEncode.bat）

实体 `GameEncode`（IEntity，Space=""，Parent="IEntity"）；属性
`Console`/`EchoSock`；方法 `Start/Stop`（恒 true）、
`Encode(id, complexity)`（4≤c≤0x80，错则 `[GameEncode::Encode]complex degree error`）、
`Split(id, segments, length)`（1≤s≤8、0x40≤l≤0x200、s·4·l≤0x1000，
三段错误文案独立）。源文件路径字符串
`d:\blackgoldserverengine\serverengine\branches\server(20120807)\fsverenc\gameencode.cpp`。

### 2.2 GenEncodeFiles（@0x100429D0）管线

```
srand(time(NULL))                    ← 种子非确定：官方表不可重放（by design）
g_pswd_mask[4]  = rand_dword ×4
g_pswd_key[16]  = rand_byte ×16
g_rand_0..8     = 9 × 0x800000 rand_byte 池
for i in 0..0x7FF:
    name = "Light is my strength" + itoa(i) + itoa(rand_dword())
    g_cond_value[i] = MD5(name) 十六进制串前 16 个 ASCII 字符   ← cond16 全 hex 字符
    g_cond[i][j*8+b] = (cond16[i][j] >> b) & 1                  ← 与 1.5 闭环
CreateDirectoryA("ver%d\\"); 写 <id>Mes.h / <id>Encode.h / <id>Encode.cpp
```

`rand_byte = rand()>>7`（@0x10040430，`shl 8 / 0x8000` 有符号除）；
`rand_dword = 4×rand_byte 大端拼`（@0x100404A0）；MSVC LCG
`seed=seed*214013+2531011, rand=(seed>>16)&0x7FFF`。

### 2.3 发射器与模板（字符串面实证）

- `gen_mes_h`（服务端）：`MES_MESSAGE_VERSION %d`、`MES_ACCOUNT_MASK_0..7`
  /`MES_PASSWORD_MASK_0..7`（=field 掩码）、`MES_ACCOUNT_KEY_%d 0x%02X`
  （随机）与 `MES_PASSWORD_KEY_0..15`（固定=field2Key）、
  `MES_ENCODE_ORDER UI32 g_encode_order[%d]`、`MES_COND_ENCODE`、
  `#define MES_ENCODE_%d void encode_func_%d(...) { UI08 content[%d]; UI08*p=content; … }`
  宏体（服务端游标式写法 ↔ void 签名）。
- `gen_encode_cpp/h`（客户端）：`bool encode_func_%d(UI08*, UI08*)`、
  `UI08 content[%d]` 槽位式（↔ FsVerCli 反汇编所见）、
  `UI08* content = (UI08*)_CORE_API->MemAllocate(%d)`（=oracle 假 VAPI 桩的依据）、
  `if (g_cond_%d) { __asm _emit 0x%02X }`、
  `if (_CORE_API->GetAppInstance() == NULL)…` / `GetAppHWND` /
  `if (IS_DEBUGGER_PRESENT)…`（=vt[2]/vt[9]/g_isdp 调用点全对合）、
  链模板 `(input[%d] ^ 0x%02X) - 0x%02X` 等 8 式、
  `alg.Update(content, %d); Finalize(); memcpy(output, alg.Digest(), 16)`、
  `login_send`/`set_cond_value` 全文模板（与 1.4/1.6 反汇编互证）。
- `SplitCodeFiles`：按 segments×length 生成客户端 code 记录段（对应 op02
  帧 [0x41..) 记录表 / g_client_code 变长 blob）。
- `read_client_map("<dir>FsVerCli.map")` / `read_client_dll("<dir>FsVerCli.dll")`：
  读客户端链接 map 与 DLL 本体，供记录表定位。

## 3. Go 复刻

| 组件 | 对应 | 测试 |
|---|---|---|
| `internal/fsvercli` | 传输 XOR、SetCondValue 恒等映射、SendMessage 五类 preimage/令牌、RecvMessage op02 全流程+记录表游走、LoginSend | 与 fsverify 表交叉验证（握手令牌=EncodeFunc(id, md5(pre))；登录槽 DecodeAccount/DecodePassword 回环） |
| `internal/fsverenc` | MSVC LCG（含 SeedAt 倍增跳跃）、rand_byte/rand_dword、管线（mask/key/9 池流式/cond16=md5hex("Light is my strength"+i+r)）、Lua 面校验、三件发射模板 | srand(0)/srand(1) 公开向量、SeedAt=顺序、PoolByte=顺序、cond16 ASCII hex 全表、参数文案 |
| `cmd/verencode` | GameEncode.bat 宿主 | 冒烟（生成 ver1 三件） |
| `tools/fsvercli_oracle.py` | unicorn 真值 oracle | 2048×3 对拍零失配 |

复刻边界：FsVerEnc 发射器的链体选取策略（gen_encode_cpp 0xBC0 内池索引）
为结构级——官方产物因 `srand(time(NULL))` 本不可逐字节重放；其余管线
逐指令等价。FsVerify 侧复刻见 `fsverify-verify插件复刻.md`（本次补充：
客户端令牌为 md5 摘要再编码的双重 md5 语义）。
