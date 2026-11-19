# 九阴服务端

## 运行

## 编译


编译产物写入 `build`，不会自动覆盖根目录中正在测试的服务端 EXE。

```sh
# 全部进程入口，产物统一 game_<包名>.exe 前缀（与 build.bat 一致）
for d in cmd/*/; do go build -o "build/game_$(basename "$d").exe" "./$d"; done
go generate ./...              # structgen / protogen 代码生成
go test ./...                  # 全量测试（含 golden 对拍）
```

本地无 make 时直接使用上述 go 命令；Makefile 仅作等价包装。

### 直读原版数据（可选）

structgen / protogen 默认读仓库内 `testdata/` 快照（无外部依赖、构建可复现）。
设置环境变量后**直读原版目录，不经过拷贝**：

```sh
export NINEYIN_DATA="E:\Workspace\project\game\九阴真经ol\BGserverCJ"
go generate ./...   # protogen → $NINEYIN_DATA/Res/lua/share；structgen → $NINEYIN_DATA/Res/struct
```

优先级：`-in` 显式参数 > `NINEYIN_DATA` > `testdata/` 快照。
（运行时 script 层同理：solo 的 `-scripts` 参数直接挂原版 `Res\lua`，只读。）

## 源码目录

```
cmd/                    进程入口（对应原版 game_*.exe）
  solo/                 单进程整合服（第一里程碑，FsSoloLoader 等价）
  entry/ lister/        登录网关 / 区服列表（第二里程碑）
  member/ world/ public/ 角色服务 / 场景服 / 公共数据（第二里程碑）
  probe/                协议探测与对拍客户端（oracle 基础设施）
internal/
  ecs/                  sparse-set ECS 内核（零依赖，对象模型基础）
  engine/               引擎壳：配置驱动启动、tick 循环、System 装配
  component/            structgen 生成产物（只放生成代码，禁止手改）
  prop/                 属性运行时：Record 表格、行操作日志
  sync/                 可视性/脏同步（public/private/realtime → 协议推送）
  aoi/                  九宫格 AOI、进出视野广播
  scene/                场景生命周期（含 Clonable 副本语义）
  net/                  TCP、封包帧、编码/校验/加密
  proto/                消息号 + 序列化（protogen 产物 + 手写部分）
  script/               Lua 行为层（gopher-lua，nx_* API 句柄化）
  config/               Object/Property XML、struct XML 加载（GBK）
  persist/              MySQL 存取队列、存档 blob 序列化
  proc/                 进程拓扑：Bus 接口（inproc ↔ TCP 可换）
  login/                登录→选角→进场景→移动 主干（首批 System）
tools/
  structgen/            struct XML → 组件/元数据 生成器
  protogen/             消息规格 → 序列化代码 生成器
testdata/               规格输入与对拍资产（见 testdata/README.md）
build/                  编译产物（gitignore）
```

各包职责与设计边界的详细说明见各包 `doc.go`。

## 任务与进度

里程碑清单、已完成工作明细、关键决策记录与下一步计划见 [docs/任务与进度.md](docs/任务与进度.md)。

许可证
本仓库的自有代码（cmd/、internal/、tools、及根工程配置）以 BSD 3-Clause 许可发布
注意：本项目仅限于研究，严禁商用。