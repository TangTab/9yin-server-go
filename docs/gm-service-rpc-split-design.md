# GM 服务独立化设计：gm-panel 独立进程 + 内部 HTTP/JSON API

- 日期：2026-11-07
- 状态：**已实施**（阶段 1-4 落地于同日；本文件为设计基线 + 实施记录）
- 范围：`cmd/protocol-probe`（游戏服 + 内嵌 GM 面板）→ 拆出独立 `gm-panel.exe`，与游戏服之间以内部 HTTP/JSON API 通信
- 原则：**功能零裁剪**（GM 面板所有板块保留，未实现命令保持"等待服务端支持"提示），适配只动路由/传输层

## 实施记录（2026-11-07）

- 落地物：`internal/gmapi`（类型+Validate+场景/门派/天气目录+界值常量）、
  `internal/gmweb`（共享前端资产）、`cmd/protocol-probe/gm_internal.go`
  （`/internal/v1`：health/sessions/command/catalog 四类）、`cmd/gm-panel`
  （静态直出+代理+500ms 微缓存+预校验）、`gm_internal_test.go` /
  `cmd/gm-panel/main_test.go`（11 个行为测试）。
- 与基线的偏差：
  - `-gm-internal-listen` 默认非空（127.0.0.1:19063）但 token 为空时
    fail-soft 记日志不开监听（基线写 fail-fast；实测零配置启动不该被绊倒）。
  - 并行批次新增了 `/api/items` 目录端点（game_items 中文名搜索），已同步
    纳入 internal API（`/internal/v1/catalog/items`）与面板代理。
  - 版本戳用 build.bat 的 `-ldflags -X` 注入编译时间戳（`gmInternalVersion`
    / `panelVersion`）；面板启动比对，不一致仅记 WARNING。
  - 前端 `console-core.js` refresh() 增加 503 分支（"游戏服不可达"提示）；
    index.html 原有 offline 卡片逻辑不动。
- 冒烟实测（gmtest-sandbox，端口 19093/19094/19098/19099）：health 免鉴权
  200、无/错 token 401、会话/目录/命令代理逐字透传、杀游戏服后面板
  503 + 页面存活、预校验 400 不产生后端往返，全部符合 §5 契约。

### 增补：概览页「记录服务器」登记册（同日晚些时候）

参考 Java gm-web index.html 的 gm_servers 语义（列表存浏览器 localStorage、
卡片轮询在线状态、进入控制台/改名/移除）：

- `internal/gmapi/manage.go`：`MountManageAPI(mux, boundGameAddr, version)` 挂
  `GET /api/manage/status?s=host:port`（同源探活代理——浏览器跨端口被 CORS
  拦，由面板代打目标服免鉴权 health，映射回 Java 网关同形
  `{online, status:{running, listen_addr, server_root, connected_clients,
  uptime_sec, phase}}`；非法 s 400 文案逐字对齐 Java）与
  `GET /api/panel-info`（面板自述绑定地址，登记册据此给卡片打「本面板所连」
  徽标——只有该卡片的进入控制台有效，其余仅监控）。host:port 输入面严格
  收紧（仅 IP/localhost + 端口域），探活只打 health 不带 token。
- 两种面板形态共用上述挂载：独立面板绑 `-game-addr`；内嵌面板（serveGM）
  绑 `-gm-internal-listen`（未启用时空串，登记册无本面板所连卡片但探活可用）。
- 游戏服 `/internal/v1/health` 增补 Java GmRemote status 同形字段
  （running/listenAddr/serverRoot），healthInfo 由 main() 装配。
- `index.html` 新增登记册区块：添加表单（默认端口 19063）、卡片网格、
  改名/移除/进入控制台；首用预填「本机 127.0.0.1:19063」。
  `gm_panel_web_test.go` 契约同步二次修订（登记册与 /api/manage/status 是
  面板自身端点，不再是网关残留；仍禁 `__s=`/`console.html?s=`）。
- 冒烟：双形态 /api/panel-info 与 /api/manage/status 全过（含 online 映射、
  offline {online:false}、非法 s 400）；未开 internal API 的 Java 参考服
  （19081）与非本系服表现为未连接，属预期。

---

## 1. 背景与动机

当前 GM 面板是游戏服进程内的一个 HTTP server（`-gm-listen 127.0.0.1:19062`，前端 `go:embed web`）。
拆分动机：

1. **远程管理**：现面板只绑 loopback（鉴权即"本机可达"），拆分后面板可部署在其他机器。
2. **面板不随游戏服重启掉线**：浏览器打开的面板页面在游戏服重启后仍然存活，重连即可。
3. **前端独立迭代**：面板 UI 改动不再需要重编/重启游戏服。
4. **为多服管理预留**：Go 服 / Java 参考服 / 9yin_server1（另一位开发的新版 Go 服）未来若都暴露
   internal API，一套面板可管多后端（本文只预留版本化路径，不做多后端）。

## 2. 目标 / 非目标

**目标**

- GM 面板成为独立二进制 `gm-panel.exe`，与游戏服经内部 API（HTTP/JSON + Bearer token）交互。
- 游戏服进程内命令执行路径**原样保留**：单写者纪律、channel 交接、超时语义不变。
- 内嵌面板模式（`-gm-listen`）保留为回滚路径，双模式可共存。
- 现有 mux 行为级测试（`gm_panel_flow_test.go`、`gm_panel_web_test.go`、`gm_catalog_test.go`）在
  代码抽取后继续通过。

**非目标**

- 不引入 gRPC / protobuf（依赖代价见 §12）。
- 不做多游戏服聚合管理（只在 internal API 路径上版本化 `/internal/v1/...`）。
- 不做面板自身的账号体系（单机场景；远程部署的登录是开放问题，见 §14）。
- 不改变任何 GM 动作的执行语义（校验口径、饱和规则、帧序列都不动）。

## 3. 现状架构（拆分基线）

### 3.1 组件与位置

| 组件 | 位置 | 职责 |
|---|---|---|
| HTTP mux + 校验 | `cmd/protocol-probe/gm_panel.go:250`（`newGMMux`） | 路由、参数校验大 switch（gm_panel.go:333-573）、入队、等 reply |
| gmHub / gmSession | `cmd/protocol-probe/gm_panel.go:22` | 在线会话表、快照 `gmPlayerState`、`enqueue`（750ms 入队超时） |
| 命令 channel | `gmSession.commands`（无缓冲） | HTTP 侧 → 会话循环的唯一交接通道 |
| 命令执行器 | `cmd/protocol-probe/main.go:500`（`processGMCommand` 闭包） | 34 个 action，持有 `player`/`link`/各 store/场景运行时 |
| 会话循环消费 | `cmd/protocol-probe/main.go:1141` | 主 `select` 里 `case command := <-commandQueue` |
| 目录端点 | `cmd/protocol-probe/gm_catalog.go` | fashion/cards/hair：`fashionSuits`（运行时从客户端 form 资源加载，fashion_catalog.go:62）+ `hub.itemNames`（game_items 批查，main.go:220） |
| 天气目录 | `cmd/protocol-probe/gm_weather_presets.go:88` | 纯静态闭名单，无 DB / 无运行时资源 |
| 前端 | `cmd/protocol-probe/web/`（index/console.html + static/） | 轮询 `/api/sessions`，POST `/api/sessions/{id}/command` |
| 部署 | `build.bat`（单 exe）→ `deploy.bat`（build\* 全量覆盖）→ `server-cmd/start-9yin.bat` | 单进程启动 |

已知边角：前端 `web/static/console-core.js:204` 会 `fetch("/api/clones")`，Go 侧无此 handler（404）。
迁移时按死端点处理，**不**在 internal API 中补做。

### 3.2 命令数据流（现状）

```
浏览器 ──POST /api/sessions/{id}/command──▶ HTTP handler（校验 + enqueue 750ms）
                                              │ gmCommand{reply: chan error(1)}
                                              ▼
                              gmSession.commands（无缓冲 channel）
                                              │
                                              ▼
              连接主循环 select（main.go:1141）──▶ processGMCommand（唯一帧写者）
                                              │ 写 FxNet2 帧 + 更新 gmSession 快照
                                              ▼
                                    reply chan ──▶ HTTP handler（2s 等待）
                                              │
                     200 {"status":"sent"} / 409 / 504 ◀── 浏览器
```

### 3.3 必须保留的不变量

1. **单写者纪律**：任何路径（HTTP、今后的 RPC handler）都只允许向 `gmSession.commands` 入队；
   FxNet2 帧只由会话循环写。这是防双 goroutine 交叉写连接的设计（gm_panel.go:19 注释）。
2. **无缓冲交接语义**：GM 点击 = "游戏循环下一轮发送"，不是"留到之后某个场景再发"。
3. **超时不撒谎**：入队失败/执行未确认必须如实报 409/504，绝不报 success。
4. **校验与执行分离**：HTTP 侧校验输入形状（含 `maxPlayerSP+maxNeiGongRageCapAdd` 等
   需要进程内常量的界），执行侧仍按运行态饱和（如 set_sp 按当前有效上限）。

## 4. 目标架构

### 4.1 进程拓扑

```
┌─────────────────────────┐         ┌──────────────────────────────────┐
│  gm-panel.exe（可远程） │         │  9yin-game-server.exe            │
│                         │  HTTP   │                                  │
│  web/ 静态资源(embed)   │─JSON───▶│  internal API 127.0.0.1:19063    │
│  参数预校验(gmapi)      │ Bearer  │   ├ ListSessions / Health        │
│  /api/* 代理 + 微缓存   │         │   ├ ExecuteCommand(入队+等reply) │
│                         │         │   └ Catalog(fashion/cards/hair)  │
└─────────────────────────┘         │        │ gmCommand               │
                                    │        ▼ gmSession.commands      │
        浏览器 ──▶ 19064 ◀──────────│  会话循环 processGMCommand（不变）│
                                    │  内嵌面板 -gm-listen 19062（保留）│
                                    └──────────────────────────────────┘
```

端口分配：19061 游戏TCP（现状）/ 19062 内嵌面板（保留）/ **19063 internal API（新）** /
**19064 独立面板（新）**。

### 4.2 职责划分

| 职责 | 去向 |
|---|---|
| 前端静态资源、页面路由 | gm-panel（`internal/gmweb` 共享 embed） |
| 请求体形状校验（大 switch） | `internal/gmapi` 共享包：面板预校验 + 游戏服权威校验（同一份代码） |
| 天气目录（静态） | gm-panel 本地直出（共享包数据） |
| fashion/cards/hair 目录（依赖客户端 form 资源 + game_items 名字） | 游戏服 internal API，面板代理 |
| 在线会话表、命令执行、帧写入、DB 写 | 游戏服（不变） |
| token 校验 | 游戏服 internal API 侧（面板只持有 token） |

## 5. 内部 API 契约（v1）

Base：`http://127.0.0.1:19063/internal/v1`，全部 JSON。

### 5.1 端点

| 方法 | 路径 | 语义 | 对应现端点 |
|---|---|---|---|
| GET | `/health` | `{uptimeSec, sessions, version}`；无 token 也可访问（探活） | —（新） |
| GET | `/sessions` | `[]gmPlayerState`，与现 `/api/sessions` 同构 | `/api/sessions` |
| POST | `/sessions/{id}/command` | body = 现 command JSON body；校验→入队(750ms)→等 reply(2s) | `/api/sessions/{id}/command` |
| GET | `/catalog/fashion` | 身份装 7 组 + 时装 4 段（含 game_items 中文名） | `/api/fashion-catalog` |
| GET | `/catalog/cards` | 全量皮肤卡 | `/api/cards` |
| GET | `/catalog/hair-cards` | 发型卡 | `/api/hair-cards` |

状态码语义与现状**逐字保留**：

| 场景 | 状态码 | body |
|---|---|---|
| 校验失败 | 400 | `{"error": ...}`（文案不变） |
| 会话不在线 / 入队超时（busy） | 409 | 同现 `enqueue` 错误文案 |
| 已入队但 2s 内未确认帧发送 | 504 | `"game session did not confirm the frame send"` |
| token 缺失/错误 | 401 | `{"error":"unauthorized"}` |
| 游戏服 internal API 未启用 | —（不监听） | 面板侧表现为 503 |

### 5.2 鉴权

- `Authorization: Bearer <NINEYIN_GM_TOKEN>`；`subtle.ConstantTimeCompare` 比对。
- token 为空 ⇒ internal API 整体不启动（fail-fast，日志说明）。
- 默认只绑 `127.0.0.1`。绑非 loopback 属运维自担风险，文档明示（token 明文过网线需 TLS 或
  ssh 隧道，本文不内建 TLS，见 §14）。

### 5.3 超时预算（两跳总账）

| 环节 | 预算 | 备注 |
|---|---|---|
| 入队（internal API handler 内） | 750ms | 不变 |
| 等 reply | 2s | 不变 |
| internal API handler 总计 | ≤ ~2.8s | |
| 面板→游戏服 HTTP client | 3s（command）/ 1.5s（GET） | 超时映射为面板 503/502 |
| 浏览器最长等待 | ~3.2s | 面板自身处理 <100ms |

### 5.4 载荷限制

- command body：两端均套 `http.MaxBytesReader(64KB)`（lua_eval 上限 16KB 源码 + JSON 开销，
  与 internal/luascript 16KB 闸一致）。
- lua_eval 沙盒（超时+指令预算）只在游戏服执行，面板不解析源码。

## 6. 代码组织（搬迁与新增）

### 6.1 新增共享包

```
internal/gmapi/        # RPC 契约层（无 chan、无 DB、无 net 依赖）
  types.go             #   CommandBody(JSON)、Command(校验后纯数据)、PlayerState
  validate.go          #   Validate(body) (Command, error) —— 现 gm_panel.go:333-573 大 switch 原样抽取
  weather.go           #   天气闭名单 + 目录（自 gm_weather_presets.go 拆静态部分）
internal/gmweb/        # 前端资产共享
  embed.go             #   //go:embed web → 导出 fs.FS + Index/Static handler
  web/                 #   自 cmd/protocol-probe/web/ 迁入（两进程共享同一份前端）
```

抽取规则：`Validate` 是 `body → (Command, error)` 的纯函数；依赖的进程内常量
（`maxPlayerSP`、`maxNeiGongRageCapAdd`、`gmBaseAttributeCap` 等）以包级变量注入或搬入 gmapi。
`gmCommand` 中的 `reply chan error` 留在 protocol-probe，包装 `gmapi.Command`。

### 6.2 游戏服侧改动（cmd/protocol-probe）

- `newGMMux` 改为消费 `gmapi.Validate` 与 `gmweb`（行为不变，测试守门）。
- 新增 `cmd/protocol-probe/gm_internal.go`：internal API mux（端点见 §5.1），复用
  `hub.enqueue` 与 reply 等待逻辑（与现 command handler 共用一个小函数，不复制）。
- 新 flag：
  - `-gm-internal-listen`（默认 `127.0.0.1:19063`，空=禁用）
  - `-gm-internal-token`（默认取 env `NINEYIN_GM_TOKEN`；启用监听但 token 为空 ⇒ 启动报错）
- `-gm-listen` 内嵌面板**保留**（回滚路径 + httptest 测试载体）。

### 6.3 新进程 cmd/gm-panel

- flag：`-listen`（默认 `127.0.0.1:19064`）、`-game-addr`（默认 `127.0.0.1:19063`）、
  `-game-token`（默认取 env `NINEYIN_GM_TOKEN`）、`-token`（面板自身鉴权，默认空=不启用，见 §14）。
- 路由：
  - `/`、`/console.html`、`/static/*` → gmweb 本地直出；
  - `/api/weather` → gmapi 静态目录本地直出；
  - `/api/sessions`、`/api/sessions/{id}/command`、`/api/fashion-catalog`、`/api/cards`、
    `/api/hair-cards` → 代理 internal API（路径映射 `/api/X` → `/internal/v1/Y`）；
  - command 代理前先用 `gmapi.Validate` 预校验（快速 400，游戏服侧仍权威校验）。
- 会话列表微缓存 500ms（前端轮询 + 多标签页去抖）；`/health` 探活失败时面板返回
  `503 {"error":"game server unreachable"}`，页面顶部横幅提示（前端小改，功能位不动）。
- 不持有 DB、不读客户端资源、不写任何游戏状态 ⇒ 游戏服重启对它是普通网络故障。

### 6.4 新增 GM 动作的演进路径（不因拆分变复杂）

加一个 GM 动作 = `gmapi` 加字段 + `Validate` 加 case + `processGMCommand` 加 case +
前端按钮。两进程同仓同版本发布，无契约漂移窗口；`/health` 返回 `version`（编译时间戳）供
面板检测版本不匹配并提示。

## 7. 故障与生命周期语义

| 场景 | 行为 |
|---|---|
| 游戏服重启中 | 面板代理得 connection refused → 503 + 页面横幅"游戏服不可达"；页面不白屏 |
| 游戏服起来了、会话未上 | `/sessions` 返回空数组（现状语义）；command 409 "not online" |
| 面板先于游戏服启动 | 正常；首次代理请求前探 `/health` |
| 面板挂了 | 只影响管理入口，游戏零影响 |
| token 轮换 | 改 env.server 重启两进程；面板 401 时提示"token 失效"，不重试风暴（指数退避） |

## 8. 配置与启动

### 8.1 env.server（deploy.bat 生成、永不覆盖的本地文件）新增一行

```
set "NINEYIN_GM_TOKEN=<guid>"
```

### 8.2 start-9yin.bat 变更（server-cmd\* 按 skip-existing 部署，需手动同步目标机）

1. 读 env.server（现机制不变）；若 `NINEYIN_GM_TOKEN` 未定义 ⇒ 用 PowerShell
   `[guid]::NewGuid().ToString("N")` 生成一次并**追加**写入 env.server（幂等）。
2. 传递给两进程：游戏服 `-gm-internal-token=%NINEYIN_GM_TOKEN%`；面板
   `-game-token=%NINEYIN_GM_TOKEN%`。
3. 面板作为第二个前台窗口（`start "GM Panel" gm-panel.exe ...`）；游戏服退出时面板自然残留，
   属可接受（探活失败自显横幅）。

### 8.3 build.bat 变更

追加第二条 `go build -trimpath -o build\gm-panel.exe ./cmd/gm-panel`。
deploy.bat 无需改动（build\* 全量覆盖自动带走新 exe）；README 启动说明补一段。

## 9. 测试计划

| 层 | 测试 | 基线 |
|---|---|---|
| gmapi 抽取 | 现有 mux 测试不改断言直接跑（`gm_panel_flow_test.go`、`gm_panel_web_test.go`、`gm_catalog_test.go`） | 阶段1门禁 |
| gmapi.Validate | 表驱动：34 action × 合法/非法载荷，文案逐字对齐现 handler | 新增 |
| internal API | httptest：401（无/错 token）、400/409/504 映射、catalog 降级（itemNames=nil 回退 ConfigID） | 新增 |
| 面板代理 | httptest 假后端：透传状态码/body、超时→503、微缓存、健康检查 | 新增 |
| 端到端冒烟 | 本机起游戏服(internal API)+面板，真实浏览器走一遍 12 个已落地动作 | 手工清单 |

## 10. 分阶段实施（每阶段可独立合入）

1. **抽取 gmapi/gmweb**：纯代码搬家，行为零变化，现测试守门。
2. **游戏服 internal API**：新 flag + 新 mux + 测试；`-gm-listen` 照旧。
3. **cmd/gm-panel**：代理 + 静态直出 + 前端横幅小改 + 测试。
4. **构建部署**：build.bat / start-9yin.bat / env.server.default / README；F: 目标机手动同步
   server-cmd（按既有规则）。
5. **观察期后收尾**：内嵌面板标记 deprecated（但保留一个版本周期），docs 更新。

## 11. 明确不做 / 待后续批次

- GM 批次2（记录帧）/ 批次3（实体）/ 批次4（副本）仍按原计划在内嵌路径上落地——拆分与
  功能批次正交，新动作天然同时进两进程（§6.4）。

## 12. 备选方案与取舍

| 方案 | 结论 | 理由 |
|---|---|---|
| gRPC + protobuf | 否 | go.mod 仅 3 个直接依赖（mysql 驱动/sqlmock/x/text，golua 本地 replace），引 grpc 树与项目极简依赖+单二进制哲学冲突；面板本就讲 HTTP/JSON，收益仅是强类型——用共享 Go 包 `gmapi` 已获得同等类型安全 |
| stdlib net/rpc | 否 | 半弃用、无 HTTP 语义（超时/状态码/中间件），与现有 mux 复用目标相悖 |
| 保持内嵌 | 可接受的退路 | 若最终只在单机用，`-gm-listen` 保留即此方案，零成本回滚 |
| WebSocket 推送会话列表 | 否（暂） | 前端现为 2s 轮询 + 微缓存已够；流式留给多服聚合时再做 |

## 13. 风险与回滚

| 风险 | 缓解 |
|---|---|
| 校验双份漂移（面板/游戏服） | 单一来源 `gmapi.Validate`，两处调用同一函数 |
| 超时语义在代理层被吞 | 状态码逐字映射表（§5.1）+ 代理层测试断言 |
| token 泄露面（env.server 明文） | 单机场景接受；文档注明远程部署需隧道；面板 401 退避 |
| 部署多一个常驻进程被漏启动 | start-9yin.bat 一并拉起；缺面板时内嵌 `-gm-listen` 仍在 |
| 双写者回归（有人在代理层直接写帧） | internal API handler 只接触 `hub.enqueue`；代码评审红线 + 注释 |

**回滚**：不启动 gm-panel.exe、移除 start 脚本第二行即可回到纯内嵌模式；internal API flag
置空即完全关闭。游戏服命令执行路径全程未改，无协议层回滚需求。

## 14. 开放问题

1. 面板远程部署时自身的登录方案（token 页面 / IP 白名单 / 保持仅隧道访问）——v1 不做，默认仍
   绑 loopback，远程靠 ssh 隧道。
2. 非 TLS 明网传输是否要内建自签 HTTPS——倾向否，等真实部署需求。
3. `/api/clones`（前端 fetch 但无 handler 的死端点）在迁移时是否顺手摘除——建议摘，单独提交。
4. 多后端聚合（面板同时管 Go/Java/9yin_server1）何时做——等 9yin_server1 是否暴露 internal API
   有结论再立项；`/internal/v1` 版本化已为此预留。
