# GPKitSvr 全解与 Go 复刻

> 对象:`GPKitSvr.dll`(x86,2009-09-10)/ `GPKitSvr64.dll`(x64,2013-02-22)/
> `GPKitSvrD.dll`(x86 Debug,2009-09-10),BGserverCJ/Debug 与 release 两份完全相同。
> 无 PDB 随包(GPKitSvrD 内嵌路径 `e:\GPKBuild\src\server\GPKitSvr\Dll_Debug\GPKitSvr.pdb`
> 证明是独立源码树的外部产品,与 Fs* 系列不同源)。
> 分析方法:capstone 静态反汇编 + RTTI 重建(x64)+ 32 位 PowerShell P/Invoke
> 动态行为验证(自制无害载荷,见 tools/gpk_probe/)。

## 1. 定位:不是公共库

- 全套 BGserverCJ(Debug+release)中,**唯一消费者是 release/FsGame.dll**(即
  game_world 进程),导入表直连:
  - `?GPKCreateSvrDynCode@SGPK@@YGPAVIGPKSvrDynCode@1@XZ` → `SGPK::GPKCreateSvrDynCode()`
  - 全局 `?g_pGpk@@3PAVIGPKSvrDynCode@SGPK@@A`(`g_pGpk`,FsGame.pdb @0x108a7da8),
    唯一使用者 **PlayerBaseModule.obj**
- Debug/FsGame.dll 零 GPK 引用(Debug 构建未带 GPK)。
- GPKitSvr64 是 x64,根本无法载入 x86 game_world——SDK 套件原样随包,未用。
- 目录中不存在 `gpkconf.ini` / `GPKLog.txt` / `DynCodeBin\` / `AuthData.dat`,
  说明本快照从未实际配置过 GPK 检查资源。

GPK = Game Protect Kit,命名空间 `SGPK`。服务器侧"动态码"反外挂组件:
服务器生成/持有动态校验码 → 随消息下发 → 客户端执行回报 → 服务器校验 →
失败计数/惩罚(AddBuffToPlayerForGpk 上 debuff)。

## 2. 导出面与类层级

三个 DLL 均只导出一个符号:

```c
SGPK::IGPKSvrDynCode* __stdcall GPKCreateSvrDynCode();
```

实现 = `new CSvrDynCode(0x414 字节) + 构造`。

类层级(x64 RTTI 确认):

| 类 | 接口 | 版本 |
|---|---|---|
| `CSvrDynCode` | `SGPK::IGPKSvrDynCode` | x86+x64 |
| `CMainDynCode` | (内部) | x86+x64 |
| `CCSAuth` | `SGPK::IGPKCSAuth` | **x64(2013)新增** |
| `CCodeLoader` | (内部) | x64 |

### CSvrDynCode 布局(x86,总 0x414)

| 偏移 | 类型 | 含义 |
|---|---|---|
| +0x000 | ptr | vtable(@0x1001f620,9 槽) |
| +0x004 | u8 | flag(构造=1;Init 成功后=0) |
| +0x008 | ptr[250] | CMainDynCode* 数组(码组槽) |
| +0x3F0 | u16 | 码组计数(Init 成功置 1) |
| +0x3F2 | u16 | 备用 |
| +0x3F4 | ptr | 当前活跃 CMainDynCode(析构用) |
| +0x3F8 | 0x18 | CRITICAL_SECTION |

### CMainDynCode 布局(x86,总 0x414;vtable @0x1001f6c4)

| 偏移 | 类型 | 含义 |
|---|---|---|
| +0x000 | ptr | vtable(9 槽) |
| +0x004 | ptr[256] | CodeEntry* 数组 |
| +0x404 | u32 | 已加载条目数 count |
| +0x408 | u32 | 调用序号(每个接口调用 ++,防重放) |
| +0x40C/+0x410 | u32 | 备用 |

### CodeEntry 布局(x86,new 0x58)

| 偏移 | 含义 |
|---|---|
| +0x00 | codebase = VirtualAlloc(NULL, sz, MEM_COMMIT, PAGE_EXECUTE_READWRITE) |
| +0x04 | codesize |
| +0x08 | bin 数据缓冲(fopen 数据文件内容) |
| +0x0C | bin 数据大小 |
| +0x10 | **fn1** = codebase(1) 的返回值(Encode 入口) |
| +0x14 | **fn2** = codebase(2) 的返回值(Check/Decode 入口) |
| +0x18..0x57 | 代码文件全路径名(strncpy 0x3F) |
| +0x57 | NUL 终止 |

## 3. IGPKSvrDynCode 接口(9 槽 vtable,x86 地址)

| 槽 | x86 | 签名 | 语义 |
|---|---|---|---|
| 0 | 0x100159C0 | dtor | 析构(删除 250 码组+AuthData 缓冲+CS) |
| 1 | 0x10015410 | `(codeDir, dataDir)→int` | **Init**:装载动态码组,返回加载数 |
| 2 | 0x10015560 | `()→int` | **RandomPick** = `rand() % count`,空则 -1 |
| 3 | 0x100155B0 | `(packed, &out)→int` | **GetData**:hiword(packed)=idx → out=数据指针,ret=数据大小 |
| 4 | 0x100155F0 | `(a, b, codeid)→bool` | **Call1** → entry.fn1(b, codeid&0xFFFF) |
| 5 | 0x10015630 | `(a, b, codeid)→bool` | **Call2** → entry.fn2(b, codeid&0xFFFF) |
| 6 | 0x10015BF0 | Release | delete this |
| 7 | 0x10015700 | `(path)` | 惰性加载 AuthData.dat(全局一次性) |
| 8 | 0x10015740 | `()→obj` | 创建 0x110 辅助对象(x64 版已移除,返 0) |

CMainDynCode 的 9 槽一一对应(槽 1=装载、槽 2=rand 选择、槽 3=取数据、
槽 4/5=转调 entry.fn1/fn2、槽 6=Release),vtable @0x1001f6c4。
**CsVrDynCode 槽 2-5 按 hiword(codeid) 定位 CMainDynCode 后转发。**

## 4. Init 装载协议(动态验证定案)

FsGame 侧调用(release/FsGame.dll @0x100c93b0 PlayerBaseModule GPKInit):

```
g_pGpk = GPKCreateSvrDynCode();            // 失败 → "GPK Init Failed"
g_pGpk->Init("DynCodeBin\Server", "DynCodeBin\Client");  // slot1
                                          // ret<=0 → "GPK Code Load Failed"
```

slot1 → new CMainDynCode → CMainDynCode::Clear(旧条目 VirtualFree+free)
→ `FindFirstFileA(codeDir + "\*.bin")` 枚举,GetFileAttributes 过滤目录,
对每个命中文件(动态验证,tools/gpk_probe/matrix.ps1 二分定案):

1. **代码文件** = 枚举到的 `codeDir\X.bin` 本身:
   `fopen(rb) + fread(≤0x2800)` → `VirtualAlloc(RWX)` 拷入 →
   **`call codebase(1)` 与 `call codebase(2)`**(cdecl,载荷必须 `ret 4`),
   返回值即 entry.fn1 / entry.fn2。
2. **数据文件** = `codeDir + basename(dataDir) + "\" + basename(codeDir命中文件)`
   = **`DynCodeBin\Server\Client\X.bin`**(嵌套子目录!与直觉的
   `DynCodeBin\Client\X.bin` 不同;x64 2013 版实现为 `dataDir\X.bin`,版本差异)。
   读入 → entry+0x08/+0x0C。
3. entry 名字段 = 代码文件全路径;`count++`(上限 256)。

**即官方载荷形态:`Server\*.bin` 是一段以 `call(1)/call(2)` 自举的
x86 机器码(返回 Encode/Check 两个函数指针),`Server\Client\*.bin`
是配套挑战数据。服务器把任意代码当函数执行——这是"动态码"的本义。**

载荷 ABI 细节(探针实测):
- call(1)/call(2):cdecl、参数 1/2、需 `ret 4`;
- fn1/fn2:调用方 `push arg2; push arg3; call fn` 且不清栈 → 载荷需
  `ret 8`;参数由 CMainDynCode 槽位转发(探针返回值显示转发时对
  codeid 低 16 位有重排,见 tools/gpk_probe/chain.ps1)。

## 5. 一次性初始化(构造函数内)

`GPKCreateSvrDynCode` 构造时若全局 `inited==0`:

1. `init_tableA()`(0x14AA0):表 A @0x10022CC8(**.bss**)填 **70 个函数指针**
   (0x10001A30…0x10014440);
2. `init_tableB()`(0x14D60):表 B @0x10022E58 填 **70 个函数指针**;
3. 解析 **`gpkconf.ini`**(自制 INI 解析器 0x15020,`#` 注释、`key = value`),
   7 个开关键:CheckMem / CheckProcName / CheckWindow / CheckModName /
   CheckCode / CheckDriver / CheckScanMod → 全局 0x10022FCC…0x10022FE4;
4. `load_AuthData("AuthData.dat")`(0x14A10):fopen+fread **0x3E80(16000)字节**
   → 全局缓冲 [0x10022FC0](旧指针换出至 [0x10022FC4])。该缓冲即
   CCodeLoader(x64)/70 对检查函数的密钥素材来源。

### hack 警告表与 70 对检查

`.rdata` 0x1F160 起为 **`"hack 10 warning"` … `"hack 79 warning"` 共 70 条**
(等宽 16 字节槽),另有 `"run on windows 64bit"`、`"client no admin"` 两条
特殊告警。70 条 ↔ 70 对函数(表 A = 生成器/编码器,表 B = 校验器)严格对应:

- A[i] 生成第 i+10 号检查的动态码/密钥材料(反汇编 A[0]@0x10001A30:
  TEA 风格多轮混合,常量 0x23132312/0x21524111/0x23973235/0x42A48222,
  全局密钥 @0x1001F150..0x1001F15C = `3990458D 850FE045 0761EE8A 18A164C3`);
- B[i] 校验客户端回报。

x64(2013)将两张表合并为 .data @0x43DB0 / @0x44080 各 70 项共 140 个函数
(初始化代码 0x15400+ 段提取,见 tools/gpk_probe/gpk64_tables.json)。

## 6. FsGame 协议(PlayerBaseModule,gpk 调用簇 @0x100C5F00-0x100C9500)

- **OnReady**(@0x100C8820 处理器):周期挑战
  `idx = g_pGpk->RandomPick()` → 玩家自定义属性
  `gpk_code = idx`、`NpState = 0`;`g_pGpk->Call1(outbuf, …)` 编码
  (失败 → `"GPK Encode Failed!!"`)→ 转码(0x100C5C20)→ 消息字段
  **`gpk_msg`** 下发;取客户端码失败 → `"GPK GetCltDynCode Failed!!"`。
- **回执处理**(@0x100C6540):读消息 `gpk_code` 字段,缺失 →
  `"GPK Not Find gpk_code!!"`;`g_pGpk->Call2(gpk_code_str, 0xC4, out196)`
  (slot5,输出 196 字节,首字节 `0x69` 为正常签名,随后取 out[1..3] 组合值)
  → 失败:`NpMisCount += 2`、`gpk_code = 0`、`NpState = 0`。
- **惩罚**:`PlayerBaseModule::AddBuffToPlayerForGpk(IKernel*, PERSISTID&)`
  (FsGame.map @0x100C85C0)——检测命中给玩家上惩罚 buff。
- 关闭(@0x100C5F00):`g_pGpk->Release()`。

## 7. x64(2013)新增:CCSAuth::CheckAuthReply

`CCSAuth`(vtable @0x3_6B10,3 槽:dtor/CheckAuthReply/dtor)实现
`SGPK::IGPKCSAuth`。**CheckAuthReply(this, &out, pkt, len)** @0x16C10:

1. `len != 0x80` → throw `"packet length error!"`(**回执包固定 128 字节**);
2. `*(u16)pkt == this->expectIdx`(对象 +0x0C),否则
   throw `"CheckAuthReply Idx Exception!"`(序号防重放);
3. **mask 校验:`pkt[0:4] == ~pkt[4:8]`**(not 后比较),否则
   throw `"packet mask check error!"`;
4. idx 分派:
   - `0x0A < idx < 0x15` 且某全局开关关闭 → 直接通过(跳过);
   - `idx ≤ 0x14` 且非 (0x14,0x50) 且 ≠ 0x0A →
     `ret = table2[idx](0x78, this->state(+0x10), 0)`(table2=校验函数表);
   - 其余 → throw `"check error"`;
5. 异常路径写 **`GPKLog.txt`**(ofstream)并返回错误码。

x64 CMainDynCode 槽数组扩为 256×8 字节,Init 后对象 vtable 由 CSvrDynCode
切换为 CMainDynCode(接口状态机),`IsDebuggerPresent` 亦在 x64 导入表。

## 8. Go 复刻(internal/gpkit)

机制级复刻 + Go 载荷 ABI(官方 `.bin` 机器码载荷本就不随包分发,二进制
兼容无意义;载荷由 `PayloadLoader` 注入,默认实现 `GPK1` 签名格式):

```go
svr := gpkit.Create()                       // GPKCreateSvrDynCode
svr.LoadConf(iniBytes)                      // gpkconf.ini 七开关
svr.SetAuthData(dat)                        // AuthData.dat ≤16000
n   := svr.Init(codeDir, dataDir)           // Server\*.bin + Server\Client\*.bin 配对
idx := svr.RandomPick()                     // rand()%count
data, _ := svr.GetData(uint32(idx) << 16)   // 挑战数据
v, _ := svr.Call1(a, b, uint32(idx)<<16)    // entry.fn1
err := gpkit.CheckAuthReply(pkt, expectIdx, gpkit.AuthTable)  // x64 语义
gpkit.HackWarning(n)                        // "hack N warning" 10..79
```

- `Conf`:`CheckMem/CheckProcName/CheckWindow/CheckModName/CheckCode/
  CheckDriver/CheckScanMod`(自制 INI:`#` 注释、`key = value`)。
- 默认载荷格式 `GPK1|u16 genId|u16 checkId`:映射到内置 70 检查
  (xxtea 风格混合,官方全局密钥 3990458D/850FE045/0761EE8A/18A164C3)。
- `CheckAuthReply`:0x80 长度、u16 idx、`pkt[0:4]==~pkt[4:8]` mask、
  分派校验表。
- 表 A/B 原始函数地址(RVA)存档于 `tableA`/`tableB` 变量供审计。

测试:临时目录构造载荷+数据文件 → Init/RandomPick/GetData/Call1/Call2/
CheckAuthReply 正反例/Conf 解析/HackWarning 全表,`go test ./internal/gpkit` 全绿。
