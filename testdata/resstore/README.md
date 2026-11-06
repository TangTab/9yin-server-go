# 测试资源仓夹具（res-base.rep）

真实资源（仓库根的 `res-base.rep` / `resources/`）被移走时的测试替身：
`cmd/protocol-probe` 的资源访问器在主资源仓打不开时会回退到这里
（日志可见 `resource fixture store in use`），`go test ./...` 无需
真实数据即可全绿。`NINEYIN_RESOURCE_FIXTURE=off` 可强制禁用该回退。

- 生成日期：2026-11-06
- 数据来源：`G:\pojie\ai\9yin-go-server1\resources`（官方客户端解包数据，
  与 `F:\Game\Online\9yin\9yin_server1\resources` 同源，skill_new.ini 同为
  1,725,320 字节 / 2026-09-05）
- 覆盖范围：**仅测试触及的子集**（137 个文件，原始约 113 MB，MPQ 压缩后
  约 19 MB）——技能全量表、buff/内功表、动作表、item/modifypack、
  trade/rule、`npc/npcconfig` 整树（walk 根）以及 city05_chengdu /
  born03_yanyuzhuang / school31_shenjihui 三个场景的 creator XML 与
  mappath 巡逻表。**不含**全量服务端数据：真实服跑起来仍需完整资源。

## 重新生成（资源集变化时）

1. 让测试指向真实资源并记录访问轨迹（`NINEYIN_RESOURCE_TRACE`）：

   ```
   NINEYIN_SERVER_ROOT=<真实资源根> NINEYIN_RESOURCE_TRACE=trace.tsv \
     go test ./cmd/protocol-probe/
   ```

2. 汇总轨迹：`read/open/stat` 取文件本身；`readdir/walk` 的目录把
   **整棵子树**收进来（目录枚举结果会喂给名字筛选逻辑，只放触及过的
   文件会改变枚举语义）。
3. 按原路径 staging 到 `<stage>/resources/...` 后打包：

   ```
   go run ./cmd/mpq pack <stage> testdata/resstore/res-base.rep
   ```

注意：夹具是二进制 MPQ，diff 不可读；范围变化请重新走一遍上面的
轨迹流程，而不是手工往里塞文件。
