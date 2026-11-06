# MPQ 资源打包与三层读取 —— 模块权威记录

> 2026-11-03 整理。模块：`tools/mpq`（artifactId `nineyin-mpq`，包 `com.nineyin.tools.mpq`），
> 移植自本地 Rust mpq 库（`E:\Workspace\project\Rust\mpq\mpq`），仅依赖 JDK + fastjson2。
> 测试基准目录：`E:\Workspace\project\game\9yin_server1\`（Go 服务端部署目录）。

## 1. 概述

资源以 War3 式 MPQ 补丁链存储，读取固定三层，第一个命中生效：

```
本地散文件（resources/…）  >  res-patch.rep（覆盖层）  >  res-base.rep（全量基础层）
                                              …墓碑（manifest deleted）三层皆 miss
```

选择 MPQ 的原因：War3 验证过的分层覆盖模型、单文件分发、zlib 压缩 3.1×、
按名随机查找 O(1)。写侧与读取同一套 Java 实现，另有 Rust 原库做跨语言对拍。

## 2. 目录布局（资源根）

```
<根>/
  res-base.rep        全量基础层（必需）
  res-patch.rep       覆盖层（可选；只放「与 base 不同 or 新增」的文件）
  res-manifest.json   外置清单（可选；缺失时退化为 base + 本地层）
  resources/…         本地散文件（调试/热修覆盖用；生产应为空）
```

归档内文件名统一带 `resources\` 前缀、`\` 分隔（如 `resources\modern\ini\npc\monster312.ini`）；
查找不分大小写、`/` 与 `\` 等价。

## 3. res-manifest.json 格式

```json
{
  "base":  { "file": "res-base.rep",  "version": "0001" },
  "patch": { "file": "res-patch.rep", "version": "0003" },
  "local": "resources",
  "deleted": ["resources\\modern\\map\\ini\\xxx.ini"]
}
```

- `patch` 段可缺省（无补丁层）；`local` 缺省为 `resources`；整份清单可缺省（首次部署零配置）。
- `version`：人读版本号，供启动校验与热重载比对。
- `deleted`：**墓碑**。MPQ 覆盖链只能"顶掉"不能"删掉"，删除靠清单表达；名字统一小写规范化存储。

## 4. CLI 手册

shaded 包：`tools/mpq/target/nineyin-mpq-0.1.0-SNAPSHOT.jar`（`mvnw -pl tools/mpq package`）。

```
mpq info    <归档>                       头部与条目概览
mpq list    <归档>                       列出文件（大小/压缩后/标志/名字）
mpq extract <归档> [destDir] [名字…]     解包（跳过 (listfile)）
mpq pack    <目录> <out.rep> [-v 1-4]    目录递归打包（默认 v1；base/patch 均用它压）
mpq add     <归档> <本地文件=归档名>…    添加/替换（写回原归档）
mpq remove  <归档> <名字>…              删除（写回原归档）
mpq convert <归档> <out.rep> -v 1-4     头版本转换
mpq store   info|list|read <资源根>…    三层资源仓
```

`store` 子命令：
- `store info <根>` —— 三层条目数、版本、合并清单大小、告警
- `store list <根> [前缀]` —— 合并视图（层/大小/名字）
- `store read <根> <名字>` —— 显示命中层 + 预览

## 5. Java API

```java
// 只读单个归档
try (MpqArchive a = MpqArchive.open(Path.of("res-base.rep"))) {
    List<FileEntry> entries = a.entries();      // 文件列表（含名字/大小/标志）
    byte[] data = a.readFile("resources\\a.ini");
}

// 三层资源仓（推荐入口）
try (ResStore store = ResStore.open(Path.of("9yin_server1"))) {
    String layer = store.layer("resources/a.ini");   // local | patch | base | null
    byte[] data  = store.read("resources/a.ini");    // 三层自动解析 + 墓碑
    List<ResStore.ResFile> all = store.list();       // 合并清单
    store.reload();                                   // 热重载（换 patch 后原子换句柄）
}

// 写：创建 / 修改（保留文件字节原样复制，仅重写容器）
byte[] empty = MpqWriter.create(MpqWriter.CreateOptions.ofVersion(V1));
var result = MpqWriter.modify(empty, List.of(),
        new LinkedHashMap<>(Map.of("resources\\a.ini", bytes)));   // 增/替换
MpqWriter.modify(data, List.of(new MpqWriter.Deletion.ByName("x")), map);  // 删
MpqWriter.convert(data, V4);
```

## 6. 实测性能（2026-11-03，9yin_server1，52926 文件 / 808MB）

| 场景 | 散文件直读 | MPQ | 结论 |
| --- | --- | --- | --- |
| 全量扫描（启动） | 1.6s（491 MB/s） | 4.4s（184 MB/s） | 一次性多 ~2.8s |
| 随机单读（热缓存） | 35 µs/次 | 77 µs/次（解压占 65µs） | 每次多 ~41µs |
| 名字查找 | 需开句柄 ~30µs | 0.6 µs | MPQ 更快 |
| 磁盘/分发体积 | 808 MB | 259 MB（3.1×） | 冷缓存/网络分发 MPQ 占优 |

服务端"启动读入 + 运行时偶读"模式下影响可忽略；高频读可在 ResStore 上加进程内缓存。

## 7. 运维流程

1. **压 base**：`mpq pack resources res-base.rep -v 1`（v1 容器最小；v4 带全量 MD5 校验和）。
2. **发 patch**：改动文件按归档名组织后 `mpq pack` 成 res-patch.rep，version 递增；
   删除的文件写进 manifest `deleted`。
3. **校验**：`mpq store info` 看层数；全量内容指纹用
   `VerifyAll`（Java）/ `cargo run --release --example verify_rep`（Rust），两边指纹一致 = 零差异。
4. **rebase**（patch 超过 base 的 ~20-30% 时）：解包 patch → 与 base 合并 → 重压新 base →
   version 归档保留供回滚（manifest 指回旧版即回滚）。
5. 生产形态下 `resources/` 本地层应清空——本地文件会静默遮蔽 patch/base，造成"部署了补丁不生效"。

## 8. 与 Rust 原库的差异（有意为之）

- 不支持 bzip2（0x10）/ LZMA（0x12）压缩方法，遇到明确报错（自产归档走 zlib，无影响）。
- 不做缺失 (listfile) 时的名字恢复启发式（魔兽清单）；不解析 (attributes)。
- 不支持 Rust 库自定义的密码 AEAD 扩展（遇保护归档明确报错，需先在 Rust 端 unprotect）。

## 9. 验证状态（2026-11-03）

- JUnit 26/26：Storm/Jenkins/PKWARE 参考向量、v1-v4 往返、三层覆盖/墓碑/热重载/路径穿越。
- 跨语言对拍：Java 读 Rust 生成归档（含加密/key-v2/sector CRC/PKWARE）逐文件 sha256 一致；
  Rust 读 Java 写出归档 6/6 一致；259MB 真实归档全量 52928 文件双方聚合 sha256 一致、0 失败 0 告警。

## 10. 已知坑与注意

- **名字大小写**：所有比较（含 manifest 墓碑）必须走规范化小写；曾因此修过两个 bug。
- **路径**：ResStore 本地层已防 `..` 穿越；归档名统一 `resources\` 前缀。
- **测量陷阱**：磁盘上有后台进程（杀毒/同步）时随机读数据会被严重污染（曾测出 2.26ms/次
  的假值，复测 77µs）；基准工具 `ResBench` / `ResBenchProfile` 在 test 源码目录，换机器可复测。
- 9yin_server1 当前为演示状态：res-patch.rep 含 monster312.ini 覆盖 + patchdemo\hello.txt 新增，
  manifest 墓碑指向 `resources\modern\map\ini\weather\...\pppixelrefraction.ini`；
  因 base 就是从当前 resources/ 树压出，本地层全量遮蔽 base（合并 52927 = local 52926 + patch 1）。

## 11. 源码地图

| 文件 | 职责 |
| --- | --- |
| `tools/mpq/…/MpqArchive.java` | 读：头部定位/经典表+HET/BET/查找/sector 读取/CRC-MD5 告警 |
| `tools/mpq/…/MpqWriter.java` | 写：create/modify（原样复制+重建表+自检）/convert/key-v2 重加密 |
| `tools/mpq/…/MpqMain.java` | CLI（含 store 子命令） |
| `tools/mpq/…/res/ResStore.java` | 三层 VFS（本地>patch>base、墓碑、热重载、防穿越） |
| `tools/mpq/…/res/ResManifest.java` | res-manifest.json 解析/序列化 |
| `tools/mpq/…/StormCrypt.java` 等 | 算法层：storm 加密、Jenkins、PKWARE、zlib、头部/表编解码 |
| Rust 端 `examples/gen_java_fixtures.rs`、`verify_java_archives.rs`、`verify_rep.rs` | 对拍夹具生成 / 反向验证 / 全量指纹 |
