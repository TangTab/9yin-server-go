# FsVerify.dll 全解与 Go 复刻

> 权威档案：BGserverCJ 官方快照（2014-03-22，PDB 全符号）的 `FsVerify.dll`
> 静态反汇编 + unicorn 动态 oracle，全功能 Go 复刻落位 `internal/fsverify`。
> 工具链：`tools/pdb_syms.py`（PDB 符号）、`tools/fsverify_oracle.py`
> （x86 模拟执行）、`tools/fsverify_gen.py`（提升器/生成器）。

## 一、模块定位

FsVerify.dll 是 Fs 框架接入服（FsSoloEntry）的**可选消息校验插件**：
`game_entry.ini` 里 `//MessageVerifyDll=fsverify.dll`（官方样本默认注释
=禁用）。加载路径（SoloEntry::LoadVerifyDll @0x100a1500）：
LoadLibrary → `FxModule_GetType`（类型校验，不符报 "type error"）→
`FxVerify_GetInterface` 拿 IVerify* 存 SoloEntity+0x184。

导出仅两个：

| 导出 | RVA | 语义 |
|---|---|---|
| FxModule_GetType | 0xd9390 | Fs 模块类型标识 |
| FxVerify_GetInterface | 0xd93d0 | `CVerify::NewInstace` 工厂 |

**接口（??_7CVerify vtable @0x271f98，slot 序即声明序）**：

| slot | 方法 | 反汇编实证 |
|---|---|---|
| 0/1 | dtor / Release | 常规 |
| 2 | GetVersion() | `mov eax, 0x12c` = **300**（本构建；2016 线上版本门=333 由另版供给） |
| 3 | GetEncodeNum() | `mov eax, 0x800` = **2048** |
| 4 | GetEncodeId(i) | `g_encode_order[i]`（u32[2048] @0x285b28，0..2047 的**固定置换**） |
| 5 | GetClientCond(i) | `get_client_cond(i)`@0xd9420：跳转表 2048 项 → `g_cond_value[i]` 16B |
| 6 | GetClientCode(i) | `get_client_code(i)`@0xe2040：跳转表 2048 项 → 变长 blob 头指针 |
| 7 | EncodeFunc(id,in,out) | `encode_func`@0x200a50 派发（见 §四） |
| 8 | DecodeAccount(p,len) | 原地解密账号槽（见 §三） |
| 9 | DecodePassword(p,len) | 原地解密密码槽（同构独立键系） |

CVerify 对象无状态（NewInstace 分配 4B 纯 vtable 指针）——所有键、
掩码、表全部编译期常量。

## 二、消费方协议（FsSoloEntry 反汇编）

`COuterHook::CheckMessageIllegal` @0x10097820 按 msg[0] 跳转表（1..10）：

| op | 名称 | 门 | 语义 |
|---|---|---|---|
| 1 | CLIENT_RET_ENCODE | len≥0x35 | buf = `client.f10(4B) ‖ msg[0x21](4B) ‖ msg[1..0x21](32B)`，`VerifyMessage(vfy, client.encodeID, msg[0x25..0x35], buf)` 比对 16B 令牌；失败断连（错误码 3） |
| 2 | ClientLogin | len≥0x15 | 版本门：`vfy->GetVersion()` vs 帧 `[1:5]` u32，不符回 `{03, 85 4E 00 00}` 断连；再 `DecodeLogin(vfy,msg,len)`（≤0x200B，账号槽 u32len≤0x40 → DecodeAccount 原地解密 → 密码槽同） |
| 4 | choose_role | len≥0x54 | 令牌比对（buf = f10 ‖ msg[1..0x43]） |
| 7 | select | len≥0x21 | 令牌比对（buf = f10, msg[0x11], msg[0x1d], msg[0x15], msg[0x19]） |
| 10 | （重置类） | len 门 | 令牌比对 |
| 其余 | — | — | 直接放行 |

`COuterHook::VerifyMessage` @0x10097640（C2S 令牌验证）：
`EncodeFunc(id, src)` 算期望 16B 与客户端令牌 memcmp；失配且
`PrevEncodeIdInvalid()==false` 且 `prevID != id` 时以 prev 再试一次
（编码 id 周期轮换期间的宽限）。

`COuterHook::SendClientEncode` @0x10097360（S2C 下发，opcode=2）：
帧 = `[0]=2` + `[0x21]=f10` + `[0x25]=encodeID` + `[0x29]=f28` +
`[1..0x21) 清零` + `[0x2d..0x3d)=GetClientCond(id)` +
`[0x3d..0x4d)=GetClientCode(id)`——客户端拿 cond/code 参与自己的应答编码。

编码 id 轮换策略（SoloEntry 层，非 DLL）：`GetEncodeId()=counter[0x4c] %
GetEncodeNum()`、prev 用 `[0x50]`；SetCurrentEncode 以时间混合更新计数。

**vfy==NULL 时 CheckMessageIllegal 恒放行**——生产配置禁用即整个握手
静默跳过；现代客户端首飞直接 op02（golden 实证），无需本插件在线。

## 三、op02 双槽加密（DecodeAccount/DecodePassword）

两方法同构（@0x210b10/@0x210d50），仅常量不同：

```c
BYTE key[16] = {域固定键};
des2key(key, 1);                    // KnL=解密(kA=key[0:8]) KnR=加密(kB=key[8:16])
for (i = 0; i < len>>4; i++) {      // 16B 块
    D2des(p, p);                    // 三级阶梯（见下）
    p 块 ^= mask[i&1];              // 偶/奇块 4×LE32 掩码
}
return TRUE;
```

`D2des(from,to)`（d3des 双密钥扩展 @0xd6b40）：

```
work1=scrunch(in[0:8])  work2=scrunch(in[8:16])   // BE 载入
desfunc(work1,kA') desfunc(work2,kA')              // ①D_kA 两半独立
swap(work1.R, work2.L)                             // 跨半交换
desfunc(work1,kB) desfunc(work2,kB)                // ②E_kB（此时配对=左左/右右）
swap(work1.R, work2.L)                             // 换回
desfunc(work1,kA') desfunc(work2,kA')              // ③D_kA 再独立
unscrun → out
```

即编码方向（客户端）为 XOR 掩码 → `E_kA(两半) → D_kB(左左/右右交叉) →
E_kA` 的 EDE 阶梯——与 B 线 fork 客户端逆向
（official-2433-b-line `internal/auth/field_cipher.go`）**逐字节同构**。

**常量三方实证一致**（官方 DLL 立即数 = B 线常量 = golden 密文回环）：

| 域 | 16B 键 | 偶块掩码（4×LE32） | 奇块掩码 |
|---|---|---|---|
| 账号 | `66f928e8a1009b7629a01f8def458919` | f8ebd0a7 c4badc9c a84fd0a2 74cf24b7 | 26e24c92 38efda09 c3b2d047 28efd49c |
| 密码 | `106eba557761c77e1eea0deb676293dc` | f24d83e5 ab2526d8 32b93688 affdfe20 | e2435ae5 136126b1 52ba6e85 3fa3fb23 |

golden `login-connect-001.bin`（现代客户端 2016 官服首飞）：
账号槽 `4578627e…` → `test001`；密码槽 → 派生形态
`drUGtwMnNPbVE9PQ==VjYzaVZab0xjQ0M3bD`（原始口令不可逆，等值比对）。

附带工具库：md5 类（标准 MD5，oracle 实测 hashlib 等价）、
d3des 全家（deskey/des/des2key/des3key/makekey/make2key/make3key/
Ddes/D2des/D3des、`Df_Key` 表）——仅双域阶梯被登录链使用。

## 四、EncodeFunc：2048 个生成函数

`encode_func(i)` @0x200a50：i≤0x7ff 跳转表（2048 项）派发到
`encode_func_0..2047`（PUB32 符号齐备）。每个函数是同一生成器的实例
（~0x220B，统一 ~20.3k 指令/次，其中 md5 占 19k）：

```c
void encode_func_N(BYTE* in, BYTE* out) {
    BYTE scratch[K];              // K = 8 或 9
    scratch[j] = in[src_j] {+,-,|,^,&} imm…;   // 每步 0~2 个子操作
    md5 m; m.Update(scratch, K); m.Finalize();
    memcpy(out, m.Digest(), 16);
}
```

复刻方式：`tools/fsverify_gen.py` 用 capstone 提升每个函数的运算程序
（轻量数据流：追踪输入指针寄存器 → 识别「字节装载+算术链+游标存储」
三段式；符号间隙溢出以首 `ret` 截断），**unicorn oracle 全量对拍
2048×3 随机输入零失配**后生成 `internal/fsverify/encode_programs.go`
（紧凑字节码）与 `testdata/fsverify/encode-vectors.txt`（2048 golden）。

三表（`tables.go`）直接自 .data/跳转表提取：
`g_encode_order`（置换）、`g_cond_value`（2048×16B）、
`g_client_code` blob 头 16B×2048（变长 blob 只有头 16B 上线）。

## 五、Go 复刻清单（internal/fsverify）

| Go | 对应 DLL |
|---|---|
| `ModuleVersion=300` / `EncodeNum=2048` | GetVersion / GetEncodeNum |
| `EncodeID(i)` | GetEncodeId（g_encode_order） |
| `ClientCond(i)` / `ClientCode(i)` | GetClientCond / GetClientCode |
| `EncodeFunc(id, in16) [16]byte` | EncodeFunc + encode_func_N 程序表 |
| `VerifyClientToken(cur, prev, prevValid, src, token)` | COuterHook::VerifyMessage 语义 |
| `DecodeAccount(slot)` / `DecodePassword(slot)` | 双槽解密（原地语义→返回明文串） |
| `EncodeAccountFields(acct, derived)` | 客户端编码方向（测试/种子） |

验证：`go test ./internal/fsverify`（golden 双槽 + 密文锁定 + 回环 +
**全量 2048 encode 向量** + 三表抽查 + 令牌三态 + 常量锁）全绿。

## 六、未接线与后续

- 现代客户端不走 op01 握手（首飞即 op02），生产配置亦禁用本插件——
  登录链保持现状（密文摘要域身份，见 §13/账号库节），本包独立自洽。
- 若需官方黑金方言全握手：gateway 加 SERVER_SET_ENCODE 下发 +
  CheckMessageIllegal 同款 op01/04/07/10 令牌门 + op02 版本门取
  `ModuleVersion`；op02 明文账号直接可由 `DecodeAccount` 取得
  （对应 B 线记忆「official 分支升级路」）。
- `g_client_code` 变长 blob 的 16B 之后内容从未上线，语义未查（客户端
  侧若消费全 blob 需另行逆向 fxgamelogic）。
