# MPQ 资源包与多层资源仓 —— 模块权威记录

> 2026-11-24 定版（Go 实现）：`internal/mpq`（读写与算法层）+ `internal/mpq/res`
> （多层资源仓、读缓存、路径式访问入口、层构建）+ `cmd/mpq`（CLI）。
> 格式与算法移植自 Java / Rust 参考实现，起源与实测见 §9。

## 1. 模型

数据根目录下的资源有两处来源：**层文件**（MPQ 归档）与**散文件**。读取顺序固定，
第一个命中生效：

```
根下散文件  >  <前缀>N.rep  >  <前缀>N-1.rep  >  …  >  <前缀>001.rep
```

- 层文件名 = 前缀 + 序号 + `.rep`，**序号即层号，越大优先级越高**。

  | 前缀 | 用途 | 例 |
  |---|---|---|
  | `b` | 仓内测试数据根（`testdata/`） | `b001.rep` `b002.rep` … |
  | `data` | 正式服务端数据根（部署的 `Res/`） | `data001.rep` `data002.rep` … |
  | `h` | 参考资源根（`testdata/references/`，黑金原档等，只读） | `h001.rep` `h002.rep` … |

- **数据根可以嵌套**：路径按离自己最近的数据根解析。`testdata/references/` 自带
  `h001.rep`，是嵌套在 `testdata/` 里的另一个数据根——两边各有同名的
  `ini\Item\…` 时各读各的；内层的层容器不会被当成外层的散文件收进 `b` 层。

- **资源名就是数据根下的相对路径**（`ini\Scene\born03\npc_creator.xml`），与散文件
  布局完全一致、不分大小写、`/` 与 `\` 等价。所以同一套相对路径在"纯散文件"与
  "打包后"两种数据根下都能用。
- 层内可登记**删除**（内部文件 `(deleted)`，每行一个名字）：某层登记的名字对其
  以下各层视为不存在；更高层重新收录、或根下出现同名散文件，即恢复。MPQ 覆盖链
  只能"顶掉"不能"删掉"，删除靠它表达。
- 生效时机：**散文件实时生效**（改完即见，不进任何缓存）；**层文件在 Reload /
  重新打开之后生效**。

## 2. 目录布局

```
<数据根>/
  b001.rep b002.rep …        仓内 testdata（或 data001.rep … 正式 Res）
  mpq/ …                     保持散文件：mpq 包自身的测试夹具
  references/h001.rep        参考资源根：黑金原档全量（嵌套的数据根，只读）
  README.md                  保持散文件：给人看的目录说明
  <其余散文件>               覆盖层：调试 / 热修 / 刚构建待验证的产物
```

testdata 首个层 `b001.rep`：1438 个文件，散文件形式 286 MiB → 59.7 MiB（约 4.8×）。
参考资源根 `references/h001.rep`：黑金服务端 Res 全量，6489 个文件，35.7 MiB。

## 3. CLI 手册

`go run ./cmd/mpq`（或 `build.bat` 产物 `game_mpq.exe`）：

```
mpq info    <mpq>                     头部与条目概览
mpq list    <mpq>                     列出文件
mpq extract <mpq> [destDir] [名字…]   解包单个归档
mpq pack    <dir> <out.mpq> [-v 1-4]  目录递归打包为单个归档（不建层）
mpq add / remove / convert            单归档的增删与头版本转换

mpq store info                  <数据根>            层清单与合并视图统计
mpq store list                  <数据根> [前缀]     合并视图：所在层 / 大小 / 名字
mpq store read                  <数据根> <名字>     命中层与内容预览
mpq store pack                  <数据根> [选项]     散文件 → 新层
mpq store unpack                <数据根> [目标目录] [前缀]   合并视图解包成散文件树
mpq store delete                <数据根> <名字>…    在最高层登记删除
```

`store pack` 选项：

| 选项 | 含义 |
|---|---|
| `-prefix b\|data\|h` | 层文件族。建首个层必填；多族并存时必填 |
| `-layer N` | 目标层号。缺省 = 现有最大层号 + 1 |
| `-replace` | 允许整层重建已存在的目标层（重建后该层只含本次源提供的内容） |
| `-from <目录>` | 源目录。缺省 = 数据根自身的散文件 |
| `-exclude <规则>` | 排除规则（`/` 分隔 glob，可重复）；命中文件本身或其任一级上级目录即排除 |
| `-v 1-4` | MPQ 格式版本，缺省 4（带全量 MD5 校验和） |
| `-full` | 关闭去重 |
| `-delete-missing` | 源中缺失而下层存在的名字在本层登记删除（源须是完整树） |
| `-prune` | 成功并回读校验后，删除已并入的源散文件 |

层容器文件（`bNNN.rep` / `dataNNN.rep` / `hNNN.rep`）本身不是资源，不会被收进任何层。

## 4. 构建与更新流程

```bash
# 首个层：把 testdata 的数据打成 b001（测试夹具与目录说明保持散文件；
# references/h001.rep 是层容器，天然不会被收进来）
go run ./cmd/mpq store pack testdata -prefix b -exclude mpq -exclude README.md

# 增量层：改几个散文件后——只收与下层内容不同的文件，得到增量层；-prune 清掉已并入的散文件
go run ./cmd/mpq store pack testdata -prefix b -prune -exclude mpq -exclude README.md

# 服务端数据（NPC / 任务 / 物品 / 技能）先重建为散文件，验证后再走上面的增量层
go run ./tools/datagen

# 正式服：从完整资源树建 / 更新
go run ./cmd/mpq store pack <Res> -prefix data -from <新版完整树> -delete-missing

# 看当前生效内容、解出整棵树、登记删除
go run ./cmd/mpq store list  <数据根> ini/Scene
go run ./cmd/mpq store read  <数据根> ini/Scene/born03/npc_creator.xml
go run ./cmd/mpq store unpack <数据根>
go run ./cmd/mpq store delete <数据根> ini/old.xml
```

要点：

- **去重**是缺省行为：与下层逐字节相同的文件不进新层。所以"根下散文件 → 下一层"
  天然得到增量层；没有任何下层时即全量层。没有可写内容时不落新文件（MPQ 归档
  不能为空），并报告"无变化"。
- **`-prune` 的安全性**：层文件先写临时文件、重新打开逐个回读比对、再原子替换，
  全部通过后才删除散文件。被 `-exclude` 挡掉的文件不动。
- **整层重建**要显式 `-replace`：用不完整的源重建会把该层冲成只剩源里的内容。
  重建保留本层原有的删除名单。调用方自己持有的 `res.Store` 须先关闭（见 §7）。
- **删除**只能登记进最高层；根下若还有同名散文件，CLI 会警告——散文件盖过墓碑，
  须先移走。
- 正式服的更新形态是"新增更高层号的层"（`data002.rep`…），低层不动。

## 5. Go API

```go
// 资源仓：名字相对数据根，\ 或 / 分隔，不分大小写
store, _ := res.Open("testdata")                  // 层文件族自动探测
data, _ := store.ReadFile("ini/Scene/born03/npc_creator.xml")
layer := store.Layer("ini/Scene/born03/npc_creator.xml") // "local" / "b001" …
files, _ := store.List()                          // 合并视图，按名字升序
store.Reload()                                    // 层文件变化后重新打开
store.Close()

// 带读缓存（服务端常驻）：层内文件的字节与解析结果各缓存一份
cached, _ := res.OpenCached(dir, res.CacheOptions{})
fsys := cached.FS()                               // fs.FS 视图：fs.ReadFile / fs.WalkDir / fs.ReadDir

// 构建与删除
res.BuildLayer(dir, res.BuildOptions{Prefix: res.PrefixTest, Prune: true, Exclude: []string{"mpq"}})
res.Delete(dir, res.Options{}, "ini/old.xml")
```

## 6. 读取侧接入：路径式访问入口

`res` 包提供 `os.*` / `filepath.*` 的等价替换。调用方继续传主机路径，入口自己判断
路径是否落在数据根之下（自身或任一级祖先目录里有层文件）：是则走资源仓，否则
直读文件系统。**调用方不需要知道数据根在哪、有没有打包。**

| 入口 | 等价于 |
|---|---|
| `res.ReadFile(path)` | `os.ReadFile`——返回的切片归调用方所有 |
| `res.ReadFileShared(path)` | 同上，但返回的可能是读缓存内部对象，**只能读不能改**；用于对同一大文件反复取小片段（免整文件拷贝） |
| `res.Stat` / `res.Lstat` | `os.Stat` / `os.Lstat` |
| `res.ReadDir(dir)` | `os.ReadDir`——按名字排序，条目名保留原样大小写 |
| `res.WalkDir(root, fn)` | `filepath.WalkDir`——SkipDir / SkipAll 语义相同 |
| `res.Glob(pattern)` | `filepath.Glob` |

规则与注意：

- **读数据根（`testdata`、部署的 `Res`、`ResourcePath` 之类）内容的代码一律走
  `res.*`**，包括存在性检查（`os.Stat`）与目录枚举。漏掉的位置在散文件被并入层
  之后就找不到文件了。
- 错误与 `os` 同形：`*fs.PathError` 带主机路径，`errors.Is(err, fs.ErrNotExist)` 与
  `os.IsNotExist(err)` 都可用。
- 散文件实时可见（枚举也会即时并入新落盘的散文件）；层文件被替换后，同进程内要
  `res.ResetPathCache()` 才重新打开。`BuildLayer` / `Delete` 已内置。
- 层内文件没有有意义的时间戳：`Stat` 的 `ModTime` 为零值。依赖 mtime 判断变更
  的逻辑对层内文件等于"永远未变"。
- 同一数据根在进程内只打开一次（带 256 MiB 字节缓存缺省）；测试里用完调
  `res.ResetPathCache()` 释放句柄，否则 Windows 上临时目录清理不掉。

## 7. 已知限制

- **Windows 文件占用**：已打开的层文件无法被替换。`store pack -replace` 报
  "替换失败（文件被占用？）"即此——先停掉占用进程（或关闭自己持有的
  `res.Store`）。层文件被原地改写时，已打开的句柄会直接读到新字节（可能撕裂），
  不是受支持的更新方式。
- **单层上限 4 GB**：写出侧用 32 位偏移。超出须拆层。
- 归档只支持 zlib（及读取 PKWARE implode）；bzip2 / LZMA 压缩方法与 Rust 库的
  密码 AEAD 扩展不支持，遇到明确报错。
- 同一目录下**仅 ASCII 大小写不同**的同名文件无法共存（MPQ 按名寻址不分大小写），
  构建时报错。
- 同一数据根下并存多族层文件（`b…`、`data…`、`h…` 里不止一族）是配置错误，打开
  报错，须显式指定前缀。
- 内部文件名（`(listfile)` `(deleted)` 及 MPQ 保留名）不是资源，读不到。

## 8. 验证状态（2026-11-24）

- `internal/mpq`：Storm / Jenkins / PKWARE 参考向量，v1–v4 往返，加密与 sector CRC。
- `internal/mpq/res`：41 项——多层覆盖顺序、登记删除与恢复、层号规则（数值序而非
  字典序、补零与否等价、重复 / 零号 / 损坏 / 多族并存的拒绝）、热重载、构建的去重 /
  修剪 / 重建 / 删除登记 / 解包、参考资源族与嵌套数据根（外层 `b001` 内嵌
  `references/h001`，同名文件各读各的），以及路径式入口与 `os.*` 的逐项一致性对比
  （同一棵树的散文件版与打包版，对文件 / 目录 / 遍历 / glob 的结果必须相同）。
- `testdata/b001.rep` 经底层 `extract` 解出后与原散文件逐字节一致。

## 9. 历史：Java / Rust 起源（2026-11-03）

模块原为 Java（`tools/mpq`，artifactId `nineyin-mpq`），移植自本地 Rust mpq 库
（`E:\Workspace\project\Rust\mpq\mpq`），仅依赖 JDK + fastjson2。当时的布局是
清单制三层（`res-base.rep` / `res-patch.rep` / `res-manifest.json`，归档名带
`resources\` 前缀）；2026-11-24 改为上文的编号多层模型，清单取消，删除名单移入
层内 `(deleted)`。

选择 MPQ 的原因：War3 验证过的分层覆盖模型、单文件分发、按名随机查找 O(1)。

实测（2026-11-03，52926 文件 / 808 MB）：

| 场景 | 散文件直读 | MPQ | 结论 |
| --- | --- | --- | --- |
| 全量扫描（启动） | 1.6s（491 MB/s） | 4.4s（184 MB/s） | 一次性多 ~2.8s |
| 随机单读（热缓存） | 35 µs/次 | 77 µs/次（解压占 65µs） | 每次多 ~41µs |
| 名字查找 | 需开句柄 ~30µs | 0.6 µs | MPQ 更快 |
| 磁盘/分发体积 | 808 MB | 259 MB（3.1×） | 冷缓存/网络分发 MPQ 占优 |

服务端"启动读入 + 运行时偶读"模式下影响可忽略。跨语言对拍：Java 读 Rust 生成
归档逐文件 sha256 一致；259MB 真实归档全量 52928 文件双方聚合 sha256 一致。

| 文件（Go） | 职责 |
| --- | --- |
| `internal/mpq/archive.go` | 读：头部定位 / 经典表 + HET/BET / 按名查找 / sector 读取 / CRC-MD5 告警 |
| `internal/mpq/writer.go` | 写：create / modify（原样复制 + 重建表 + 自检）/ convert |
| `internal/mpq/storm.go` 等 | 算法层：storm 加密、Jenkins、PKWARE、zlib、头部 / 表编解码 |
| `internal/mpq/res/store.go` | 多层资源仓：散文件 > 高层 > 低层、登记删除、热重载 |
| `internal/mpq/res/build.go` | 层构建（去重 / 修剪 / 重建 / 登记删除）、解包 |
| `internal/mpq/res/path.go` | 路径式访问入口与数据根发现 |
| `internal/mpq/res/fsfs.go` | `fs.FS` 视图与名称索引 |
| `internal/mpq/res/cache.go` | 读缓存（层内文件缓存，散文件直读） |
| `cmd/mpq/store.go` | CLI `store` 子命令 |
