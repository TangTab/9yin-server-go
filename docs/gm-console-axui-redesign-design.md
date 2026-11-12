# GM 控制台 AXUI 重设计 · 页面与文档设计（基线）

- 日期：2026-11-12
- 状态：**实施中**（P0 服务端加固 + P1 骨架 + P2 首批 4 组 + P3 全量迁移已落地；附录 A 45/45 板块完成，旧版下线待用户验收后执行）
- 范围：`internal/gmweb/web`（前端全面重写）+ 游戏/面板服务端目录与轮询接口的性能加固
- 关联：`docs/gm-service-rpc-split-design.md`（面板两种形态与 internal API 契约，本文不改其架构，只在其上加固）
- 前端基座：`internal/gmweb/web/static/` 下已就位的 AXUI v3 三件套
  （`ax.esm.js` 41k 行运行时、`ax.css` 30k 行样式、`ax-api.ts` 1 万行 AI 友好类型声明）

---

## 实施记录（2026-11-12 · P1 骨架 + P2 首批 4 组）

**落地物**：

- 新页 `web/console-ax.html` + `static/app/`（main/api/state/ui/app.css + panels/registry、
  role、item、fashion、scene、pending）——`internal/gmweb/gmweb.go` 增挂
  `/console-ax.html` 路由；行为测试 `TestGMConsoleAxPageServed`（页面引用 + 全部
  模块资产可取）。
- 已迁移分组（附录 A 打勾）：**组1 角色状态、组2 货币与物品、组4 外观与坐骑、
  组6 传送与场景**——共 21 个板块；其余 8 组走 `pending.js` 占位（说明 + 跳旧版）。
- 传送/踢馆 183 个 preset 从旧 console.html **脚本提取**（tmp/inject_presets.py，
  gitignored）注入 scene.js，零手抄。
- 框架行为：目录并发闸 ≤2 + 内存/sessionStorage 双层缓存 + 300ms 去抖搜索 +
  AbortController 取消过期请求；sessions 轮询 5s + 页面不可见暂停；命令全局互斥
  （M2）；Toast + 操作历史（M5）。**页面冷启动只发 1 个 sessions 请求**，目录
  全部跟随板块按需拉取（F1/F2/F3/F4/F5/F6 全部落地；ETag 部分待 P0）。

**与基线的偏差**：

1. **新页并行而非一步切换**（用户决策）：`/console-ax.html` 与旧版
   `/console.html` 并存，旧页零改动（§10 的"一步切换"风险项作废）；迁移完成前
   每页有「旧版控制台」出口。
2. **AXUI ESM 不会自动注册 Web Component**——ax-api.ts 头部声明与运行时不符
   （`regComp` 已导出但从不被调用；实测未注册时 193 个 ax-btn 0 尺寸）。
   main.js 启动时遍历命名空间对 `*Elem` 类逐个 `regComp()`。此为风险表第 1 条
   的实锤，已在该处文档化。
3. picker 初始拉取策略分两类：本地过滤型目录（时装/礼包/坐骑/站姿/天气）进板块
   即拉一次（每会话缓存后零成本）；服务端搜索型（道具）与全量无意义目录（称号，
   当前部署 3474 行全空名）保持输入触发。

**冒烟实测**（gm-panel 指向真服 127.0.0.1:19063，浏览器逐面板验证）：

- 组件注册后 193 个传送按钮正常渲染；「少林」过滤跨组命中 2 按钮、其余组隐藏；
  翻页 2/26 页正确。
- 时装目录 1255 条分页 50/页、分类下拉自动填充；坐骑 36 条；天气 49 条
  （Label 自带昼/夜标注）；道具服务端搜索「久仙刀」3 条真实命中（300 上限内）。
- 无游戏服/错 token 时：顶栏「游戏服不可达」、目录区错误提示、命令 Toast
  「游戏服不可达，命令未发送」——三处降级全部符合预期。
- `go test`：TestGM* 全绿、gm-panel 全绿、gmweb 无测试文件；唯一失败
  `TestShapedMultihitPicksUpNewEntrants` 为工作树预先存在的战斗系统失败
  （stash 验证与本次无关）。

**服务端数据缺口（非前端问题，记录备查）**：真服 `/api/titles` 3474 行**全部空名**
（称号数据.json 未装载，旧版同样只显示 ID）；`/api/idle-stances` 返回空数组。

**下一步**：P0 服务端加固（S1 名称内存缓存 / S2 目录预热 / S3+S4 ETag / S5 代理
缓存）→ P3 迁移其余 8 组（按附录 A 逐项）。

---

## 实施记录（2026-11-12 · P0 服务端加固）

**落地物**（S1-S5 全部，行为级测试覆盖）：

- **共享层** `internal/gmapi/etag.go`：`ETagMatches`（RFC 9110 弱比较、
  两侧剥 `W/` 前缀、逗号列表、`*` 全匹配）/ `SetETag` / `WriteNotModified` /
  `HashTag`（FNV-1a + 分隔符防拼接歧义）/ `WeatherTag`。两端共用同一口径。
- **S1** `cmd/protocol-probe/gm_name_cache.go`：`gmItemNames` 包级缓存，
  boot 后台全表 `SELECT item_id,item_name FROM game_items` 入内存
  （>50 万行放弃内存化回落批查；DB 不可用 60s 重试；fail-soft——未就绪
  时 `hub.itemNames` 走原批查路径，行为与旧版逐字一致）。`generation`
  每次成功装载 +1，供依赖名称的目录标签失效。
- **S2** `warmGMCatalogDirs()`（gm_catalog_dirs.go）：boot 后台触发 6 个
  sync.Once 目录 + neigong 源目录；`main.go` 起 goroutine 调用。
- **S3** 全部目录端点带 `W/` ETag + If-None-Match→304（`gm_etag.go` 的
  `gmWriteTaggedJSON`/`gmWriteContentTagged`，304 路径零序列化）：
  - Once 静态目录（clones/scenes/titles/mounts/idle-stances/recast-affixes/
    taolus/战斗假人套路）：装载后哈希一次，`gmStaticTagOf` memo；
  - 名称依赖目录（fashion-catalog/cards/hair-cards）：标签拼名称代数，
  名称缓存重载自动失效；items 搜索：标签拼 (q,view)，每 URL 独立；
  - 运行态小端点（meridians/giftboxes/neigongs/repute/origin/minimal-login）：
  内容标签或值标签。
- **manifest** `GET /api/catalog/manifest`（内嵌面板）与
  `/internal/v1/catalog/manifest`（internal API，面板代理
  `/api/catalog/manifest`）：`{version: "<boot>-g<名称代数>", catalogs:{…}}`。
- **S4** sessions 304：`gmSession.stateVer`（9 个 set* 方法统一 bumpState）
  + `hub.attachCount`（attach/detach）→ `sessionsTag`；两处 sessions 出口
  （/api/sessions 与 /internal/v1/sessions）共用 `writeSessionsTagged`。
- **S5** `cmd/gm-panel`：目录代理 30s 微缓存（key=目录名+查询串，失败不
  缓存）+ ETag 透传 + INM 命中直回 304；sessions 代理透传 ETag 并支持
  304；`gameRequest` 转发客户端 INM 并回传后端 ETag；面板本地天气目录
  也带 ETag/304。

**测试**：`internal/gmapi/etag_test.go`（匹配/哈希/天气标签）；
`cmd/protocol-probe/gm_p0_test.go`（名称缓存就绪/回退/放弃、15 端点
ETag 304 往返、manifest 形状、sessions 304 与 attach/setAction/detach
失效、预热并发幂等）；`cmd/gm-panel/etag_proxy_test.go`（ETag 透传、
INM 304 不打后端、30s 微缓存去抖、查询串分键、manifest 路由）。
全绿（唯一失败 `TestShapedMultihitPicksUpNewEntrants` 为工作树预先
存在的战斗系统问题，与本项无关）。

**与基线的偏差**：

1. S1 装载成功即停（不做周期重载）——名称表变动频率极低，重启进程刷新；
   代数（generation）机制保留，后续若加定时重载可直接复用。
2. 面板 sessions 微缓存保持 500ms（基线建议透传 304，实际两者都做了：
   缓存窗口内 INM 命中直接 304，不落窗口则转发 INM 给游戏服协商）。

**冒烟**（真服 19063 + 新面板 19075）：面板目录 30s 缓存生效（同键多请求
仅一次后端往返）；旧游戏服无 ETag/manifest → 空头透传/404 透传，无回归。
**S1-S4 的线上生效需重新编译部署游戏服**（start-9yin.bat 流程），部署后
§5.3 指标 1（itemNames SQL=0）/ 指标 2（二次刷新全 304）即达成——行为级
测试已在 httptest 层面验证同一语义。

---

## 实施记录（2026-11-12 · P3 全量迁移）

**落地物**：剩余 8 组 24 个板块全部迁入 AXUI 版（附录 A 45/45 ✔）——

- `panels/equip.js`（装备与铸造 5 板块）、`skill.js`（武学与修炼 5）、
  `battle.js`（战斗与演练 4）、`params.js`（游戏参数 3）、`quest.js`
  （任务与身份 5）、`uipref.js`（界面与操作 2）、`debug.js`（调试工具 3）、
  `misc.js`（其他·GM 邮件 1）；registry.js 全部切换为真实模块，
  `pending.js` 保留作后续新增板块的占位模板。
- `ui.js` 新增 `actionBtn`（参数命令按钮统一封装：build() 点击时构造
  payload → 确认/互斥/Toast/历史/会话刷新），全部参数表单类按钮收敛到它。
- 洗练槽位表（22 槽）与词条折叠区（真服实测 184 词条）、门派 34 项、
  快捷键 keyArgs 文本命令族、S2C 探针常用 18 项矩阵等全部按旧版语义迁入。

**本轮修的两个坑**：

1. skill.js 7 处 numField 结果裸传（漏 `.node`）导致整组加载失败
  （appendChild 非 Node 崩溃）；已写静态扫描脚本全模块复查归零。
2. **embed 静态资产无缓存头 → 浏览器把旧 JS 模块缓存死**：修完 bug
  重新部署后页面仍执行旧代码。`gmweb.go` 给 /static/ 与页面直出加
  `Cache-Control: no-cache`（本地面板正确性优先；这也保护旧版页的
  脚本更新）。

**冒烟**（真服 + 新面板，浏览器逐板块）：28 个新板块全部渲染正常
（含组5 修复后复验）；洗练词条 184 复选项、槽位 22 项、门派 34 项、
探针 19 按钮等抽查通过。`go test`：gmapi / gm-panel / TestGM* 全绿。

**事故记录**：调试中 `taskkill /IM gm-panel.exe` 误杀用户在跑的 19064
面板，已立即用 start-gm-panel.bat 恢复并验证 200；后续进程管理一律
按监听端口取 PID 精准操作。

**待办**：用户验收新版后删除旧版 console.html / console-core.js /
console-features.js / console.css 与 pending.js（附录 A 完成即满足
§9-P3 删除条件）。

---

## 实施记录（2026-11-12 · P4 体验增强 + F3 通道补完）

**落地物**：

- **F3 目录/会话 304 通道前端补完**（`api.js` 重写缓存层）：目录缓存键
  = `url@manifest版本`，每 60s 静默校验 `/api/catalog/manifest`，版本
  变化（游戏服重启/名称缓存重载）即全量失效；TTL 过期条目带
  If-None-Match 协商刷新，304 空体续期零传输。sessions 轮询同样带
  INM。旧游戏服（无 manifest/ETag）自动退化为纯 TTL 缓存，fail-soft。
- **Ctrl+K 全局命令面板**（`palette.js`）：跨 12 组搜板块直达 +
  148 常用传送 + 35 踢馆全量直达（选中即发 switch_scene_preset），
  ↑↓/Enter/Esc 键盘流；顶栏 🔍 同入口。
- **操作历史抽屉**（M5 收口，`ui.js` mountHistoryDrawer）：顶栏 🕰 呼出
  最近 100 条（时间/板块/结果/回执），复制 JSON / 一键清空。
- **亮暗主题切换**：顶栏 🌓，body scheme 属性 + localStorage 记忆；
  app.css 补亮色变量组（ax.css 组件自带亮色默认）。

**冒烟**（真服 + 新面板）：Ctrl+K 默认 10 项、「少林」跨板块+传送命中
3 项（前缀优先排序）；历史抽屉空态/复制/清空；亮色生效（bg 245,246,248）
并可切回暗色记忆。全部 JS 语法检查通过；go test 无回归。

---

## 1. 背景与问题诊断

### 1.1 现状

| 层 | 现物 | 规模 | 问题 |
|---|---|---|---|
| 页面 | `web/console.html` | 1242 行，40+ 个 `<section>` 手写 HTML | 单文件巨页，控件风格原始（裸 `<select size=10>`/`<button>` 堆叠） |
| 逻辑 | `static/console-core.js` + `console-features.js` | 282 + 1224 行 | 全局函数 + 内联 `onclick`，无模块化；所有目录在页面加载时全量拉取 |
| 目录 API | 游戏服 `/internal/v1/catalog/*` ← 面板 `/api/*` 代理 | 19 个目录/状态端点 | 无缓存、无 ETag；部分端点每次请求同步打 MySQL |
| 轮询 | `/api/sessions` | 2.5s 间隔 | 无 304；GM 命令发送前还额外先拉一次 sessions |

### 1.2 「加载数据时卡服务器」根因分析（代码级）

打开/刷新 console.html 时的请求链与阻塞点：

```
页面加载 (console-features.js:1205-1224)
 ├─ refresh() + setInterval(refresh, 2500)          → /api/sessions（轻，hub 快照）
 └─ 18 个目录加载并发发出：loadReputeStatus / loadClones / loadGiftBoxes /
    loadWeather / loadScenes / loadCardCategories / loadAllCards / loadAllTitles /
    loadMeridians / loadHairCards / loadFashionCatalog / loadMounts /
    loadIdleStances / loadMinimalLogin / loadOriginStatus /
    loadCombatDummyTaolus / loadGMTaolus / loadGMNeigongs
```

1. **MySQL 批查风暴（主因）**：`fashion-catalog` / `cards` / `hair-cards`
   每次请求都经 `hub.itemNames`（`cmd/protocol-probe/inventory_mutations.go:657`）
   按 **64 条/批、批间串行** 查 `game_items`。时装目录 500+ 套装、皮肤卡牌几千条 ID
   → 单端点几十条串行 SQL；三端点并发且**零缓存**。GM 面板把 DB 连接池/查询吞吐打满，
   与游戏循环的持久化操作同库竞争 → 表现为「服务器卡」。
2. **首访懒解析大表**：`gm_catalog_dirs.go` 的 clones/scenes/titles/mounts/
   idle-stances/recast-affixes 用 `sync.Once` 在 **HTTP handler 内同步解析** GBK INI
   （全表扫描 + 解码），与游戏循环抢 CPU（首次打开页面时集中爆发）。
3. **零 HTTP 缓存**：目录响应无 ETag/版本号，任何刷新全量重拉；无 304。
4. **全量下发 + 前端渲染压力**：全量目录一次性 JSON（道具目录全表 10 万+ 行虽已限搜 300，
   但卡牌/称号/时装仍是全量），浏览器端 `<select>` 渲染数千 option 也卡。
5. **命令通道放大**：`command()`（console-core.js:241）每次点击先 `await refresh()`
   再 POST；命令经无缓冲 channel 交给**游戏会话循环**同步执行
   （`gm_panel.go:103-118`，750ms 超时）——高频连点时游戏循环被 GM 命令淹没。

### 1.3 结论

「卡服务器」不是前端渲染单方面问题，是 **前端加载策略（并发风暴+全量拉取）× 服务端目录实现（每请求打库+懒解析）× 命令通道（前置轮询+会话循环执行）** 三层叠加。重设计必须前后端一起改，只换 UI 皮不解决问题。

---

## 2. 目标与非目标

**目标**

1. G1 视觉与操作现代化：AXUI v3 组件体系（Web Components），暗色主题为默认，统一表单/反馈/确认交互。
2. G2 数据加载不卡服：页面加载对游戏服的冲击从「18 并发请求 × DB 批查」降到「≤2 并发 × 纯内存快照 + 304 复用」。
3. G3 操作效率：高频操作（发道具/选套装/传送）点击路径 ≤2 步；有搜索、有历史、有危险操作确认。
4. G4 可维护：前端 ES 模块化按组拆分；板块-命令映射有单一事实清单（文档即清单）。
5. G5 零功能裁剪：现有 40+ 板块、53 个命令 action（`internal/gmapi/validate.go`）全部保留，未实现命令继续给「等待服务端支持」提示。

**非目标**

- 不改 `gm-service-rpc-split-design.md` 定下的进程拓扑（游戏服内嵌面板 / 独立 gm-panel.exe 两形态与 Bearer internal API 不动）。
- 不引入构建工具链（webpack/vite）：保持 `go:embed` 直出 + 原生 ES Modules，浏览器直跑。
- 不做多角色多会话管理界面（单服单角色定位不变，会话列表仅作状态展示）。
- 不做鉴权体系（localhost-only + 现有 token 机制不变）。

---

## 3. 总体方案概览

```
┌────────────────────────── 浏览器 ──────────────────────────┐
│  console.html（壳）                                        │
│   ├─ <ax.css> + <script type=module src=static/app/main.js>│
│   ├─ app/main.js    启动、路由（hash）、全局状态订阅          │
│   ├─ app/api.js     fetch 封装：并发闸(≤2) / ETag缓存 /     │
│   │                 去抖搜索 / 统一错误规范 / 命令队列        │
│   ├─ app/ui.js      通用件工厂：CatalogPicker / CmdBtn /    │
│   │                 StatChips / HistoryDrawer               │
│   └─ app/panels/<组>.js  每组一个模块，import() 按需加载      │
│        （首次切入板块才拉该板块目录 → 请求数从 18 并发降为 0） │
└──────────────┬─────────────────────────────────────────────┘
               │ /api/*（同源）
┌──────────────▼──────────────┐    ┌──────────────────────────┐
│ gm-panel.exe（独立形态）      │    │ 游戏服内嵌面板（serveGM） │
│  代理 + 目录微缓存 30s/ETag  │    │ 同一套 handler           │
└──────────────┬──────────────┘    └──────────────┬───────────┘
               │ /internal/v1/* (Bearer)          │
┌──────────────▼──────────────────────────────────▼───────────┐
│ 游戏服目录层加固（本文 §5）：                                  │
│  · itemNameCache：boot 后台全量载入 game_items → 纯内存查名    │
│  · 目录预热：sync.Once 保留，boot 后台 goroutine 全部先触发     │
│  · ETag/版本：目录响应带弱校验，If-None-Match → 304            │
│  · sessions：状态版本号 + 304；轮询间隔 2.5s→5s                 │
└──────────────────────────────────────────────────────────────┘
```

三层各自独立见效：服务端加固（P0）单独上线即可消除卡服；前端重写（P1-P3）替换体验；两者通过 §6 的 API 契约对齐。

---

## 4. 页面设计

### 4.1 信息架构（导航重组）

沿用 `console-core.js:88-100` 已定稿的 11 组分组（2026-11-02 定版），不动分组语义，只动呈现：

| # | 分组 | 板块（现有 h2 标题） | 主要交互形态 |
|---|---|---|---|
| 1 | 角色状态 | 在线角色 / 五维属性 / 实力等级 / 活力属性 / 性格善恶 / 称号管理 | 状态卡 + 表单 |
| 2 | 货币与物品 | 碎银官银黄金怒气 / 礼包与物品 / 世界宝箱 | 快捷数值表单 + 目录搜索 |
| 3 | 装备与铸造 | 所有道具与换装 / 洗练 / 模板生成 / 冶炼 / 兵录设置 | 目录搜索 + 槽位选择 |
| 4 | 外观与坐骑 | 时装 / 时装身份装发放 / 风物志卡牌 / 风物志头发 / 坐骑 / 待机站姿 | 目录搜索（大头） |
| 5 | 武学与修炼 | 门派 / 武学套路 / 修为 / 经脉 / 经脉周天设置 | 下拉 + 数值表单 |
| 6 | 传送与场景 | 常用传送 / 踢馆传送 / 自定义切图 / 场景天气 / 副本 / 世界 BOSS | 按钮矩阵 + 目录搜索 |
| 7 | 战斗与演练 | 训练木桩 / 战斗假人 / 伙伴系统 / 御风神水 | 参数表单 + 开关 |
| 8 | 游戏参数 | 怪物刷新 / 数据存盘 / 移动参数 | 开关 + 数值表单 |
| 9 | 任务与身份 | 场景任务 / 暮色剧情 / 九阴志 / 身份系统 / 门派声望 | 步骤按钮 + 状态 |
| 10 | 界面与操作 | 快捷栏位 / 快捷键方案 | 工具按钮 |
| 11 | 调试工具 | S2C 探针 / 聊天气泡 / 动作剧情 | 参数表单 + 探针矩阵 |

导航规则：
- 左侧 `ax-menu`（树形，两级：组 → 板块），当前板块高亮，组可折叠（记住状态，沿用现有 localStorage 键 `gm-tab` / `gm-group-collapsed` 语义平滑迁移）。
- URL hash 路由：`#/group/section`，可收藏/刷新恢复/后退。
- 未列入分组的板块自动落入「其他」组的兜底规则保留（改名不失联）。

### 4.2 布局设计（框架级线框）

```
┌────────────────────────────────────────────────────────────────────┐
│ 顶栏（常驻）                                                         │
│ [☰] 九阴 GM   [●当前服 127.0.0.1:19063]   ─ 角色卡 ax-stats ─        │
│   角色 名 ｜ 门派 ｜ 场景 ｜ 坐标 ｜ 碎银/官银/黄金 ｜ 怒气▾ ｜ 修为     │
│                                    [🕐操作历史] [◐ 主题] [← 服务器概览] │
├──────────┬─────────────────────────────────────────────────────────┤
│ 左导航     │  面包屑：货币与物品 / 礼包与物品                            │
│ ax-menu   │ ┌─────────────────────────────────────────────────┐     │
│ ▾角色状态  │ │ 板块标题 + 说明（ax-callout info，可折叠）          │     │
│  在线角色  │ │                                                 │     │
│  五维属性  │ │  … 板块内容（骨架屏 ax-skeleton → 数据到达后渲染）…   │     │
│ ▸货币与物品│ │                                                 │     │
│  礼包与物品│ │  [主操作 ax-btn prim] [次操作 ax-btn] [危险 ax-btn   │     │
│ ▸装备与铸造│ │   error + confirm]                               │     │
│  …        │ └─────────────────────────────────────────────────┘     │
│           │  Toast 反馈浮层（右下，命令成功/失败/排队）                  │
├──────────┴─────────────────────────────────────────────────────────┤
│ 底栏：命令通道状态（●已连接 / ○游戏服不可达·自动恢复中）｜最后刷新时间     │
└────────────────────────────────────────────────────────────────────┘
```

- 顶栏角色卡 = 现在 `#player-card` chips 的升级：`ax-stats` 组件化，数字变化时高亮闪一下；断线时整条变 `ax-callout warn`「游戏服不可达，自动恢复中」（沿用现有 503 分支语义）。
- 底栏命令通道状态替代现在顶栏的 `#command-status` 文本，全局可见。
- 窄屏（<960px）左导航自动收起为抽屉（`Drawer` 模块），顶栏 ☰ 按钮呼出。

### 4.3 全局组件与交互模式（本设计的核心抽象）

以下 5 个模式覆盖全部 40+ 板块，新板块开发 = 组合这 5 件，不再各写各的。

#### M1 CatalogPicker · 目录选择器（治卡服的前端一半）

统一替代现在 8 处「全量拉目录 → `<select size=N>` / 搜索按钮 + 下拉」的写法。

```
┌──────────────────────────────────────────────┐
│ 🔍 输入中文名或 ID…（去抖 300ms）        [分类▾]│
├──────────────────────────────────────────────┤
│ ▸ 服务端搜索（/api/items?q=…&view=…，上限300） │
│ ▸ ax-select search + async + manual 模式       │
│ ▸ 结果 ax-pagination（前端分页 50/页）          │
│ ▸ 最近选用（localStorage per-catalog，前 8 条）  │
│ ▸ 选中行展示副信息（configId/分类/品质色点）      │
└──────────────────────────────────────────────┘
```

- AXUI 落点：`ax-select`（`search` + `async` + `manual` 属性，见 ax-api.ts `AxSelectAttributes`）承载下拉与搜索框，分页用 `ax-pagination`，加载中 `ax-skeleton`。
- 服务端搜索的目录（道具已有 300 上限搜索）：输入即查，绝不拉全量。
- 小目录（<200 行：经脉/周天/天气/站姿/难度档）：一次拉全量 + 前端过滤，sessionStorage 缓存。
- **所有目录缓存键带版本号**（§5.3 manifest version），服务端目录重建后自动失效。

#### M2 CmdBtn · 命令按钮

统一替代 `onclick="command({action:...})"` 散写：

- 三档视觉：普通 `ax-btn`、主操作 `prim`、危险 `error`（删除/清空/重置类，强制 `ax.confirm` 二次确认，确认文案带影响面，如「将删除全部已学武学套路」）。
- 行为：点击 → 禁用自身 + 顶栏转圈 → POST 命令 → 结果 Toast（成功含服务端回执摘要；`unsupported action` 显示「等待服务端支持」原语义）→ 刷新会话快照（单次，不再前置 refresh）。
- 并发纪律：同一时刻同一会话只允许 1 条在飞命令（按钮全局互斥），排队提示，杜绝连点淹没会话循环。

#### M3 StatForm · 数值设置表单

碎银/怒气/五维/修为/移动参数等「数值 + 应用」类板块统一为：

- `ax-fields` 布局 + `ax-number`（min/max/step 来自现有 input 的界值，服务端校验口径不变，`internal/gmapi` 界值常量为准）。
- 预设快捷片（如怒气 +10/+50/回满）做成 `ax-btn` 组。
- 应用成功后该字段绿色 `ax-flag` 短暂标注新值。

#### M4 ToggleRow · 开关行

怪物自动追击/每日真气硬闸/伙伴辅助/最小登录模式等布尔+参数类：

- `ax-checkbox`（或 `ax-switch` 形态）+ 关联参数输入，`立即应用` 或 `保存` 语义与现状一致（哪些是进程级/角色级/会话级在说明文案保留现有描述）。

#### M5 FeedbackFlow · 反馈流

- **Toast**（右下堆叠，自动消失）：命令结果、复制成功、缓存失效提示。
- **操作历史** `Drawer`（顶栏 🕐）：最近 100 条 {时间, 板块, 动作, 参数摘要, 结果}，localStorage 持久；一键复制 JSON（排查用）。
- **原始状态**：`#state`（会话 JSON）保留为板块内 `<details>` 折叠 + 复制按钮，不再裸占屏。

### 4.4 AXUI 组件选型总表

| 用途 | AXUI | 备注 |
|---|---|---|
| 左导航 | `ax-menu`（Menu 模块） | 两级树、折叠、高亮；窄屏转 Drawer |
| 面包屑 | `ax-more`/自研轻量 | hash 路由联动 |
| 角色状态卡 | `ax-stats` + `ax-badge`/`ax-flag` | 顶栏常驻 |
| 板块容器 | 原生 `<section>` + ax.css 布局类 | 每板块一卡 |
| 说明文案 | `ax-callout type=info` | 现 `.muted` 长说明迁入，可折叠 |
| 表单 | `ax-fields` / `ax-input` / `ax-number` / `ax-select` / `ax-checkbox` / `ax-textarea` | M1/M3/M4 |
| 目录选择 | `ax-select`（search/async/manual）+ `ax-pagination` + `ax-skeleton` | M1 |
| 按钮 | `ax-btn` | M2 |
| 确认 | `ax.confirm`（Promise） | 危险操作 |
| 反馈 | Toast 模块 / `ax-flag` | M5 |
| 历史/详情 | Drawer / Dialog 模块 | M5、词条列表等 |
| 进度类 | `ax-progress` | 怒气/真气等 0-100 值 |
| 主题切换 | `ax-twilight`（dark 默认） | 暗色与现版一致 |
| 空态/错误 | `ax-result` | 目录缺失、游戏服不可达 |

> 类型口径：前端一律对照 `static/ax-api.ts`（组件 Attributes 为 kebab-case、模块 Options 为 camelCase；`elem.set()/reset()/clear()/on()` 统一操作）。

### 4.5 板块级设计要点（只列有特殊设计的，其余套 M1-M5）

- **在线角色**：`ax-stats` 大卡（含 `ax-progress` 怒气条）+ 会话 JSON 折叠 + 「弹出游戏 GM 面板」「刷新」。
- **所有道具与换装**：M1 目录搜索（服务端搜索已是口径）；选中后数量 `ax-number` + 「加入背包 / 立即穿戴」；品质色点取 colorLevel；宝物洗炼两个入口按钮挪为选中宝物后才可用（降低误触）。
- **传送矩阵（常用/踢馆）**：现有 100+ 按钮改为**分组按钮矩阵 + 顶部过滤框**（输入「少林」即时过滤所有分组），每按钮 CmdBtn；保留全部 preset 键。视觉密度用 ax.css 栅格收紧，不再 10 行 `<p>` 堆叠。
- **洗练词条列表**：词条全量列表（PropBuilt/SkillPack 两大折叠组）迁入 Dialog 全屏选择器，复选 + 筛选 + 「武器类型」过滤，确认后批量提交（现 `recastChecked` 语义）。
- **战斗假人**：预设套路下拉 + 招式 `ax-select multiple`（替代顿号分隔 textarea，校验错配红框）；生成/清除分离，清除为危险操作。
- **GM 邮件**：`ax-fields` 分组表单（收件/标题/正文/附件），发送为危险确认（不可撤回）；成功 Toast + 清空表单。
- **S2C 探针**：常用 ID 矩阵保留，自定义区 `ax-number`×2；发送记录进操作历史。
- **冶炼/最小登录**：ToggleRow + 状态回显，说明文案原样保留（含 D:\ 对齐排障提示）。
- **副本**：目录下拉（懒加载该板块时才拉）+ 难度联动禁用（现 `syncCloneFun` 逻辑保留）。

### 4.6 主题与视觉

- 默认 **暗色**（`<html theme="dark">`，与现版配色接近：底 #0f1115 系），`ax-twilight` 提供亮色切换，选择存 localStorage。
- 视觉基调：金色 accent（延续现版 `--accent: #e8b34b` 的武侠金）+ AXUI 主题变量微调（只覆盖少量 CSS 变量，不 fork ax.css）。
- 品质色/状态色映射集中在 `app/ui.js` 一张表。

---

## 5. 数据加载与性能设计（治「卡服务器」）

### 5.1 服务端改造（游戏服 + 面板，P0，可独立先行上线）

| # | 改造 | 位置 | 说明 |
|---|---|---|---|
| S1 | **itemNameCache：全量内存化** | `cmd/protocol-probe`（新 `gm_name_cache.go`） | boot 后台 goroutine 一次性 `SELECT item_id,item_name FROM game_items` 进内存 map（数万行、几 MB，启动即查完）；`hub.itemNames` 改查内存。DB 不可用时 60s 重试直到成功；期间回退现状（ConfigID 显示）。**目录端点从此零 DB 查询** |
| S2 | **目录启动预热** | `main.go` 起 `go func` 逐个触发 `gm_catalog_dirs.go` 的 7 个 `sync.Once` | sync.Once 机制保留（仍是唯一装载点），只是把「首用户请求时解析」提前到 boot 后台；HTTP handler 永不再做同步 INI 解析（预热未完成的罕见窗口按现状等待 Once） |
| S3 | **目录版本 + ETag** | 各 catalog handler | 版本 = 目录内容 hash（装载完成时算一次）+ 进程 boot 时间戳；响应头 `ETag: W/"<ver>"`，`If-None-Match` 命中回 304 空体。新增 `GET /internal/v1/catalog/manifest` 返回全部目录版本汇总 |
| S4 | **sessions 版本 + 304** | `hub.snapshots()` 路径 | gmSession 状态每次变更 `stateVer++`，hub 汇总 hash 做 ETag；轮询未变化回 304 |
| S5 | **面板代理缓存升级** | `cmd/gm-panel/main.go` | 目录代理微缓存 500ms → 30s + 透传 ETag/304；sessions 代理透传 304 |
| S6 | **命令通道不加前置轮询** | 前端配合（见 5.2） | 服务端不变；`enqueue` 的 750ms 超时语义保留 |

> S1/S2 消除 DB 与 CPU 冲击（卡服主因）；S3/S4/S5 把「每次刷新全量重拉」变「304 空体」；三层叠加后页面加载对游戏服的负担 ≈ 0。

### 5.2 前端策略

| # | 策略 | 说明 |
|---|---|---|
| F1 | **按需加载目录** | 目录请求跟随板块：首次切入板块才拉该板块目录（`import()` 面板模块时发起）。首屏只拉 sessions。页面冷启动请求 18 → **1** |
| F2 | **并发闸** | `app/api.js` 全局目录请求队列，同时在飞 ≤2（304 命中不占闸）；搜索类即时请求单独快道（不与目录装载争抢） |
| F3 | **双层缓存** | 内存 Map（会话期）+ sessionStorage（键含 §5.1-S3 版本号，版本不符即弃）；manifest 每 60s 静默校验一次 |
| F4 | **去抖搜索** | CatalogPicker 输入 300ms 去抖 + AbortController 取消在飞旧请求 |
| F5 | **轮询降频** | sessions 轮询 2.5s→5s + ETag（304 空体近零成本）；页面不可见（`visibilitychange`）暂停 |
| F6 | **命令直发** | CmdBtn 不再前置 refresh；命令完成回执后单次刷新会话快照 |

### 5.3 指标与验收（P0+P1 完成后冒烟口径）

1. 冷启动 console.html：游戏服日志中 `itemNames` 相关 SQL 查询 **0 条**；目录 handler 无 INI 解析耗时（预热完成）。
2. 连续刷新页面 10 次：第二次起目录请求全部 304；游戏循环 tick 耗时无可感知抖动（对比刷新前基线）。
3. 首屏 JS 体积 ≤ 现 console-*.js 总和的 1.5 倍（AXUI 首次引入的一次性成本，之后不变）；首屏仅加载当前板块模块。
4. 全部 53 个 action 的按钮行为回归：成功/失败/未实现三态提示正确。

---

## 6. API 契约（新增/调整，均为增量）

| 端点 | 方法 | 变化 | 说明 |
|---|---|---|---|
| `/internal/v1/catalog/manifest` | GET | **新增** | `{version, catalogs:{items:"v12", cards:"v3", …}}`；面板代理为 `/api/catalog/manifest` |
| `/internal/v1/catalog/*`（19 个） | GET | 加 ETag/304 | 响应体形状**零变化**（前端旧页/新页/测试共用） |
| `/internal/v1/sessions` | GET | 加 ETag/304 | 同上 |
| `/api/*`（面板代理层） | GET | 缓存升级 + 304 透传 | 透传规则见 5.1-S5 |
| 命令端点 | POST | 不变 | `/api/sessions/{id}/command` 与 internal 版原样 |

前端模块 `app/api.js` 对上表的抽象：`catalog(name)`（带缓存/版本）、`sessions()`（ETag 轮询）、`command(action, params)`（互斥队列）。业务面板不允许直接 `fetch`。

---

## 7. 前端工程结构（替换 console-core/features）

```
internal/gmweb/web/
├─ index.html            # 服务器概览页（本次不动，仅换 ax.css 变量可选）
├─ console.html          # 重写为壳：顶栏/导航/路由出口，≤150 行
└─ static/
   ├─ ax.esm.js / ax.css / ax-api.ts   # 基座（已就位）
   └─ app/
      ├─ main.js         # 启动：路由(hash)、导航构建、全局状态
      ├─ api.js          # §6 抽象 + 并发闸/缓存/去抖
      ├─ state.js        # 会话快照订阅（顶栏角色卡消费）
      ├─ ui.js           # M1-M5 工厂 + 主题 + 品质色表
      └─ panels/         # 每组一文件，动态 import()
         ├─ role.js      # 组1 角色状态（6 板块）
         ├─ item.js      # 组2 货币与物品
         ├─ equip.js     # 组3 装备与铸造
         ├─ fashion.js   # 组4 外观与坐骑
         ├─ skill.js     # 组5 武学与修炼
         ├─ scene.js     # 组6 传送与场景
         ├─ battle.js    # 组7 战斗与演练
         ├─ params.js    # 组8 游戏参数
         ├─ quest.js     # 组9 任务与身份
         ├─ ui-prefs.js  # 组10 界面与操作
         └─ debug.js     # 组11 调试工具 + 其他兜底
```

- 模块全为 `<script type="module">`（ax.esm.js 是 ESM，天然配套）；无构建步骤，`go:embed` 直出（`gmweb.go` 只需把 `/static/app/` 一并嵌入——fs.Sub 已覆盖整个 static 目录，**无需改动**；仅当新增页面路径时才动 `gmweb.go` 的 known map，本次不加页面）。
- 每个面板模块导出 `render(container, ctx)` 纯挂载函数；板块注册表 `{id, group, title, module}` 单一事实来源，导航与「其他」兜底都从它生成。
- 旧文件 `console-core.js`/`console-features.js`/`console.css` 在 P3 完成后删除。

---

## 8. 文档设计（配套文档体系）

| 文档 | 时机 | 内容 | 维护规则 |
|---|---|---|---|
| `docs/gm-console-axui-redesign-design.md`（本文） | 现在 | 设计基线 + 实施记录 | 每阶段完成追加「实施记录」小节（体例同 gm-service-rpc-split-design.md），偏差写明 |
| 附录 A（本文内） | 现在 | 板块-功能-命令映射总表 = 迁移清单 | 每迁完一个板块打 ✔ |
| `docs/gm-console-api-reference.md` | P0 落地时 | 面板 `/api/*` ↔ internal `/internal/v1/*` ↔ handler ↔ 数据源 对照表（含 ETag 语义） | 新增/改端点必须同步，PR 检查项 |
| `static/ax-api.ts` | 已就位 | AXUI 组件 API 权威声明（本地副本） | 只随 axui 升级整体替换，不手改 |

写作口径：面向「下次打开本文的自己」，所有结论带 file:line 或实测数据；实施记录写偏差与原因，不重写设计。

---

## 9. 实施计划

| 阶段 | 内容 | 交付判据 | 依赖 |
|---|---|---|---|
| **P0 服务端加固** | S1 名称内存缓存 / S2 目录预热 / S3+S4 ETag / S5 代理缓存（§5.1） | §5.3 指标 1、2；`go test ./...` 全绿（gm_internal_test / main_test 增补 ETag 与 manifest 用例） | 无，**可先行单独上线**，旧页面立即受益 |
| **P1 前端骨架** | console.html 壳 + main/api/state/ui + 路由 + 顶栏角色卡 + 命令通道（M2/M5） | 空板块框架可跑；命令三态反馈正确；暗色主题 | P0 的 ETag（不依赖也可跑，304 只是优化） |
| **P2 高频板块** | 组 1/2/4（角色状态、货币物品、外观坐骑——目录选择器大头） | M1 CatalogPicker 上线；这些板块目录请求全部按需+304 | P1 |
| **P3 全量迁移** | 其余 8 组 + 附录 A 逐项打勾；删除旧 console-*.{js,css} | 附录 A 100% ✔；功能回归（53 action 三态） | P2 |
| **P4 体验增强**（可选） | 全局 Ctrl+K 目录搜索、传送过滤矩阵、操作历史 Drawer、亮色主题打磨 | — | P3 |

每阶段一个 commit 系列，P0 与前端解耦发布。

---

## 10. 风险与回滚

| 风险 | 缓解 |
|---|---|
| AXUI 组件行为与 ax-api.ts 声明有出入（声明由源码静态分析生成） | 以 `ax.esm.js` 实际行为为准，发现偏差修声明注释；P1 先做组件冒烟页（仅开发期，不入嵌入资产） |
| game_items 全表载入内存的峰值（极端大表） | 数万行量级实测；载入后立刻释放 rows；上限保护（>50 万行放弃内存化，回落批查+缓存） |
| ETag hash 计算本身的开销 | 目录装载完成时算一次（内容不变不复算）；sessions 用自增版本号拼串，不 hash 大 JSON |
| 新旧前端并存期混乱 | console.html 一步切换到新壳（P1），不留双页；旧 JS 在 P3 末删除 |
| 回滚 | 前端：git revert console.html + static/app 即回旧页；服务端 S1-S5 全部 fail-soft（缓存未命中回落原路径），无需开关 |

---

## 附录 A：板块-功能-命令映射总表（迁移清单）

> 「命令」列为 `command({action})` 的 action 值（契约层 `internal/gmapi/validate.go`，共 53 个白名单分支）；
> 「现 UI 形态」指旧 console.html 的控件；「新设计」引用 §4.3 模式。✔ = 已迁移。

| 组 | 板块 | 命令 / 端点 | 现 UI 形态 | 新设计 | ✔ |
|---|---|---|---|---|---|
| 角色状态 | 在线角色 | GET /api/sessions；popup_gm_panel | pre JSON + 按钮 | ax-stats 大卡 + 折叠 JSON | ✔ |
| | 五维属性 | set_five_attributes | 5×number+按钮 | M3（0=不改语义保留） | ✔ |
| | 实力等级 | set_power_level | number+按钮 | M3 | ✔ |
| | 活力属性 | set_vitals | 9×number | M3 分组 | ✔ |
| | 性格善恶 | set_alignment | 3×number | M3 | ✔ |
| | 称号管理 | GET /api/titles；grant_title / wear_title / cancel_title / clear_titles | 搜索+select size8 | M1 + 危险确认（clear） | ✔ |
| 货币与物品 | 碎银/官银/黄金/怒气 | set_silver / set_guanyin / set_gold / set_sp / add_sp / fill_sp / set_rage_recovery | 4 组 number+按钮 | M3 + 快捷片 + ax-progress(怒气) | ✔ |
| | 礼包与物品 | GET /api/giftboxes, /api/items；grant_item / wear_item / add_item / send_test_mail / giftbox_merchant_here | 过滤+select size6+通用输入 | M1 ×2 | ✔ |
| | 世界宝箱 | spawn_chest | 按钮 | CmdBtn | ✔ |
| 装备与铸造 | 所有道具与换装 | GET /api/items；grant_item / wear_item；tersure_wash(old/new) | 搜索+下拉+数量 | M1 + 数量 + 洗炼入口 |✔ |
| | 洗练 | GET /api/recast-affixes；recast_slot_prop/dire/polish/skill（slot 参数）；recast_add_affix / recast_clear_affix；tersure_wash_open | 槽位下拉+词条折叠列表 | 槽位 M1 + 词条 Dialog 全屏选择器 |✔ |
| | 模板生成 | gm_generate_weapon / gm_generate_armor / gm_generate_bracer | 下拉组 | M3 组合 |✔ |
| | 冶炼 | GET/POST /api/minimal-login | 按钮+状态 | M4 + 排障说明保留 |✔ |
| | 兵录 | binglu（bingluArgs：set/max 文本命令） | 下拉+3 number | M3 |✔ |
| 外观与坐骑 | 时装 | grant_all_fashion / grant_identity_suits / set_showequip_type | 按钮+下拉 | CmdBtn + M4(显示类型) | ✔ |
| | 时装/身份装发放 | GET /api/fashion-catalog；grant_fashion | 分类下拉+select size10+过滤 | M1（分类 tab 化） | ✔ |
| | 风物志卡牌 | GET /api/cards；grant_all_cards / equip_card / unequip_card | 分类+搜索+select size8 | M1 | ✔ |
| | 风物志头发 | GET /api/hair-cards；equip_hair | select size10 | M1（复用卡牌） | ✔ |
| | 坐骑 | GET /api/mounts；mount_ride | select size8 | M1 | ✔ |
| | 待机站姿 | GET /api/idle-stances；idle_stance_set（stanceAction） | 预设按钮+select size8 | 预设片+M1 | ✔ |
| 武学与修炼 | 门派 | set_faction；learn_all_neigong | 大分组下拉 | M1（分组 optgroup 保留） |✔ |
| | 武学套路 | GET /api/taolus, /api/neigongs；learn_taolu / forget_taolu / learn_neigong / forget_neigong / learn_all_taolu / forget_all_taolu | 下拉+层数 | M1×2 + 层数 number + 危险确认 |✔ |
| | 修为 | set_faculty / fill_current_neigong / set_fill_speed / set_act_day_limit / clear_team_fatigue / add_livepoint / set_hold_faculty / set_live_speed_power | 长表单 | M3 分组 + 危险确认 |✔ |
| | 经脉 | set_zhenqi_cap | checkbox+number | M4 |✔ |
| | 经脉周天 | GET /api/meridians；set_jingmai_tier | 2 下拉 | M1(小目录)+M3 |✔ |
| 传送与场景 | 常用传送 | switch_scene_preset（preset 键 ×100+） | 按钮矩阵 | 分组矩阵+过滤框 | ✔ |
| | 踢馆传送 | switch_scene_preset（preset=tiguan_* ×35） | 按钮矩阵 | 同上 | ✔ |
| | 自定义切图 | GET /api/scenes；switch_scene / teleport | 输入+过滤下拉+坐标 | M1(场景)+坐标表单+「填入当前坐标」 | ✔ |
| | 场景天气 | GET /api/weather；set_weather | 下拉+过渡秒 | M1(小目录)+M3 | ✔ |
| | 副本 | GET /api/clones；enter_clone / reenter_clone / destroy_clone | 2 下拉+按钮 | M1+难度联动禁用 | ✔ |
| | 世界 BOSS | worldboss_spawn / clear / list | 3 按钮 | CmdBtn（clear 危险） | ✔ |
| 战斗与演练 | 训练木桩 | spawn_dummy(hp) | number+2 按钮 | M3+快捷片 |✔ |
| | 战斗假人 | GET /api/combat-dummy/taolus；spawn_combat_dummy / clear_combat_dummies | 名字/血量/伤害+套路 textarea | 表单+ax-select multiple；清除危险确认 |✔ |
| | 伙伴系统 | bot_ensure/pull_here/follow_start/follow_stop/revive/support_configure | 按钮组+3 参数 | M4 + 按钮组 + bot 状态卡 |✔ |
| | 御风神水 | yufeng_add / yufeng_remove；add_item(ride_*) | 按钮 ×6 | CmdBtn 组 |✔ |
| 游戏参数 | 怪物刷新 | set_respawn_interval / set_corpse_ttl / set_lootbox_ttl / set_npc_chase | 3 number+checkbox | M3+M4 |✔ |
| | 数据存盘 | set_autosave_interval | number | M3 |✔ |
| | 移动参数 | set_move_speed / set_gravity；fightword_probe | 2 number | M3 |✔ |
| 任务与身份 | 场景任务验证 | scene_task_start / scene_task_submit_ready / scene_task_remove；authored_task_* ×4 | 按钮+任务 ID | 步骤条+ID 表单 |✔ |
| | 暮色剧情验证 | twilight_mechanism_* ×2 / twilight_npc_move_* ×2 / twilight_stages_* ×2 / twilight_difficulty ×4 | 按钮组 | 分组卡+难度档 M4 |✔ |
| | 九阴志 | ninjiuzhi_set/reset/status/event | 下拉+按钮 | M1(小目录)+状态回显 |✔ |
| | 身份系统 | GET /api/origin/status；origin_activate_all | 按钮+状态 | CmdBtn+状态卡 |✔ |
| | 门派声望 | GET /api/repute/status；repute_apply（reputeName/Mode/Value） | 下拉+number | M1+M3 |✔ |
| 界面与操作 | 快捷栏位 | save_shortcuts | 按钮 | CmdBtn |✔ |
| | 快捷键方案 | keybind（keyArgs 文本命令：bind/mode/default/save/load…） | 键位小表单+方案按钮 | M3+方案按钮组 |✔ |
| 调试工具 | S2C 探针 | custom_probe（msgId/subType） | 2 number+常用矩阵 | 表单+矩阵 |✔ |
| | 聊天气泡 | chat_broadcast / speech_len / speech_nul | 输入+3 按钮 | 表单+CmdBtn |✔ |
| | 动作剧情测试 | red_super_armor / action_state_sgmd / action_state_hsqs05 / action_state_stop / npc_bubble / drama_prompt | 按钮 | CmdBtn 组 |✔ |
| 其他（兜底组） | GM 邮件 | send_mail（收件/标题/正文/物品附件） | 表单 | M3 分组表单 + 危险确认（见 §4.5） |✔ |

> 板块与 action 的完整对照以代码为准：前端入口 `console.html` + `console-features.js`；白名单 `internal/gmapi/validate.go`；迁移时逐行核对，发现文档与代码不符**改文档并记录**。
