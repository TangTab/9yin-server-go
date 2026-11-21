# fork 登录加密逆向与「注册账号」移植全记录

> 2026-11-11。对象：`E:\Workspace\project\game\9yin_server1\build\9yin-game-native-menu.exe`
> （fork 线服务端，同模块 `github.com/local/9yin-go-server` 的姊妹构建，含
> `internal/auth` 的登录密码学完整实现，无源码）。本文记录逆向方法、算法
> 全貌、验证手段与移植落点。

## 一、要解决的问题

本仓 `internal/auth` 只实现了哈希身份边界（`ParseLoginAccountIdentity`：
op02 账号槽密文 → `acct:v1:` 键），拿不到明文账号，也因此做不了 fork GM 页
那样的「注册账号 + 登录验密」。fork exe 的 GM 页（127.0.0.1:19062）有
`POST /api/register {account,password}`，且登录不再自动建号、注册后强制验密。

## 二、逆向方法

1. **符号与行号**：`go tool objdump -s "internal/auth\."` 拿到全部函数的
   指令流 + pclntab 行号（源文件名都在：`field_cipher.go`/`login_password.go`/
   `login_credentials.go`/`account_key.go`）。`go tool nm` 拿数据符号地址，
   PE 节表算 VA→文件偏移，直接从 `.data`/`.rdata` 抽表与常量。
2. **指令级仿真器**（`tmp/fork-auth-re/emulate.py`）：对 objdump 文本逐条
   执行（寄存器/栈/全局表），零转录风险。关键坑：Go asm 统一 **(src, dst)**
   操作数序；`CMPQ a, b` 语义是 `a-b`；栈增长检查的 `0x10(R14)` 是**低位**
   栈守卫（守卫须小于 SP 才不触发 morestack）。
3. **fork oracle**：把 fork exe 在隔离目录拉起（`NINEYIN_SERVER_ROOT` 指向
   junction 资源树；资源缺口用 `resources.7z` 全量解包补齐），JSON 存储模式，
   调真实 `/api/register` 落库，拿账号键与口令校验子做逐字节比对。

## 三、算法全貌（全部经 oracle 五组向量验证）

### 3.1 域密码 `field_cipher.go`

- **fcDesCore = 标准 DES**。d3des 表系（pc1/pc2/totrot/bigbyte/bytebit/SP）
  与公版逐值一致；KAT `key=133457799BBCDFF1, pt=0123456789ABCDEF →
  85E813540F0AB405` 双证（仿真器 + Go `crypto/des`）。**移植直接用
  `crypto/des`**，无自研轮函数。
- **双域固定键**（.data 抽取）：
  - field1（账号槽）：`66f928e8a1009b76 29a01f8def458919`
  - field2（密码槽）：`106eba557761c77e 1eea0deb676293dc`
  - 每域 [0:8]=阶梯 1/3 级加密键，[8:16]=第 2 级解密键（EDE 三级）。
- **奇偶预白化掩码**（块序号 %2 选择，4×LE32 与块词异或后再 bswap 进阶梯）：
  field1 Even/Odd、field2 Even/Odd 四个 16 字节常量（见 `field_cipher.go`）。
- **fcTransform128 阶梯**（16 字节块 → 4×BE32 半块对）：
  ```
  (a,b)=E(w0,w1)  (c,d)=E(w2,w3)    两半独立（键A 加密）
  (e,f)=D(a,c)    (g,h)=D(b,d)      左左/右右交叉（键B 解密）
  (i,j)=E(e,g)    (k,l)=E(f,h)      再独立（键A 加密）；输出=(i,j,k,l)
  ```
- **pad16**：明文零填充到 16 整倍（**不是** PKCS7）。
- **EncryptFields(account, password) → (账号槽密文, 密码槽密文)**，确定性，
  全程无解密路径——客户端/服务端各自加密后比对密文。

### 3.2 口令派生 `login_password.go`

```
EncryptPassword(pw)      = base64( AES-256-CBC(passwordKey, 零IV, PKCS7(pw)) )
ComputeLoginPassword(pw) = rotate_half( base64( md5和低字节 || EncryptPassword(pw) ) )
passwordKey              = "nengbupojiemexiechengxuhaoxinku" + 0x00   （32B）
ComputePasswordVerifier  = EncryptFields(账号, 登录密码)[1]   // 落库的校验子
```

### 3.3 账号键 `account_key.go`

`AccountKeyFor(name) = "acct:v1:" + hex(SHA256("nineyin/login-account/v1"
+ 0x00 + LE32(len(ct1)) + ct1))`，其中 `ct1 = EncryptFields(name,"x")[0]`。
**与本仓登录侧公式完全同域**——注册键与登录帧密文哈希天然一致，存量
`acct:v1` 键零迁移（oracle 实证）。

### 3.4 登录帧 `login_credentials.go`

`ParseLoginCredentials(frame) → (账号槽密文, 密码槽密文)`：头 25B、
[0x15:0x19]=LE32 账号槽长，账号槽后 u32 密码槽长 + 密码槽，结构与
`ParseLoginAccountIdentity` 同帧同源。

## 四、fork 实证语义（GM 页行为）

| 输入 | 响应 |
|---|---|
| 新账号注册 | `{"account":名,"account_id":N}` |
| 已注册账号再注册（换密码） | `{"account":名,"updated":true}`（**upsert＝改密**） |
| 空账号 / 空密码 | 400「账号不能为空」/「密码不能为空」 |
| 未注册登录（fork 默认） | 拒绝（登录不再自动建号） |

## 五、移植落点（本仓）

| 文件 | 内容 |
|---|---|
| `internal/auth/field_cipher.go` | 双域 DES 阶梯 + `EncryptFields`（`crypto/des` 为核） |
| `internal/auth/login_password.go` | 口令派生链 + `ComputePasswordVerifier` + `AccountKeyFor` |
| `internal/auth/login_credentials.go` | `ParseLoginCredentials` 槽提取 |
| `internal/auth/field_cipher.go` 同时提供逆阶梯 `DecryptAccountFields`：登录槽密文解出明文账号（EDE 反序换向），供 m655/m607 回传真实用户名（`officialAccountLabel`）——非 ASCII 名回空（tag6 窄串约束） |
| `internal/auth/field_cipher_test.go` | fork oracle 五组向量（1/2 块账号、多长密码）+ 逆解回环 + 非 ASCII 过滤 |
| `internal/role` | `Account.PasswordHash` + `RegisterAccount`（JSON/MySQL 双后端；accounts 表 0001 schema 已带 `password_hash VARBINARY(255)`，**零迁移**） |
| `cmd/protocol-probe/role_store.go` | `login` 验密：注册账号强制比对、存量账号（无校验子）免密放行；`NINEYIN_REQUIRE_REGISTRATION=1` 时未注册拒绝（fork 行为），默认关闭兼容存量自动建号 |
| `cmd/protocol-probe/gm_register.go` + `gm_panel.go` | GM 页 `POST /api/register`（upsert 语义对齐 fork） |
| `internal/gmweb/web/index.html` | 首页「注册账号」区块 |
| `cmd/protocol-probe/login_frame_test.go` | 客户端形态 op02 帧全链回放（注册键=登录键、同帧重放放行、换密帧拒绝） |

## 六、验证链

1. 仿真器 = fork 字节码（构造性等价）。
2. 仿真器 + 包装层 → fork exe `/api/register` 落库值：5 组账号（键 + 校验子
   base64）**逐字节相等**（含 2 块账号掩码路径与 4 块长密码）。
3. Go 移植 → 同 5 组向量全绿（`field_cipher_test.go`）。
4. Go `crypto/des` 单级与仿真器 core 随机对拍 3 组 + KAT 相等。
5. 客户端形态帧集成测试：`TestClientStyleLoginFrameRoundTrip` 全链
   （`ParseLoginAccountIdentity` ↔ `AccountKeyFor` 键一致、验密生效）。

## 七、逆向期中间产物

`tmp/fork-auth-re/`：objdump 全文（`auth-disasm.txt`）、符号表、表抽取与
仿真器脚本、oracle 向量（`oracle-vectors.json`）、fork 落库快照。fork 服务端
隔离运行目录 `tmp/fork-run/`（553MB 解包资源 + junction）验证后已删除。
