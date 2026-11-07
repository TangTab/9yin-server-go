# 九阴兼容服务端（可运行、可编译副本）

这个目录只保留服务端运行和编译所需内容，不包含 IDA 数据库、反汇编、
审计缓存、历史 EXE 或旧客户端副本。

## 运行

双击 `启动本地九阴服务.bat`。固定端口：

- 区服列表：`127.0.0.1:4000`
- 游戏服务：`127.0.0.1:19061`
- GM 页面（独立面板进程，优先）：`http://127.0.0.1:19064/` → `http://127.0.0.1:19064/console.html`
- GM 页面（游戏服内嵌，回滚形态）：`http://127.0.0.1:19062/`
- GM internal API（仅供 gm-panel 调用，仅本机 + token）：`127.0.0.1:19063`

GM 面板有两种等价形态：独立进程 `gm-panel.exe`（19064，经 internal API
19063 与游戏服交互，游戏服重启后面板不掉线）与游戏服内嵌（19062，回滚
路径）。两形态共用同一份前端与命令校验；token 由 start-9yin.bat 首次
启动自动生成到 env.server（`NINEYIN_GM_TOKEN`）。单独拉起/重开独立面板
用 `server-cmd\start-gm-panel.bat`（读同一份 token；全无 token 时首次
运行也会自动生成，但要求游戏服用同一 token 启动）。设计见
`docs/gm-service-rpc-split-design.md`。

账号：test001
密码：test001

## 编译

双击 `build.bat`，或在本目录运行：

```powershell
go test ./...
go build -trimpath -o .\build\9yin-game-server.exe ./cmd/protocol-probe
go build -trimpath -o .\build\gm-panel.exe ./cmd/gm-panel
```

编译产物写入 `build`，不会自动覆盖根目录中正在测试的服务端 EXE。

源码目录：

- `cmd`：服务端和配套命令
- `internal`：协议、角色、资源及网络实现
- `migrations`：MySQL 表结构迁移
- `scripts`：服务端 Lua 脚本（见下节）
- `docs`：研究笔记与协议知识库（原 `spec/` 现为 `docs/spec/`，文本抽取数据在 `docs/text-extract/`）
- `testdata`：测试所需的最小协议夹具

## 数据库迁移重放

`cmd/sql-replay` 按版本序重放 `migrations/` 下的迁移（文件约定见
`migrations/README.md`，沿用 Java 服务端权威存档）：

- `NNNN_*.sql.gz` — tar+gzip 单成员存档（mysqldump 风格多语句，文件自身
  幂等），校验和取内部 SQL 文本
- `NNNN_*.sql` — 本仓库手写增量，语句以 `-- migrate:statement` 分隔

台账记在独立的 `sql_replay_ledger` 表（`schema_migrations` 是随 0002
配置数据 `REPLACE INTO` 的普通表，不承载版本判定）；已应用且校验和一致
的版本自动跳过，改动已应用版本会拒绝执行。

```powershell
go run ./cmd/sql-replay -status   # 对照台账查看已应用/待应用（只读）
go run ./cmd/sql-replay           # 重放待应用版本
go run ./cmd/sql-replay -force    # 无视台账全量重放（缺表补建、按存档恢复配置）
go run ./cmd/sql-replay -list     # 不连库，列出文件与校验和
```

DSN 依次取 `-dsn` 参数、`NINEYIN_MYSQL_DSN` 环境变量、`env.server`。
`-embedded` 可改用编译时嵌入的副本（默认读磁盘上的 `migrations/`）。

## Lua 脚本

服务端内嵌 Lua 5.5 运行时（纯 Go，见 `third_party/golua`）。`scripts/`
目录下的脚本在启动时加载，GM 面板可执行任意 Lua 代码并热重载脚本目录。
脚本运行在沙盒内（无 io/os，500ms 超时），目前用于编写原生未实现的
NPC 服务（如"任务"）逻辑。API 参考见 [scripts/README.md](scripts/README.md)。

## 资源读取（MPQ 三层仓 + 读缓存）

游戏资源经 MPQ 三层资源仓读取（`internal/mpq/res`），根目录布局：

- `res-base.rep`：全量基础层（MPQ 归档，必需）
- `res-patch.rep`：覆盖层（可选）
- `res-manifest.json`：外置清单（补丁版本与墓碑）
- `resources/`：本地散文件层

查找顺序为 **本地散文件 > patch > base**，第一个命中生效；读取结果
进入读缓存（LRU，缺省 256 MiB），同一文件只解压一次。本地未解包
的文件（如各场景 `map/ter/*/terrain.ini`）会直接从 `res-base.rep`
提供，无需完整解包客户端资源。

命令行工具（`cmd/mpq`）支持查看 / 解包 / 打包 / 增删：

```powershell
go run ./cmd/mpq store info .          # 三层仓概览
go run ./cmd/mpq store list . prefix   # 按前缀列文件（含所在层）
go run ./cmd/mpq store read . <名字>   # 读单个文件并预览
```

环境变量（`cmd/protocol-probe`）：

- `NINEYIN_RESOURCE_STORE=off`：禁用资源仓，回退直接文件访问（排查用）
- `NINEYIN_RESOURCE_CACHE_MB=<n>`：读缓存上限（MiB，缺省 256）

## 许可证

- 本仓库的初始代码来源自论坛，其真实作者未知，亦未声明许可证。故本仓库自有代码（`cmd/`、`internal/(!ecs)`、及根工程配置）以
  [BSD 3-Clause](LICENSE.txt) 许可发布。
- `internal/ecs/` 目录为内嵌的 [unitoftime/ecs](https://github.com/unitoftime/ecs)
  框架源码，保持上游原样、不做修改，遵循其上游 MIT License 许可证
  （随附原文件 [internal/ecs/LICENSE](internal/ecs/LICENSE)，
  不继承根工程的 BSD-3）。
- `third_party/golua/` 目录为内嵌的 [iceisfun/golua](https://github.com/iceisfun/golua)
  v2（Lua 5.5 运行时）源码，遵循其上游 MIT License 许可证（随附原文件
  [third_party/golua/LICENSE](third_party/golua/LICENSE)）。相对上游仅打了
  Windows 进程组编译兼容补丁，差异与来源见
  [third_party/golua/FORK.md](third_party/golua/FORK.md)，不继承根工程的 BSD-3。

  
<strong>注意：本项目仅限于研究，严禁用于商用。游戏版权归蜗牛游戏公司所有。</strong>